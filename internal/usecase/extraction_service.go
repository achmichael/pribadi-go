package usecase

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fumiama/go-docx"
	"github.com/rs/zerolog"
)

// ExtractionService defines the interface for text extraction from files
type ExtractionService interface {
	ExtractText(ctx context.Context, fileBytes []byte, mimeType string) (string, error)
}

type extractionService struct {
	logger *zerolog.Logger
}

// NewExtractionService creates a new extraction service
func NewExtractionService(logger *zerolog.Logger) ExtractionService {
	return &extractionService{
		logger: logger,
	}
}

// ExtractText extracts text from various file types
func (s *extractionService) ExtractText(ctx context.Context, fileBytes []byte, mimeType string) (string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.logger.Info().
			Str("mime_type", mimeType).
			Dur("duration", duration).
			Msg("Text extraction completed")
	}()

	var text string
	var err error

	switch mimeType {
	case "image/png", "image/jpeg", "image/jpg":
		text, err = s.extractFromImage(ctx, fileBytes)
	case "application/pdf":
		text, err = s.extractFromPDF(ctx, fileBytes)
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		text, err = s.extractFromDOCX(ctx, fileBytes)
	default:
		return "", fmt.Errorf("unsupported mime type: %s", mimeType)
	}

	if err != nil {
		return "", err
	}

	s.logger.Info().
		Int("char_count", len(text)).
		Str("mime_type", mimeType).
		Msg("Text extracted successfully")

	return text, nil
}

// extractFromImage extracts text from images using OCR
// Note: This is a placeholder. In production, use gosseract or HTTP call to Tesseract service
func (s *extractionService) extractFromImage(ctx context.Context, fileBytes []byte) (string, error) {
	s.logger.Warn().Msg("OCR extraction not fully implemented - returning placeholder")
	
	// TODO: Implement OCR using one of these approaches:
	// 1. gosseract (requires CGO): github.com/otiai10/gosseract/v2
	// 2. HTTP call to local Tesseract REST API
	// 3. Cloud OCR service (Google Vision, AWS Textract, etc.)
	
	// For now, return a placeholder message
	return "[OCR extraction not available - please implement gosseract or Tesseract REST wrapper]", nil
}

func (s *extractionService) extractFromPDF(ctx context.Context, fileBytes []byte) (string, error) {
	s.logger.Warn().Msg("PDF text extraction not fully implemented - returning placeholder")
	return "[PDF extraction not available - please implement using a suitable PDF text extraction library]", nil
}

// extractFromDOCX extracts text from DOCX files
func (s *extractionService) extractFromDOCX(ctx context.Context, fileBytes []byte) (string, error) {
	reader := bytes.NewReader(fileBytes)
	
	// Read DOCX file
	doc, err := docx.Parse(reader, int64(len(fileBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to parse DOCX: %w", err)
	}

	var textBuilder strings.Builder

	// Extract text from paragraphs
	for _, item := range doc.Document.Body.Items {
		switch item.(type) {
		case *docx.Paragraph:
			para := item.(*docx.Paragraph)
			for _, child := range para.Children {
				switch child.(type) {
				case *docx.Run:
					run := child.(*docx.Run)
					for _, rc := range run.Children {
						switch rc.(type) {
						case *docx.Text:
							text := rc.(*docx.Text)
							textBuilder.WriteString(text.Text)
						}
					}
				}
			}
			textBuilder.WriteString("\n")
		case *docx.Table:
			// Extract text from tables
			table := item.(*docx.Table)
			for _, row := range table.TableRows {
				for _, cell := range row.TableCells {
					for _, para := range cell.Paragraphs {
						for _, child := range para.Children {
							if run, ok := child.(*docx.Run); ok {
								for _, rc := range run.Children {
									if text, ok := rc.(*docx.Text); ok {
										textBuilder.WriteString(text.Text)
										textBuilder.WriteString(" ")
									}
								}
							}
						}
					}
					textBuilder.WriteString("\t")
				}
				textBuilder.WriteString("\n")
			}
		}
	}

	return textBuilder.String(), nil
}
