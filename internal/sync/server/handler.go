package server

import (
	"encoding/json"
	"net/http"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
)

// Handler handles sync API requests.
type Handler struct {
	ReceiveEvents *usecases.ReceiveEvents
}

// NewHandler creates a new handler with the given use cases.
func NewHandler(receiveEvents *usecases.ReceiveEvents) *Handler {
	return &Handler{
		ReceiveEvents: receiveEvents,
	}
}

// PushRequest is the request body for pushing events.
type PushRequest struct {
	ClientID string               `json:"clientId"`
	Events   []*syncevent.SyncEvent `json:"events"`
}

// PushResponse is the response body for pushing events.
type PushResponse struct {
	Accepted []string                 `json:"accepted"`
	Rejected []usecases.RejectedEvent `json:"rejected"`
}

// SyncRequest is the request body for bidirectional sync.
type SyncRequest struct {
	ClientID string                   `json:"clientId"`
	Cursor   int64                    `json:"cursor"`
	Events   []*syncevent.SyncEvent   `json:"events"`
}

// SyncResponse is the response body for bidirectional sync.
type SyncResponse struct {
	Accepted  []string                 `json:"accepted"`
	Rejected  []usecases.RejectedEvent `json:"rejected"`
	Entities  []syncevent.EntityState  `json:"entities"`
	NewCursor int64                    `json:"newCursor"`
}

// HandlePushEvents handles POST /api/v1/events requests.
func (h *Handler) HandlePushEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.ClientID == "" {
		http.Error(w, "clientId is required", http.StatusBadRequest)
		return
	}

	result, err := h.ReceiveEvents.Execute(&usecases.ReceiveRequest{
		ClientID: req.ClientID,
		Events:   req.Events,
	})
	if err != nil {
		http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := PushResponse{
		Accepted: result.Accepted,
		Rejected: result.Rejected,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleHealth handles GET /health requests.
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleSync handles POST /api/v1/sync requests for bidirectional sync.
func (h *Handler) HandleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.ClientID == "" {
		http.Error(w, "clientId is required", http.StatusBadRequest)
		return
	}

	result, err := h.ReceiveEvents.ExecuteSync(&usecases.SyncRequest{
		ClientID: req.ClientID,
		Cursor:   req.Cursor,
		Events:   req.Events,
	})
	if err != nil {
		http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := SyncResponse{
		Accepted:  result.Accepted,
		Rejected:  result.Rejected,
		Entities:  result.Entities,
		NewCursor: result.NewCursor,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
