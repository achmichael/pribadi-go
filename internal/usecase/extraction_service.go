package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fumiama/go-docx"
	"github.com/ledongthuc/pdf"
	"github.com/rs/zerolog"
)

// ExtractionService defines the interface for text extraction from files
type ExtractionService interface {
	ExtractText(ctx context.Context, fileBytes []byte, mimeType string) (string, error)
}

type extractionService struct {
	logger      *zerolog.Logger
	florenceURL string
	httpClient  *http.Client
}

// NewExtractionService creates a new extraction service.
// florenceURL is the base URL of the Florence FastAPI container, e.g. "http://localhost:8100"
func NewExtractionService(logger *zerolog.Logger, florenceURL string) ExtractionService {
	return &extractionService{
		logger:      logger,
		florenceURL: strings.TrimRight(florenceURL, "/"),
		httpClient:  &http.Client{Timeout: 120 * time.Second},
	}
}

// ExtractText extracts text from various file types
func (s *extractionService) ExtractText(ctx context.Context, fileBytes []byte, mimeType string) (string, error) {
	start := time.Now()
	defer func() {
		s.logger.Info().
			Str("mime_type", mimeType).
			Dur("duration", time.Since(start)).
			Msg("Text extraction completed")
	}()

	switch mimeType {
	case "image/png", "image/jpeg", "image/jpg", "image/webp":
		return s.extractFromImageViaFlorence(ctx, fileBytes, mimeType)
	case "application/pdf":
		return s.extractFromPDF(ctx, fileBytes)
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return s.extractFromDOCX(ctx, fileBytes)
	default:
		return "", fmt.Errorf("unsupported mime type: %s", mimeType)
	}
}

// extractFromImageViaFlorence calls the Florence-2 container HTTP API.
// It runs detailed captioning + OCR and merges both results.
func (s *extractionService) extractFromImageViaFlorence(ctx context.Context, fileBytes []byte, mimeType string) (string, error) {
	// Determine file extension from mime type for multipart filename
	ext := "jpg"
	switch mimeType {
	case "image/png":
		ext = "png"
	case "image/webp":
		ext = "webp"
	}

	caption, err := s.florencePost(ctx, "/detailed-caption", fileBytes, ext)
	if err != nil {
		return "", fmt.Errorf("florence detailed-caption failed: %w", err)
	}

	ocrText, err := s.florencePost(ctx, "/ocr", fileBytes, ext)
	if err != nil {
		// OCR failure is non-fatal — proceed with caption only
		s.logger.Warn().Err(err).Msg("Florence OCR failed, using caption only")
		ocrText = ""
	}

	var sb strings.Builder
	sb.WriteString("[Image Description]\n")
	sb.WriteString(caption)
	if ocrText != "" {
		sb.WriteString("\n\n[Text in Image]\n")
		sb.WriteString(ocrText)
	}
	return sb.String(), nil
}

// florenceResponse mirrors the JSON shapes returned by /detailed-caption and /ocr
type florenceCaptionResp struct {
	Caption string `json:"caption"`
}
type florenceOCRResp struct {
	Text string `json:"text"`
}

// florencePost sends an image as multipart/form-data to the given Florence endpoint.
func (s *extractionService) florencePost(ctx context.Context, endpoint string, imgBytes []byte, ext string) (string, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	fw, err := mw.CreateFormFile("file", "image."+ext)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err = fw.Write(imgBytes); err != nil {
		return "", fmt.Errorf("write form file: %w", err)
	}
	mw.Close()

	url := s.florenceURL + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("florence %s returned %d: %s", endpoint, resp.StatusCode, string(raw))
	}

	// Parse response based on endpoint
	switch endpoint {
	case "/ocr":
		var r florenceOCRResp
		if err := json.Unmarshal(raw, &r); err != nil {
			return "", fmt.Errorf("parse ocr response: %w", err)
		}
		return r.Text, nil
	default: // /detailed-caption, /caption
		var r florenceCaptionResp
		if err := json.Unmarshal(raw, &r); err != nil {
			return "", fmt.Errorf("parse caption response: %w", err)
		}
		return r.Caption, nil
	}
}

// extractFromPDF extracts text from all pages of a PDF.
// If a page yields no text (scanned/image PDF), it falls back to Florence OCR.
func (s *extractionService) extractFromPDF(ctx context.Context, fileBytes []byte) (string, error) {
	start := time.Now()

	// ledongthuc/pdf needs a io.ReaderAt + size — use temp file
	tmpFile, err := os.CreateTemp("", "pdfextract-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create temp pdf: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(fileBytes); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("write temp pdf: %w", err)
	}
	tmpFile.Close()

	pdfFile, reader, err := pdf.Open(tmpPath)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer pdfFile.Close()

	totalPages := reader.NumPage()
	s.logger.Info().
		Int("pages", totalPages).
		Dur("parse_duration", time.Since(start)).
		Msg("PDF opened")

	var textBuilder strings.Builder
	emptyPages := 0

	for i := 1; i <= totalPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			emptyPages++
			continue
		}

		pageText, err := page.GetPlainText(nil)
		if err != nil {
			s.logger.Warn().
				Int("page", i).
				Err(err).
				Msg("Failed to extract text from PDF page")
			emptyPages++
			continue
		}

		trimmed := strings.TrimSpace(pageText)
		if trimmed == "" {
			emptyPages++
			continue
		}

		if textBuilder.Len() > 0 {
			textBuilder.WriteString("\n\n")
		}
		textBuilder.WriteString(trimmed)
	}

	// If all pages empty → likely scanned PDF, fallback to Florence vision
	if textBuilder.Len() == 0 && emptyPages > 0 {
		s.logger.Info().
			Int("empty_pages", emptyPages).
			Msg("PDF has no text layer, falling back to Florence OCR")
		return s.extractFromImageViaFlorence(ctx, fileBytes, "image/png")
	}

	s.logger.Info().
		Int("pages_with_text", totalPages-emptyPages).
		Int("empty_pages", emptyPages).
		Int("total_chars", textBuilder.Len()).
		Msg("PDF text extraction done")

	return textBuilder.String(), nil
}

// extractFromDOCX extracts text from DOCX files
func (s *extractionService) extractFromDOCX(_ context.Context, fileBytes []byte) (string, error) {
	reader := bytes.NewReader(fileBytes)

	doc, err := docx.Parse(reader, int64(len(fileBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to parse DOCX: %w", err)
	}

	var textBuilder strings.Builder

	for _, item := range doc.Document.Body.Items {
		switch v := item.(type) {
		case *docx.Paragraph:
			for _, child := range v.Children {
				if run, ok := child.(*docx.Run); ok {
					for _, rc := range run.Children {
						if t, ok := rc.(*docx.Text); ok {
							textBuilder.WriteString(t.Text)
						}
					}
				}
			}
			textBuilder.WriteString("\n")
		case *docx.Table:
			for _, row := range v.TableRows {
				for _, cell := range row.TableCells {
					for _, para := range cell.Paragraphs {
						for _, child := range para.Children {
							if run, ok := child.(*docx.Run); ok {
								for _, rc := range run.Children {
									if t, ok := rc.(*docx.Text); ok {
										textBuilder.WriteString(t.Text)
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
