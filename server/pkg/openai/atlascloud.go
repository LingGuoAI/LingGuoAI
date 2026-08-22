package openai

import (
	"os"
	"strings"
)

const (
	AtlasCloudProvider       = "atlascloud"
	AtlasCloudProviderAlias  = "atlas-cloud"
	AtlasProviderAlias       = "atlas"
	AtlasCloudDefaultBaseURL = "https://api.atlascloud.ai/v1"
	AtlasCloudDefaultModel   = "qwen/qwen3.5-flash"
	AtlasCloudReasoningModel = "deepseek-ai/deepseek-v4-pro"
)

// IsAtlasCloudProvider reports whether a provider value targets Atlas Cloud.
func IsAtlasCloudProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case AtlasCloudProvider, AtlasCloudProviderAlias, AtlasProviderAlias:
		return true
	default:
		return false
	}
}

// ApplyAtlasCloudTextConfig maps Atlas Cloud onto the existing OpenAI-compatible text path.
func ApplyAtlasCloudTextConfig(cfg *Config, baseURL, apiKey, model string) {
	cfg.Provider = "openai"
	cfg.OpenAIBaseURL = normalizeAtlasCloudBaseURL(baseURL)
	cfg.OpenAIKey = resolveAtlasCloudAPIKey(apiKey)
	cfg.OpenAIModel = strings.TrimSpace(model)
	if cfg.OpenAIModel == "" {
		cfg.OpenAIModel = AtlasCloudDefaultModel
	}
}

func normalizeAtlasCloudBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return AtlasCloudDefaultBaseURL
	}
	return baseURL
}

func resolveAtlasCloudAPIKey(apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	switch apiKey {
	case "", "ATLASCLOUD_API_KEY", "$ATLASCLOUD_API_KEY", "${ATLASCLOUD_API_KEY}":
		if envKey := os.Getenv("ATLASCLOUD_API_KEY"); envKey != "" {
			return envKey
		}
	case "ATLAS_CLOUD_API_KEY", "$ATLAS_CLOUD_API_KEY", "${ATLAS_CLOUD_API_KEY}":
		if envKey := os.Getenv("ATLAS_CLOUD_API_KEY"); envKey != "" {
			return envKey
		}
	}
	return apiKey
}
