package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/conversation"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/rs/zerolog"
)

// ─── Interface ─────────────────────────────────────────────────────

// Interceptor processes behavioral instructions and corrections.
// It updates conversation state and preferences without forwarding
// the message to the LLM for a full response (Principle 4, 8).
type Interceptor interface {
	// Process handles a classified message. Returns:
	// - intercepted=true if message was a pure instruction (no LLM response needed)
	// - ackMessage: short acknowledgement to send back (empty if not intercepted)
	// - error
	Process(ctx context.Context, params InterceptParams) (intercepted bool, ackMessage string, err error)
}

// InterceptParams holds inputs for instruction interception.
type InterceptParams struct {
	UserID    string
	SessionID string
	UserText  string
	Class     *Classification
	State     *domain.StateData
}

// ─── Implementation ────────────────────────────────────────────────

type interceptor struct {
	stateManager conversation.StateManager
	prefManager  conversation.PreferenceManager
	logger       *zerolog.Logger
	llm          *ollama.OllamaClient
	dashRepo     repository.DashboardRepository
}

// NewInterceptor creates an Interceptor.
func NewInterceptor(
	stateManager conversation.StateManager,
	prefManager conversation.PreferenceManager,
	logger *zerolog.Logger,
	llm *ollama.OllamaClient,
	dashRepo repository.DashboardRepository,
) Interceptor {
	return &interceptor{
		stateManager: stateManager,
		prefManager:  prefManager,
		logger:       logger,
		llm:          llm,
		dashRepo:     dashRepo,
	}
}

func (i *interceptor) Process(ctx context.Context, params InterceptParams) (bool, string, error) {
	start := time.Now()

	if params.Class == nil {
		return false, "", nil
	}

	// Not an instruction or correction → not intercepted
	if !params.Class.IsInstruction && !params.Class.IsCorrection {
		return false, "", nil
	}

	// Handle preference updates
	if params.Class.IsInstruction && len(params.Class.ExtractedPrefs) > 0 {
		ack, err := i.applyPreferences(ctx, params)
		if err != nil {
			return false, "", err
		}

		i.logger.Info().
			Str("user_id", params.UserID).
			Int("prefs_applied", len(params.Class.ExtractedPrefs)).
			Dur("ms", time.Since(start)).
			Msg("[interceptor] preferences applied")

		return true, ack, nil
	}

	// Handle correction
	if params.Class.IsCorrection {
		err := i.handleCorrection(ctx, params)
		if err != nil {
			return false, "", err
		}

		i.logger.Info().
			Str("user_id", params.UserID).
			Dur("ms", time.Since(start)).
			Msg("[interceptor] correction noted")

		// Corrections still need LLM response with corrected context
		// So we DON'T intercept — but we've updated the state
		return false, "", nil
	}

	return false, "", nil
}

// applyPreferences updates both session state and persistent preferences.
func (i *interceptor) applyPreferences(ctx context.Context, params InterceptParams) (string, error) {
	state := params.State
	if state == nil {
		defaultState := domain.DefaultStateData()
		state = &defaultState
	}

	var ackParts []string

	for _, pref := range params.Class.ExtractedPrefs {
		// Update persistent preference
		if err := i.prefManager.SetExplicit(ctx, params.UserID, pref.Key, pref.Value); err != nil {
			i.logger.Warn().Err(err).
				Str("key", pref.Key).
				Msg("[interceptor] failed to persist preference")
		}

		// Update session state
		switch pref.Key {
		case domain.PrefKeyLanguage:
			state.Language = pref.Value
			ackParts = append(ackParts, i.langAck(pref.Value))
		case domain.PrefKeyTone:
			state.Tone = pref.Value
			ackParts = append(ackParts, fmt.Sprintf("Tone → %s", pref.Value))
		case domain.PrefKeyVerbosity:
			state.Verbosity = pref.Value
			ackParts = append(ackParts, i.verbosityAck(pref.Value))
		case domain.PrefKeyResponseFormat:
			state.ResponseFormat = pref.Value
			ackParts = append(ackParts, fmt.Sprintf("Format → %s", pref.Value))
		case domain.PrefKeyDisplayName:
			state.UserDisplayName = pref.Value
			ackParts = append(ackParts, fmt.Sprintf("Baik, saya panggil kamu %s 👋", pref.Value))
		default:
			// Store as custom directive
			if state.CustomDirectives == nil {
				state.CustomDirectives = make(map[string]string)
			}
			state.CustomDirectives[pref.Key] = pref.Value
			ackParts = append(ackParts, fmt.Sprintf("%s → %s", pref.Key, pref.Value))
		}
	}

	// Persist updated state
	if err := i.stateManager.Update(ctx, params.UserID, params.SessionID, state,
		string(IntentSetPreference), string(ClassPreference), ""); err != nil {
		return "", fmt.Errorf("update state after preference: %w", err)
	}

	if len(ackParts) == 0 {
		return "✅", nil
	}
	
	// Generate natural acknowledgement using LLM
	ackMsg := i.generateLLMAck(ctx, params.UserText, ackParts)
	if ackMsg != "" {
		return ackMsg, nil
	}

	if len(ackParts) == 1 {
		return "✅ " + ackParts[0], nil
	}
	return "✅ " + joinAcks(ackParts), nil
}

