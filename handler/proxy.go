package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

type ChatCompletionRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. 读取并解析请求体，以便提取和清洗 model
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var req ChatCompletionRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Printf("Error parsing JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 2. 清洗模型前缀 (如 oc/big-pickle -> big-pickle)
	originalModel := req.Model
	cleanedModel := originalModel
	if strings.HasPrefix(originalModel, "oc/") {
		cleanedModel = strings.TrimPrefix(originalModel, "oc/")
	} else if strings.HasPrefix(originalModel, "opencode/") {
		cleanedModel = strings.TrimPrefix(originalModel, "opencode/")
	}

	// 如果模型名发生了改变，重新序列化 Request Body
	var upstreamBody io.Reader = bytes.NewReader(bodyBytes)
	if cleanedModel != originalModel {
		var rawMap map[string]interface{}
		_ = json.Unmarshal(bodyBytes, &rawMap)
		rawMap["model"] = cleanedModel
		modifiedBytes, _ := json.Marshal(rawMap)
		upstreamBody = bytes.NewReader(modifiedBytes)
		log.Printf("Cleaned model prefix: %s -> %s", originalModel, cleanedModel)
	}

	// 3. 构建发往 OpenCode 官方接口的请求
	upstreamURL := "https://opencode.ai/zen/v1/chat/completions"
	reqUpstream, err := http.NewRequestWithContext(r.Context(), "POST", upstreamURL, upstreamBody)
	if err != nil {
		log.Printf("Error creating upstream request: %v", err)
		http.Error(w, "Failed to create upstream request", http.StatusInternalServerError)
		return
	}

	// 4. 伪装桌面客户端 Headers
	reqUpstream.Header.Set("Content-Type", "application/json")
	reqUpstream.Header.Set("Authorization", "Bearer public")
	reqUpstream.Header.Set("x-opencode-client", "desktop")
	if req.Stream {
		reqUpstream.Header.Set("Accept", "text/event-stream")
	} else {
		reqUpstream.Header.Set("Accept", "application/json")
	}

	// 5. 发送请求
	client := &http.Client{}
	resp, err := client.Do(reqUpstream)
	if err != nil {
		log.Printf("Error calling upstream: %v", err)
		http.Error(w, "Failed to call OpenCode upstream: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 6. 复制响应头与响应状态
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// 7. 流式/普通响应拷贝
	if req.Stream {
		// 流式写入，实时 Flush
		flusher, ok := w.(http.Flusher)
		if !ok {
			// 如果 ResponseWriter 不支持 Flush，则只能普通写入
			_, _ = io.Copy(w, resp.Body)
			return
		}

		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				_, _ = w.Write(buf[:n])
				flusher.Flush() // 保证实时推送到客户端
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Streaming read error: %v", err)
				break
			}
		}
	} else {
		_, _ = io.Copy(w, resp.Body)
	}
}
