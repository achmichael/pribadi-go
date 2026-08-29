package llm

import "sync"

type hybridRouter struct {
	mu     sync.RWMutex
	ollama Client
	openai Client
	// can add gemini, claude, etc.
}

func NewHybridRouter(ollama Client, openai Client) Router {
	return &hybridRouter{
		ollama: ollama,
		openai: openai,
	}
}

func (r *hybridRouter) AddProvider(provider Provider, client Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch provider {
	case ProviderOpenAI:
		r.openai = client
	}
}

func (r *hybridRouter) Route(intent string, needsRAG, needsMemory bool) Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Selective routing logic (zero data exposure)
	// If the task involves personal documents (RAG) or personal memory, ALWAYS use local (Ollama)
	if needsRAG || needsMemory {
		return ProviderOllama
	}

	// For general tasks, we can route to cloud providers if configured
	// E.g., chitchat, general knowledge, translation
	switch intent {
	case "chitchat", "ask_information", "clarify", "translation", "summarize_general":
		if r.openai != nil {
			return ProviderOpenAI
		}
	}

	// Default fallback is always local
	return ProviderOllama
}

func (r *hybridRouter) GetClient(provider Provider) (Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	switch provider {
	case ProviderOllama:
		return r.ollama, nil
	case ProviderOpenAI:
		if r.openai == nil {
			return r.ollama, nil // fallback if not configured
		}
		return r.openai, nil
	default:
		return r.ollama, nil
	}
}

func (r *hybridRouter) GetDefaultClient() Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ollama
}
