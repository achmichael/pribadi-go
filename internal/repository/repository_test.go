package repository

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/achmichael/pribadi-go/internal/repository/sqlc"
)

func TestSQLiteRepository(t *testing.T) {
	// Create temp directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	schemaPath := "../../db/schema.sql"

	// Create repository
	repo, err := NewSQLiteRepository(dbPath, schemaPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Test message operations
	t.Run("InsertMessage", func(t *testing.T) {
		msg, err := repo.InsertMessage(ctx, sqlc.InsertMessageParams{
			WaID:      "test-wa-id-1",
			FromJid:   "user1@whatsapp.net",
			ToJid:     "user2@whatsapp.net",
			Content:   "Hello, World!",
			MediaType: sql.NullString{},
			Timestamp: time.Now().Unix(),
		})
		if err != nil {
			t.Fatalf("Failed to insert message: %v", err)
		}
		if msg.ID == 0 {
			t.Error("Expected non-zero message ID")
		}
	})

	// Test project operations
	t.Run("InsertProject", func(t *testing.T) {
		proj, err := repo.InsertProject(ctx, sqlc.InsertProjectParams{
			Name:     "Test Project",
			OwnerJid: "owner@whatsapp.net",
			Status:   "active",
		})
		if err != nil {
			t.Fatalf("Failed to insert project: %v", err)
		}
		if proj.ID == 0 {
			t.Error("Expected non-zero project ID")
		}
	})
}

func TestFileRepository(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create repository
	repo, err := NewFileRepository(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create file repository: %v", err)
	}

	ctx := context.Background()

	t.Run("SaveAndReadFile", func(t *testing.T) {
		testData := []byte("test file content")
		
		// Save file
		filePath, err := repo.SaveTempFile(ctx, testData, ".txt")
		if err != nil {
			t.Fatalf("Failed to save file: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("File was not created")
		}

		// Read file
		data, err := repo.ReadFile(ctx, filePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(data) != string(testData) {
			t.Errorf("File content mismatch: got %s, want %s", data, testData)
		}

		// Delete file
		if err := repo.DeleteFile(ctx, filePath); err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		// Verify file is deleted
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Error("File was not deleted")
		}
	})

	t.Run("FileSizeLimit", func(t *testing.T) {
		// Create data larger than 50MB
		largeData := make([]byte, 51*1024*1024)
		
		_, err := repo.SaveTempFile(ctx, largeData, ".bin")
		if err == nil {
			t.Error("Expected error for file size exceeding limit")
		}
	})

	t.Run("CleanupOldFiles", func(t *testing.T) {
		// Create a test file
		testData := []byte("old file")
		filePath, err := repo.SaveTempFile(ctx, testData, ".txt")
		if err != nil {
			t.Fatalf("Failed to save file: %v", err)
		}

		// Change file modification time to 2 hours ago
		oldTime := time.Now().Add(-2 * time.Hour)
		if err := os.Chtimes(filePath, oldTime, oldTime); err != nil {
			t.Fatalf("Failed to change file time: %v", err)
		}

		// Cleanup files older than 1 hour
		if err := repo.CleanupOldFiles(ctx, 1*time.Hour); err != nil {
			t.Fatalf("Failed to cleanup old files: %v", err)
		}

		// Verify file is deleted
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Error("Old file was not deleted")
		}
	})
}
