package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/achmichael/pribadi-go/pkg/utils"
	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SearchResult represents a search result from the vector database.
type SearchResult struct {
	ID       string
	Content  string
	Score    float32
	Metadata map[string]string
}

// VectorRepository defines the interface for vector database operations.
type VectorRepository interface {
	UpsertDocument(ctx context.Context, id string, content string, metadata map[string]string) error
	Search(ctx context.Context, query string, topK int, filterDocID string) ([]SearchResult, error)
	SearchWithFilters(ctx context.Context, query string, topK int, filters map[string]string) ([]SearchResult, error)
	PurgeRAGDocuments(ctx context.Context, userID string) error
	Close() error
}

// qdrantVectorRepo implements VectorRepository using Qdrant gRPC.
type qdrantVectorRepo struct {
	conn        *grpc.ClientConn
	points      pb.PointsClient
	collections pb.CollectionsClient
	embedder    *utils.OllamaEmbedder
	collName    string
	vectorSize  uint64
	logger      *zerolog.Logger
}

// NewVectorRepository creates a new vector repository backed by Qdrant (gRPC).
//
// qdrantAddr should be "host:port" e.g. "localhost:6334".
func NewVectorRepository(qdrantAddr string, ollamaBaseURL string, logger *zerolog.Logger) (VectorRepository, error) {
	start := time.Now()

	conn, err := grpc.NewClient(
		qdrantAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("qdrant grpc dial: %w", err)
	}

	logger.Info().
		Str("addr", qdrantAddr).
		Dur("dial_ms", time.Since(start)).
		Msg("[vector_repo] qdrant gRPC connected")

	pointsClient := pb.NewPointsClient(conn)
	collectionsClient := pb.NewCollectionsClient(conn)

	embedder := utils.NewOllamaEmbedder(ollamaBaseURL, "nomic-embed-text")

	collName := "documents"
	const vectorSize uint64 = 768 // nomic-embed-text dimension

	repo := &qdrantVectorRepo{
		conn:        conn,
		points:      pointsClient,
		collections: collectionsClient,
		embedder:    embedder,
		collName:    collName,
		vectorSize:  vectorSize,
		logger:      logger,
	}

	if err := repo.ensureCollection(context.Background()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ensure collection: %w", err)
	}

	logger.Info().
		Str("collection", collName).
		Uint64("vector_size", vectorSize).
		Dur("total_ms", time.Since(start)).
		Msg("[vector_repo] qdrant ready")

	return repo, nil
}

