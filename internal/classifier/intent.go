// Package classifier provides LLM-based message classification.
// It identifies user intent and message type before the orchestrator
// decides how to handle the message (Principles 1, 2).
package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/rs/zerolog"
)

// ─── Types ─────────────────────────────────────────────────────────

// MessageClass represents the type of user message.
type MessageClass string

const (
	ClassQuestion     MessageClass = "question"
	ClassInstruction  MessageClass = "instruction"
	ClassCommand      MessageClass = "command"
	ClassCorrection   MessageClass = "correction"
	ClassPreference   MessageClass = "preference_update"
	ClassFeedback     MessageClass = "feedback"
	ClassClarification MessageClass = "clarification"
	ClassCasual       MessageClass = "casual"
	ClassTaskRequest  MessageClass = "task_request"
)

// Intent represents what the user wants to achieve.
type Intent string

const (
	IntentAskInfo       Intent = "ask_information"
	IntentAskDocument   Intent = "ask_document"
	IntentSetPreference Intent = "set_preference"
	IntentCorrect       Intent = "correct_previous"
	IntentContinueTask  Intent = "continue_task"
	IntentNewTask       Intent = "new_task"
	IntentChitChat      Intent = "chitchat"
	IntentGiveCommand   Intent = "give_command"
	IntentClarify       Intent = "clarify"
	IntentFeedback      Intent = "feedback"
)

// Classification is the full result of analyzing a user message.
type Classification struct {
	MessageClass     MessageClass `json:"message_class"`
	Intent           Intent       `json:"intent"`
	IsInstruction    bool         `json:"is_instruction"`
	IsCorrection     bool         `json:"is_correction"`
	RequiresClarify  bool         `json:"requires_clarification"`
	ExtractedPrefs   []PrefUpdate `json:"extracted_preferences,omitempty"`
	Confidence       string       `json:"confidence"` // "high", "medium", "low"
	ContinuesPrevious bool       `json:"continues_previous"`
}

// PrefUpdate represents a detected preference change.
type PrefUpdate struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ─── Interface ─────────────────────────────────────────────────────

// IntentClassifier classifies user messages.
type IntentClassifier interface {
	// Classify analyzes a user message and returns classification.
	Classify(ctx context.Context, params ClassifyParams) (*Classification, error)

	// HeuristicOnly runs only the fast heuristic classifier (no LLM).
	HeuristicOnly(params ClassifyParams) *Classification
}

// ClassifyParams holds inputs for classification.
type ClassifyParams struct {
	UserText       string
	LastIntent     string // previous turn's intent
	LastClass      string // previous turn's class
	ActiveTask     string // current active task description
	TurnCount      int    // how many turns in this session
	HasActiveDoc   bool   // whether a document is currently focused
}

// ─── Implementation ────────────────────────────────────────────────

type intentClassifier struct {
	llm    *ollama.OllamaClient
	logger *zerolog.Logger
}

// NewIntentClassifier creates an IntentClassifier.
func NewIntentClassifier(llm *ollama.OllamaClient, logger *zerolog.Logger) IntentClassifier {
	return &intentClassifier{
		llm:    llm,
		logger: logger,
	}
}

const classificationPrompt = `You are a message classification agent. Analyze the user message and classify it.

CONTEXT:
- Previous intent: %s
- Previous class: %s
- Active task: %s
- Turn count: %d
- Has active document: %v

CLASSIFICATION RULES:
1. message_class: one of [question, instruction, command, correction, preference_update, feedback, clarification, casual, task_request]
2. intent: one of [ask_information, ask_document, set_preference, correct_previous, continue_task, new_task, chitchat, give_command, clarify, feedback]
3. is_instruction: true if user is changing HOW you should respond (language, tone, format, name, etc.)
4. is_correction: true if user is correcting a previous wrong answer
5. requires_clarification: true if message is ambiguous and could mean multiple things
6. extracted_preferences: array of {key, value} pairs if preferences detected. Valid keys: language, tone, verbosity, response_format, display_name
7. confidence: high/medium/low
8. continues_previous: true if this message is a follow-up to the previous task/topic

PREFERENCE DETECTION EXAMPLES:
- "Jawab dalam bahasa Inggris" → {key: "language", value: "en"}
- "Answer in English" → {key: "language", value: "en"}
- "Jawab singkat saja" / "Be concise" → {key: "verbosity", value: "concise"}
- "Jelaskan detail" / "Explain in detail" → {key: "verbosity", value: "detailed"}
- "Panggil aku Michael" / "Call me Michael" → {key: "display_name", value: "Michael"}
- "Gunakan markdown" / "Use markdown" → {key: "response_format", value: "markdown"}
- "Pakai nada formal" / "Be formal" → {key: "tone", value: "formal"}
- "Santai saja" → {key: "tone", value: "casual"}

CORRECTION DETECTION:
- "Bukan, maksudku..." / "No, I meant..."
- "Salah, yang benar adalah..." / "Wrong, the correct one is..."
- "Koreksi: ..." / "Actually, ..."
- Any message that contradicts or fixes a previous assistant response

Return ONLY valid JSON, no other text:
{
  "message_class": "",
  "intent": "",
  "is_instruction": false,
  "is_correction": false,
  "requires_clarification": false,
  "extracted_preferences": [],
  "confidence": "",
  "continues_previous": false
}

USER MESSAGE: %s`