func (i *interceptor) generateLLMAck(ctx context.Context, userText string, ackParts []string) string {
	if i.llm == nil {
		return ""
	}

	personaName := "Assistant"
	personaDesc := "Saya adalah asisten AI yang cerdas dan efisien."
	voiceGuide := "Gunakan bahasa natural dan hangat, TAPI tidak berlebihan. Jangan gunakan filler exclamation di awal kalimat kecuali relevan."
	
	if i.dashRepo != nil {
		if conf, err := i.dashRepo.GetAgentConfigByKey(ctx, "persona_name"); err == nil && conf != nil {
			json.Unmarshal([]byte(conf.ValueJSON), &personaName)
		}
		if conf, err := i.dashRepo.GetAgentConfigByKey(ctx, "persona_description"); err == nil && conf != nil {
			json.Unmarshal([]byte(conf.ValueJSON), &personaDesc)
		}
		if conf, err := i.dashRepo.GetAgentConfigByKey(ctx, "voice_guidelines"); err == nil && conf != nil {
			json.Unmarshal([]byte(conf.ValueJSON), &voiceGuide)
		}
	}

	systemPrompt := fmt.Sprintf(`ANDA ADALAH: %s
%s
Pedoman Gaya Bahasa:
%s

TUGAS ANDA:
Pengguna baru saja memberikan instruksi atau informasi pribadi. Sistem telah mencatat perubahan berikut:
%s

Berikan respons SINGKAT (maksimal 1-2 kalimat) untuk mengonfirmasi bahwa Anda telah mengingat informasi tersebut. 
Berespons secara natural dan hangat sesuai pedoman gaya bahasa Anda. Jangan mengulangi format sistem (jangan gunakan bullet point atau tanda panah). Langsung berikan balasan percakapan.`, personaName, personaDesc, voiceGuide, joinAcks(ackParts))

	messages := []ollama.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userText},
	}
	
	reply, err := i.llm.Chat(ctx, messages)
	if err != nil {
		i.logger.Warn().Err(err).Msg("[interceptor] failed to generate LLM ack, falling back to static")
		return ""
	}
	return strings.TrimSpace(reply)
}

// handleCorrection marks correction in state.
func (i *interceptor) handleCorrection(ctx context.Context, params InterceptParams) error {
	state := params.State
	if state == nil {
		defaultState := domain.DefaultStateData()
		state = &defaultState
	}

	state.LastCorrectionAt = time.Now().UTC().Format(time.RFC3339)

	return i.stateManager.Update(ctx, params.UserID, params.SessionID, state,
		string(IntentCorrect), string(ClassCorrection), "")
}

func (i *interceptor) langAck(lang string) string {
	switch lang {
	case "en":
		return "Language → English"
	case "id":
		return "Bahasa → Indonesia"
	default:
		return fmt.Sprintf("Language → %s", lang)
	}
}

func (i *interceptor) verbosityAck(v string) string {
	switch v {
	case "concise":
		return "Mode → ringkas"
	case "detailed":
		return "Mode → detail"
	default:
		return fmt.Sprintf("Verbosity → %s", v)
	}
}

func joinAcks(parts []string) string {
	result := ""
	for idx, p := range parts {
		if idx > 0 {
			result += " | "
		}
		result += p
	}
	return result
}
