package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

const maxFileSize = 50 * 1024 * 1024 // 50MB

// FileRepository defines the interface for file operations
type FileRepository interface {
	SaveTempFile(ctx context.Context, data []byte, ext string) (string, error)
	DeleteFile(ctx context.Context, path string) error
	ReadFile(ctx context.Context, path string) ([]byte, error)
	CleanupOldFiles(ctx context.Context, olderThan time.Duration) error
}

type fileRepo struct {
	basePath string
}

// NewFileRepository creates a new file repository
func NewFileRepository(basePath string) (FileRepository, error) {
	// Create directory if not exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &fileRepo{
		basePath: basePath,
	}, nil
}

func (r *fileRepo) SaveTempFile(ctx context.Context, data []byte, ext string) (string, error) {
	// Check file size limit
	if len(data) > maxFileSize {
		return "", fmt.Errorf("file size exceeds 50MB limit")
	}

	// Generate unique filename
	filename := uuid.New().String() + ext
	filePath := filepath.Join(r.basePath, filename)

	// Write file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filePath, nil
}

func (r *fileRepo) DeleteFile(ctx context.Context, path string) error {
	// Verify path is within basePath (security check)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	absBasePath, err := filepath.Abs(r.basePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute base path: %w", err)
	}

	if !filepath.HasPrefix(absPath, absBasePath) {
		return fmt.Errorf("path is outside base directory")
	}

	// Delete file
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (r *fileRepo) ReadFile(ctx context.Context, path string) ([]byte, error) {
	// Verify path is within basePath (security check)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	absBasePath, err := filepath.Abs(r.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute base path: %w", err)
	}

	if !filepath.HasPrefix(absPath, absBasePath) {
		return nil, fmt.Errorf("path is outside base directory")
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

func (r *fileRepo) CleanupOldFiles(ctx context.Context, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)

	entries, err := os.ReadDir(r.basePath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(r.basePath, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(filePath); err != nil {
				// Log error but continue with other files
				continue
			}
		}
	}

	return nil
}
