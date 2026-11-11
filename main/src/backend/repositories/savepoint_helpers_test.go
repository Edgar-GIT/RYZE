package repositories_test

import "gorm.io/gorm"

// inSavepoint runs a statement a test expects the database to reject inside
// a savepoint of the test transaction. PostgreSQL aborts a transaction after
// any failed statement; the savepoint is rolled back instead, so the test can
// keep using its transaction afterwards.
func inSavepoint(tx *gorm.DB, fn func(sp *gorm.DB) error) error {
	return tx.Transaction(fn)
}
