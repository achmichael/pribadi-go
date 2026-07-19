package reasoning

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

func setupTestRepo(t *testing.T) repository.Repository {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-reasoning-*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	repo, err := repository.NewSQLiteRepository(tmpFile.Name(), "../../db/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func TestVerifier_SimpleAck(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	v := &verifier{
		repo:   repo,
		logger: &logger,
	}

	cases := []string{
		"ok",
		"OK!",
		"baik",
		"✅",
		"👍",
		"yes",
	}

	for _, tc := range cases {
		if !v.isSimpleAck(tc) {
			t.Errorf("expected %q to be simple ack", tc)
		}
	}
}

func TestVerifier_NotSimpleAck(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	v := &verifier{
		repo:   repo,
		logger: &logger,
	}

	cases := []string{
		"This is a detailed explanation about machine learning.",
		"Machine learning is a subset of AI that focuses on...",
		"okay, but I need more information",
	}

	for _, tc := range cases {
		if v.isSimpleAck(tc) {
			t.Errorf("expected %q NOT to be simple ack", tc)
		}
	}
}

func TestVerifier_ContainsCertainLanguage(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	v := &verifier{
		repo:   repo,
		logger: &logger,
	}

	cases := []struct {
		text     string
		contains bool
	}{
		{"This is definitely true", true},
		{"I am absolutely certain", true},
		{"Pasti demikian", true},
		{"It might be true", false},
		{"I think so", false},
		{"Always happens", true},
		{"Selalu begitu", true},
	}

	for _, tc := range cases {
		result := v.containsCertainLanguage(tc.text)
		if result != tc.contains {
			t.Errorf("text=%q: expected contains=%v, got=%v", tc.text, tc.contains, result)
		}
	}
}

func TestVerifier_FallbackVerification(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	v := &verifier{
		repo:   repo,
		logger: &logger,
	}

	// Response with certainty but no context
	params := VerifyParams{
		UserQuestion: "What is AI?",
		Response:     "AI is definitely the future of technology and absolutely will change everything.",
		RAGContext:   "", // no context
	}

	result := v.fallbackVerification(params)

	if result.IsValid {
		t.Error("expected invalid for certain claims without context")
	}
	if result.HallucinationRisk != "medium" {
		t.Errorf("expected medium risk, got %s", result.HallucinationRisk)
	}
	if len(result.Issues) == 0 {
		t.Error("expected issues to be reported")
	}
}

func TestVerifier_FallbackVerification_VerboseResponse(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	v := &verifier{
		repo:   repo,
		logger: &logger,
	}

	params := VerifyParams{
		UserQuestion: "Hi",
		Response:     strings.Repeat("This is a very long response. ", 50), // >500 chars
	}

	result := v.fallbackVerification(params)

	if len(result.Issues) == 0 {
		t.Error("expected issue about verbosity")
	}
}

func TestVerifier_SkipSimpleResponse(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	v := NewVerifier(nil, repo, nil, &logger)

	state := domain.DefaultStateData()
	params := VerifyParams{
		UserQuestion: "Are you ready?",
		Response:     "Yes",
		State:        &state,
	}

	result, err := v.Verify(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsValid {
		t.Error("expected simple ack to be valid")
	}
	if result.ConfidenceScore != 1.0 {
		t.Errorf("expected confidence=1.0 for simple ack, got %f", result.ConfidenceScore)
	}
}

func TestReflector_FallbackReflection(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	r := &reflector{logger: &logger}

	result := r.fallbackReflection()

	if len(result.WhatWorked) == 0 {
		t.Error("expected at least one item in what_worked")
	}
	if result.ConfidenceLevel == "" {
		t.Error("expected confidence level")
	}
}

func TestReflector_SkipShortResponse(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	r := NewReflector(nil, &logger)

	params := ReflectionParams{
		UserQuestion: "Hello",
		Response:     "Hi there!",
		Plan:         "direct",
	}

	result, err := r.Reflect(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.WhatWorked) == 0 {
		t.Error("expected fallback reflection")
	}
}