func (c *intentClassifier) Classify(ctx context.Context, params ClassifyParams) (*Classification, error) {
	start := time.Now()

	// Fast path: very short messages can be classified heuristically
	if result := c.heuristicClassify(params); result != nil {
		c.logger.Info().
			Str("class", string(result.MessageClass)).
			Str("intent", string(result.Intent)).
			Str("method", "heuristic").
			Dur("ms", time.Since(start)).
			Msg("[classifier] classified")
		return result, nil
	}

	prompt := fmt.Sprintf(classificationPrompt,
		params.LastIntent,
		params.LastClass,
		params.ActiveTask,
		params.TurnCount,
		params.HasActiveDoc,
		params.UserText,
	)

	messages := []ollama.ChatMessage{
		{Role: "user", Content: prompt},
	}

	reply, err := c.llm.ChatJSON(ctx, messages)
	if err != nil {
		c.logger.Warn().Err(err).Msg("[classifier] LLM classification failed, using fallback")
		return c.fallbackClassify(params), nil
	}

	reply = strings.TrimSpace(reply)
	if reply == "" || reply == "{}" {
		return c.fallbackClassify(params), nil
	}

	var result Classification
	if err := json.Unmarshal([]byte(reply), &result); err != nil {
		c.logger.Warn().Err(err).Str("reply", reply).Msg("[classifier] JSON parse failed")
		return c.fallbackClassify(params), nil
	}

	c.logger.Info().
		Str("class", string(result.MessageClass)).
		Str("intent", string(result.Intent)).
		Bool("is_instruction", result.IsInstruction).
		Bool("is_correction", result.IsCorrection).
		Int("prefs_detected", len(result.ExtractedPrefs)).
		Str("confidence", result.Confidence).
		Dur("ms", time.Since(start)).
		Msg("[classifier] classified")

	return &result, nil
}

func (c *intentClassifier) HeuristicOnly(params ClassifyParams) *Classification {
	if result := c.heuristicClassify(params); result != nil {
		return result
	}
	return c.fallbackClassify(params)
}

