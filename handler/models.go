package handler

import (
	"encoding/json"
	"net/http"
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

func ModelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := ModelsResponse{
		Object: "list",
		Data: []ModelInfo{
			{ID: "big-pickle", Object: "model", OwnedBy: "opencode"},
			{ID: "qwen3.6-plus-free", Object: "model", OwnedBy: "opencode"},
			{ID: "nemotron-3-super-free", Object: "model", OwnedBy: "opencode"},
			{ID: "minimax-m2.5-free", Object: "model", OwnedBy: "opencode"},
			{ID: "trinity-large-preview-free", Object: "model", OwnedBy: "opencode"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
