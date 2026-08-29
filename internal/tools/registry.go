package tools

import (
	"github.com/achmichael/pribadi-go/pkg/llm"
)

type Registry struct {
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Schemas() []llm.Tool {
	return []llm.Tool{
		{
			Type: "function",
			Function: llm.Function{
				Name:        "get_document_metadata",
				Description: "Retrieve structured metadata (like author, title, year) for a specific document ID. Use this when the user asks about factual information like 'who is the author of this document', 'what is the title', or 'who wrote it'.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"document_id": map[string]interface{}{
							"type":        "string",
							"description": "The ID of the document to get metadata for",
						},
					},
					"required": []string{"document_id"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.Function{
				Name:        "search_documents",
				Description: "Search uploaded documents for relevant information. Call ONLY if the question requires referencing specific document content that was previously uploaded by the user. Do NOT call for casual chat, general knowledge questions, greetings, or thanks.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query to find relevant document chunks",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.Function{
				Name:        "recall_memory",
				Description: "Retrieve stored personal facts and preferences about the user. Call ONLY if the question asks about user-specific information (their name, preferences, past facts they told you) that you don't already have in context. Do NOT call for greetings, thanks, or general conversation.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query to find relevant personal facts",
						},
					},
					"required": []string{"query"},
				},
			},
		},
	}
}
