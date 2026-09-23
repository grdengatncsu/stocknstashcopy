// Package database owns SQLite setup and persistence operations.
//
// HTTP concerns stay in package api; this package receives model values and
// enforces multi-step consistency with transactions where needed.
package database
