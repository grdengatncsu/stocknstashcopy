package database

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/jae-white/stock-n-stash/server/models"
)

// IsScanProcessed reports whether the server has already committed a scan ID.
func IsScanProcessed(db *sql.DB, scanID string) (bool, error) {
	// SELECT EXISTS returns a single boolean without loading the matching row.
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM processed_scans WHERE scan_id = ?
		)
	`, scanID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// MarkScanProcessed records a scan ID so later submissions can be recognized
// as retries. ProcessScanReport normally performs this inside its transaction.
func MarkScanProcessed(db *sql.DB, scanID string, processedAt string) error {
	_, err := db.Exec(`
		INSERT INTO processed_scans (scan_id, processed_at)
		VALUES (?, ?)
	`, scanID, processedAt)
	return err
}

// ProcessScanReport routes accepted items to inventory and uncertain items to
// pending review. The returned bool is true when this scan was handled earlier.
func ProcessScanReport(db *sql.DB, report models.ScanReport, processedAt string) (bool, error) {
	// All items and the processed marker must commit as one unit. Otherwise, a
	// retry after a partial failure could create duplicate inventory records.
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	// Rollback is a no-op after a successful Commit and covers all error returns.
	defer tx.Rollback()

	var exists bool

	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM processed_scans WHERE scan_id = ?
		)
	`, report.ScanID).Scan(&exists)

	if err != nil {
		return false, err
	}

	if exists {
		// A repeated scan is a successful no-op. This makes client retries safe.
		return true, nil
	}

	for _, item := range report.Items {
		if item.RequiresReview {
			// Preserve the model's guess for a person to confirm or correct.
			pendingID := uuid.NewString()
			_, err = tx.Exec(`
				INSERT INTO pending_results (id, scan_id, association_id, suggested_name, suggested_upc, confidence, quantity, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, pendingID, report.ScanID, item.AssociationID, item.Name, item.UPC, item.Confidence, item.Quantity, processedAt, processedAt)
			if err != nil {
				return false, err
			}
		} else {
			// High-confidence items can enter inventory immediately.
			inventoryID := uuid.NewString()
			_, err = tx.Exec(`
				INSERT INTO inventory_items (id, name, upc, quantity, expiration_date, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, inventoryID, item.Name, item.UPC, item.Quantity, nil, processedAt, processedAt)
			if err != nil {
				return false, err
			}
		}
	}

	// Write this marker last: its presence means every item above was stored.
	_, err = tx.Exec(`
		INSERT INTO processed_scans (scan_id, processed_at)
		VALUES (?, ?)
	`, report.ScanID, processedAt)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	return false, nil
}
