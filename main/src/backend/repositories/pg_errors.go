package repositories

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// uniqueViolation is the SQLSTATE PostgreSQL reports when an INSERT or UPDATE
// would break a unique constraint.
const uniqueViolation = "23505"

// isDuplicateEntry reports whether an error is a unique-constraint violation.
func isDuplicateEntry(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}

// violatedConstraint returns the name of the constraint a PostgreSQL error
// reports, or "" when the error carries none.
func violatedConstraint(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

// inSavepoint runs fn in its own transaction scope: a savepoint when db is
// already inside a transaction, a short transaction otherwise. PostgreSQL
// aborts the whole surrounding transaction after any failed statement, so a
// statement whose constraint violation a repository maps to a domain error
// runs here: the caller's transaction stays usable afterwards, as it did with
// MariaDB.
func inSavepoint(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}
