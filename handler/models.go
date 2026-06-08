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

// 缓存相关的全局变量
var (
	cachedModels []ModelInfo
	lastChecked  time.Time
	cachedMutex  sync.RWMutex
	isChecking   bool
	checkMutex   sync.Mutex
)

// 默认兜底及初始模型列表
var defaultBackupModels = []ModelInfo{
	{ID: "big-pickle", Object: "model", OwnedBy: "opencode"},
	{ID: "nemotron-3-super-free", Object: "model", OwnedBy: "opencode"},
	{ID: "deepseek-v4-flash-free", Object: "model", OwnedBy: "opencode"},
	{ID: "mimo-v2.5-free", Object: "model", OwnedBy: "opencode"},
	{ID: "nemotron-3-ultra-free", Object: "model", OwnedBy: "opencode"},
}

func init() {
	// 初始状态下直接将缓存设置为默认备份列表
	cachedModels = defaultBackupModels
	// 注意：此处不再启动任何后台定时器（Idle状态下完全静默）
}

// 探测函数：并发向 OpenCode 验证每个候选模型，并更新缓存与时间戳
func probeModels() {
	log.Println("[Probe] Revalidating models list from upstream...")

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
				log.Printf("[Probe] Model %s failed health check. Skip.", model.ID)
			}
		}(candidate)
	}
	wg.Wait()

	log.Printf("[Probe] Revalidation finished. Verified %d models.", len(verifiedModels))

	// 更新内存缓存及时间戳
	cachedMutex.Lock()
	if len(verifiedModels) > 0 {
		cachedModels = verifiedModels
		lastChecked = time.Now()
	}
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

// 测试单个模型的可用性
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

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// 接口处理器：采用 Stale-While-Revalidate 机制，0ms 延时瞬间响应，后台异步更新
func ModelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. 判断缓存是否过期 (24小时)
	cachedMutex.RLock()
	needRevalidate := time.Since(lastChecked) > 24*time.Hour
	cachedMutex.RUnlock()

	// 2. 若过期，则惰性触发异步检测（仅启动单个任务，防并发冲突）
	if needRevalidate {
		checkMutex.Lock()
		if !isChecking {
			isChecking = true
			go func() {
				defer func() {
					checkMutex.Lock()
					isChecking = false
					checkMutex.Unlock()
				}()
				probeModels()
			}()
		}
		checkMutex.Unlock()
	}

	// 3. 瞬间返回当前缓存的数据（Stale-While-Revalidate 核心，保障 0ms 响应）
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	cachedMutex.RLock()
	defer cachedMutex.RUnlock()

	var resp ModelsResponse
	resp.Object = "list"
	resp.Data = cachedModels

	_ = json.NewEncoder(w).Encode(resp)
}
