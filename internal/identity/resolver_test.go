package identity

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

func newTestResolver(t *testing.T) (*Resolver, repository.Repository) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	schemaPath := "../../db/schema.sql"

	log := zerolog.Nop()
	repo, err := repository.NewSQLiteRepository(dbPath, schemaPath)
	if err != nil {
		t.Fatalf("NewSQLiteRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return NewResolver(repo, &log), repo
}

func TestAutoProvision(t *testing.T) {
	r, _ := newTestResolver(t)
	ctx := context.Background()

	uid1, err := r.ResolveUserID(ctx, "whatsapp", "628111@s.whatsapp.net")
	if err != nil {
		t.Fatalf("ResolveUserID: %v", err)
	}
	if uid1 == "" {
		t.Fatal("expected non-empty UserID")
	}

	// Same identity → same UserID
	uid2, err := r.ResolveUserID(ctx, "whatsapp", "628111@s.whatsapp.net")
	if err != nil {
		t.Fatal(err)
	}
	if uid2 != uid1 {
		t.Errorf("same identity returned different UIDs: %s vs %s", uid1, uid2)
	}

	// Different identity → different UserID
	uid3, err := r.ResolveUserID(ctx, "whatsapp", "628222@s.whatsapp.net")
	if err != nil {
		t.Fatal(err)
	}
	if uid3 == uid1 {
		t.Error("different identity returned same UID")
	}
}

func TestCrossPlatformLink(t *testing.T) {
	r, _ := newTestResolver(t)
	ctx := context.Background()

	// WA user auto-provisioned
	uid, err := r.ResolveUserID(ctx, "whatsapp", "628111@s.whatsapp.net")
	if err != nil {
		t.Fatal(err)
	}

	// Link Telegram to same user
	err = r.LinkPlatform(ctx, uid, "telegram", "12345678")
	if err != nil {
		t.Fatalf("LinkPlatform: %v", err)
	}

	// Resolve from Telegram → same UserID
	resolved, err := r.ResolveUserID(ctx, "telegram", "12345678")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != uid {
		t.Errorf("cross-platform link failed: WA=%s, TG=%s", uid, resolved)
	}
}

func TestLinkNonExistentUser(t *testing.T) {
	r, _ := newTestResolver(t)
	ctx := context.Background()

	err := r.LinkPlatform(ctx, "nonexistent-uuid", "telegram", "999")
	if err == nil {
		t.Error("expected error linking to nonexistent user")
	}
}
