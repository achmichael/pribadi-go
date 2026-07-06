package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/delivery/whatsapp"
	"github.com/achmichael/pribadi-go/internal/factmemory"
	"github.com/achmichael/pribadi-go/internal/identity"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/usecase/rag"
	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Orchestrator interface {
	Handle(ctx context.Context, msg whatsapp.IncomingMessage) error
}

type orchestrator struct {
	waClient      *whatsapp.Client
	transcription TranscriptionService
	extraction    ExtractionService
	ragIngest     rag.IngestionService
	ragRetrieve   rag.RetrievalService
	ollama        *ollama.OllamaClient
	repo          repository.Repository
	dashboardRepo repository.DashboardRepository
	memory        factmemory.MemoryManager
	resolver      *identity.Resolver
	refResolver   ReferenceResolver
	logger        *zerolog.Logger
}

// DocMeta represents structured metadata extracted from documents
type DocMeta struct {
	DocumentType    string `json:"document_type"`
	Title           string `json:"title"`
	Authors         string `json:"authors"`
	Institution     string `json:"institution"`
	PublicationYear string `json:"publication_year"`
	DOI             string `json:"doi"`
	Keywords        string `json:"keywords"`
	Abstract        string `json:"abstract"`
	Supervisor      string `json:"supervisor"`
	Advisor         string `json:"advisor"`
}

func NewOrchestrator(
	waClient *whatsapp.Client,
	transcription TranscriptionService,
	extraction ExtractionService,
	ragIngest rag.IngestionService,
	ragRetrieve rag.RetrievalService,
	ollamaClient *ollama.OllamaClient,
	repo repository.Repository,
	dashboardRepo repository.DashboardRepository,
	memory factmemory.MemoryManager,
	resolver *identity.Resolver,
	refResolver ReferenceResolver,
	logger *zerolog.Logger,
) Orchestrator {
	return &orchestrator{
		waClient:      waClient,
		transcription: transcription,
		extraction:    extraction,
		ragIngest:     ragIngest,
		ragRetrieve:   ragRetrieve,
		ollama:        ollamaClient,
		repo:          repo,
		dashboardRepo: dashboardRepo,
		memory:        memory,
		resolver:      resolver,
		refResolver:   refResolver,
		logger:        logger,
	}
}

// maxContextChars caps how much RAG context goes into the system prompt.
const maxContextChars = 4000

