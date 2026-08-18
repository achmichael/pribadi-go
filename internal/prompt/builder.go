// Package prompt builds system prompts for LLM inference.
// It encodes the behavioral framework principles into the prompt
// and merges dynamic context (RAG, memory, state, preferences).
package prompt

import (
	"fmt"
	"strings"

	"github.com/achmichael/pribadi-go/internal/domain"
)

// maxContextChars caps RAG context injected into prompt.
const maxContextChars = 2000

// Persona holds agent identity fetched from dashboard config.
type Persona struct {
	Name            string
	Description     string
	Tone            string
	VoiceGuidelines string
}

// SessionContext holds all dynamic context for a single turn.
type SessionContext struct {
	// State
	State     *domain.StateData
	TurnCount int

	// Persona (from dashboard agent_config)
	Persona Persona

	// Template (from dashboard agent_config)
	PromptTemplate string

	// RAG
	RAGContext string
	HasRAG     bool
	RAGSources []string

	// Memory
	MemoryContext string

	// Document isolation
	MetadataBlock string
	TargetDocID   string

	// User preferences as readable block
	PreferencesBlock string
}

// Builder constructs system prompts.
type Builder struct{}

// NewBuilder creates a prompt Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build assembles the full system prompt from all context sources.
func (b *Builder) Build(sc SessionContext) string {
	if sc.TargetDocID != "" && sc.MetadataBlock != "" {
		return b.buildDocumentIsolatedPrompt(sc)
	}
	return b.buildConversationalPrompt(sc)
}

// buildConversationalPrompt handles normal (non-document-focused) messages.
func (b *Builder) buildConversationalPrompt(sc SessionContext) string {
	// Start with template if available, otherwise build from scratch
	if sc.PromptTemplate != "" {
		return b.buildFromTemplate(sc)
	}
	return b.buildFromScratch(sc)
}

// buildFromTemplate uses the dashboard-configured template with variable substitution.
func (b *Builder) buildFromTemplate(sc SessionContext) string {
	sysPrompt := sc.PromptTemplate

	sysPrompt = strings.ReplaceAll(sysPrompt, "{{persona_name}}", sc.Persona.Name)
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{persona_description}}", sc.Persona.Description)
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{tone_preference}}", b.resolveTone(sc))
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{voice_guidelines}}", sc.Persona.VoiceGuidelines)

	// Behavioral framework injection
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{behavioral_rules}}", b.behavioralRules(sc))

	// RAG placeholder removal (moved to user prompt)
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{rag_context}}", "")
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{user_facts}}", "")

	return sysPrompt
}

// buildFromScratch constructs prompt without a template.
func (b *Builder) buildFromScratch(sc SessionContext) string {
	var sb strings.Builder

	// Identity
	sb.WriteString(fmt.Sprintf("ANDA ADALAH: %s\n%s\n", sc.Persona.Name, sc.Persona.Description))
	sb.WriteString(fmt.Sprintf("Gaya Bahasa: %s\n\n", b.resolveTone(sc)))

	// Behavioral rules
	rules := b.behavioralRules(sc)
	if rules != "" {
		sb.WriteString(rules)
		sb.WriteString("\n\n")
	}
	
	// Few-Shot Examples
	sb.WriteString(b.fewShotExamples(sc))
	sb.WriteString("\n")

	// RAG and Memory omitted here. Moved to user prompt to avoid Lost in the Middle.
	return sb.String()
}

// buildDocumentIsolatedPrompt handles document-focused messages with isolation rules.
func (b *Builder) buildDocumentIsolatedPrompt(sc SessionContext) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ANDA ADALAH: %s\n%s\nGaya Bahasa: %s\n\n", sc.Persona.Name, sc.Persona.Description, b.resolveTone(sc)))

	// Behavioral rules
	rules := b.behavioralRules(sc)
	if rules != "" {
		sb.WriteString(rules)
		sb.WriteString("\n\n")
	}
	
	// Few-Shot Examples
	sb.WriteString(b.fewShotExamples(sc))
	sb.WriteString("\n")

	sb.WriteString("ATURAN PENGGUNAAN KONTEKS DOKUMEN:\n")
	sb.WriteString("1. Dokumen yang SEDANG AKTIF adalah dokumen dengan metadata berikut:\n")
	sb.WriteString(sc.MetadataBlock)
	sb.WriteString("\n")
	sb.WriteString("2. Jika pengguna menanyakan informasi spesifik mengenai isi dokumen ini, jawablah berdasarkan potongan teks (chunks) di bawah ini.\n\n")
	sb.WriteString("3. Anda sudah memiliki kemampuan membaca dokumen melalui konteks teks di bawah ini. Bertindaklah seolah Anda membaca dokumen tersebut secara langsung.\n\n")
	sb.WriteString("4. Jika pengguna menanyakan pertanyaan umum (general knowledge) atau di luar konteks dokumen, jawablah secara natural menggunakan pengetahuan umum Anda yang luas. Jika ada kaitan yang menarik dengan dokumen aktif, Anda boleh menyebutkannya.\n\n")
	sb.WriteString(fmt.Sprintf("5. Jika pengguna secara spesifik merujuk pada dokumen lain (document_id berbeda dari %s), beri tahu mereka bahwa dokumen yang sedang aktif saat ini adalah dokumen ini.\n\n", sc.TargetDocID))

	// RAG omitted here, moved to User Prompt.
	sb.WriteString("Silakan merujuk pada konteks dokumen yang dilampirkan bersama pertanyaan pengguna.\n")
	
	return sb.String()
}