// heuristicClassify handles obvious cases without LLM call.
func (c *intentClassifier) heuristicClassify(params ClassifyParams) *Classification {
	text := strings.TrimSpace(strings.ToLower(params.UserText))

	// Greeting and casual patterns
	greetings := []string{"hi", "halo", "hello", "hey", "hai", "selamat pagi", "selamat siang", "selamat malam", "pagi", "siang", "malam", "aamiiinn", "amin", "amiiin"}
	for _, g := range greetings {
		if text == g || (strings.HasPrefix(text, g+" ") && len(text) < 30) {
			return &Classification{
				MessageClass: ClassCasual,
				Intent:       IntentChitChat,
				Confidence:   "high",
			}
		}
	}
	
	// Thanks patterns
	thanks := []string{"makasih", "makasih lo", "terima kasih", "thanks", "thank you", "makasih ya", "terimakasih"}
	for _, t := range thanks {
		if text == t || (strings.HasPrefix(text, t+" ") && len(text) < 30) {
			return &Classification{
				MessageClass: ClassCasual,
				Intent:       "thanks",
				Confidence:   "high",
			}
		}
	}

	// Preference: language switch
	langPatterns := map[string]string{
		"jawab dalam bahasa inggris": "en",
		"answer in english":          "en",
		"use english":                "en",
		"jawab dalam bahasa indonesia": "id",
		"pakai bahasa indonesia":     "id",
	}
	for pattern, lang := range langPatterns {
		if strings.Contains(text, pattern) {
			return &Classification{
				MessageClass:  ClassPreference,
				Intent:        IntentSetPreference,
				IsInstruction: true,
				Confidence:    "high",
				ExtractedPrefs: []PrefUpdate{
					{Key: "language", Value: lang},
				},
			}
		}
	}

	// Preference: verbosity
	if strings.Contains(text, "jawab singkat") || strings.Contains(text, "be concise") || text == "singkat" {
		return &Classification{
			MessageClass:  ClassPreference,
			Intent:        IntentSetPreference,
			IsInstruction: true,
			Confidence:    "high",
			ExtractedPrefs: []PrefUpdate{
				{Key: "verbosity", Value: "concise"},
			},
		}
	}

	if strings.Contains(text, "jelaskan detail") || strings.Contains(text, "explain in detail") || strings.Contains(text, "jelaskan secara detail") {
		return &Classification{
			MessageClass:  ClassPreference,
			Intent:        IntentSetPreference,
			IsInstruction: true,
			Confidence:    "high",
			ExtractedPrefs: []PrefUpdate{
				{Key: "verbosity", Value: "detailed"},
			},
		}
	}

	// Preference: display name
	namePatterns := []string{"panggil aku ", "panggil saya ", "call me ", "nama saya ", "my name is "}
	for _, p := range namePatterns {
		if strings.Contains(text, p) {
			idx := strings.Index(text, p)
			name := strings.TrimSpace(text[idx+len(p):])
			if strings.HasPrefix(name, "adalah ") {
				name = strings.TrimSpace(name[7:])
			}
			// Clean trailing punctuation
			name = strings.TrimRight(name, ".,!?;:")
			if name != "" {
				return &Classification{
					MessageClass:  ClassPreference,
					Intent:        IntentSetPreference,
					IsInstruction: true,
					Confidence:    "high",
					ExtractedPrefs: []PrefUpdate{
						{Key: "display_name", Value: name},
					},
				}
			}
		}
	}

	// Preference: tone
	if strings.Contains(text, "nada formal") || strings.Contains(text, "be formal") || strings.Contains(text, "pakai formal") {
		return &Classification{
			MessageClass:  ClassPreference,
			Intent:        IntentSetPreference,
			IsInstruction: true,
			Confidence:    "high",
			ExtractedPrefs: []PrefUpdate{
				{Key: "tone", Value: "formal"},
			},
		}
	}

	// Preference: markdown
	if strings.Contains(text, "gunakan markdown") || strings.Contains(text, "use markdown") || strings.Contains(text, "pakai markdown") {
		return &Classification{
			MessageClass:  ClassPreference,
			Intent:        IntentSetPreference,
			IsInstruction: true,
			Confidence:    "high",
			ExtractedPrefs: []PrefUpdate{
				{Key: "response_format", Value: "markdown"},
			},
		}
	}

	return nil // not heuristically classifiable
}

// fallbackClassify provides a safe default when LLM fails.
func (c *intentClassifier) fallbackClassify(params ClassifyParams) *Classification {
	text := strings.TrimSpace(params.UserText)

	// Question heuristic: ends with ? or starts with question words
	isQuestion := strings.HasSuffix(text, "?")
	questionStarters := []string{"apa", "siapa", "kapan", "dimana", "mengapa", "kenapa", "bagaimana", "berapa",
		"what", "who", "when", "where", "why", "how", "which", "apakah", "bisakah", "can", "could", "is", "are", "do", "does", "deskripsikan"}
	lower := strings.ToLower(text)
	for _, q := range questionStarters {
		if strings.HasPrefix(lower, q+" ") || strings.HasPrefix(lower, q+",") {
			isQuestion = true
			break
		}
	}

	class := ClassCasual
	intent := IntentChitChat

	if isQuestion {
		class = ClassQuestion
		intent = IntentAskInfo
		if params.HasActiveDoc {
			intent = IntentAskDocument
		}
	} else if len(text) > 50 {
		class = ClassTaskRequest
		intent = IntentNewTask
	}

	// Continuity: if previous task exists and this is short, likely continuation
	if params.ActiveTask != "" && params.TurnCount > 1 {
		return &Classification{
			MessageClass:      class,
			Intent:            IntentContinueTask,
			ContinuesPrevious: true,
			Confidence:        "low",
		}
	}

	return &Classification{
		MessageClass: class,
		Intent:       intent,
		Confidence:   "low",
	}
}
