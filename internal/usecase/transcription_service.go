package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

// resolveLibPath resolves the library path relative to the whisper binary.
// It looks for a "lib" directory next to the binary's parent directory.
func resolveLibPath(whisperBinPath string) string {
	absBin, err := filepath.Abs(whisperBinPath)
	if err != nil {
		return ""
	}
	// bin is at e.g. /project/bin/whisper → lib at /project/lib
	return filepath.Join(filepath.Dir(filepath.Dir(absBin)), "lib")
}

// TranscriptionService defines the interface for audio transcription
type TranscriptionService interface {
	Transcribe(ctx context.Context, audioBytes []byte) (string, error)
}

type transcriptionService struct {
	fileRepo         repository.FileRepository
	whisperBinPath   string
	whisperModelPath string
	logger           *zerolog.Logger
}

// NewTranscriptionService creates a new transcription service
func NewTranscriptionService(
	fileRepo repository.FileRepository,
	whisperBinPath string,
	whisperModelPath string,
	logger *zerolog.Logger,
) TranscriptionService {
	return &transcriptionService{
		fileRepo:         fileRepo,
		whisperBinPath:   whisperBinPath,
		whisperModelPath: whisperModelPath,
		logger:           logger,
	}
}

// Transcribe transcribes audio using Whisper.cpp
func (s *transcriptionService) Transcribe(ctx context.Context, audioBytes []byte) (string, error) {
	start := time.Now()
	s.logger.Info().Int("audio_size", len(audioBytes)).Msg("Starting transcription")

	// Save audio to temp file
	tempAudioPath, err := s.fileRepo.SaveTempFile(ctx, audioBytes, ".ogg")
	if err != nil {
		return "", fmt.Errorf("failed to save temp audio file: %w", err)
	}
	defer func() {
		if err := s.fileRepo.DeleteFile(ctx, tempAudioPath); err != nil {
			s.logger.Warn().Err(err).Str("path", tempAudioPath).Msg("Failed to delete temp audio file")
		}
	}()

	// Convert OGG to WAV (16kHz, 1 channel) using FFmpeg
	tempWavPath := strings.TrimSuffix(tempAudioPath, filepath.Ext(tempAudioPath)) + ".wav"
	ffmpegCmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", tempAudioPath, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", tempWavPath)
	if err := ffmpegCmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg conversion failed: %w", err)
	}
	defer func() {
		if err := s.fileRepo.DeleteFile(ctx, tempWavPath); err != nil {
			s.logger.Warn().Err(err).Str("path", tempWavPath).Msg("Failed to delete temp wav file")
		}
	}()

	// Output path for transcription
	outputDir := filepath.Dir(tempWavPath)
	outputBaseName := strings.TrimSuffix(filepath.Base(tempWavPath), filepath.Ext(tempWavPath))
	outputTxtPath := filepath.Join(outputDir, outputBaseName+".txt")

	defer func() {
		if _, err := os.Stat(outputTxtPath); err == nil {
			if err := os.Remove(outputTxtPath); err != nil {
				s.logger.Warn().Err(err).Str("path", outputTxtPath).Msg("Failed to delete output txt file")
			}
		}
	}()

	// Execute whisper.cpp CLI
	cmd := exec.CommandContext(ctx,
		s.whisperBinPath,
		"--model", s.whisperModelPath,
		"--language", "auto",
		"--output-txt",
		"--file", tempWavPath,
	)

	// Set LD_LIBRARY_PATH so whisper can find its shared libraries
	libPath := resolveLibPath(s.whisperBinPath)
	cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH="+libPath)

	// Capture output for logging
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	s.logger.Info().
		Str("bin", s.whisperBinPath).
		Str("model", s.whisperModelPath).
		Str("audio", tempAudioPath).
		Msg("Executing whisper.cpp")

	// Run command
	if err := cmd.Run(); err != nil {
		s.logger.Error().
			Err(err).
			Str("stdout", stdout.String()).
			Str("stderr", stderr.String()).
			Msg("Whisper execution failed")
		return "", fmt.Errorf("whisper execution failed: %w\nstderr: %s", err, stderr.String())
	}

	s.logger.Debug().
		Str("stdout", stdout.String()).
		Msg("Whisper execution output")

	// Read transcription output
	audioPath := tempWavPath + ".txt"
	transcriptBytes, err := os.ReadFile(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to read transcription output: %w", err)
	}

	transcript := strings.TrimSpace(string(transcriptBytes))

	duration := time.Since(start)
	s.logger.Info().
		Dur("duration", duration).
		Int("char_count", len(transcript)).
		Msg("Transcription completed")

	return transcript, nil
}
