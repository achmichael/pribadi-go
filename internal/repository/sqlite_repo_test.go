package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// newTestRepo creates an ephemeral SQLite repo with all migrations applied.
func newTestRepo(t *testing.T) Repository {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Copy schema and migrations into temp dir so relative paths work.
	// We need db/schema.sql and db/migrations/ accessible from CWD.
	// Since tests run from package dir, we use ../../db/...
	schemaPath := "../../db/schema.sql"
	if _, err := os.Stat(schemaPath); err != nil {
		t.Fatalf("schema file not found at %s: %v", schemaPath, err)
	}

	repo, err := NewSQLiteRepository(dbPath, schemaPath)
	if err != nil {
		t.Fatalf("NewSQLiteRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func TestWALEnabled(t *testing.T) {
	repo := newTestRepo(t)
	var mode string
	err := repo.GetDB().QueryRow("PRAGMA journal_mode;").Scan(&mode)
	if err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("expected WAL, got %s", mode)
	}
}

func TestMigrationTablesExist(t *testing.T) {
	repo := newTestRepo(t)
	tables := []string{"users", "platform_identities", "messages_v2", "user_facts"}
	for _, tbl := range tables {
		var name string
		err := repo.GetDB().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", tbl, err)
		}
	}
}

func TestFTSVirtualTablesExist(t *testing.T) {
	repo := newTestRepo(t)
	tables := []string{"messages_fts", "messages_fts_trigram"}
	for _, tbl := range tables {
		var name string
		err := repo.GetDB().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl,
		).Scan(&name)
		if err != nil {
			t.Errorf("FTS table %s not found: %v", tbl, err)
		}
	}
}

func TestUserCRUDAndPlatformIdentity(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Create user
	uid := "test-uuid-001"
	err := repo.CreateUser(ctx, uid, "Budi")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Get user
	u, err := repo.GetUserByID(ctx, uid)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if u.DisplayName != "Budi" {
		t.Errorf("display_name: got %q, want %q", u.DisplayName, "Budi")
	}

	// Link WA identity
	err = repo.UpsertPlatformIdentity(ctx, "whatsapp", "6281234567890@s.whatsapp.net", uid)
	if err != nil {
		t.Fatalf("UpsertPlatformIdentity: %v", err)
	}

	// Resolve back
	resolved, err := repo.GetUserIDByPlatform(ctx, "whatsapp", "6281234567890@s.whatsapp.net")
	if err != nil {
		t.Fatalf("GetUserIDByPlatform: %v", err)
	}
	if resolved != uid {
		t.Errorf("resolved user_id: got %q, want %q", resolved, uid)
	}
}

