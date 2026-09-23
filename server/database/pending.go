package database

import (
	"database/sql"

	"github.com/jae-white/stock-n-stash/server/models"
)

// AddPendingResult stores a scan result that requires human review.
func AddPendingResult(db *sql.DB, result models.PendingResult) error {
	updatedAt := result.UpdatedAt
	if updatedAt == "" {
		updatedAt = result.CreatedAt
	}
	_, err := db.Exec(`
		INSERT INTO pending_results (id, scan_id, association_id, suggested_name, suggested_upc, confidence, quantity, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, result.ID, result.ScanID, result.AssociationID, result.SuggestedName, result.SuggestedUPC, result.Confidence, result.Quantity, result.CreatedAt, updatedAt)

	return err
}

// GetPendingResults returns every unresolved result for the review screen.
func GetPendingResults(db *sql.DB) ([]models.PendingResult, error) {
	rows, err := db.Query(`
		SELECT id, scan_id, association_id, suggested_name, suggested_upc, confidence, quantity, version, created_at, updated_at, deleted_at
		FROM pending_results
		WHERE deleted_at IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]models.PendingResult, 0)
	for rows.Next() {
		var result models.PendingResult
		if err := rows.Scan(&result.ID, &result.ScanID, &result.AssociationID, &result.SuggestedName, &result.SuggestedUPC, &result.Confidence, &result.Quantity, &result.Version, &result.CreatedAt, &result.UpdatedAt, &result.DeletedAt); err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// GetPendingResultByID returns one unresolved result or sql.ErrNoRows.
func GetPendingResultByID(db *sql.DB, id string) (models.PendingResult, error) {
	row := db.QueryRow(`
		SELECT id, scan_id, association_id, suggested_name, suggested_upc, confidence, quantity, version, created_at, updated_at, deleted_at
		FROM pending_results
		WHERE id = ? AND deleted_at IS NULL
	`, id)

	var result models.PendingResult
	if err := row.Scan(&result.ID, &result.ScanID, &result.AssociationID, &result.SuggestedName, &result.SuggestedUPC, &result.Confidence, &result.Quantity, &result.Version, &result.CreatedAt, &result.UpdatedAt, &result.DeletedAt); err != nil {
		return result, err
	}

	return result, nil
}

// DeletePendingResult removes a result after it no longer needs review.
func DeletePendingResult(db *sql.DB, id string) error {
	result, err := db.Exec(`
		DELETE FROM pending_results
		WHERE id = ?
	`, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ResolvePendingResult atomically promotes a reviewed result to inventory.
// Both database changes succeed together, or neither is kept.
func ResolvePendingResult(db *sql.DB, pendingID string, expectedVersion int, item models.InventoryItem) error {
	// The transaction prevents a failure between INSERT and DELETE from leaving
	// the same physical item in both tables.
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	// Rollback is harmless after Commit and protects every early-return path.
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE pending_results
		SET deleted_at = ?, updated_at = ?, version = version + 1
		WHERE id = ? AND version = ? AND deleted_at IS NULL
	`, item.UpdatedAt, item.UpdatedAt, pendingID, expectedVersion)
	if err != nil {
		return err
	}

	// DELETE does not fail when nothing matches, so inspect the affected count.
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrVersionConflict
	}

	_, err = tx.Exec(`
		INSERT INTO inventory_items (id, name, upc, quantity, expiration_date, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.Name, item.UPC, item.Quantity, item.ExpirationDate, item.Version, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return err
	}

	// Commit makes both changes visible to other database users.
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
