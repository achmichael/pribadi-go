package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/classifier"
	contextpkg "github.com/achmichael/pribadi-go/internal/context"
	"github.com/achmichael/pribadi-go/internal/conversation"
	"github.com/achmichael/pribadi-go/internal/delivery/whatsapp"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/factmemory"
	"github.com/achmichael/pribadi-go/internal/identity"
	"github.com/achmichael/pribadi-go/internal/planner"
	"github.com/achmichael/pribadi-go/internal/prompt"
	"github.com/achmichael/pribadi-go/internal/reasoning"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/response"
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

	stateManager      conversation.StateManager
	prefManager       conversation.PreferenceManager
	intentClassifier  classifier.IntentClassifier
	taskClassifier    classifier.TaskClassifier
	interceptor       classifier.Interceptor
	planner           planner.Planner
	contextBuilder    contextpkg.Builder
	promptBuilder     *prompt.Builder
	promptComposer    prompt.Composer
	verifier          reasoning.Verifier
	reflector         reasoning.Reflector
	responseProcessor response.Processor
	interactionLogger response.InteractionLogger
	logger            *zerolog.Logger
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
	stateManager conversation.StateManager,
	prefManager conversation.PreferenceManager,
	intentClassifier classifier.IntentClassifier,
	taskClassifier classifier.TaskClassifier,
	interceptor classifier.Interceptor,
	planner planner.Planner,
	contextBuilder contextpkg.Builder,
	promptBuilder *prompt.Builder,
	promptComposer prompt.Composer,
	verifier reasoning.Verifier,
	reflector reasoning.Reflector,
	responseProcessor response.Processor,
	interactionLogger response.InteractionLogger,
	logger *zerolog.Logger,
) Orchestrator {
	return &orchestrator{
		waClient:          waClient,
		transcription:     transcription,
		extraction:        extraction,
		ragIngest:         ragIngest,
		ragRetrieve:       ragRetrieve,
		ollama:            ollamaClient,
		repo:              repo,
		dashboardRepo:     dashboardRepo,
		memory:            memory,
		resolver:          resolver,
		refResolver:       refResolver,
		stateManager:      stateManager,
		prefManager:       prefManager,
		intentClassifier:  intentClassifier,
		taskClassifier:    taskClassifier,
		interceptor:       interceptor,
		planner:           planner,
		contextBuilder:    contextBuilder,
		promptBuilder:     promptBuilder,
		promptComposer:    promptComposer,
		verifier:          verifier,
		reflector:         reflector,
		responseProcessor: responseProcessor,
		interactionLogger: interactionLogger,
		logger:            logger,
	}
}

