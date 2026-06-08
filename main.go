package main

import (
	"log"
	"net/http"
	"os"
	"opencode-proxy/handler"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "20128"
	}

	// 注册路由
	http.HandleFunc("/v1/models", handler.ModelsHandler)
	http.HandleFunc("/v1/chat/completions", handler.ProxyHandler)

	log.Printf("OpenCode Minimal Go Proxy starting on port %s...", port)
	log.Printf("Forwarding requests to https://opencode.ai/zen/v1/chat/completions")
	
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
