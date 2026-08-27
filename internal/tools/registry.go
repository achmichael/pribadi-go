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
			Function: ollama.Function{
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
