package llm

type Provider string

const (
	ProviderOllama Provider = "ollama"
	ProviderOpenAI Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGemini Provider = "gemini"
	ProviderGrok Provider = "grok"
	// add others as needed
)

type Router interface {
	Route(intent string, needsRAG, needsMemory bool) Provider
	GetClient(provider Provider) (Client, error)
	GetDefaultClient() Client
	AddProvider(provider Provider, client Client)
}