func TestInsertMessageV2AndFTSSearch(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// Setup user
	uid := "test-uuid-fts"
	repo.CreateUser(ctx, uid, "TestUser")

	// Insert messages
	msgs := []InsertMessageV2Params{
		{UserID: uid, Platform: "whatsapp", SessionID: "s1", Role: "user", Content: "Saya sedang belajar Golang untuk project kantor", TokenCount: 10},
		{UserID: uid, Platform: "whatsapp", SessionID: "s1", Role: "assistant", Content: "Golang cocok untuk backend services yang performant", TokenCount: 8},
		{UserID: uid, Platform: "whatsapp", SessionID: "s1", Role: "user", Content: "Saya tinggal di kota Pasuruan, Jawa Timur", TokenCount: 9},
		{UserID: uid, Platform: "whatsapp", SessionID: "s1", Role: "user", Content: "Tolong buatkan reminder untuk meeting besok jam 10 pagi", TokenCount: 11},
	}

	for _, m := range msgs {
		if _, err := repo.InsertMessageV2(ctx, m); err != nil {
			t.Fatalf("InsertMessageV2: %v", err)
		}
	}

	// FTS standard search: "golang"
	results, err := repo.SearchMessagesFTS(ctx, uid, "golang", 10)
	if err != nil {
		t.Fatalf("SearchMessagesFTS(golang): %v", err)
	}
	if len(results) != 2 {
		t.Errorf("FTS 'golang': got %d results, want 2", len(results))
	}

	// FTS standard search: "pasuruan"
	results, err = repo.SearchMessagesFTS(ctx, uid, "pasuruan", 10)
	if err != nil {
		t.Fatalf("SearchMessagesFTS(pasuruan): %v", err)
	}
	if len(results) != 1 {
		t.Errorf("FTS 'pasuruan': got %d results, want 1", len(results))
	}

	// Trigram substring search: "remind" (partial match of "reminder")
	results, err = repo.SearchMessagesTrigram(ctx, uid, "remind", 10)
	if err != nil {
		t.Fatalf("SearchMessagesTrigram(remind): %v", err)
	}
	if len(results) != 1 {
		t.Errorf("trigram 'remind': got %d results, want 1", len(results))
	}

	// FTS should NOT match other users
	repo.CreateUser(ctx, "other-user", "Other")
	results, err = repo.SearchMessagesFTS(ctx, "other-user", "golang", 10)
	if err != nil {
		t.Fatalf("SearchMessagesFTS other user: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("FTS for wrong user: got %d results, want 0", len(results))
	}
}

func TestTrigramSearchMixedLang(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	uid := "test-uuid-trigram"
	repo.CreateUser(ctx, uid, "Tester")

	// Indonesian + English mixed content
	repo.InsertMessageV2(ctx, InsertMessageV2Params{
		UserID: uid, Platform: "whatsapp", SessionID: "s1", Role: "user",
		Content: "Saya pakai framework NextJS untuk frontend dan microservice pakai gRPC", TokenCount: 12,
	})

	// Trigram search for technical term substring
	results, err := repo.SearchMessagesTrigram(ctx, uid, "gRPC", 10)
	if err != nil {
		t.Fatalf("SearchMessagesTrigram(gRPC): %v", err)
	}
	if len(results) != 1 {
		t.Errorf("trigram 'gRPC': got %d results, want 1", len(results))
	}

	// Trigram search for partial: "NextJ" (substring of NextJS)
	results, err = repo.SearchMessagesTrigram(ctx, uid, "NextJ", 10)
	if err != nil {
		t.Fatalf("SearchMessagesTrigram(NextJ): %v", err)
	}
	if len(results) != 1 {
		t.Errorf("trigram 'NextJ': got %d results, want 1", len(results))
	}
}

func TestListMessagesByUserSession(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	uid := "test-uuid-session"
	repo.CreateUser(ctx, uid, "Tester")

	// Insert messages in 2 sessions
	repo.InsertMessageV2(ctx, InsertMessageV2Params{UserID: uid, Platform: "whatsapp", SessionID: "s1", Role: "user", Content: "Hello session 1"})
	repo.InsertMessageV2(ctx, InsertMessageV2Params{UserID: uid, Platform: "whatsapp", SessionID: "s1", Role: "assistant", Content: "Hi back in s1"})
	repo.InsertMessageV2(ctx, InsertMessageV2Params{UserID: uid, Platform: "whatsapp", SessionID: "s2", Role: "user", Content: "New session here"})

	// List s1
	results, err := repo.ListMessagesByUserSession(ctx, uid, "s1", 100)
	if err != nil {
		t.Fatalf("ListMessagesByUserSession: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("session s1: got %d results, want 2", len(results))
	}

	// List s2
	results, err = repo.ListMessagesByUserSession(ctx, uid, "s2", 100)
	if err != nil {
		t.Fatalf("ListMessagesByUserSession s2: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("session s2: got %d results, want 1", len(results))
	}
}

func TestUserFactsCRUD(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	uid := "test-uuid-facts"
	repo.CreateUser(ctx, uid, "Tester")

	// Insert facts
	id1, err := repo.InsertFact(ctx, InsertFactParams{
		UserID: uid, FactText: "Tinggal di Pasuruan", Category: "biodata",
	})
	if err != nil {
		t.Fatalf("InsertFact: %v", err)
	}

	id2, err := repo.InsertFact(ctx, InsertFactParams{
		UserID: uid, FactText: "Suka kopi hitam tanpa gula", Category: "preference",
	})
	if err != nil {
		t.Fatalf("InsertFact 2: %v", err)
	}

	// List active
	facts, err := repo.ListActiveFacts(ctx, uid, 100)
	if err != nil {
		t.Fatalf("ListActiveFacts: %v", err)
	}
	if len(facts) != 2 {
		t.Errorf("active facts: got %d, want 2", len(facts))
	}

	// Deactivate one
	if err := repo.DeactivateFact(ctx, id1); err != nil {
		t.Fatalf("DeactivateFact: %v", err)
	}

	facts, err = repo.ListActiveFacts(ctx, uid, 100)
	if err != nil {
		t.Fatalf("ListActiveFacts after deactivate: %v", err)
	}
	if len(facts) != 1 {
		t.Errorf("active facts after deactivate: got %d, want 1", len(facts))
	}
	if facts[0].ID != id2 {
		t.Errorf("remaining fact ID: got %d, want %d", facts[0].ID, id2)
	}

	// Update timestamp
	if err := repo.UpdateFactTimestamp(ctx, id2); err != nil {
		t.Fatalf("UpdateFactTimestamp: %v", err)
	}
}

func TestFTSTriggerAfterDelete(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	uid := "test-uuid-del"
	repo.CreateUser(ctx, uid, "Tester")

	msg, err := repo.InsertMessageV2(ctx, InsertMessageV2Params{
		UserID: uid, Platform: "whatsapp", SessionID: "s1", Role: "user",
		Content: "Ini pesan yang akan dihapus berisi kata unik xyzzy123",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Confirm FTS finds it
	results, err := repo.SearchMessagesFTS(ctx, uid, "xyzzy123", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 FTS result before delete, got %d", len(results))
	}

	// Delete directly
	_, err = repo.GetDB().ExecContext(ctx, "DELETE FROM messages_v2 WHERE id = ?", msg.ID)
	if err != nil {
		t.Fatal(err)
	}

	// FTS should no longer find it
	results, err = repo.SearchMessagesFTS(ctx, uid, "xyzzy123", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 FTS results after delete, got %d", len(results))
	}
}

func TestFTSTriggerAfterUpdate(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	uid := "test-uuid-upd"
	repo.CreateUser(ctx, uid, "Tester")

	msg, err := repo.InsertMessageV2(ctx, InsertMessageV2Params{
		UserID: uid, Platform: "whatsapp", SessionID: "s1", Role: "user",
		Content: "Kata lama uniqueword999",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Update content
	_, err = repo.GetDB().ExecContext(ctx,
		"UPDATE messages_v2 SET content = ? WHERE id = ?",
		"Kata baru replacedterm888", msg.ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Old term gone from FTS
	results, err := repo.SearchMessagesFTS(ctx, uid, "uniqueword999", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("old term still in FTS after update: %d results", len(results))
	}

	// New term found
	results, err = repo.SearchMessagesFTS(ctx, uid, "replacedterm888", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("new term not in FTS after update: %d results", len(results))
	}
}