// maxContextChars caps how much RAG context goes into the system prompt.
const maxContextChars = 2000

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
	sessionID := "default" // TODO: multi-session

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
			// Normalisasi spasi dan newline agar LLM tidak bingung dengan format OCR
			extractText = strings.Join(strings.Fields(extractText), " ")

			metaPrompt := `ANDA ADALAH: DOCUMENT METADATA EXTRACTION AGENT

TUGAS ANDA:
Ekstrak metadata terstruktur dari potongan teks dokumen di bawah ini. Teks ini adalah hasil OCR/ekstraksi otomatis dari PDF dan SERING MENGANDUNG NOISE seperti: header jurnal, nomor volume/issue, kode artikel, running title dari artikel lain dalam volume yang sama, nomor halaman, watermark, atau metadata penerbit yang TIDAK BOLEH disalahartikan sebagai judul dokumen.

ATURAN WAJIB UNTUK MENENTUKAN "TITLE" (JUDUL):
1. Judul dokumen yang benar SELALU berada tepat SEBELUM daftar nama penulis (authors), bukan sebelum nama jurnal/header/volume.
2. JANGAN PERNAH mengambil teks yang terlihat seperti: nama jurnal, "Volume X Nomor Y", "JN-xxx"/kode artikel, ISSN, nama institusi penerbit, atau judul artikel lain yang mungkin muncul sebagai running header di bagian atas/bawah halaman.
3. Judul asli biasanya merupakan frasa nominal yang menjelaskan topik penelitian (metode, subjek, objek penelitian) — bukan kalimat administratif jurnal.
4. WAJIB lakukan cross-check internal sebelum finalisasi: setelah Anda mengekstrak "Title" dan "Abstract"/"Keywords", PERIKSA apakah keduanya secara tematik konsisten satu sama lain. Jika "Title" yang Anda temukan TIDAK relevan secara tema dengan "Abstract" atau "Keywords" yang Anda temukan di teks yang sama, maka "Title" tersebut SALAH — cari ulang kandidat judul lain di teks, atau jika benar-benar tidak ditemukan, kembalikan string kosong daripada memaksakan judul yang tidak konsisten.
5. Jika dalam satu potongan teks Anda menemukan LEBIH DARI SATU kandidat judul (misalnya satu di bagian atas sebagai header, satu lagi tepat sebelum nama penulis), PILIH yang posisinya TEPAT SEBELUM/BERDEKATAN dengan daftar nama penulis.
6. EKSTRAK SECARA VERBATIM (SAMA PERSIS): Salin teks judul asli kata demi kata. JANGAN PERNAH meringkas, menghilangkan kata, atau memodifikasi judul asli menjadi bentuk lain (contoh: teks asli yang panjang TIDAK BOLEH disingkat menjadi lebih pendek).

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

			o.repo.InsertMessageV2(context.Background(), repository.InsertMessageV2Params{
				UserID:        userID,
				Platform:      "whatsapp",
				PlatformMsgID: msg.ID,
				SessionID:     sessionID,
				Role:          "user",
				Content:       "[Mengirim Dokumen: " + fileName + "]",
				TokenCount:    0,
			})

			o.repo.InsertMessageV2(context.Background(), repository.InsertMessageV2Params{
				UserID:     userID,
				Platform:   "whatsapp",
				SessionID:  sessionID,
				Role:       "assistant",
				Content:    summary, // Variabel pesan "✅ Dokumen berhasil diproses..."
				TokenCount: 0,
			})
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

	// Intercept explicit command (Reset)
	if strings.TrimSpace(strings.ToLower(userText)) == "/reset" {
		o.logger.Info().Str("userID", userID).Str("sessionID", sessionID).Msg("[Orchestrator] Explicit commmand received")

		// clean chat from DB
		err = o.repo.DeleteMessagesByUserSession(ctx, userID, sessionID)
		if err != nil {
			o.logger.Warn().Err(err).Msg("[orchestrator] failed to delete messages from sqlite")
		}

		// Reset conversation state
		err = o.stateManager.DeleteState(ctx, userID, sessionID)
		if err != nil {
			o.logger.Warn().Err(err).Msg("[orchestrator] failed to delete conversation state")
		}

		// purge user fact from qdrant
		err = o.memory.PurgeMemory(ctx, userID)
		if err != nil {
			o.logger.Warn().Err(err).Msg("[orchestrator] failed to purge user fact")
		}

		// delete documents from sqlite and chunks from qdrant
		err = o.ragRetrieve.PurgeRAGDocuments(ctx, userID)
		if err != nil {
			o.logger.Warn().Err(err).Msg("[orchestrator] failed to delete user documents from sqlite")
		}

		err = o.repo.DeleteAllUserDocuments(ctx, userID)
		if err != nil {
			o.logger.Warn().Err(err).Msg("[orchestrator] failed to purge user documents from qdrant")
		}

		// Kirim balasan ke user bahwa riwayat berhasil dihapus
		sendErr := o.waClient.SendText(ctx, msg.SenderJID, "🧹 *Sesi Diatur Ulang*\nRiwayat percakapan dan memori sesi Anda telah berhasil dibersihkan.")
		if sendErr != nil {
			o.logger.Warn().Err(sendErr).Msg("[orchestrator] failed to send reset confirmation")
		}

		return sendErr
	}

	// Load Session State
	state, err := o.stateManager.Load(ctx, userID, sessionID)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] state load failed, continuing with empty state")
	}
	var stateData *domain.StateData
	var rawState *repository.ConversationStateRow
	var turnCount int
	if state != nil {
		stateData = state
		if raw, _ := o.stateManager.GetRaw(ctx, userID, sessionID); raw != nil {
			rawState = raw
			turnCount = raw.TurnCount
		}
	}

	// ── 2. Reference Resolution (Extract Context Needs) ─────────────
	ragQuery := userText
	targetDocID := ""
	if msg.MessageType == whatsapp.MessageTypeText || msg.MessageType == whatsapp.MessageTypeVoiceNote {
		ragQuery, targetDocID = o.refResolver.ResolveQuery(ctx, userID, userText, sessionID)
		if targetDocID != "" {
			doc, err := o.repo.GetUserDocumentByID(ctx, targetDocID)
			if err == nil && doc != nil {
				// --- Query Contextualization Agent ---
				if doc.MetadataJSON != "" && doc.MetadataJSON != "{}" {
					contextualizePrompt := fmt.Sprintf(`ANDA ADALAH: QUERY CONTEXTUALIZATION AGENT untuk sistem RAG multilingual.

