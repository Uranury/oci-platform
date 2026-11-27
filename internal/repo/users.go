package repo

import (
	"context"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo/dbgen"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
)

type User interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Create(ctx context.Context, id uuid.UUID, username, email, password string) error
	// Delete removes the user and everything they own. It runs several
	// statements: call it on a transaction's Executor (RunInTx).
	Delete(ctx context.Context, id uuid.UUID) error
}

type user struct {
	q *dbgen.Queries
}

func NewUser(exec dbutil.Executor) User {
	return &user{q: dbgen.New(exec)}
}

func toUser(r dbgen.User) *models.User {
	return &models.User{
		ID: r.ID, Username: r.Username, Email: r.Email, Password: r.Password,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func (u *user) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	r, err := u.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, translate(err)
	}
	return toUser(r), nil
}

func (u *user) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	r, err := u.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, translate(err)
	}
	return toUser(r), nil
}

func (u *user) Create(ctx context.Context, id uuid.UUID, username, email, password string) error {
	return translate(u.q.CreateUser(ctx, dbgen.CreateUserParams{
		ID: id, Username: username, Email: email, Password: password,
	}))
}

func (u *user) Delete(ctx context.Context, id uuid.UUID) error {
	if err := u.q.DeleteUserWorkouts(ctx, id); err != nil {
		return translate(err)
	}
	if err := u.q.DeleteUserTemplates(ctx, id); err != nil {
		return translate(err)
	}
	return affected(u.q.DeleteUser(ctx, id))
}
