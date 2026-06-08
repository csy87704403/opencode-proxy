package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// 判断模型是否被环境变量配置的黑名单过滤
func isBlacklisted(modelID string) bool {
	blacklistEnv := os.Getenv("BLACKLIST_MODELS")
	if blacklistEnv == "" {
		return false
	}
	// 将环境变量通过逗号分割
	models := strings.Split(blacklistEnv, ",")
	for _, m := range models {
		if strings.TrimSpace(m) == modelID {
			return true
		}
	}
	return false
}

type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

func ModelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. 尝试动态从 OpenCode 上游拉取最新模型列表并过滤
	client := &http.Client{Timeout: 5 * time.Second}
	reqUpstream, err := http.NewRequestWithContext(r.Context(), "GET", "https://opencode.ai/zen/v1/models", nil)
	if err == nil {
		reqUpstream.Header.Set("Authorization", "Bearer public")
		reqUpstream.Header.Set("x-opencode-client", "desktop")

		resp, err := client.Do(reqUpstream)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()

			var upstreamResp ModelsResponse
			if err := json.NewDecoder(resp.Body).Decode(&upstreamResp); err == nil {
				// 过滤逻辑：排除环境变量黑名单，且只保留以 "-free" 结尾的模型或 "big-pickle"
				var filteredData []ModelInfo
				for _, model := range upstreamResp.Data {
					if isBlacklisted(model.ID) {
						continue
					}
					if strings.HasSuffix(model.ID, "-free") || model.ID == "big-pickle" {
						filteredData = append(filteredData, model)
					}
				}
				upstreamResp.Data = filteredData

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(upstreamResp)
				return
			}
		}
	}

	// 2. 如果上游请求失败，降级使用本地备用列表
	fallbackResp := ModelsResponse{
		Object: "list",
		Data: []ModelInfo{
			{ID: "big-pickle", Object: "model", OwnedBy: "opencode"},
			{ID: "nemotron-3-super-free", Object: "model", OwnedBy: "opencode"},
			{ID: "deepseek-v4-flash-free", Object: "model", OwnedBy: "opencode"},
			{ID: "mimo-v2.5-free", Object: "model", OwnedBy: "opencode"},
			{ID: "nemotron-3-ultra-free", Object: "model", OwnedBy: "opencode"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(fallbackResp)
}
