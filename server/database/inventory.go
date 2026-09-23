package database

import (
	"database/sql"
	"errors"

	"github.com/jae-white/stock-n-stash/server/models"
)

// AddInventoryItem inserts one confirmed product into persistent inventory.
func AddInventoryItem(db *sql.DB, item models.InventoryItem) error {
	_, err := db.Exec(`
		INSERT INTO inventory_items (id, name, upc, quantity, expiration_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.Name, item.UPC, item.Quantity, item.ExpirationDate, item.CreatedAt, item.UpdatedAt)
	return err
}

// GetInventoryItems returns all confirmed products. The empty result is an
// allocated slice so the JSON response is [] rather than null.
func GetInventoryItems(db *sql.DB) ([]models.InventoryItem, error) {
	rows, err := db.Query(`
		SELECT id, name, upc, quantity, expiration_date, version, created_at, updated_at, deleted_at
		FROM inventory_items
		WHERE deleted_at IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.InventoryItem, 0)
	// rows.Next advances one database record at a time; Scan copies its columns
	// into the matching Go fields in the same order as the SELECT statement.
	for rows.Next() {
		var item models.InventoryItem
		if err := rows.Scan(&item.ID, &item.Name, &item.UPC, &item.Quantity, &item.ExpirationDate, &item.Version, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

var ErrVersionConflict = errors.New("version conflict")

// UpdateInventoryItem replaces editable fields while preserving the original
// ID and creation timestamp.
func UpdateInventoryItem(db *sql.DB, item models.InventoryItem) error {
	result, err := db.Exec(`
		UPDATE inventory_items
		SET name = ?, upc = ?, quantity = ?, expiration_date = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND version = ? AND deleted_at IS NULL
	`, item.Name, item.UPC, item.Quantity, item.ExpirationDate, item.UpdatedAt, item.ID, item.Version)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// SQL UPDATE is not an error when its WHERE clause matches no rows, so turn
	// that case into an explicit error for callers.
	if rowsAffected == 0 {
		return ErrVersionConflict
	}

	return nil
}

// GetInventoryItemByID returns one confirmed product or sql.ErrNoRows when the
// requested ID does not exist.
func GetInventoryItemByID(db *sql.DB, id string) (models.InventoryItem, error) {
	row := db.QueryRow(`
		SELECT id, name, upc, quantity, expiration_date, version, created_at, updated_at, deleted_at
		FROM inventory_items
		WHERE id = ? AND deleted_at IS NULL
	`, id)

	var item models.InventoryItem
	if err := row.Scan(&item.ID, &item.Name, &item.UPC, &item.Quantity, &item.ExpirationDate, &item.Version, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt); err != nil {
		return models.InventoryItem{}, err
	}
	return item, nil
}

// DeleteInventoryItem removes one product and reports a missing ID as an error.
func DeleteInventoryItem(db *sql.DB, id string, expectedVersion int, deletedAt string) error {
	result, err := db.Exec(`
		UPDATE inventory_items
		SET deleted_at = ?, updated_at = ?, version = version + 1
		WHERE id = ? AND version = ? AND deleted_at IS NULL
	`, deletedAt, deletedAt, id, expectedVersion)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrVersionConflict
	}

	return nil
}
