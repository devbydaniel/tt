package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/devbydaniel/tt/internal/app"
	"github.com/devbydaniel/tt/internal/domain/area"
)

// AreaHandler handles area-related HTTP requests.
type AreaHandler struct {
	App *app.App
}

// NewAreaHandler creates a new AreaHandler.
func NewAreaHandler(application *app.App) *AreaHandler {
	return &AreaHandler{
		App: application,
	}
}

// extractAreaUUID extracts the UUID from a URL path like /api/v1/areas/{uuid}
func extractAreaUUID(path string) string {
	path = strings.TrimPrefix(path, "/api/v1/areas/")
	if idx := strings.Index(path, "/"); idx != -1 {
		return path[:idx]
	}
	return path
}

// HandleAreas handles POST /api/v1/areas (create) and GET /api/v1/areas (list).
// @Summary Create or list areas
// @Tags areas
// @Accept json
// @Produce json
// @Security BearerAuth
// @Router /api/v1/areas [post]
// @Router /api/v1/areas [get]
func (h *AreaHandler) HandleAreas(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreateArea(w, r)
	case http.MethodGet:
		h.handleListAreas(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleCreateArea handles POST /api/v1/areas.
// @Summary Create a new area
// @Description Create a new area with the given name
// @Tags areas
// @Accept json
// @Produce json
// @Param request body CreateAreaRequest true "Area creation parameters"
// @Success 201 {object} area.Area
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/areas [post]
func (h *AreaHandler) handleCreateArea(w http.ResponseWriter, r *http.Request) {
	var req CreateAreaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Name is required")
		return
	}

	a, err := h.App.CreateArea.Execute(req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create area: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, a)
}

// handleListAreas handles GET /api/v1/areas.
// @Summary List areas
// @Description List all areas
// @Tags areas
// @Produce json
// @Success 200 {array} area.Area
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/areas [get]
func (h *AreaHandler) handleListAreas(w http.ResponseWriter, _ *http.Request) {
	areas, err := h.App.ListAreas.Execute()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list areas: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, areas)
}

// HandleAreaByUUID handles GET/PATCH/DELETE /api/v1/areas/{uuid}.
// @Summary Get, update, or delete an area
// @Tags areas
// @Accept json
// @Produce json
// @Param uuid path string true "Area UUID"
// @Security BearerAuth
// @Router /api/v1/areas/{uuid} [get]
// @Router /api/v1/areas/{uuid} [patch]
// @Router /api/v1/areas/{uuid} [delete]
func (h *AreaHandler) HandleAreaByUUID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetArea(w, r)
	case http.MethodPatch:
		h.handleUpdateArea(w, r)
	case http.MethodDelete:
		h.handleDeleteArea(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGetArea handles GET /api/v1/areas/{uuid}.
// @Summary Get an area by UUID
// @Description Retrieve a single area by its UUID
// @Tags areas
// @Produce json
// @Param uuid path string true "Area UUID"
// @Success 200 {object} area.Area
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/areas/{uuid} [get]
func (h *AreaHandler) handleGetArea(w http.ResponseWriter, r *http.Request) {
	uuid := extractAreaUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	a, err := h.App.GetAreaByUUID.Execute(uuid)
	if err != nil {
		if errors.Is(err, area.ErrAreaNotFound) {
			writeError(w, http.StatusNotFound, "Area not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get area: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, a)
}

// handleUpdateArea handles PATCH /api/v1/areas/{uuid}.
// @Summary Rename an area
// @Description Update an area's name
// @Tags areas
// @Accept json
// @Produce json
// @Param uuid path string true "Area UUID"
// @Param request body UpdateAreaRequest true "New area name"
// @Success 200 {object} area.Area
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/areas/{uuid} [patch]
func (h *AreaHandler) handleUpdateArea(w http.ResponseWriter, r *http.Request) {
	uuid := extractAreaUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	// Get the area first to get its current name
	existingArea, err := h.App.GetAreaByUUID.Execute(uuid)
	if err != nil {
		if errors.Is(err, area.ErrAreaNotFound) {
			writeError(w, http.StatusNotFound, "Area not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get area: "+err.Error())
		return
	}

	var req UpdateAreaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Name is required")
		return
	}

	// Rename using oldName -> newName
	a, err := h.App.RenameArea.Execute(existingArea.Name, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to rename area: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, a)
}

// handleDeleteArea handles DELETE /api/v1/areas/{uuid}.
// @Summary Delete an area
// @Description Delete an area by its UUID
// @Tags areas
// @Param uuid path string true "Area UUID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/areas/{uuid} [delete]
func (h *AreaHandler) handleDeleteArea(w http.ResponseWriter, r *http.Request) {
	uuid := extractAreaUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	// Get the area first to get its name (DeleteArea uses name)
	existingArea, err := h.App.GetAreaByUUID.Execute(uuid)
	if err != nil {
		if errors.Is(err, area.ErrAreaNotFound) {
			writeError(w, http.StatusNotFound, "Area not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get area: "+err.Error())
		return
	}

	_, err = h.App.DeleteArea.Execute(existingArea.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete area: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
