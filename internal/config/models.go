package config

// ProviderModels maps providers to their available models
var ProviderModels = map[string][]string{
	"anthropic": {
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
		"claude-2.1",
		"claude-2.0",
		"claude-instant-1.2",
	},
	"openai": {
		"gpt-4o",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
	},
	"openrouter": {
		"anthropic/claude-3-opus-20240229",
		"anthropic/claude-3-sonnet-20240229",
		"anthropic/claude-3-haiku-20240307",
		"openai/gpt-4o",
		"openai/gpt-4-turbo",
		"openai/gpt-4",
		"openai/gpt-3.5-turbo",
		"google/gemini-1.5-pro",
		"google/gemini-1.0-pro",
		"meta-llama/llama-3-70b-instruct",
		"meta-llama/llama-3-8b-instruct",
		"mistral/mistral-large-latest",
		"mistral/mistral-medium-latest",
		"mistral/mistral-small-latest",
	},
}

// GetModelsForProvider returns the available models for a provider
func GetModelsForProvider(provider string) []string {
	if models, ok := ProviderModels[provider]; ok {
		return models
	}
	return []string{}
}

// IsValidModel checks if a model is valid for a provider
func IsValidModel(provider, model string) bool {
	models := GetModelsForProvider(provider)
	for _, m := range models {
		if m == model {
			return true
		}
	}
	return false
}