KONTEKS:
Sistem ini menyimpan dokumen yang isinya bisa berbahasa apa saja (Inggris, Indonesia, dll). 
Pengguna sedang memberikan query: "%s"

TUGAS ANDA:
Anda menerima query asli dari pengguna beserta metadata dokumen aktif (jika ada). Tugas Anda BUKAN menjawab pertanyaan, tetapi menghasilkan SATU query pencarian semantik yang dioptimalkan untuk vector retrieval (RAG), dengan ATURAN KETAT:

1. Jika pertanyaan pengguna sangat umum (misal: "apa isi dokumen tersebut?", "tolong ringkas dokumennya", "jelaskan isi file ini"), kembalikan SAJA query asli tersebut ("apa isi dari dokumen tersebut?") TANPA tambahan kata-kata lain. JANGAN mengarang kata kunci dari metadata jika tidak diminta!
2. Hanya lakukan penambahan istilah spesifik dari metadata jika pengguna menanyakan sesuatu yang sangat butuh penyeimbang bahasa (misal terjemahan).
3. Output HANYA berupa query hasil akhir dalam bentuk teks polos, tanpa prefix "Query pencarian semantik:", tanpa penjelasan, tanpa markdown. JANGAN MENGARANG!

INPUT METADATA: %s`, userText, doc.MetadataJSON)

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

	// ── 3. Classification (Phase 2) ─────────────────────────────────
	var classResult *classifier.Classification
	var activeTask string
	var lastIntent string
	var lastClass string
	if rawState != nil {
		activeTask = rawState.ActiveTask
		lastIntent = rawState.LastIntent
		lastClass = rawState.LastMessageClass
	}

	intentRes, err := o.intentClassifier.Classify(ctx, classifier.ClassifyParams{
		UserText:     userText,
		LastIntent:   lastIntent,
		LastClass:    lastClass,
		ActiveTask:   activeTask,
		TurnCount:    turnCount,
		HasActiveDoc: targetDocID != "",
	})
	if err == nil {
		classResult = intentRes
	}

	taskRes, _ := o.taskClassifier.Classify(ctx, classifier.TaskClassifyParams{
		UserText:   userText,
		ActiveTask: activeTask,
		LastIntent: lastIntent,
		SessionID:  sessionID,
		UserID:     userID,
		TurnCount:  turnCount,
	})
	if classResult != nil && taskRes != nil {
		classResult.ContinuesPrevious = taskRes.ContinuesTask
	}

	// ── 4. Interceptor (Phase 2) ────────────────────────────────────
	if classResult != nil {
		intercepted, ackMsg, err := o.interceptor.Process(ctx, classifier.InterceptParams{
			UserID:    userID,
			SessionID: sessionID,
			UserText:  userText,
			Class:     classResult,
			State:     stateData,
		})
		if err != nil {
			o.logger.Warn().Err(err).Msg("[orchestrator] interceptor error")
		}
		if intercepted {
			o.logger.Info().Str("msg_id", msg.ID).Msg("[orchestrator] request intercepted (e.g. preference update)")
			sendErr := o.waClient.SendText(ctx, msg.SenderJID, ackMsg)
			
			bgCtx := context.Background()
			
			// 1. Persist to history so the LLM knows what happened
			o.repo.InsertMessageV2(bgCtx, repository.InsertMessageV2Params{
				UserID:        userID,
				Platform:      "whatsapp",
				PlatformMsgID: msg.ID,
				SessionID:     sessionID,
				Role:          "user",
				Content:       userText,
				TokenCount:    estimateTokens(userText),
			})
			o.repo.InsertMessageV2(bgCtx, repository.InsertMessageV2Params{
				UserID:     userID,
				Platform:   "whatsapp",
				SessionID:  sessionID,
				Role:       "assistant",
				Content:    ackMsg,
				TokenCount: estimateTokens(ackMsg),
			})
			
			// 2. State update
			o.responseProcessor.UpdateState(bgCtx, userID, sessionID, classResult)
			
			// 3. Trigger memory sync so facts are extracted
			o.responseProcessor.TriggerMemorySync(userID, userText, ackMsg)
			
			// 4. Log interaction
			o.interactionLogger.Log(bgCtx, response.InteractionLog{
				UserID:      userID,
				SessionID:   sessionID,
				Timestamp:   time.Now(),
				UserMessage: userText,
				Response:    ackMsg,
				Metadata:    map[string]interface{}{"intercepted": true},
				Duration:    time.Since(handleStart).Milliseconds(),
				Success:     sendErr == nil,
			})

			return sendErr
		}
	}

	// ── 5. Planner (Phase 2) ────────────────────────────────────────
	plan, err := o.planner.CreatePlan(ctx, planner.PlanParams{
		UserText:       userText,
		Classification: classResult,
		State:          stateData,
		HasActiveDoc:   targetDocID != "",
		TurnCount:      turnCount,
	})
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] planning failed, fallback to defaults")
	}

	// ── 6. Context Builder & Prompt Composer (Phase 3) ──────────────
	sc, err := o.contextBuilder.Build(ctx, contextpkg.BuildParams{
		UserID:       userID,
		SessionID:    sessionID,
		UserText:     userText,
		RAGQuery:     ragQuery,
		TargetDocID:  targetDocID,
		HistoryLimit: 10,
		SkipRAG:      plan != nil && !plan.NeedsRAG,
		SkipMemory:   plan != nil && !plan.NeedsMemory,
	})
	if err != nil {
		return fmt.Errorf("context builder: %w", err)
	}

	var convHistory []prompt.HistoryMessage
	if historyMsgs, err := o.repo.ListMessagesByUserSession(ctx, userID, sessionID, 10); err == nil {
		for _, m := range historyMsgs {
			convHistory = append(convHistory, prompt.HistoryMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	composed, err := o.promptComposer.Compose(ctx, prompt.ComposeParams{
		UserText:            userText,
		Classification:      classResult,
		Plan:                plan,
		SessionContext:      sc,
		ConversationHistory: convHistory,
	})
	if err != nil {
		return fmt.Errorf("prompt composer: %w", err)
	}

	// ── 7. LLM Generation ───────────────────────────────────────────
	messages := []ollama.ChatMessage{
		{Role: "system", Content: composed.SystemPrompt},
	}

	for _, histMsg := range convHistory {
		messages = append(messages, ollama.ChatMessage{
			Role:    histMsg.Role,
			Content: histMsg.Content,
		})
	}

	messages = append(messages, ollama.ChatMessage{
		Role:    "user",
		Content: composed.UserPrompt,
	})

	o.logger.Info().
		Str("msg_id", msg.ID).
		Int("sys_len", len(composed.SystemPrompt)).
		Int("user_len", len(composed.UserPrompt)).
		Msg("[orchestrator] sending to Ollama")

	llmStart := time.Now()
	rawResponse, err := o.ollama.Chat(ctx, messages)
	llmDur := time.Since(llmStart)
	if err != nil {
		o.logger.Error().Err(err).Dur("ms", llmDur).Msg("[orchestrator] Ollama generation failed")

		// Log error
		o.interactionLogger.LogError(ctx, userID, sessionID, userText, err.Error(), time.Since(handleStart).Milliseconds())
		return err
	}

	// ── 8. Verifier (Phase 4) ───────────────────────────────────────
	verification, _ := o.verifier.Verify(ctx, reasoning.VerifyParams{
		UserID:       userID,
		SessionID:    sessionID,
		UserQuestion: userText,
		Response:     rawResponse,
		State:        stateData,
		RAGContext:   sc.RAGContext,
		RAGSources:   sc.RAGSources,
	})

	// ── 9. Response Processor (Phase 5) ─────────────────────────────
	finalRes, err := o.responseProcessor.Process(ctx, response.ProcessParams{
		UserID:         userID,
		SessionID:      sessionID,
		UserMessage:    userText,
		RawResponse:    rawResponse,
		Classification: classResult,
		Plan:           plan,
		ComposedPrompt: composed,
		Verification:   verification,
		RAGSources:     sc.RAGSources,
		State:          stateData,
	})
	if err != nil {
		return fmt.Errorf("response processor: %w", err)
	}

	// Populate durations in metadata
	finalRes.Metadata.TotalDurationMs = time.Since(handleStart).Milliseconds()
	finalRes.Metadata.LLMInferenceMs = llmDur.Milliseconds()

	// ── 10. Send Reply ──────────────────────────────────────────────
	sendTextErr := o.waClient.SendText(ctx, msg.SenderJID, finalRes.Text)

	// ── 11. Async Post-processing ───────────────────────────────────
	// Note: We use background context for async work to not be cancelled if request context ends
	bgCtx := context.Background()

	// 1. Persist Raw Messages
	o.repo.InsertMessageV2(bgCtx, repository.InsertMessageV2Params{
		UserID:        userID,
		Platform:      "whatsapp",
		PlatformMsgID: msg.ID,
		SessionID:     sessionID,
		Role:          "user",
		Content:       userText,
		TokenCount:    estimateTokens(userText),
	})
	o.repo.InsertMessageV2(bgCtx, repository.InsertMessageV2Params{
		UserID:     userID,
		Platform:   "whatsapp",
		SessionID:  sessionID,
		Role:       "assistant",
		Content:    finalRes.Text,
		TokenCount: estimateTokens(finalRes.Text),
	})

	// 2. Update State
	if finalRes.ShouldUpdate {
		o.responseProcessor.UpdateState(bgCtx, userID, sessionID, classResult)
	}

	// 3. Trigger Memory Sync
	if finalRes.ShouldUpdate {
		o.responseProcessor.TriggerMemorySync(userID, userText, finalRes.Text)
	}

	// 4. Reflector & Logging
	go func() {
		// Reflect
		reflection, _ := o.reflector.Reflect(bgCtx, reasoning.ReflectionParams{
			UserQuestion: userText,
			Response:     finalRes.Text,
			Plan:         fmt.Sprintf("%+v", plan),
			RAGUsed:      finalRes.Metadata.RAGUsed,
			MemoryUsed:   finalRes.Metadata.MemoryUsed,
			HistoryUsed:  finalRes.Metadata.HistoryUsed,
		})
		finalRes.Metadata.Reflection = reflection

		// Convert metadata to map[string]interface{}
		metaMap := map[string]interface{}{
			"llm_ms": finalRes.Metadata.LLMInferenceMs,
		}
		if classResult != nil {
			metaMap["intent"] = classResult.Intent
			metaMap["class"] = classResult.MessageClass
		}
		if plan != nil {
			metaMap["strategy"] = plan.ResponseStrategy
		}
		if verification != nil {
			metaMap["confidence_score"] = verification.ConfidenceScore
			metaMap["hallucination_risk"] = verification.HallucinationRisk
			metaMap["is_valid"] = verification.IsValid
		}

		// Log Interaction
		o.interactionLogger.Log(bgCtx, response.InteractionLog{
			UserID:      userID,
			SessionID:   sessionID,
			Timestamp:   time.Now(),
			UserMessage: userText,
			Response:    finalRes.Text,
			Metadata:    metaMap,
			Duration:    finalRes.Metadata.TotalDurationMs,
			Success:     sendTextErr == nil,
		})
	}()

	return sendTextErr
}

// estimateTokens rough word-based token count.
func estimateTokens(s string) int {
	words := 0
	for range strings.Fields(s) {
		words++
	}
	return int(float64(words) * 1.3)
}
