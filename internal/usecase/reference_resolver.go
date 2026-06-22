package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/rs/zerolog"
)

type ReferenceResolver interface {
	ResolveQuery(ctx context.Context, userID, userText, sessionID string) (string, string)
}

type referenceResolver struct {
	llm    *ollama.OllamaClient
	repo   repository.Repository
	logger *zerolog.Logger
}

func NewReferenceResolver(llm *ollama.OllamaClient, repo repository.Repository, logger *zerolog.Logger) ReferenceResolver {
	return &referenceResolver{
		llm:    llm,
		repo:   repo,
		logger: logger,
	}
}

type ResolutionResult struct {
	RewrittenQuery   string `json:"rewritten_query"`
	TargetDocumentID string `json:"target_document_id"`
}

const referenceResolutionPrompt = `You are the DOCUMENT REFERENCE RESOLUTION AGENT.
You are responsible for resolving document references before answering any question.
The user may refer to previously uploaded documents without explicitly mentioning their names.

You have access to:
* Uploaded document registry
* Conversation history

## DOCUMENT REGISTRY
%s

## CONVERSATION HISTORY (Last 5 messages)
%s

## REFERENCE RESOLUTION RULES
Step 1: Analyze whether the user is referring to a document indirectly.
Possible references include: dokumen ini, dokumen tersebut, file ini, file tersebut, laporan ini, laporan tersebut, dll.
Step 2: Attempt to resolve the reference using priority:
1. Most recently uploaded document
2. Most recently discussed document
3. Document referenced in the previous user message
4. Document referenced in the previous assistant response
Step 3: If exactly one document can be resolved, rewrite the internal query using the resolved document.
Example: User: "What is the title of that document?", Rewritten: "What is the title of document_id=doc_002?"

IMPORTANT: Return ONLY a valid JSON object with the following structure, and no other text:
{
  "rewritten_query": "the rewritten query containing the context, or the original query if no reference is found",
  "target_document_id": "the resolved document ID, or empty string if none"
}

Current User Query: %s
`

func (r *referenceResolver) ResolveQuery(ctx context.Context, userID, userText, sessionID string) (string, string) {
	// 1. Fetch latest documents
	docs, err := r.repo.GetLatestUserDocuments(ctx, userID, 3)
	if err != nil {
		r.logger.Warn().Err(err).Msg("[resolver] failed to fetch latest documents")
	}

	if len(docs) == 0 {
		// No documents uploaded, no need to resolve references
		return userText, ""
	}

	var docRegistry strings.Builder
	for _, doc := range docs {
		docRegistry.WriteString(fmt.Sprintf("- document_id: %s\n  file_name: %s\n  title: %s\n  author: %s\n  uploaded_at: %s\n\n",
			doc.ID, doc.FileName, doc.Title, doc.Author, doc.CreatedAt.Format("2006-01-02 15:04:05")))
	}

	// 2. Fetch conversation history
	msgs, err := r.repo.ListMessagesByUserSession(ctx, userID, sessionID, 5)
	if err != nil {
		r.logger.Warn().Err(err).Msg("[resolver] failed to fetch conversation history")
	}

	var history strings.Builder
	for _, msg := range msgs {
		history.WriteString(fmt.Sprintf("%s: %s\n", strings.ToUpper(msg.Role), msg.Content))
	}

	// 3. Prepare prompt
	prompt := fmt.Sprintf(referenceResolutionPrompt, docRegistry.String(), history.String(), userText)
	messages := []ollama.ChatMessage{
		{Role: "user", Content: prompt},
	}

	// 4. Call LLM
	reply, err := r.llm.ChatJSON(ctx, messages)
	if err != nil {
		r.logger.Error().Err(err).Msg("[resolver] LLM call failed")
		return userText, ""
	}

	reply = strings.TrimSpace(reply)
	if reply == "" || reply == "{}" {
		return userText, ""
	}

	// 5. Parse JSON
	var result ResolutionResult
	if err := json.Unmarshal([]byte(reply), &result); err != nil {
		r.logger.Warn().Err(err).Str("reply", reply).Msg("[resolver] failed to parse resolution JSON")
		return userText, ""
	}

	if result.RewrittenQuery != "" {
		r.logger.Info().
			Str("original", userText).
			Str("rewritten", result.RewrittenQuery).
			Str("target_doc", result.TargetDocumentID).
			Msg("[resolver] query resolved")
		return result.RewrittenQuery, result.TargetDocumentID
	}

	return userText, ""
}
