package identity

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Resolver maps (platform, platformUserID) → internal UserID.
// Auto-provisions new users on first encounter.
type Resolver struct {
	repo   repository.Repository
	logger *zerolog.Logger
	mu     sync.Mutex // serialize auto-provisioning to avoid race
}

func NewResolver(repo repository.Repository, logger *zerolog.Logger) *Resolver {
	return &Resolver{repo: repo, logger: logger}
}

// ResolveUserID returns the internal UserID for a platform identity.
// If no mapping exists, creates a new user + mapping (auto-provisioning).
func (r *Resolver) ResolveUserID(ctx context.Context, platform, platformUserID string) (string, error) {
	// Fast path: already mapped
	uid, err := r.repo.GetUserIDByPlatform(ctx, platform, platformUserID)
	if err == nil && uid != "" {
		return uid, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("resolve lookup: %w", err)
	}

	// Slow path: auto-provision
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring lock
	uid, err = r.repo.GetUserIDByPlatform(ctx, platform, platformUserID)
	if err == nil && uid != "" {
		return uid, nil
	}

	// Create new user
	newUID := uuid.New().String()
	if err := r.repo.CreateUser(ctx, newUID, ""); err != nil {
		return "", fmt.Errorf("create user: %w", err)
	}
	if err := r.repo.UpsertPlatformIdentity(ctx, platform, platformUserID, newUID); err != nil {
		return "", fmt.Errorf("upsert identity: %w", err)
	}

	r.logger.Info().
		Str("user_id", newUID).
		Str("platform", platform).
		Str("platform_user_id", platformUserID).
		Msg("[identity] auto-provisioned new user")

	return newUID, nil
}

// LinkPlatform links a new platform identity to an existing user.
// Used for manual account linking (e.g. /link command).
func (r *Resolver) LinkPlatform(ctx context.Context, userID, platform, platformUserID string) error {
	// Verify user exists
	_, err := r.repo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user %s not found: %w", userID, err)
	}
	return r.repo.UpsertPlatformIdentity(ctx, platform, platformUserID, userID)
}
