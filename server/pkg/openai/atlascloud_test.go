package openai

import "testing"

func TestIsAtlasCloudProvider(t *testing.T) {
	for _, provider := range []string{"atlascloud", "atlas-cloud", "atlas", " AtlasCloud "} {
		if !IsAtlasCloudProvider(provider) {
			t.Fatalf("expected %q to be recognized as Atlas Cloud", provider)
		}
	}

	if IsAtlasCloudProvider("openai") {
		t.Fatal("openai should not be recognized as Atlas Cloud")
	}
}

func TestApplyAtlasCloudTextConfigDefaults(t *testing.T) {
	var cfg Config

	ApplyAtlasCloudTextConfig(&cfg, "", "", "")

	if cfg.Provider != "openai" {
		t.Fatalf("expected provider to use OpenAI-compatible path, got %q", cfg.Provider)
	}
	if cfg.OpenAIBaseURL != AtlasCloudDefaultBaseURL {
		t.Fatalf("expected default base URL %q, got %q", AtlasCloudDefaultBaseURL, cfg.OpenAIBaseURL)
	}
	if cfg.OpenAIModel != AtlasCloudDefaultModel {
		t.Fatalf("expected default model %q, got %q", AtlasCloudDefaultModel, cfg.OpenAIModel)
	}
}

func TestApplyAtlasCloudTextConfigResolvesAPIKeyAlias(t *testing.T) {
	t.Setenv("ATLAS_CLOUD_API_KEY", "alias-key")

	var cfg Config
	ApplyAtlasCloudTextConfig(&cfg, "https://api.atlascloud.ai/v1/", "ATLAS_CLOUD_API_KEY", AtlasCloudReasoningModel)

	if cfg.OpenAIBaseURL != AtlasCloudDefaultBaseURL {
		t.Fatalf("expected normalized base URL %q, got %q", AtlasCloudDefaultBaseURL, cfg.OpenAIBaseURL)
	}
	if cfg.OpenAIKey != "alias-key" {
		t.Fatalf("expected env-resolved API key, got %q", cfg.OpenAIKey)
	}
	if cfg.OpenAIModel != AtlasCloudReasoningModel {
		t.Fatalf("expected model %q, got %q", AtlasCloudReasoningModel, cfg.OpenAIModel)
	}
}
