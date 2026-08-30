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
		{
			Type: "function",
			Function: llm.Function{
				Name:        "add_monitor_task",
				Description: "Add a new monitoring task to watch a value from an API or website. The task will run in the background and notify the user when the configured condition is met.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Name for this monitor task",
						},
						"category": map[string]interface{}{
							"type":        "string",
							"description": "Free-form category label (e.g. stock, forex, competitor, weather)",
						},
						"source_type": map[string]interface{}{
							"type":        "string",
							"description": "Data source type: 'http_api' or 'web_scrape'",
							"enum":        []string{"http_api", "web_scrape"},
						},
						"source_config": map[string]interface{}{
							"type":        "string",
							"description": "JSON config for the data source. For http_api: {url, method, headers, extract_path}. For web_scrape: {url, description}",
						},
						"condition_mode": map[string]interface{}{
							"type":        "string",
							"description": "Condition evaluation mode: 'structural' (numeric comparison, no LLM) or 'nl_judge' (natural language, uses LLM per cycle)",
							"enum":        []string{"structural", "nl_judge"},
						},
						"condition_config": map[string]interface{}{
							"type":        "string",
							"description": "JSON config for condition. For structural: {operator, value, text}. For nl_judge: {prompt}",
						},
					},
					"required": []string{"name", "source_type", "source_config", "condition_mode", "condition_config"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.Function{
				Name:        "remove_monitor_task",
				Description: "Remove (delete) an existing monitor task by its ID.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "number",
							"description": "The ID of the monitor task to remove",
						},
					},
					"required": []string{"id"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.Function{
				Name:        "list_monitor_tasks",
				Description: "List all monitor tasks for the current user, optionally filtered by category.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"category": map[string]interface{}{
							"type":        "string",
							"description": "Optional category filter",
						},
					},
				},
			},
		},
	}
}
