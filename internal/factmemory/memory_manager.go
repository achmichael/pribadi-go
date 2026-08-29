// Package factmemory implements persistent semantic memory for conversational
// facts. It stores facts both in SQLite (audit trail, dedup) and Qdrant
// (semantic search), using a separate "user_facts" collection from the
// document RAG pipeline's "documents" collection.
package factmemory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/pkg/llm"
	"github.com/achmichael/pribadi-go/pkg/utils"
	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ─── Interface ─────────────────────────────────────────────────────

// ScoredFact represents a fact with confidence metadata.
type ScoredFact struct {
	FactText   string  `json:"fact_text"`
	Category   string  `json:"category"`
	Score      float32 `json:"score"`
	Confidence string  `json:"confidence"` // "high", "medium", "low"
}

// MemoryManager is the top-level interface consumed by the orchestrator.
type MemoryManager interface {
	// PrefetchRelevant returns a formatted context block of relevant facts
	// for the given user+message, ready to inject into system prompt.
	PrefetchRelevant(ctx context.Context, userID string, currentMessage string) (string, error)

	// PrefetchWithScores returns scored facts with confidence levels.
	// Used by the intelligence layer for confidence-aware retrieval (Principle 6).
	PrefetchWithScores(ctx context.Context, userID string, currentMessage string) ([]ScoredFact, error)

	// SyncAsync fires a background goroutine to extract facts from the
	// latest conversation turn and store them. Non-blocking.
	SyncAsync(userID string, userMessage string, assistantMessage string)

	// DeactivateFact deactivates a fact in SQLite and removes it from Qdrant.
	DeactivateFact(ctx context.Context, userID string, factID int64) error

	PurgeMemory(ctx context.Context, userID string) error

	// Close shuts down the worker pool and Qdrant connection.
	Close()
}

// ─── Config ────────────────────────────────────────────────────────

const (
	collectionName           = "user_facts"
	vectorSize               = 768 // nomic-embed-text
	prefetchTopK             = 5
	prefetchMinScore float32 = 0.6
	dedupThreshold   float32 = 0.92 // cosine similarity above this = duplicate
	workerCount              = 2
	channelCap               = 50
)

// ─── Implementation ────────────────────────────────────────────────

type manager struct {
	repo        repository.Repository
	embedder    *utils.Embedder
	llm         llm.Client
	conn        *grpc.ClientConn
	points      pb.PointsClient
	collections pb.CollectionsClient
	logger      *zerolog.Logger
	jobs        chan syncJob
	done        chan struct{}
}

type syncJob struct {
	UserID           string
	UserMessage      string
	AssistantMessage string
}

// NewMemoryManager creates a MemoryManager backed by Qdrant + SQLite.
// qdrantAddr is "host:port" (gRPC), e.g. "localhost:6334".
func NewMemoryManager(
	repo repository.Repository,
	llm llm.Client,
	qdrantAddr string,
	ollamaBaseURL string,
	logger *zerolog.Logger,
) (MemoryManager, error) {
	conn, err := grpc.NewClient(
		qdrantAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("qdrant dial: %w", err)
	}

	m := &manager{
		repo:        repo,
		embedder:    utils.NewEmbedder(ollamaBaseURL, "nomic-embed-text"),
		llm:         llm,
		conn:        conn,
		points:      pb.NewPointsClient(conn),
		collections: pb.NewCollectionsClient(conn),
		logger:      logger,
		jobs:        make(chan syncJob, channelCap),
		done:        make(chan struct{}),
	}

	if err := m.ensureCollection(context.Background()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ensure user_facts collection: %w", err)
	}

	// Start worker pool
	for i := 0; i < workerCount; i++ {
		go m.worker(i)
	}

	logger.Info().
		Str("collection", collectionName).
		Int("workers", workerCount).
		Msg("[factmemory] initialized")

	return m, nil
}

// ─── Prefetch ──────────────────────────────────────────────────────

