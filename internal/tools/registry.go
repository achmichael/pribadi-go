package tools

import (
	"github.com/achmichael/pribadi-go/pkg/ollama"
)

type Registry struct {
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Schemas() []ollama.Tool {
	return []ollama.Tool{
		{
			Type: "function",
			Function: ollama.Function{
				Name:        "search_documents",
				Description: "Search uploaded documents (RAG) for relevant information. Use when user asks about uploaded files, papers, or documents.",
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
			Function: ollama.Function{
				Name:        "search_memory",
				Description: "Search personal facts and memories about the user. Use when user asks about their own information (name, preferences, past conversations).",
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
