package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"spiritFruit/pkg/logger"
)

var (
	globalHTTPClient *http.Client
	once             sync.Once
)

func initGlobalHTTPClient() {
	apiTransport := &http.Transport{
		// 项目部署环境可能需要通过 HTTP(S)_PROXY 访问境外/跨区域模型服务。
		// 自定义 Transport 不会自动继承代理配置，必须显式设置。
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 120 * time.Second,
	}
	globalHTTPClient = &http.Client{
		Timeout:   300 * time.Second,
		Transport: apiTransport,
	}
}

// NewProvider 工厂方法：根据配置返回对应的实现
func NewProvider(cfg Config) Provider {
	once.Do(func() { initGlobalHTTPClient() })
	return newProvider(cfg, globalHTTPClient)
}

// NewProviderWithTimeout 用于配置测试等短请求，避免网络不可达时占用长连接。
func NewProviderWithTimeout(cfg Config, timeout time.Duration) Provider {
	once.Do(func() { initGlobalHTTPClient() })
	client := *globalHTTPClient
	client.Timeout = timeout
	return newProvider(cfg, &client)
}

func newProvider(cfg Config, client *http.Client) Provider {
	switch cfg.Provider {
	case "getgoapi":
		return &GetGoAPIClient{Config: cfg, client: client}
	case "gemini":
		return &GeminiClient{Config: cfg, client: client}
	case "doubao", "volces", "volcengine":
		return &DoubaoClient{Config: cfg, client: client}
	case "vertex", "gcp":
		return &VertexClient{Config: cfg, client: client}
	case "openai":
		fallthrough
	default:
		return &OpenAIClient{Config: cfg, client: client}
	}
}

// doRequest 通用泛型请求
func doRequest[T any](client *http.Client, method, url string, headers map[string]string, payload interface{}) (T, error) {
	var result T
	var reqBody io.Reader

	if payload != nil {
		jsonBytes, _ := json.Marshal(payload)
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return result, fmt.Errorf("AI request failed (%s %s): %w", method, url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		logger.Error("AI API Error", zap.String("url", url), zap.String("body", string(bodyBytes)))
		message := string(bodyBytes)
		if len(message) > 1000 {
			message = message[:1000]
		}
		return result, fmt.Errorf("API error: %d: %s", resp.StatusCode, message)
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return result, fmt.Errorf("unmarshal failed: %v", err)
	}

	return result, nil
}
