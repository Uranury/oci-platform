package repo

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrDuplicate is returned when an insert/update hits a unique constraint.
	ErrDuplicate = errors.New("repo: duplicate")
	// ErrInUse is returned when a delete is blocked by a foreign key that
	// still references the row.
	ErrInUse = errors.New("repo: still referenced")
)

// translatePgError maps constraint violations to repo sentinels so callers
// never need to know Postgres error codes.
func translatePgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return ErrDuplicate
		case "23503": // foreign_key_violation
			return ErrInUse
		}
	}
	return err
}
