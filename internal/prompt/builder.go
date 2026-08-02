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
const maxContextChars = 4000

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

	// Behavioral framework injection
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{behavioral_rules}}", b.behavioralRules(sc))

	// Memory
	if sc.MemoryContext != "" {
		sysPrompt = strings.ReplaceAll(sysPrompt, "{{user_facts}}", sc.MemoryContext)
	} else {
		sysPrompt = strings.ReplaceAll(sysPrompt, "{{user_facts}}", "")
	}

	// RAG
	if sc.HasRAG {
		ragCtx := b.truncateRAG(sc.RAGContext)
		sysPrompt = strings.ReplaceAll(sysPrompt, "{{rag_context}}", ragCtx)
		if len(sc.RAGSources) > 0 {
			sysPrompt += "\n\nSources: " + strings.Join(sc.RAGSources, ", ")
		}
	} else {
		sysPrompt = strings.ReplaceAll(sysPrompt, "{{rag_context}}", "")
	}

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

	// Memory
	if sc.MemoryContext != "" {
		sb.WriteString(sc.MemoryContext)
		sb.WriteString("\n\n")
	}

	// RAG
	if sc.HasRAG {
		sb.WriteString("Konteks dokumen relevan:\n")
		sb.WriteString(b.truncateRAG(sc.RAGContext))
		sb.WriteString("\n\n")
		if len(sc.RAGSources) > 0 {
			sb.WriteString("Sources: " + strings.Join(sc.RAGSources, ", ") + "\n\n")
		}
	}

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

	if sc.MemoryContext != "" {
		sb.WriteString(sc.MemoryContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString("ATURAN PENGGUNAAN KONTEKS DOKUMEN:\n")
	sb.WriteString("1. Dokumen yang SEDANG AKTIF adalah dokumen dengan metadata berikut:\n")
	sb.WriteString(sc.MetadataBlock)
	sb.WriteString("\n")
	sb.WriteString("2. JIKA pengguna menanyakan informasi yang spesifik mengenai isi dokumen ini, jawablah BERDASARKAN potongan teks (chunks) di bawah ini. JANGAN mengarang informasi yang tidak ada di dalam chunk dokumen.\n\n")
	sb.WriteString("3. DILARANG KERAS mengucapkan 'Maaf saya hanya menerima teks dan tidak dapat membaca dokumen', 'Saya tidak bisa melihat dokumen', dsb. Anda SUDAH BISA membaca dokumen melalui konteks teks yang disuntikkan oleh sistem di bawah ini. Bertindaklah seolah Anda membaca dokumen tersebut secara langsung.\n\n")
	sb.WriteString("4. JIKA pengguna menanyakan pertanyaan umum (general knowledge) atau di luar konteks dokumen, JAWABLAH secara natural menggunakan pengetahuan umum Anda yang luas (seperti ChatGPT/Gemini/Claude). Anda TIDAK dibatasi hanya pada dokumen untuk pertanyaan umum. Namun, jika ada kaitan yang menarik dengan dokumen aktif, Anda boleh menyebutkannya secara sekilas.\n\n")
	sb.WriteString(fmt.Sprintf("5. Jika pengguna secara spesifik merujuk pada dokumen lain (document_id berbeda dari %s), beri tahu mereka bahwa dokumen yang sedang aktif saat ini adalah dokumen ini.\n\n", sc.TargetDocID))

	sb.WriteString("KONTEN UNTUK DIJAWAB:\n")
	if sc.HasRAG {
		sb.WriteString(b.truncateRAG(sc.RAGContext))
	} else {
		sb.WriteString("(Tidak ada chunk yang ditemukan)")
	}

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

	sb.WriteString("ATURAN PERILAKU (WAJIB DIPATUHI):\n")

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

	// Truthfulness & consistency (Principles 7, 9, 11, 17)
	sb.WriteString("4. JANGAN PERNAH mengarang informasi pribadi pengguna yang tidak pernah disebutkan.\n")
	sb.WriteString("5. Jika tidak tahu jawabannya, katakan tidak tahu. Jangan menebak.\n")
	sb.WriteString("6. Jika pengguna mengoreksi Anda, terima koreksi dan jangan ulangi kesalahan yang sama.\n")
	sb.WriteString("7. Jika pengguna menanyakan sesuatu tentang DIRINYA SENDIRI (nama, preferensi, dsb) dan informasi tersebut TIDAK ADA di <informasi_latar_belakang> atau User Preferences, JAWAB dengan natural: sampaikan bahwa Anda belum punya info tersebut, dan tawarkan agar pengguna memberitahu supaya Anda ingat. JANGAN gunakan bahasa penolakan/disclaimer privasi seperti 'saya tidak bisa membantu mencari informasi tentang Anda' — Anda BUKAN menolak, Anda hanya belum tahu.\n")

	// Voice Guidelines
	if sc.Persona.VoiceGuidelines != "" {
		sb.WriteString("8. Pedoman Gaya Bahasa (Voice Guidelines):\n")
		sb.WriteString(sc.Persona.VoiceGuidelines)
		sb.WriteString("\n")
	}

	// Custom directives from user
	if sc.State != nil && len(sc.State.CustomDirectives) > 0 {
		sb.WriteString("9. Instruksi khusus dari pengguna:\n")
		i := 0
		for k, v := range sc.State.CustomDirectives {
			sb.WriteString(fmt.Sprintf("   - %s: %s\n", k, v))
			i++
		}
	}

	// User display name
	if sc.State != nil && sc.State.UserDisplayName != "" {
		sb.WriteString(fmt.Sprintf("10. Nama pengguna adalah '%s' (Gunakan secara natural jika perlu, tidak perlu selalu disebutkan).\n", sc.State.UserDisplayName))
	}

	return sb.String()
}
