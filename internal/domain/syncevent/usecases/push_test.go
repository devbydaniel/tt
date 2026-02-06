package usecases_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devbydaniel/tt/internal/domain/syncevent"
	"github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/testutil"
)

func setupPushTest(t *testing.T, handler http.HandlerFunc) (*usecases.PushEvents, *syncevent.Repository, *httptest.Server) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := syncevent.NewRepository(db)

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := syncevent.NewClient(server.URL, "test-api-key")

	push := &usecases.PushEvents{
		Repo:     repo,
		Client:   client,
		ClientID: "test-client",
	}

	return push, repo, server
}

func TestPushRejectedEventsGetFailureTracking(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req syncevent.PushRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Reject all events
		rejected := make([]syncevent.RejectedEvent, len(req.Events))
		for i, e := range req.Events {
			rejected[i] = syncevent.RejectedEvent{
				EventUUID: e.EventUUID,
				Reason:    "bad client id",
			}
		}

		resp := syncevent.PushResponse{
			Accepted: []string{},
			Rejected: rejected,
		}
		json.NewEncoder(w).Encode(resp)
	})

	push, repo, _ := setupPushTest(t, handler)

	createUnpushedEvent(repo, "entity-1", "event-1")

	// First push attempt — failure_count goes to 1
	result, err := push.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Rejected != 1 {
		t.Errorf("rejected = %d, want 1", result.Rejected)
	}

	// Event should still be in unpushed (not yet permanently failed)
	unpushed, _ := repo.GetUnpushed(10)
	if len(unpushed) != 1 {
		t.Errorf("should have 1 unpushed after 1 rejection, got %d", len(unpushed))
	}
}

func TestPushPermanentlyFailsAfterThreshold(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req syncevent.PushRequest
		json.NewDecoder(r.Body).Decode(&req)

		rejected := make([]syncevent.RejectedEvent, len(req.Events))
		for i, e := range req.Events {
			rejected[i] = syncevent.RejectedEvent{
				EventUUID: e.EventUUID,
				Reason:    "bad client id",
			}
		}

		resp := syncevent.PushResponse{
			Accepted: []string{},
			Rejected: rejected,
		}
		json.NewEncoder(w).Encode(resp)
	})

	push, repo, _ := setupPushTest(t, handler)

	createUnpushedEvent(repo, "entity-1", "event-1")

	// Push MaxFailureCount times to reach permanent failure
	for i := 0; i < syncevent.MaxFailureCount; i++ {
		push.Execute()
	}

	// Event should now be permanently failed and excluded from unpushed
	unpushed, _ := repo.GetUnpushed(10)
	if len(unpushed) != 0 {
		t.Errorf("should have 0 unpushed after %d rejections, got %d", syncevent.MaxFailureCount, len(unpushed))
	}

	failed, _ := repo.GetPermanentlyFailed()
	if len(failed) != 1 {
		t.Errorf("should have 1 permanently failed event, got %d", len(failed))
	}
}