func (m *manager) PrefetchRelevant(ctx context.Context, userID string, currentMessage string) (string, error) {
	start := time.Now()

	embedding, err := m.embedder.Embed(ctx, "search_query: "+currentMessage)
	if err != nil {
		return "", fmt.Errorf("embed query: %w", err)
	}

	// Search with user_id filter
	limit := uint64(prefetchTopK)
	withPayload := true
	scoreThreshold := prefetchMinScore

	resp, err := m.points.Search(ctx, &pb.SearchPoints{
		CollectionName: collectionName,
		Vector:         embedding,
		Limit:          limit,
		ScoreThreshold: &scoreThreshold,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: withPayload}},
		Filter: &pb.Filter{
			Must: []*pb.Condition{
				{
					ConditionOneOf: &pb.Condition_Field{
						Field: &pb.FieldCondition{
							Key: "user_id",
							Match: &pb.Match{
								MatchValue: &pb.Match_Keyword{Keyword: userID},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("search user_facts: %w", err)
	}

	results := resp.GetResult()
	if len(results) == 0 {
		m.logger.Debug().
			Str("user_id", userID).
			Dur("ms", time.Since(start)).
			Msg("[factmemory] prefetch: no relevant facts")
		return "", nil
	}

	// Build context block
	var sb strings.Builder
	sb.WriteString("<informasi_latar_belakang>\n")
	for _, scored := range results {
		factText := ""
		category := ""
		for k, v := range scored.GetPayload() {
			switch k {
			case "fact_text":
				factText = v.GetStringValue()
			case "category":
				category = v.GetStringValue()
			}
		}
		if factText != "" {
			if category != "" {
				sb.WriteString(fmt.Sprintf("- [%s] %s\n", category, factText))
			} else {
				sb.WriteString(fmt.Sprintf("- %s\n", factText))
			}
		}
	}
	sb.WriteString("</informasi_latar_belakang>")

	m.logger.Info().
		Str("user_id", userID).
		Int("facts_found", len(results)).
		Dur("ms", time.Since(start)).
		Msg("[factmemory] prefetch done")

	return sb.String(), nil
}

// PrefetchWithScores returns scored facts with confidence metadata.
func (m *manager) PrefetchWithScores(ctx context.Context, userID string, currentMessage string) ([]ScoredFact, error) {
	start := time.Now()

	embedding, err := m.embedder.Embed(ctx, "search_query: "+currentMessage)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	limit := uint64(prefetchTopK)
	withPayload := true

	resp, err := m.points.Search(ctx, &pb.SearchPoints{
		CollectionName: collectionName,
		Vector:         embedding,
		Limit:          limit,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: withPayload}},
		Filter: &pb.Filter{
			Must: []*pb.Condition{
				{
					ConditionOneOf: &pb.Condition_Field{
						Field: &pb.FieldCondition{
							Key: "user_id",
							Match: &pb.Match{
								MatchValue: &pb.Match_Keyword{Keyword: userID},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("search user_facts: %w", err)
	}

	results := resp.GetResult()
	if len(results) == 0 {
		return nil, nil
	}

	var scored []ScoredFact
	for _, r := range results {
		factText := ""
		category := ""
		for k, v := range r.GetPayload() {
			switch k {
			case "fact_text":
				factText = v.GetStringValue()
			case "category":
				category = v.GetStringValue()
			}
		}
		if factText == "" {
			continue
		}

		confidence := scoreToConfidence(r.GetScore())
		scored = append(scored, ScoredFact{
			FactText:   factText,
			Category:   category,
			Score:      r.GetScore(),
			Confidence: confidence,
		})
	}

	m.logger.Info().
		Str("user_id", userID).
		Int("facts_found", len(scored)).
		Dur("ms", time.Since(start)).
		Msg("[factmemory] prefetch with scores done")

	return scored, nil
}

// scoreToConfidence maps vector similarity score to confidence level.
func scoreToConfidence(score float32) string {
	switch {
	case score >= 0.75:
		return "high"
	case score >= 0.50:
		return "medium"
	default:
		return "low"
	}
}

// ─── Sync (async) ──────────────────────────────────────────────────

func (m *manager) SyncAsync(userID string, userMessage string, assistantMessage string) {
	job := syncJob{
		UserID:           userID,
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
	}

	select {
	case m.jobs <- job:
		m.logger.Debug().
			Str("user_id", userID).
			Msg("[factmemory] sync job queued")
	default:
		m.logger.Warn().
			Str("user_id", userID).
			Msg("[factmemory] sync job DROPPED (channel full)")
	}
}

func (m *manager) worker(id int) {
	m.logger.Debug().Int("worker_id", id).Msg("[factmemory] worker started")
	for {
		select {
		case job, ok := <-m.jobs:
			if !ok {
				m.logger.Debug().Int("worker_id", id).Msg("[factmemory] worker stopped")
				return
			}
			m.processSync(job)
		case <-m.done:
			return
		}
	}
}

func (m *manager) processSync(job syncJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	start := time.Now()

	// 1. Extract facts via LLM
	facts, err := m.extractFacts(ctx, job.UserMessage, job.AssistantMessage)
	if err != nil {
		m.logger.Error().Err(err).
			Str("user_id", job.UserID).
			Msg("[factmemory] fact extraction failed")
		return
	}

	if len(facts) == 0 {
		m.logger.Debug().
			Str("user_id", job.UserID).
			Dur("ms", time.Since(start)).
			Msg("[factmemory] no facts extracted from turn")
		return
	}

	m.logger.Info().
		Str("user_id", job.UserID).
		Int("facts_extracted", len(facts)).
		Dur("extract_ms", time.Since(start)).
		Msg("[factmemory] facts extracted")

	// 2. Dedup + store each fact
	stored := 0
	for _, f := range facts {
		if err := m.dedupAndStore(ctx, job.UserID, f); err != nil {
			m.logger.Error().Err(err).
				Str("user_id", job.UserID).
				Str("fact", f.Fact).
				Msg("[factmemory] store fact failed")
			continue
		}
		stored++
	}

	m.logger.Info().
		Str("user_id", job.UserID).
		Int("stored", stored).
		Int("total", len(facts)).
		Dur("total_ms", time.Since(start)).
		Msg("[factmemory] sync done")
}

// ─── LLM Fact Extraction ──────────────────────────────────────────

type extractedFact struct {
	Fact     string `json:"fact"`
	Category string `json:"category"`
}

const extractionPrompt = `You are a fact extraction assistant. Analyze the following conversation turn and extract discrete, personal facts about the user.

Rules:
- Only extract facts that are PERSONAL to the user (preferences, biodata, habits, projects, relationships, location, skills, etc.)
- Do NOT extract generic knowledge or conversational filler
- Each fact should be a single, self-contained statement
- If there are no personal facts, return an empty JSON array []
- Category must be one of: biodata, preference, project, skill, relationship, location, habit, general

Return ONLY a valid JSON array, no other text. Example:
[{"fact": "Tinggal di kota Pasuruan", "category": "location"}, {"fact": "Suka kopi hitam tanpa gula", "category": "preference"}]

Conversation turn:
User: %s
Assistant: %s

Extract facts (JSON array only):`

func (m *manager) extractFacts(ctx context.Context, userMsg, assistantMsg string) ([]extractedFact, error) {
	// Truncate to avoid huge prompts
	if len(userMsg) > 1000 {
		userMsg = userMsg[:1000]
	}
	if len(assistantMsg) > 500 {
		assistantMsg = assistantMsg[:500]
	}

	prompt := fmt.Sprintf(extractionPrompt, userMsg, assistantMsg)
	messages := []llm.ChatMessage{
		{Role: "user", Content: prompt},
	}

	reply, err := m.llm.ChatJSON(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("llm chat json: %w", err)
	}

	reply = strings.TrimSpace(reply)
	if reply == "" || reply == "{}" {
		return nil, nil
	}

	var facts []extractedFact
	if err := json.Unmarshal([]byte(reply), &facts); err != nil {
		// Fallback 1: for models that wrap the array in an object, e.g., {"facts": [...]}
		var fallback struct {
			Facts []extractedFact `json:"facts"`
		}
		if err2 := json.Unmarshal([]byte(reply), &fallback); err2 == nil && fallback.Facts != nil {
			facts = fallback.Facts
		} else {
			// Fallback 2: for models that return a single object instead of an array
			var singleObj extractedFact
			if err3 := json.Unmarshal([]byte(reply), &singleObj); err3 == nil && singleObj.Fact != "" {
				facts = []extractedFact{singleObj}
			} else {
				m.logger.Warn().
					Str("raw_reply", reply).
					Err(err).
					Msg("[factmemory] failed to parse LLM fact extraction response")
				return nil, nil // non-fatal: LLM gave bad JSON, skip this turn
			}
		}
	}

	// Filter empty/too-short facts
	var valid []extractedFact
	for _, f := range facts {
		f.Fact = strings.TrimSpace(f.Fact)
		f.Category = strings.TrimSpace(f.Category)
		if len(f.Fact) < 5 {
			continue
		}
		if f.Category == "" {
			f.Category = "general"
		}
		valid = append(valid, f)
	}

	return valid, nil
}

// ─── Dedup + Store ─────────────────────────────────────────────────

func (m *manager) dedupAndStore(ctx context.Context, userID string, fact extractedFact) error {
	// 1. Embed the fact with asymmetric prefix
	embedding, err := m.embedder.Embed(ctx, "search_document: "+fact.Fact)
	if err != nil {
		return fmt.Errorf("embed fact: %w", err)
	}

	// 2. Search existing facts for this user to check duplicates
	limit := uint64(3)
	withPayload := true
	resp, err := m.points.Search(ctx, &pb.SearchPoints{
		CollectionName: collectionName,
		Vector:         embedding,
		Limit:          limit,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: withPayload}},
		Filter: &pb.Filter{
			Must: []*pb.Condition{
				{
					ConditionOneOf: &pb.Condition_Field{
						Field: &pb.FieldCondition{
							Key: "user_id",
							Match: &pb.Match{
								MatchValue: &pb.Match_Keyword{Keyword: userID},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("dedup search: %w", err)
	}

	// 3. Check if any existing fact is too similar
	for _, scored := range resp.GetResult() {
		if scored.GetScore() >= dedupThreshold {
			// Duplicate found — just update timestamp of existing fact in SQLite
			factIDStr := ""
			for k, v := range scored.GetPayload() {
				if k == "fact_id" {
					factIDStr = v.GetStringValue()
				}
			}
			if factIDStr != "" {
				var factID int64
				fmt.Sscanf(factIDStr, "%d", &factID)
				if factID > 0 {
					_ = m.repo.UpdateFactTimestamp(ctx, factID)
				}
			}
			m.logger.Debug().
				Str("user_id", userID).
				Str("fact", fact.Fact).
				Float32("dup_score", scored.GetScore()).
				Msg("[factmemory] duplicate fact, updated timestamp")
			return nil
		}
	}

	// 4. Not a duplicate — store in SQLite
	factID, err := m.repo.InsertFact(ctx, repository.InsertFactParams{
		UserID:          userID,
		FactText:        fact.Fact,
		Category:        fact.Category,
		SourceMessageID: sql.NullInt64{}, // optional, not tracked here
	})

	if err != nil {
		return fmt.Errorf("insert fact: %w", err)
	}

	// 5. Store in Qdrant
	pointID := deterministicUUID(fmt.Sprintf("fact-%s-%d", userID, factID))
	waitUpsert := true

	payload := map[string]*pb.Value{
		"user_id":   {Kind: &pb.Value_StringValue{StringValue: userID}},
		"fact_id":   {Kind: &pb.Value_StringValue{StringValue: fmt.Sprintf("%d", factID)}},
		"fact_text": {Kind: &pb.Value_StringValue{StringValue: fact.Fact}},
		"category":  {Kind: &pb.Value_StringValue{StringValue: fact.Category}},
	}

	_, err = m.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collectionName,
		Wait:           &waitUpsert,
		Points: []*pb.PointStruct{
			{
				Id: &pb.PointId{
					PointIdOptions: &pb.PointId_Uuid{Uuid: pointID},
				},
				Vectors: &pb.Vectors{
					VectorsOptions: &pb.Vectors_Vector{
						Vector: &pb.Vector{Data: embedding},
					},
				},
				Payload: payload,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("qdrant upsert: %w", err)
	}

	m.logger.Info().
		Str("user_id", userID).
		Int64("fact_id", factID).
		Str("category", fact.Category).
		Str("fact", fact.Fact).
		Msg("[factmemory] new fact stored")

	return nil
}

// ─── Fact Management ───────────────────────────────────────────────

func (m *manager) DeactivateFact(ctx context.Context, userID string, factID int64) error {
	// 1. Deactivate in SQLite
	if err := m.repo.DeactivateFact(ctx, factID); err != nil {
		return fmt.Errorf("deactivate fact in sqlite: %w", err)
	}

	// 2. Delete from Qdrant
	pointID := deterministicUUID(fmt.Sprintf("fact-%s-%d", userID, factID))
	waitDelete := true

	_, err := m.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: collectionName,
		Wait:           &waitDelete,
		Points: pb.NewPointsSelector(
			&pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: pointID}},
		),
	})
	if err != nil {
		return fmt.Errorf("delete point in qdrant: %w", err)
	}

	m.logger.Info().
		Str("user_id", userID).
		Int64("fact_id", factID).
		Msg("[factmemory] fact deactivated and removed from Qdrant")

	return nil
}

// ─── Helpers ───────────────────────────────────────────────────────

func (m *manager) ensureCollection(ctx context.Context) error {
	listResp, err := m.collections.List(ctx, &pb.ListCollectionsRequest{})
	if err != nil {
		return fmt.Errorf("list collections: %w", err)
	}

	for _, c := range listResp.GetCollections() {
		if c.GetName() == collectionName {
			m.logger.Info().
				Str("collection", collectionName).
				Msg("[factmemory] collection already exists")
			return nil
		}
	}

	distance := pb.Distance_Cosine
	_, err = m.collections.Create(ctx, &pb.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     vectorSize,
					Distance: distance,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}

	m.logger.Info().
		Str("collection", collectionName).
		Msg("[factmemory] collection created")
	return nil
}

func deterministicUUID(id string) string {
	h := sha256.Sum256([]byte(id))
	u, _ := uuid.FromBytes(h[:16])
	return u.String()
}

func (m *manager) Close() {
	close(m.done)
	close(m.jobs)
	m.conn.Close()
	m.logger.Info().Msg("[factmemory] shut down")
}

// this func to purge fact in qdrant with user_id
func (m *manager) PurgeMemory(ctx context.Context, userID string) error {
	wait := true
	_, err := m.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: collectionName,
		Wait:           &wait,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Filter{
				Filter: &pb.Filter{
					Must: []*pb.Condition{
						{
							ConditionOneOf: &pb.Condition_Field{
								Field: &pb.FieldCondition{
									Key: "user_id",
									Match: &pb.Match{
										MatchValue: &pb.Match_Keyword{
											Keyword: userID,
										},
									},
								},
							},
						},
						{
							ConditionOneOf: &pb.Condition_Field{
								Field: &pb.FieldCondition{
									Key: "type",
									Match: &pb.Match{
										MatchValue: &pb.Match_Keyword{
											Keyword: strings.TrimSuffix(collectionName, "s"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	return err
}
