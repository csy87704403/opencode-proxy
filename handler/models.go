package handler

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// 缓存可用模型列表与并发读写锁
var (
	cachedModels     []ModelInfo
	cachedMutex      sync.RWMutex
	initialCheckDone = false
)

// 兜底返回模型列表（在首次启动且检测未完成时使用）
var defaultBackupModels = []ModelInfo{
	{ID: "big-pickle", Object: "model", OwnedBy: "opencode"},
	{ID: "nemotron-3-super-free", Object: "model", OwnedBy: "opencode"},
	{ID: "deepseek-v4-flash-free", Object: "model", OwnedBy: "opencode"},
	{ID: "mimo-v2.5-free", Object: "model", OwnedBy: "opencode"},
	{ID: "nemotron-3-ultra-free", Object: "model", OwnedBy: "opencode"},
}

func init() {
	// 启动后台异步检测协程
	go startPeriodicProbing()
}

// 定时循环检测模型可用性
func startPeriodicProbing() {
	// 首次启动立即执行一次检测
	probeModels()

	// 之后每隔 30 分钟在后台自动检测一次
	ticker := time.NewTicker(30 * time.Minute)
	for range ticker.C {
		probeModels()
	}
}

// 探测函数：并发向 OpenCode 验证每个候选模型
func probeModels() {
	log.Println("[Probe] Starting dynamic model health check...")

	candidates := fetchUpstreamCandidates()
	if len(candidates) == 0 {
		log.Println("[Probe] No candidates retrieved from upstream, keeping existing cache.")
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var verifiedModels []ModelInfo

	// 并发探测候选模型的可用状态
	for _, candidate := range candidates {
		wg.Add(1)
		go func(model ModelInfo) {
			defer wg.Done()
			if testModelAvailability(model.ID) {
				mu.Lock()
				verifiedModels = append(verifiedModels, model)
				mu.Unlock()
				log.Printf("[Probe] Model %s is active and free.", model.ID)
			} else {
				log.Printf("[Probe] Model %s failed health check (expired or paid). Skip.", model.ID)
			}
		}(candidate)
	}
	wg.Wait()

	log.Printf("[Probe] Verification finished. Found %d working free models.", len(verifiedModels))

	// 更新内存缓存
	cachedMutex.Lock()
	cachedModels = verifiedModels
	initialCheckDone = true
	cachedMutex.Unlock()
}

// 从上游获取所有的候选免费大模型
func fetchUpstreamCandidates() []ModelInfo {
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", "https://opencode.ai/zen/v1/models", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer public")
	req.Header.Set("x-opencode-client", "desktop")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var upstreamResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&upstreamResp); err != nil {
		return nil
	}

	var candidates []ModelInfo
	for _, model := range upstreamResp.Data {
		if strings.HasSuffix(model.ID, "-free") || model.ID == "big-pickle" {
			candidates = append(candidates, model)
		}
	}
	return candidates
}

// 测试单个模型的可用性 (发送 1 个 max_tokens 的极小请求)
func testModelAvailability(modelID string) bool {
	client := &http.Client{Timeout: 5 * time.Second}

	payload := map[string]interface{}{
		"model":      modelID,
		"messages":   []map[string]string{{"role": "user", "content": "."}},
		"max_tokens": 1,
	}
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://opencode.ai/zen/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer public")
	req.Header.Set("x-opencode-client", "desktop")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 只要接口返回 2xx 状态码，表示该模型目前在免费测试中
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// 接口处理器：瞬间从缓存中返回数据 (0ms 延时)
func ModelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	cachedMutex.RLock()
	defer cachedMutex.RUnlock()

	var resp ModelsResponse
	resp.Object = "list"

	if initialCheckDone && len(cachedModels) > 0 {
		resp.Data = cachedModels
	} else {
		resp.Data = defaultBackupModels
	}

	_ = json.NewEncoder(w).Encode(resp)
}
