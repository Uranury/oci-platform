package repo

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrNotFound is returned when no row matches (including rows the caller
	// may not see).
	ErrNotFound = errors.New("repo: not found")
	// ErrDuplicate is returned when an insert/update hits a unique constraint.
	ErrDuplicate = errors.New("repo: duplicate")
	// ErrInUse is returned when a delete is blocked by a foreign key that
	// still references the row.
	ErrInUse = errors.New("repo: still referenced")
)

// translate maps pgx/Postgres errors to repo sentinels so nothing above the
// repo layer needs to know about pgx or Postgres error codes.
func translate(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
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

// affected turns a zero-row write into ErrNotFound.
func affected(n int64, err error) error {
	if err != nil {
		return translate(err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
