package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"opencode-proxy/handler"
)

// AuthMiddleware 拦截并校验 API 密钥，防止公网端口被未授权扫描和滥用
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expectedKey := os.Getenv("PROXY_API_KEY")
		if expectedKey != "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error": "Unauthorized: Missing Authorization header"}`))
				return
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error": "Unauthorized: Invalid Authorization header format"}`))
				return
			}
			if parts[1] != expectedKey {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error": "Unauthorized: Incorrect API Key"}`))
				return
			}
		}
		next(w, r)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "20128"
	}

	// 注册路由并应用鉴权中间件
	http.HandleFunc("/v1/models", AuthMiddleware(handler.ModelsHandler))
	http.HandleFunc("/v1/chat/completions", AuthMiddleware(handler.ProxyHandler))

	log.Printf("OpenCode Minimal Go Proxy starting on port %s...", port)
	log.Printf("Forwarding requests to https://opencode.ai/zen/v1/chat/completions")
	if os.Getenv("PROXY_API_KEY") != "" {
		log.Println("Authentication is ENABLED (PROXY_API_KEY is set).")
	} else {
		log.Println("WARNING: Authentication is DISABLED (PROXY_API_KEY is not set).")
	}
	
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
