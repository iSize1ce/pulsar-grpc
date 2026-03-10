package main

import (
	"errors"
	"net/http"

	"database/sql"
)

// handleServers dispatches GET/POST/PUT/DELETE for server management.
func handleServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		servers, err := searchServers(r.URL.Query().Get("q"))
		if err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, servers)

	case http.MethodPost:
		var req struct {
			URL  string `json:"url"`
			Name string `json:"name"`
		}
		if err := decodeBody(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.URL == "" {
			http.Error(w, `{"error":"url is required"}`, http.StatusBadRequest)
			return
		}
		srv, err := createServer(req.URL, req.Name)
		if err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, srv)

	case http.MethodPut:
		var req struct {
			ID   int64  `json:"id"`
			Meta string `json:"meta"`
		}
		if err := decodeBody(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ID == 0 {
			http.Error(w, `{"error":"id is required"}`, http.StatusBadRequest)
			return
		}
		if err := updateServerMeta(req.ID, req.Meta); err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, map[string]bool{"ok": true})

	case http.MethodDelete:
		var req struct {
			ID int64 `json:"id"`
		}
		if err := decodeBody(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ID == 0 {
			http.Error(w, `{"error":"id is required"}`, http.StatusBadRequest)
			return
		}
		if err := deleteServer(req.ID); errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		} else if err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, map[string]bool{"ok": true})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSavedRequests dispatches GET/POST/PUT/DELETE for saved request management.
func handleSavedRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := searchSavedRequests(r.URL.Query().Get("q"), r.URL.Query().Get("method"))
		if err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, items)

	case http.MethodPost:
		var req struct {
			Name     string `json:"name"`
			ServerID int64  `json:"server_id"`
			Method   string `json:"method"`
			Payload  string `json:"payload"`
		}
		if err := decodeBody(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ServerID == 0 || req.Method == "" {
			http.Error(w, `{"error":"server_id and method are required"}`, http.StatusBadRequest)
			return
		}
		item, err := createSavedRequest(req.Name, req.ServerID, req.Method, req.Payload)
		if err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, item)

	case http.MethodPut:
		var req struct {
			ID       int64  `json:"id"`
			ServerID int64  `json:"server_id"`
			Method   string `json:"method"`
			Payload  string `json:"payload"`
		}
		if err := decodeBody(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ID == 0 {
			http.Error(w, `{"error":"id is required"}`, http.StatusBadRequest)
			return
		}
		if err := updateSavedRequest(req.ID, req.ServerID, req.Method, req.Payload); err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, map[string]bool{"ok": true})

	case http.MethodDelete:
		var req struct {
			ID int64 `json:"id"`
		}
		if err := decodeBody(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ID == 0 {
			http.Error(w, `{"error":"id is required"}`, http.StatusBadRequest)
			return
		}
		if err := deleteSavedRequest(req.ID); errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		} else if err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, map[string]bool{"ok": true})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleHistory dispatches GET/POST/DELETE for request history.
func handleHistory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := searchHistory(r.URL.Query().Get("q"), r.URL.Query().Get("method"))
		if err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, items)

	case http.MethodPost:
		var req struct {
			ServerID   int64  `json:"server_id"`
			Method     string `json:"method"`
			Payload    string `json:"payload"`
			Response   string `json:"response"`
			StatusCode int    `json:"status_code"`
		}
		if err := decodeBody(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ServerID == 0 || req.Method == "" {
			http.Error(w, `{"error":"server_id and method are required"}`, http.StatusBadRequest)
			return
		}
		item, err := createHistoryEntry(req.ServerID, req.Method, req.Payload, req.Response, req.StatusCode)
		if err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, item)

	case http.MethodDelete:
		var req struct {
			ID  int64 `json:"id"`
			All bool  `json:"all"`
		}
		if err := decodeBody(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.All {
			if err := clearHistory(); err != nil {
				respondError(w, err)
				return
			}
		} else {
			if req.ID == 0 {
				http.Error(w, `{"error":"id or all is required"}`, http.StatusBadRequest)
				return
			}
			if err := deleteHistoryEntry(req.ID); errors.Is(err, sql.ErrNoRows) {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			} else if err != nil {
				respondError(w, err)
				return
			}
		}
		respondJSON(w, map[string]bool{"ok": true})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