// ensureCollection creates the collection if it doesn't exist.
func (r *qdrantVectorRepo) ensureCollection(ctx context.Context) error {
	// Check if collection exists
	listResp, err := r.collections.List(ctx, &pb.ListCollectionsRequest{})
	if err != nil {
		return fmt.Errorf("list collections: %w", err)
	}

	for _, c := range listResp.GetCollections() {
		if c.GetName() == r.collName {
			r.logger.Info().
				Str("collection", r.collName).
				Msg("[vector_repo] collection already exists")
			return nil
		}
	}

	// Create collection
	distance := pb.Distance_Cosine
	_, err = r.collections.Create(ctx, &pb.CreateCollection{
		CollectionName: r.collName,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     r.vectorSize,
					Distance: distance,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}

	r.logger.Info().
		Str("collection", r.collName).
		Msg("[vector_repo] collection created")
	return nil
}

// deterministicUUID derives a stable UUID from an arbitrary string ID
// using SHA-256 → UUID v4 format. Same input always gives same UUID.
func deterministicUUID(id string) string {
	h := sha256.Sum256([]byte(id))
	u, _ := uuid.FromBytes(h[:16])
	return u.String()
}

func (r *qdrantVectorRepo) UpsertDocument(ctx context.Context, id string, content string, metadata map[string]string) error {
	start := time.Now()

	// Generate embedding
	embedStart := time.Now()
	embedding, err := r.embedder.Embed(ctx, content)
	embedDur := time.Since(embedStart)
	if err != nil {
		r.logger.Error().Err(err).
			Str("doc_id", id).
			Int("content_len", len(content)).
			Dur("embed_ms", embedDur).
			Msg("[vector_repo] embedding failed")
		return fmt.Errorf("generate embedding: %w", err)
	}

	r.logger.Debug().
		Str("doc_id", id).
		Int("content_len", len(content)).
		Int("embedding_dim", len(embedding)).
		Dur("embed_ms", embedDur).
		Msg("[vector_repo] embedding generated")

	// Build payload: store content + original ID + all metadata
	payload := map[string]*pb.Value{
		"content":     {Kind: &pb.Value_StringValue{StringValue: content}},
		"original_id": {Kind: &pb.Value_StringValue{StringValue: id}},
	}
	for k, v := range metadata {
		payload[k] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: v}}
	}
	
	if metadata["chunk_index"] == "0" {
		r.logger.Info().
			Str("doc_id", id).
			Int("payload_metadata_count", len(payload)).
			Msg("[AUDIT] vector_repo UpsertDocument checking payload for chunk 0")
	}

	// Derive deterministic UUID from string ID
	pointUUID := deterministicUUID(id)

	// Upsert point
	storeStart := time.Now()
	waitUpsert := true
	_, err = r.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: r.collName,
		Wait:           &waitUpsert,
		Points: []*pb.PointStruct{
			{
				Id: &pb.PointId{
					PointIdOptions: &pb.PointId_Uuid{Uuid: pointUUID},
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
	storeDur := time.Since(storeStart)
	if err != nil {
		r.logger.Error().Err(err).
			Str("doc_id", id).
			Dur("store_ms", storeDur).
			Msg("[vector_repo] upsert failed")
		return fmt.Errorf("upsert point: %w", err)
	}

	r.logger.Debug().
		Str("doc_id", id).
		Dur("embed_ms", embedDur).
		Dur("store_ms", storeDur).
		Dur("total_ms", time.Since(start)).
		Msg("[vector_repo] upsert done")

	return nil
}

func (r *qdrantVectorRepo) Search(ctx context.Context, query string, topK int, filterDocID string) ([]SearchResult, error) {
	start := time.Now()

	// Generate query embedding
	embedStart := time.Now()
	queryEmbedding, err := r.embedder.Embed(ctx, query)
	embedDur := time.Since(embedStart)
	if err != nil {
		r.logger.Error().Err(err).
			Int("query_len", len(query)).
			Dur("embed_ms", embedDur).
			Msg("[vector_repo] query embedding failed")
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}

	r.logger.Debug().
		Int("query_len", len(query)).
		Dur("embed_ms", embedDur).
		Msg("[vector_repo] query embedding generated")

	// Build filter if provided
	var filter *pb.Filter
	if filterDocID != "" {
		filter = &pb.Filter{
			Must: []*pb.Condition{
				{
					ConditionOneOf: &pb.Condition_Field{
						Field: &pb.FieldCondition{
							Key: "document_id",
							Match: &pb.Match{
								MatchValue: &pb.Match_Keyword{
									Keyword: filterDocID,
								},
							},
						},
					},
				},
			},
		}
	}

	r.logger.Info().
		Int("query_len", len(query)).
		Int("top_k", topK).
		Str("filterDocID", filterDocID).
		Msg("[AUDIT] vector_repo Search executing with filterDocID")

	searchStart := time.Now()
	limit := uint64(topK)
	withPayload := true
	resp, err := r.points.Search(ctx, &pb.SearchPoints{
		CollectionName: r.collName,
		Vector:         queryEmbedding,
		Limit:          limit,
		Filter:         filter,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	searchDur := time.Since(searchStart)
	if err != nil {
		r.logger.Error().Err(err).
			Dur("search_ms", searchDur).
			Msg("[vector_repo] search failed")
		return nil, fmt.Errorf("search points: %w", err)
	}

	// Convert results
	results := make([]SearchResult, 0, len(resp.GetResult()))
	for _, scored := range resp.GetResult() {
		sr := SearchResult{
			Score:    scored.GetScore(),
			Metadata: make(map[string]string),
		}

		// Extract payload
		for k, v := range scored.GetPayload() {
			switch k {
			case "content":
				sr.Content = v.GetStringValue()
			case "original_id":
				sr.ID = v.GetStringValue()
			default:
				sr.Metadata[k] = v.GetStringValue()
			}
		}

		// Fallback: if original_id not in payload, use UUID
		if sr.ID == "" {
			if uid := scored.GetId().GetUuid(); uid != "" {
				sr.ID = uid
			}
		}

		results = append(results, sr)
	}

	r.logger.Info().
		Int("query_len", len(query)).
		Int("top_k", topK).
		Int("results", len(results)).
		Dur("embed_ms", embedDur).
		Dur("search_ms", searchDur).
		Dur("total_ms", time.Since(start)).
		Msg("[vector_repo] search done")
		
	if len(results) > 0 {
		r.logger.Info().
			Str("top_result_id", results[0].ID).
			Float32("top_result_score", results[0].Score).
			Msg("[AUDIT] vector_repo Search top result")
	}

	return results, nil
}

func (r *qdrantVectorRepo) SearchWithFilters(ctx context.Context, query string, topK int, filters map[string]string) ([]SearchResult, error) {
	start := time.Now()

	embedStart := time.Now()
	queryEmbedding, err := r.embedder.Embed(ctx, query)
	embedDur := time.Since(embedStart)
	if err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}

	var conditions []*pb.Condition
	for k, v := range filters {
		if v == "" {
			continue
		}
		conditions = append(conditions, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key:   k,
					Match: &pb.Match{MatchValue: &pb.Match_Keyword{Keyword: v}},
				},
			},
		})
	}

	var filter *pb.Filter
	if len(conditions) > 0 {
		filter = &pb.Filter{Must: conditions}
	}

	r.logger.Info().
		Int("query_len", len(query)).
		Int("top_k", topK).
		Int("filter_count", len(conditions)).
		Msg("[vector_repo] SearchWithFilters executing")

	searchStart := time.Now()
	limit := uint64(topK)
	withPayload := true
	resp, err := r.points.Search(ctx, &pb.SearchPoints{
		CollectionName: r.collName,
		Vector:         queryEmbedding,
		Limit:          limit,
		Filter:         filter,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	searchDur := time.Since(searchStart)
	if err != nil {
		return nil, fmt.Errorf("search points: %w", err)
	}

	results := make([]SearchResult, 0, len(resp.GetResult()))
	for _, scored := range resp.GetResult() {
		sr := SearchResult{
			Score:    scored.GetScore(),
			Metadata: make(map[string]string),
		}
		for k, v := range scored.GetPayload() {
			switch k {
			case "content":
				sr.Content = v.GetStringValue()
			case "original_id":
				sr.ID = v.GetStringValue()
			default:
				sr.Metadata[k] = v.GetStringValue()
			}
		}
		if sr.ID == "" {
			if uid := scored.GetId().GetUuid(); uid != "" {
				sr.ID = uid
			}
		}
		results = append(results, sr)
	}

	r.logger.Info().
		Int("results", len(results)).
		Dur("embed_ms", embedDur).
		Dur("search_ms", searchDur).
		Dur("total_ms", time.Since(start)).
		Msg("[vector_repo] SearchWithFilters done")

	return results, nil
}

func (r *qdrantVectorRepo) Close() error {
	r.logger.Info().Msg("[vector_repo] closing qdrant connection")
	return r.conn.Close()
}

func (r *qdrantVectorRepo) PurgeRAGDocuments(ctx context.Context, userID string) error {
	start := time.Now()
	wait := true
	// Build filter to select only this user's documents
	filter := &pb.Filter{
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
		},
	}

	// Delete all points matching the filter
	deleteResp, err := r.points.Delete(ctx, &pb.DeletePoints{
		CollectionName: r.collName,
		Points:         &pb.PointsSelector{PointsSelectorOneOf: &pb.PointsSelector_Filter{Filter: filter}},
		Wait:           &wait,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Dur("total_ms", time.Since(start)).Msg("[vector_repo] delete failed")
		return fmt.Errorf("delete points: %w", err)
	}

	r.logger.Info().
		Str("user_id", userID).
		Int64("deleted", int64(deleteResp.Result.GetOperationId())).
		Dur("total_ms", time.Since(start)).
		Msg("[vector_repo] purge complete")

	return nil
}
