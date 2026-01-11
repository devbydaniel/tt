package syncevent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client handles HTTP communication with the sync server.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new sync client with sensible defaults.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PushRequest is the request body for pushing events to the server.
type PushRequest struct {
	ClientID string       `json:"clientId"`
	Events   []*SyncEvent `json:"events"`
}

// RejectedEvent represents an event that was rejected by the server.
type RejectedEvent struct {
	EventUUID string `json:"eventUuid"`
	Reason    string `json:"reason"`
}

// PushResponse is the response from the server after pushing events.
type PushResponse struct {
	Accepted []string        `json:"accepted"`
	Rejected []RejectedEvent `json:"rejected"`
}

// SyncRequest is the request body for bidirectional sync.
type SyncRequest struct {
	ClientID string       `json:"clientId"`
	Cursor   int64        `json:"cursor"` // 0 = first sync
	Events   []*SyncEvent `json:"events"` // Events to push
}

// SyncResponse is the response from bidirectional sync.
type SyncResponse struct {
	Accepted  []string        `json:"accepted"`  // Pushed events that were accepted
	Rejected  []RejectedEvent `json:"rejected"`  // Pushed events that were rejected
	Entities  []EntityState   `json:"entities"`  // Latest state per entity (authoritative)
	NewCursor int64           `json:"newCursor"` // New cursor to store
}

// PushEvents sends events to the sync server and returns the response.
func (c *Client) PushEvents(clientID string, events []*SyncEvent) (*PushResponse, error) {
	reqBody := PushRequest{
		ClientID: clientID,
		Events:   events,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v1/events", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed: invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var pushResp PushResponse
	if err := json.NewDecoder(resp.Body).Decode(&pushResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &pushResp, nil
}

// Sync performs bidirectional sync with the server.
// It sends local events and receives the latest state for entities that changed.
func (c *Client) Sync(clientID string, cursor int64, events []*SyncEvent) (*SyncResponse, error) {
	reqBody := SyncRequest{
		ClientID: clientID,
		Cursor:   cursor,
		Events:   events,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v1/sync", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed: invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &syncResp, nil
}
