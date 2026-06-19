package usecase

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

func TestExtractionService(t *testing.T) {
	logger := zerolog.Nop()
	service := NewExtractionService(&logger)

	t.Run("UnsupportedMimeType", func(t *testing.T) {
		ctx := context.Background()
		_, err := service.ExtractText(ctx, []byte("test"), "application/unsupported")
		if err == nil {
			t.Error("Expected error for unsupported mime type")
		}
	})

	t.Run("ImagePlaceholder", func(t *testing.T) {
		ctx := context.Background()
		text, err := service.ExtractText(ctx, []byte("fake image data"), "image/jpeg")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if text == "" {
			t.Error("Expected non-empty text")
		}
	})
}

func TestExtractionServiceMimeTypes(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		isValid  bool
	}{
		{"PNG", "image/png", true},
		{"JPEG", "image/jpeg", true},
		{"JPG", "image/jpg", true},
		{"PDF", "application/pdf", true},
		{"DOCX", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", true},
		{"TXT", "text/plain", false},
		{"MP3", "audio/mpeg", false},
	}

	logger := zerolog.Nop()
	service := NewExtractionService(&logger)
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ExtractText(ctx, []byte("dummy data"), tt.mimeType)
			hasError := err != nil
			expectedError := !tt.isValid

			if hasError != expectedError {
				t.Errorf("MimeType %s: expected error=%v, got error=%v", tt.mimeType, expectedError, hasError)
			}
		})
	}
}

func TestExtractionDOCX(t *testing.T) {
	// This test requires a valid DOCX file
	// Skipping for now as it would require test fixtures
	t.Skip("Requires test DOCX file fixtures")
}

func TestExtractionPDF(t *testing.T) {
	// This test requires a valid PDF file
	// Skipping for now as it would require test fixtures
	t.Skip("Requires test PDF file fixtures")
}
