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