// fewShotExamples returns dynamic examples to set tone and behavior.
func (b *Builder) fewShotExamples(sc SessionContext) string {
	var sb strings.Builder
	sb.WriteString("CONTOH PERCAKAPAN (Untuk referensi gaya bahasa):\n")
	
	// Default casual Indonesian examples
	sb.WriteString("User: bro lu bisa code go ga?\n")
	sb.WriteString("Assistant: Bisa dong! Mau bikin apa pake Go? API, CLI, atau mikorservis nih?\n\n")
	
	sb.WriteString("User: tolong jelasin apa itu docker\n")
	sb.WriteString("Assistant: Gampangnya, Docker itu kayak kontainer pengiriman tapi buat software. Daripada ribet mikirin OS beda-beda, code lu dibungkus di satu kotak (container) bareng semua yang dibutuhin, jadi pasti jalan di mana aja.\n\n")
	
	sb.WriteString("User: nama gw budi\n")
	sb.WriteString("Assistant: Oke, salam kenal Budi! Ada yang bisa dibantu hari ini?\n\n")

	return sb.String()
}

// resolveTone returns effective tone: session state > persona default.
func (b *Builder) resolveTone(sc SessionContext) string {
	if sc.State != nil && sc.State.Tone != "" {
		return sc.State.Tone
	}
	if sc.Persona.Tone != "" {
		return sc.Persona.Tone
	}
	return "casual"
}

// truncateRAG caps RAG context to maxContextChars.
func (b *Builder) truncateRAG(ragCtx string) string {
	if len(ragCtx) > maxContextChars {
		return ragCtx[:maxContextChars] + "\n[...context truncated...]"
	}
	return ragCtx
}

// behavioralRules returns the behavioral framework rules block
// encoded into the system prompt, adapted to current state.
func (b *Builder) behavioralRules(sc SessionContext) string {
	var sb strings.Builder

	sb.WriteString("ATURAN PERILAKU:\n")

	// Language
	lang := "Bahasa Indonesia"
	if sc.State != nil {
		switch sc.State.Language {
		case "en":
			lang = "English"
		case "id":
			lang = "Bahasa Indonesia"
		default:
			if sc.State.Language != "" {
				lang = sc.State.Language
			}
		}
	}
	sb.WriteString(fmt.Sprintf("1. Jawab dalam %s kecuali pengguna meminta bahasa lain.\n", lang))

	// Verbosity
	verbosity := "normal"
	if sc.State != nil && sc.State.Verbosity != "" {
		verbosity = sc.State.Verbosity
	}
	switch verbosity {
	case "concise":
		sb.WriteString("2. Jawab SINGKAT dan PADAT. Hindari penjelasan panjang.\n")
	case "detailed":
		sb.WriteString("2. Jawab DETAIL dan LENGKAP. Jelaskan step-by-step jika perlu.\n")
	default:
		sb.WriteString("2. Jawab dengan panjang yang proporsional terhadap kompleksitas pertanyaan.\n")
	}

	// Response format
	if sc.State != nil && sc.State.ResponseFormat == "markdown" {
		sb.WriteString("3. Gunakan format Markdown dalam respons.\n")
	} else if sc.State != nil && sc.State.ResponseFormat == "step_by_step" {
		sb.WriteString("3. Jelaskan jawaban dalam langkah-langkah bernomor.\n")
	}

	sb.WriteString("4. Bersikaplah seperti asisten AI cerdas dan natural. Jawab obrolan secara luwes, dan jawab pertanyaan teknis secara informatif.\n")
	sb.WriteString("5. Gunakan SEMUA konteks yang tersedia (Sejarah Percakapan, Memori, Fakta, Dokumen) secara mulus. Integrasikan informasi ke dalam obrolan secara natural layaknya ingatan sendiri.\n")
	sb.WriteString("6. Gunakan hanya informasi yang tersedia di konteks. Jika pengguna membagikan fakta baru (misal: kuliah/kerja di mana), cukup tanggapi obrolannya secara empatik.\n")
	sb.WriteString("7. Jawab santai jika informasi tidak ada. Contoh: \"Wah, saya kurang tahu soal itu.\"\n")
	sb.WriteString("8. JANGAN PERNAH mengawali kalimat dengan template basi seperti \"Terima kasih, [Nama]!\". Mulailah secara natural.\n")
	sb.WriteString("9. JANGAN memberikan pujian kosong (flattery) tanpa dasar yang jelas.\n")
	sb.WriteString("10. MIRROR TONE (Sesuaikan gaya bahasa): Jika pengguna santai, jawab santai. Jika pengguna formal, jawab formal.\n")
	sb.WriteString("11. JANGAN memaksakan pertanyaan di akhir kalimat. Bertanyalah HANYA JIKA benar-benar diperlukan untuk melanjutkan konteks.\n")

	// Voice Guidelines
	if sc.Persona.VoiceGuidelines != "" {
		sb.WriteString("12. Pedoman Gaya Bahasa (Voice Guidelines):\n")
		sb.WriteString(sc.Persona.VoiceGuidelines)
		sb.WriteString("\n")
	}

	// Custom directives from user
	if sc.State != nil && len(sc.State.CustomDirectives) > 0 {
		sb.WriteString("13. Instruksi khusus dari pengguna:\n")
		i := 0
		for k, v := range sc.State.CustomDirectives {
			sb.WriteString(fmt.Sprintf("   - %s: %s\n", k, v))
			i++
		}
	}

	// User display name
	if sc.State != nil && sc.State.UserDisplayName != "" {
		sb.WriteString(fmt.Sprintf("14. Anda sedang berbicara dengan '%s'. Panggil dia dengan namanya secara natural di tengah atau akhir kalimat sesekali.\n", sc.State.UserDisplayName))
	}

	return sb.String()
}
