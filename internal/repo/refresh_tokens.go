package repo

import (
	"context"
	"time"

	"github.com/Uranury/oci-platform/internal/models"
	"github.com/Uranury/oci-platform/internal/repo/dbgen"
	"github.com/Uranury/oci-platform/pkg/dbutil"
	"github.com/google/uuid"
)

type RefreshToken interface {
	Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*models.RefreshToken, error)
	GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type refreshToken struct {
	q *dbgen.Queries
}

func NewRefreshToken(exec dbutil.Executor) RefreshToken {
	return &refreshToken{q: dbgen.New(exec)}
}

func toRefreshToken(r dbgen.RefreshToken) *models.RefreshToken {
	return &models.RefreshToken{
		ID: r.ID, UserID: r.UserID, TokenHash: r.TokenHash,
		ExpiresAt: r.ExpiresAt, RevokedAt: r.RevokedAt, CreatedAt: r.CreatedAt,
	}
}

func (r *refreshToken) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*models.RefreshToken, error) {
	row, err := r.q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
		UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, translate(err)
	}
	return toRefreshToken(row), nil
}

func (r *refreshToken) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	row, err := r.q.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, translate(err)
	}
	return toRefreshToken(row), nil
}

func (r *refreshToken) Revoke(ctx context.Context, id uuid.UUID) error {
	return translate(r.q.RevokeRefreshToken(ctx, id))
}

func (r *refreshToken) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	return translate(r.q.RevokeAllRefreshTokensForUser(ctx, userID))
}
