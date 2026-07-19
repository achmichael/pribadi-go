package classifier

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
)

// These tests cover heuristic classification only (no LLM needed).

func TestHeuristic_Greeting(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := &intentClassifier{logger: &logger}

	cases := []struct {
		input string
		want  MessageClass
	}{
		{"hi", ClassCasual},
		{"halo", ClassCasual},
		{"hello", ClassCasual},
		{"hai", ClassCasual},
		{"pagi", ClassCasual},
	}

	for _, tc := range cases {
		result := c.heuristicClassify(ClassifyParams{UserText: tc.input})
		if result == nil {
			t.Errorf("expected classification for %q, got nil", tc.input)
			continue
		}
		if result.MessageClass != tc.want {
			t.Errorf("input=%q: expected class=%s, got=%s", tc.input, tc.want, result.MessageClass)
		}
	}
}

func TestHeuristic_LanguagePreference(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := &intentClassifier{logger: &logger}

	cases := []struct {
		input string
		lang  string
	}{
		{"jawab dalam bahasa inggris", "en"},
		{"Answer in English", "en"},
		{"use english", "en"},
		{"pakai bahasa indonesia", "id"},
	}

	for _, tc := range cases {
		result := c.heuristicClassify(ClassifyParams{UserText: tc.input})
		if result == nil {
			t.Errorf("expected classification for %q, got nil", tc.input)
			continue
		}
		if !result.IsInstruction {
			t.Errorf("input=%q: expected is_instruction=true", tc.input)
		}
		if len(result.ExtractedPrefs) == 0 {
			t.Errorf("input=%q: expected extracted prefs", tc.input)
			continue
		}
		if result.ExtractedPrefs[0].Value != tc.lang {
			t.Errorf("input=%q: expected lang=%s, got=%s", tc.input, tc.lang, result.ExtractedPrefs[0].Value)
		}
	}
}

func TestHeuristic_DisplayName(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := &intentClassifier{logger: &logger}

	cases := []struct {
		input string
		name  string
	}{
		{"panggil aku Michael", "michael"},
		{"call me Budi", "budi"},
		{"nama saya Ahmad", "ahmad"},
	}

	for _, tc := range cases {
		result := c.heuristicClassify(ClassifyParams{UserText: tc.input})
		if result == nil {
			t.Errorf("expected classification for %q, got nil", tc.input)
			continue
		}
		if len(result.ExtractedPrefs) == 0 {
			t.Errorf("input=%q: expected extracted prefs", tc.input)
			continue
		}
		if result.ExtractedPrefs[0].Key != "display_name" {
			t.Errorf("input=%q: expected key=display_name, got=%s", tc.input, result.ExtractedPrefs[0].Key)
		}
		if result.ExtractedPrefs[0].Value != tc.name {
			t.Errorf("input=%q: expected name=%s, got=%s", tc.input, tc.name, result.ExtractedPrefs[0].Value)
		}
	}
}

func TestHeuristic_Verbosity(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := &intentClassifier{logger: &logger}

	cases := []struct {
		input string
		value string
	}{
		{"jawab singkat saja", "concise"},
		{"be concise", "concise"},
		{"jelaskan detail", "detailed"},
		{"explain in detail", "detailed"},
	}

	for _, tc := range cases {
		result := c.heuristicClassify(ClassifyParams{UserText: tc.input})
		if result == nil {
			t.Errorf("expected classification for %q, got nil", tc.input)
			continue
		}
		if len(result.ExtractedPrefs) == 0 {
			t.Errorf("input=%q: expected extracted prefs", tc.input)
			continue
		}
		if result.ExtractedPrefs[0].Value != tc.value {
			t.Errorf("input=%q: expected value=%s, got=%s", tc.input, tc.value, result.ExtractedPrefs[0].Value)
		}
	}
}

func TestHeuristic_ToneAndFormat(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := &intentClassifier{logger: &logger}

	// Tone
	result := c.heuristicClassify(ClassifyParams{UserText: "pakai nada formal"})
	if result == nil || len(result.ExtractedPrefs) == 0 {
		t.Fatal("expected tone pref")
	}
	if result.ExtractedPrefs[0].Value != "formal" {
		t.Errorf("expected tone=formal, got=%s", result.ExtractedPrefs[0].Value)
	}

	// Markdown
	result = c.heuristicClassify(ClassifyParams{UserText: "gunakan markdown"})
	if result == nil || len(result.ExtractedPrefs) == 0 {
		t.Fatal("expected format pref")
	}
	if result.ExtractedPrefs[0].Value != "markdown" {
		t.Errorf("expected format=markdown, got=%s", result.ExtractedPrefs[0].Value)
	}
}

func TestHeuristic_NoMatch(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := &intentClassifier{logger: &logger}

	// Normal question should NOT be heuristically classified
	result := c.heuristicClassify(ClassifyParams{UserText: "Apa itu machine learning?"})
	if result != nil {
		t.Errorf("expected nil for normal question, got class=%s", result.MessageClass)
	}
}

func TestFallback_Question(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := &intentClassifier{logger: &logger}

	result := c.fallbackClassify(ClassifyParams{
		UserText: "Apa itu neural network?",
	})

	if result.MessageClass != ClassQuestion {
		t.Errorf("expected class=question, got=%s", result.MessageClass)
	}
	if result.Intent != IntentAskInfo {
		t.Errorf("expected intent=ask_information, got=%s", result.Intent)
	}
}

func TestFallback_QuestionWithActiveDoc(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := &intentClassifier{logger: &logger}

	result := c.fallbackClassify(ClassifyParams{
		UserText:     "Siapa penulisnya?",
		HasActiveDoc: true,
	})

	if result.Intent != IntentAskDocument {
		t.Errorf("expected intent=ask_document, got=%s", result.Intent)
	}
}

func TestFallback_ContinuesTask(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := &intentClassifier{logger: &logger}

	result := c.fallbackClassify(ClassifyParams{
		UserText:   "lanjutkan",
		ActiveTask: "summarize document",
		TurnCount:  3,
	})

	if !result.ContinuesPrevious {
		t.Error("expected continues_previous=true")
	}
}

// Verify Classify method works (uses heuristic path, no LLM).
func TestClassify_HeuristicPath(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := NewIntentClassifier(nil, &logger) // nil LLM — will use heuristic

	result, err := c.Classify(context.Background(), ClassifyParams{
		UserText: "halo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageClass != ClassCasual {
		t.Errorf("expected casual, got %s", result.MessageClass)
	}
}