func (o *orchestrator) Handle(ctx context.Context, msg whatsapp.IncomingMessage) error {
	handleStart := time.Now()
	o.logger.Info().
		Str("msg_id", msg.ID).
		Str("sender", msg.SenderJID).
		Str("type", string(msg.MessageType)).
		Msg("[orchestrator] handle start")

	// ── 0. Resolve platform identity → internal UserID ─────────────
	userID, err := o.resolver.ResolveUserID(ctx, "whatsapp", msg.SenderJID)
	if err != nil {
		o.logger.Error().Err(err).
			Str("sender", msg.SenderJID).
			Msg("[orchestrator] user resolution failed")
		return fmt.Errorf("resolve user: %w", err)
	}

	var userText string

	// ── 1. Preprocessing ──────────────────────────────────────────────
	switch msg.MessageType {
	case whatsapp.MessageTypeVoiceNote:
		// 1a. Download audio
		audioMsg := msg.RawMessage.GetAudioMessage()
		if audioMsg == nil {
			return fmt.Errorf("no audio message found")
		}

		dlStart := time.Now()
		data, err := o.waClient.GetClient().Download(ctx, audioMsg)
		if err != nil {
			return err
		}
		o.logger.Info().
			Str("msg_id", msg.ID).
			Int("bytes", len(data)).
			Dur("duration_ms", time.Since(dlStart)).
			Msg("[orchestrator] media download done")

		// 1b. Transcribe
		txStart := time.Now()
		text, err := o.transcription.Transcribe(ctx, data)
		if err != nil {
			return err
		}
		o.logger.Info().
			Str("msg_id", msg.ID).
			Int("text_len", len(text)).
			Dur("duration_ms", time.Since(txStart)).
			Msg("[orchestrator] transcription done")

		userText = text

	case whatsapp.MessageTypeImage, whatsapp.MessageTypeDocument:
		// 1a. Download media
		var data []byte
		var dlErr error
		var mime string

		dlStart := time.Now()
		if msg.MessageType == whatsapp.MessageTypeImage {
			imgMsg := msg.RawMessage.GetImageMessage()
			if imgMsg != nil {
				data, dlErr = o.waClient.GetClient().Download(ctx, imgMsg)
				mime = imgMsg.GetMimetype()
			}
		} else {
			docMsg := msg.RawMessage.GetDocumentMessage()
			if docMsg != nil {
				data, dlErr = o.waClient.GetClient().Download(ctx, docMsg)
				mime = docMsg.GetMimetype()
			}
		}
		if dlErr != nil || data == nil {
			return fmt.Errorf("failed to download media: %v", dlErr)
		}
		o.logger.Info().
			Str("msg_id", msg.ID).
			Str("mime", mime).
			Int("bytes", len(data)).
			Dur("duration_ms", time.Since(dlStart)).
			Msg("[orchestrator] media download done")

		if mime == "" {
			mime = msg.MediaMimetype
		}

		// 1b. Extract text
		exStart := time.Now()
		text, err := o.extraction.ExtractText(ctx, data, mime)
		if err != nil {
			return err
		}
		o.logger.Info().
			Str("msg_id", msg.ID).
			Int("text_len", len(text)).
			Dur("duration_ms", time.Since(exStart)).
			Msg("[orchestrator] extraction done")

		// Debug log for extracted text (truncated to 1000 chars to avoid overwhelming the console)
		previewText := text
		if len(previewText) > 1000 {
			previewText = previewText[:1000] + "... (truncated)"
		}
		o.logger.Info().
			Str("msg_id", msg.ID).
			Str("extracted_text_preview", previewText).
			Msg("[orchestrator] extracted text content")

		if msg.MessageType == whatsapp.MessageTypeImage {
			userText = text
			if msg.Caption != "" {
				userText = fmt.Sprintf("%s\n\nUser caption: %s", text, msg.Caption)
			}
		} else {
			// Document: ingest into RAG, reply with confirmation only.

			// === EXTRACT METADATA VIA LLM ===
			metaStart := time.Now()
			metadata := map[string]string{"source_file": msg.ID}

			extractText := text
			if len(extractText) > 2000 {
				extractText = extractText[:2000]
			}

			metaPrompt := `ANDA ADALAH: DOCUMENT METADATA EXTRACTION AGENT

TUGAS ANDA:
Ekstrak metadata terstruktur dari potongan teks dokumen di bawah ini. Teks ini adalah hasil OCR/ekstraksi otomatis dari PDF dan SERING MENGANDUNG NOISE seperti: header jurnal, nomor volume/issue, kode artikel, running title dari artikel lain dalam volume yang sama, nomor halaman, watermark, atau metadata penerbit yang TIDAK BOLEH disalahartikan sebagai judul dokumen.

ATURAN WAJIB UNTUK MENENTUKAN "TITLE" (JUDUL):
1. Judul dokumen yang benar SELALU berada tepat SEBELUM daftar nama penulis (authors), bukan sebelum nama jurnal/header/volume.
2. JANGAN PERNAH mengambil teks yang terlihat seperti: nama jurnal, "Volume X Nomor Y", "JN-xxx"/kode artikel, ISSN, nama institusi penerbit, atau judul artikel lain yang mungkin muncul sebagai running header di bagian atas/bawah halaman.
3. Judul asli biasanya merupakan frasa nominal yang menjelaskan topik penelitian (metode, subjek, objek penelitian) — bukan kalimat administratif jurnal.
4. WAJIB lakukan cross-check internal sebelum finalisasi: setelah Anda mengekstrak "Title" dan "Abstract"/"Keywords", PERIKSA apakah keduanya secara tematik konsisten satu sama lain. Jika "Title" yang Anda temukan TIDAK relevan secara tema dengan "Abstract" atau "Keywords" yang Anda temukan di teks yang sama, maka "Title" tersebut SALAH — cari ulang kandidat judul lain di teks, atau jika benar-benar tidak ditemukan, kembalikan string kosong daripada memaksakan judul yang tidak konsisten.
5. Jika dalam satu potongan teks Anda menemukan LEBIH DARI SATU kandidat judul (misalnya satu di bagian atas sebagai header, satu lagi tepat sebelum nama penulis), PILIH yang posisinya TEPAT SEBELUM/BERDEKATAN dengan daftar nama penulis.

FORMAT OUTPUT:
Jawab HANYA dalam format JSON murni, tanpa markdown, tanpa basa-basi, tanpa penjelasan tambahan. Jika sebuah field tidak ditemukan dalam teks, isi dengan string kosong "" — JANGAN mengisi dengan tebakan atau placeholder.

{
  "document_type": "",
  "title": "",
  "authors": "",
  "institution": "",
  "publication_year": "",
  "doi": "",
  "keywords": "",
  "abstract": "",
  "supervisor": "",
  "advisor": ""
}

TEKS DOKUMEN UNTUK DIANALISIS:
` + extractText
			metaMessages := []ollama.ChatMessage{
				{Role: "user", Content: metaPrompt},
			}

			var parsedMeta DocMeta
			var metaJSON string
			metaReply, err := o.ollama.ChatJSON(ctx, metaMessages)
			if err == nil {
				cleanReply := strings.TrimSpace(metaReply)
				if strings.HasPrefix(cleanReply, "```json") {
					cleanReply = strings.TrimPrefix(cleanReply, "```json")
					cleanReply = strings.TrimSuffix(cleanReply, "```")
				} else if strings.HasPrefix(cleanReply, "```") {
					cleanReply = strings.TrimPrefix(cleanReply, "```")
					cleanReply = strings.TrimSuffix(cleanReply, "```")
				}
				cleanReply = strings.TrimSpace(cleanReply)

				if err := json.Unmarshal([]byte(cleanReply), &parsedMeta); err == nil {
					metadata["Title"] = parsedMeta.Title
					metadata["Author"] = parsedMeta.Authors
					metadata["DocumentType"] = parsedMeta.DocumentType
					metadata["Institution"] = parsedMeta.Institution
					metadata["PublicationYear"] = parsedMeta.PublicationYear

					metaBytes, _ := json.Marshal(parsedMeta)
					metaJSON = string(metaBytes)
				} else {
					o.logger.Warn().Err(err).Str("reply", cleanReply).Msg("[orchestrator] failed to parse JSON metadata")
				}
			} else {
				o.logger.Warn().Err(err).Msg("[orchestrator] metadata extraction failed")
			}

			if metadata["Title"] == "" {
				metadata["Title"] = "Unknown"
			}
			if metadata["Author"] == "" {
				metadata["Author"] = "Unknown"
			}
			if metaJSON == "" {
				metaJSON = "{}"
			}

			o.logger.Info().
				Str("msg_id", msg.ID).
				Dur("duration_ms", time.Since(metaStart)).
				Msg("[orchestrator] LLM metadata extraction done")

			docID := uuid.New().String()
			metadata["document_id"] = docID

			ingStart := time.Now()
			count, _ := o.ragIngest.IngestText(ctx, text, metadata)
			o.logger.Info().
				Str("msg_id", msg.ID).
				Int("chunks_stored", count).
				Int("text_len", len(text)).
				Dur("duration_ms", time.Since(ingStart)).
				Msg("[orchestrator] RAG ingestion done")

			fileName := "unknown"
			if docMsg := msg.RawMessage.GetDocumentMessage(); docMsg != nil && docMsg.GetFileName() != "" {
				fileName = docMsg.GetFileName()
			}

			errDoc := o.repo.InsertUserDocument(ctx, repository.InsertUserDocumentParams{
				ID:            docID,
				UserID:        userID,
				PlatformMsgID: msg.ID,
				FileName:      fileName,
				Title:         metadata["Title"],
				Author:        metadata["Author"],
				MetadataJSON:  metaJSON,
			})
			if errDoc != nil {
				o.logger.Warn().Err(errDoc).Msg("[orchestrator] failed to save document metadata to sqlite")
			} else {
				o.logger.Info().Str("doc_id", docID).Msg("[orchestrator] saved document metadata to sqlite")
			}

			sendStart := time.Now()
			summary := fmt.Sprintf("✅ Dokumen berhasil diproses.\n📄 %d bagian disimpan ke memori.\n\nKamu bisa langsung bertanya tentang isi dokumen ini.", count)
			err = o.waClient.SendText(ctx, msg.SenderJID, summary)
			o.logger.Info().
				Str("msg_id", msg.ID).
				Dur("duration_ms", time.Since(sendStart)).
				Dur("total_ms", time.Since(handleStart)).
				Msg("[orchestrator] WA send done (document)")
			return err
		}

	default:
		userText = msg.TextContent
	}

	o.logger.Info().
		Str("msg_id", msg.ID).
		Int("user_text_len", len(userText)).
		Dur("preprocess_ms", time.Since(handleStart)).
		Msg("[orchestrator] preprocessing complete, starting retrieval + memory")

	sessionID := "default"

	// ── 1.5 Reference Resolution ────────────────────────────────────
	ragQuery := userText
	targetDocID := ""
	metadataBlock := ""
	if msg.MessageType == whatsapp.MessageTypeText || msg.MessageType == whatsapp.MessageTypeVoiceNote {
		ragQuery, targetDocID = o.refResolver.ResolveQuery(ctx, userID, userText, sessionID)
		if targetDocID != "" {
			doc, err := o.repo.GetUserDocumentByID(ctx, targetDocID)
			if err == nil && doc != nil {
				metadataJSON := doc.MetadataJSON
				if metadataJSON == "" || metadataJSON == "{}" {
					basicMeta := map[string]string{
						"title":  doc.Title,
						"author": doc.Author,
					}
					bBytes, _ := json.Marshal(basicMeta)
					metadataJSON = string(bBytes)
				}

				var b strings.Builder
				b.WriteString(fmt.Sprintf("--- DOCUMENT METADATA (AKTIF: document_id=%s) ---\n", doc.ID))
				b.WriteString(metadataJSON)
				b.WriteString("\n--- END METADATA ---\n")
				metadataBlock = b.String()

				// --- Query Contextualization Agent ---
				if doc.MetadataJSON != "" && doc.MetadataJSON != "{}" {
					contextualizePrompt := fmt.Sprintf(`ANDA ADALAH: QUERY CONTEXTUALIZATION AGENT untuk sistem RAG multilingual.

KONTEKS:
Sistem ini menyimpan dokumen yang isinya bisa berbahasa apa saja (Inggris, Indonesia, dll), namun pengguna bisa bertanya dalam bahasa apa saja juga. Embedding model yang dipakai (nomic-embed-text) memiliki kemampuan cross-lingual retrieval yang lemah — artinya query Bahasa Indonesia kemungkinan tidak akan menemukan chunk relevan yang ditulis dalam Bahasa Inggris, dan sebaliknya.

TUGAS ANDA:
Anda menerima query asli dari pengguna beserta metadata dokumen aktif (jika ada, dari SQLite). Tugas Anda BUKAN menjawab pertanyaan, tetapi menghasilkan SATU query pencarian semantik yang dioptimalkan untuk retrieval, dengan ATURAN:

1. Deteksi bahasa asli query pengguna.
2. Jika tersedia metadata dokumen aktif (title, keywords, abstract) dan bahasa metadata tersebut BERBEDA dari bahasa query pengguna, gabungkan inti pertanyaan pengguna dengan istilah-istilah kunci (entity/topik) dari metadata tersebut ke dalam query akhir — agar query memiliki representasi leksikal di KEDUA bahasa.
3. JANGAN menerjemahkan seluruh kalimat pengguna secara kaku. Cukup perkaya query dengan istilah kunci yang relevan dari metadata dokumen, dalam bahasa aslinya.
4. Jika tidak ada metadata dokumen aktif yang relevan, kembalikan query asli tanpa perubahan.
5. Output HANYA berupa query hasil akhir dalam bentuk teks polos, tanpa penjelasan, tanpa markdown.

INPUT:
Query pengguna: "%s"
Metadata dokumen aktif (JSON): %s`, userText, doc.MetadataJSON)

					ctxMsgs := []ollama.ChatMessage{{Role: "user", Content: contextualizePrompt}}
					enrichedQuery, errCtx := o.ollama.Chat(ctx, ctxMsgs)
					if errCtx == nil {
						ragQuery = strings.TrimSpace(enrichedQuery)
						o.logger.Info().Str("original_query", userText).Str("enriched_query", ragQuery).Msg("[orchestrator] query contextualization successful")
					} else {
						o.logger.Warn().Err(errCtx).Msg("[orchestrator] query contextualization failed, falling back to original query")
					}
				}
				// ------------------------------------
			}
		}
	}

	// ── 2. Retrieval (RAG docs + fact memory in parallel) ──────────
	type ragResult struct {
		ctx rag.PromptContext
		err error
	}
	type memResult struct {
		context string
		err     error
	}

	ragCh := make(chan ragResult, 1)
	memCh := make(chan memResult, 1)

	go func() {
		retStart := time.Now()
		ctxInfo, err := o.ragRetrieve.Retrieve(ctx, ragQuery, targetDocID)
		o.logger.Info().
			Str("msg_id", msg.ID).
			Bool("has_results", ctxInfo.HasResults).
			Int("context_len", len(ctxInfo.Context)).
			Dur("ms", time.Since(retStart)).
			Msg("[orchestrator] RAG retrieval done")
		ragCh <- ragResult{ctx: ctxInfo, err: err}
	}()

	go func() {
		memStart := time.Now()
		memCtx, err := o.memory.PrefetchRelevant(ctx, userID, userText)
		o.logger.Info().
			Str("msg_id", msg.ID).
			Int("memory_len", len(memCtx)).
			Dur("ms", time.Since(memStart)).
			Msg("[orchestrator] memory prefetch done")
		memCh <- memResult{context: memCtx, err: err}
	}()

	ragRes := <-ragCh
	memRes := <-memCh

	ctxInfo := ragRes.ctx
	if ragRes.err != nil {
		o.logger.Warn().Err(ragRes.err).Msg("[orchestrator] RAG retrieval failed")
		ctxInfo = rag.PromptContext{HasResults: false}
	}

	memoryContext := ""
	if memRes.err != nil {
		o.logger.Warn().Err(memRes.err).Msg("[orchestrator] memory prefetch failed")
	} else {
		memoryContext = memRes.context
	}

	// ── 3. LLM Generation ─────────────────────────────────────────────
	systemPrompt := o.buildSystemPrompt(ctx, ctxInfo, memoryContext, metadataBlock, targetDocID)

	userMsgContent := userText
	if targetDocID != "" {
		userMsgContent = "PERTANYAAN PENGGUNA:\n" + userText
	}

	messages := []ollama.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsgContent},
	}

	o.logger.Info().
		Str("msg_id", msg.ID).
		Int("system_prompt_len", len(systemPrompt)).
		Int("user_text_len", len(userText)).
		Msg("[orchestrator] sending to Ollama")

	llmStart := time.Now()
	reply, err := o.ollama.Chat(ctx, messages)
	llmDur := time.Since(llmStart)
	if err != nil {
		o.logger.Error().Err(err).
			Str("msg_id", msg.ID).
			Dur("duration_ms", llmDur).
			Msg("[orchestrator] Ollama chat FAILED")
		return err
	}
	o.logger.Info().
		Str("msg_id", msg.ID).
		Int("reply_len", len(reply)).
		Dur("duration_ms", llmDur).
		Msg("[orchestrator] Ollama chat done")

	// ── 4. Send reply ─────────────────────────────────────────────────
	sendStart := time.Now()
	err = o.waClient.SendText(ctx, msg.SenderJID, reply)
	o.logger.Info().
		Str("msg_id", msg.ID).
		Dur("send_ms", time.Since(sendStart)).
		Dur("total_ms", time.Since(handleStart)).
		Msg("[orchestrator] handle complete")

	// ── 5. Async: persist conversation + extract facts ────────────────
	// Store messages in messages_v2
	sessionID = "default" // TODO: session rotation based on idle time
	o.repo.InsertMessageV2(ctx, repository.InsertMessageV2Params{
		UserID:        userID,
		Platform:      "whatsapp",
		PlatformMsgID: msg.ID,
		SessionID:     sessionID,
		Role:          "user",
		Content:       userText,
		TokenCount:    estimateTokens(userText),
	})
	o.repo.InsertMessageV2(ctx, repository.InsertMessageV2Params{
		UserID:     userID,
		Platform:   "whatsapp",
		SessionID:  sessionID,
		Role:       "assistant",
		Content:    reply,
		TokenCount: estimateTokens(reply),
	})

	// Fire async fact extraction
	o.memory.SyncAsync(userID, userText, reply)

	return err
}

