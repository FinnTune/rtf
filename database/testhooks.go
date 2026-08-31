package database

import "database/sql"

// MigrateForTest exposes migrate to external test packages.
func MigrateForTest(db *sql.DB) error {
	return migrate(db)
}

// ColumnExistsForTest exposes columnExists to external test packages.
func ColumnExistsForTest(db *sql.DB, table, column string) (bool, error) {
	return columnExists(db, table, column)
}
