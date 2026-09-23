package models

// InventoryItem is a confirmed product stored in the user's inventory.
// Pointer fields become JSON null when optional information is unavailable.
type InventoryItem struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	UPC            *string `json:"upc"`
	Quantity       int     `json:"quantity"`
	ExpirationDate *string `json:"expiration_date"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	Version        int     `json:"version"`
	DeletedAt      *string `json:"deleted_at"`
}

// PendingResult is a low-confidence scan result waiting for human review.
// Suggested fields preserve the model's best guess so the reviewer can accept
// it as-is or replace it while resolving the result.
type PendingResult struct {
	ID            string  `json:"id"`
	ScanID        string  `json:"scan_id"`
	AssociationID string  `json:"association_id"`
	SuggestedName string  `json:"suggested_name"`
	SuggestedUPC  *string `json:"suggested_upc"`
	Confidence    float64 `json:"confidence"`
	Quantity      int     `json:"quantity"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	DeletedAt     *string `json:"deleted_at"`
	Version       int     `json:"version"`
}

// ScanReportItem is one physical item reported by the edge device. It is
// already consolidated from any matching observations across the three cameras.
type ScanReportItem struct {
	AssociationID  string  `json:"association_id"`
	Name           string  `json:"name"`
	UPC            *string `json:"upc"`
	Confidence     float64 `json:"confidence"`
	Quantity       int     `json:"quantity"`
	RequiresReview bool    `json:"requires_review"`
}

// ScanReport is the complete edge-device output for one uniquely identified
// pass through the scanner.
type ScanReport struct {
	ScanID string           `json:"scan_id"`
	Items  []ScanReportItem `json:"items"`
}