// estimateTokens rough word-based token count.
func estimateTokens(s string) int {
	words := 0
	for range strings.Fields(s) {
		words++
	}
	return int(float64(words) * 1.3)
}

// buildSystemPrompt constructs system prompt with bounded RAG context + memory.
func (o *orchestrator) buildSystemPrompt(ctx context.Context, ctxInfo rag.PromptContext, memoryContext, metadataBlock string, targetDocID string) string {
	// Fetch dynamic configuration from Dashboard
	personaName := "Assistant"
	personaDesc := "Saya adalah asisten AI yang cerdas dan efisien."
	tonePref := "formal"
	promptTemplate := "Anda adalah asisten AI. Identitas Anda: {{persona_name}} ({{persona_description}}). Gaya bahasa: {{tone_preference}}.\n\nFakta: {{user_facts}}\nDokumen: {{active_document_metadata}}\nRAG Context: {{rag_context}}"

	if conf, err := o.dashboardRepo.GetAgentConfigByKey(ctx, "persona_name"); err == nil && conf != nil {
		json.Unmarshal([]byte(conf.ValueJSON), &personaName)
	}
	if conf, err := o.dashboardRepo.GetAgentConfigByKey(ctx, "persona_description"); err == nil && conf != nil {
		json.Unmarshal([]byte(conf.ValueJSON), &personaDesc)
	}
	if conf, err := o.dashboardRepo.GetAgentConfigByKey(ctx, "tone_preference"); err == nil && conf != nil {
		json.Unmarshal([]byte(conf.ValueJSON), &tonePref)
	}
	if conf, err := o.dashboardRepo.GetAgentConfigByKey(ctx, "system_prompt_template"); err == nil && conf != nil {
		json.Unmarshal([]byte(conf.ValueJSON), &promptTemplate)
	}

	if targetDocID != "" && metadataBlock != "" {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("ANDA ADALAH: %s\n%s\nGaya Bahasa: %s\n\n", personaName, personaDesc, tonePref))
		sb.WriteString("ATURAN ISOLASI KONTEKS DOKUMEN (WAJIB DIPATUHI):\n")
		sb.WriteString("1. Dokumen yang SEDANG AKTIF dan menjadi rujukan tunggal untuk pertanyaan pengguna saat ini adalah dokumen dengan metadata berikut:\n")
		sb.WriteString(metadataBlock)
		sb.WriteString("\n")
		sb.WriteString("2. Potongan teks (chunks) yang diberikan ke Anda di bawah ini HANYA berasal dari dokumen aktif tersebut. Jika ada potongan teks yang isinya tampak tidak konsisten dengan metadata di atas, abaikan potongan tersebut dan jangan gunakan sebagai dasar jawaban.\n\n")
		sb.WriteString("3. JANGAN PERNAH mencampur, menggabungkan, atau membandingkan informasi dari dokumen ini dengan dokumen lain yang mungkin pernah dibahas SEBELUMNYA dalam riwayat percakapan ini, KECUALI pengguna secara eksplisit memerintahkan perbandingan.\n\n")
		sb.WriteString(fmt.Sprintf("4. Jika dalam riwayat percakapan terdapat pembahasan tentang dokumen lain (document_id berbeda dari %s), perlakukan pembahasan tersebut sebagai TIDAK RELEVAN untuk menjawab pertanyaan saat ini. Fokus jawaban Anda HARUS 100%% bersumber dari metadata dan chunk dokumen aktif saja.\n\n", targetDocID))
		sb.WriteString("5. Sebelum menjawab, lakukan VERIFIKASI INTERNAL: pastikan setiap metode, hasil, atau istilah teknis yang Anda sebutkan dalam jawaban benar-benar muncul dalam chunk/metadata dokumen aktif ini.\n\n")

		sb.WriteString("KONTEN UNTUK DIJAWAB:\n")
		if ctxInfo.HasResults {
			ragCtx := ctxInfo.Context
			if len(ragCtx) > maxContextChars {
				ragCtx = ragCtx[:maxContextChars] + "\n[...context truncated...]"
			}
			sb.WriteString(ragCtx)
		} else {
			sb.WriteString("(Tidak ada chunk yang ditemukan)")
		}

		return sb.String()
	}

	// Normal dynamic prompt
	sysPrompt := promptTemplate
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{persona_name}}", personaName)
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{persona_description}}", personaDesc)
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{tone_preference}}", tonePref)
	
	if memoryContext != "" {
		sysPrompt = strings.ReplaceAll(sysPrompt, "{{user_facts}}", memoryContext)
	} else {
		sysPrompt = strings.ReplaceAll(sysPrompt, "{{user_facts}}", "(Tidak ada fakta relevan)")
	}
	
	sysPrompt = strings.ReplaceAll(sysPrompt, "{{active_document_metadata}}", "(Tidak ada dokumen aktif)")

	if ctxInfo.HasResults {
		ragCtx := ctxInfo.Context
		if len(ragCtx) > maxContextChars {
			ragCtx = ragCtx[:maxContextChars] + "\n[...context truncated...]"
		}
		sysPrompt = strings.ReplaceAll(sysPrompt, "{{rag_context}}", ragCtx)
		if len(ctxInfo.Sources) > 0 {
			sysPrompt += "\n\nSources: " + strings.Join(ctxInfo.Sources, ", ")
		}
	} else {
		sysPrompt = strings.ReplaceAll(sysPrompt, "{{rag_context}}", "")
	}

	return sysPrompt
}
