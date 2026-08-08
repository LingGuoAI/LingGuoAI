package services

import (
	"spiritFruit/pkg/config"
	"spiritFruit/pkg/openai"
	"strings"
)

func NewTextProvider(adminID uint64) (openai.Provider, string) {
	cfg := openai.Config{Provider: config.GetString("ai.provider", "openai"), OpenAIBaseURL: config.GetString("ai.openai.base_url"), OpenAIKey: config.GetString("ai.openai.api_key"), OpenAIModel: config.GetString("ai.openai.model"), GetGoAPIBaseURL: config.GetString("ai.getgoapi.base_url"), GetGoAPIKey: config.GetString("ai.getgoapi.api_key"), GetGoAPIModel: config.GetString("ai.getgoapi.model"), GeminiBaseURL: config.GetString("ai.gemini.base_url"), GeminiKey: config.GetString("ai.gemini.api_key"), GeminiModel: config.GetString("ai.gemini.model"), DoubaoBaseURL: config.GetString("ai.doubao.base_url"), DoubaoKey: config.GetString("ai.doubao.api_key"), DoubaoModel: config.GetString("ai.doubao.model"), VertexKey: config.GetString("ai.vertex.api_key"), VertexModel: config.GetString("ai.vertex.model")}
	if err, db := new(AiConfigService).GetActiveConfigByType("text", &adminID); err == nil && db.ID > 0 {
		p := strings.ToLower(*db.Provider)
		model := ""
		if len(db.Model) > 0 {
			model = db.Model[0]
		}
		cfg.Provider = p
		switch p {
		case "getgoapi":
			cfg.GetGoAPIBaseURL = *db.BaseUrl
			cfg.GetGoAPIKey = *db.ApiKey
			cfg.GetGoAPIModel = model
		case "gemini", "google":
			cfg.Provider = "gemini"
			cfg.GeminiBaseURL = *db.BaseUrl
			cfg.GeminiKey = *db.ApiKey
			cfg.GeminiModel = model
		case "doubao", "volcengine", "volces":
			cfg.Provider = "doubao"
			cfg.DoubaoBaseURL = *db.BaseUrl
			cfg.DoubaoKey = *db.ApiKey
			cfg.DoubaoModel = model
		case "vertex":
			cfg.VertexKey = *db.ApiKey
			cfg.VertexModel = model
		default:
			cfg.Provider = "openai"
			cfg.OpenAIBaseURL = *db.BaseUrl
			cfg.OpenAIKey = *db.ApiKey
			cfg.OpenAIModel = model
		}
		return openai.NewProvider(cfg), p + ":" + model
	}
	return openai.NewProvider(cfg), cfg.Provider
}
