package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/devbydaniel/tt/internal/app"
	"github.com/devbydaniel/tt/internal/domain/task"
	taskusecases "github.com/devbydaniel/tt/internal/domain/task/usecases"
)

// ProjectHandler handles project-related HTTP requests.
type ProjectHandler struct {
	App *app.App
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(application *app.App) *ProjectHandler {
	return &ProjectHandler{
		App: application,
	}
}

// extractProjectUUID extracts the UUID from a URL path like /api/v1/projects/{uuid} or /api/v1/projects/{uuid}/...
func extractProjectUUID(path string) string {
	path = strings.TrimPrefix(path, "/api/v1/projects/")
	if idx := strings.Index(path, "/"); idx != -1 {
		return path[:idx]
	}
	return path
}

// getProjectFromUUID looks up a project by UUID and verifies it's a project.
func (h *ProjectHandler) getProjectFromUUID(uuid string) (*task.Task, error) {
	t, err := h.App.GetTaskByUUID.Execute(uuid)
	if err != nil {
		return nil, err
	}
	if t.TaskType != task.TaskTypeProject {
		return nil, task.ErrTaskNotFound
	}
	return t, nil
}

// HandleProjects handles POST /api/v1/projects (create) and GET /api/v1/projects (list).
// @Summary Create or list projects
// @Tags projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Router /api/v1/projects [post]
// @Router /api/v1/projects [get]
func (h *ProjectHandler) HandleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreateProject(w, r)
	case http.MethodGet:
		h.handleListProjects(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleCreateProject handles POST /api/v1/projects.
// @Summary Create a new project
// @Description Create a new project with the given parameters
// @Tags projects
// @Accept json
// @Produce json
// @Param request body CreateProjectRequest true "Project creation parameters"
// @Success 201 {object} task.Task
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects [post]
func (h *ProjectHandler) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "Title is required")
		return
	}

	opts := &taskusecases.CreateProjectOptions{
		Someday: req.Someday,
	}

	if req.Description != nil {
		opts.Description = *req.Description
	}
	if req.AreaName != nil {
		opts.AreaName = *req.AreaName
	}
	opts.PlannedDate = req.PlannedDate
	opts.DueDate = req.DueDate

	p, err := h.App.CreateProject.Execute(req.Title, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create project: "+err.Error())
		return
	}

	// Set tags if provided
	if len(req.Tags) > 0 {
		p, err = h.App.SetTags.Execute(p.ID, req.Tags)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to set tags: "+err.Error())
			return
		}
	}

	writeJSON(w, http.StatusCreated, p)
}

// handleListProjects handles GET /api/v1/projects.
// @Summary List projects
// @Description List all active projects
// @Tags projects
// @Produce json
// @Param area query string false "Filter by area name"
// @Param all query bool false "Include all projects (not just active)"
// @Success 200 {array} task.Task
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects [get]
func (h *ProjectHandler) handleListProjects(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	areaName := query.Get("area")
	all := query.Get("all") == "true"

	var projects []task.Task
	var err error

	if areaName != "" {
		// Use ListTasks with project type and area filter
		opts := &task.ListOptions{
			TaskType: task.TaskTypeProject,
			AreaName: areaName,
		}
		projects, err = h.App.ListTasks.Execute(opts)
	} else if all {
		projects, err = h.App.ListAllProjects.Execute()
	} else {
		projects, err = h.App.ListProjects.Execute()
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list projects: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, projects)
}

// HandleProjectByUUID handles GET/PATCH/DELETE /api/v1/projects/{uuid} and action routes.
// @Summary Get, update, or delete a project
// @Tags projects
// @Accept json
// @Produce json
// @Param uuid path string true "Project UUID"
// @Security BearerAuth
// @Router /api/v1/projects/{uuid} [get]
// @Router /api/v1/projects/{uuid} [patch]
// @Router /api/v1/projects/{uuid} [delete]
func (h *ProjectHandler) HandleProjectByUUID(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Check for sub-routes
	if strings.Contains(path, "/complete") {
		h.handleComplete(w, r)
		return
	}
	if strings.Contains(path, "/uncomplete") {
		h.handleUncomplete(w, r)
		return
	}

	// Handle basic CRUD operations
	switch r.Method {
	case http.MethodGet:
		h.handleGetProject(w, r)
	case http.MethodPatch:
		h.handleUpdateProject(w, r)
	case http.MethodDelete:
		h.handleDeleteProject(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGetProject handles GET /api/v1/projects/{uuid}.
// @Summary Get a project by UUID
// @Description Retrieve a single project by its UUID
// @Tags projects
// @Produce json
// @Param uuid path string true "Project UUID"
// @Success 200 {object} task.Task
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{uuid} [get]
func (h *ProjectHandler) handleGetProject(w http.ResponseWriter, r *http.Request) {
	uuid := extractProjectUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	p, err := h.getProjectFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Project not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get project: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, p)
}

// handleUpdateProject handles PATCH /api/v1/projects/{uuid}.
// @Summary Update a project
// @Description Update a project's properties (partial update)
// @Tags projects
// @Accept json
// @Produce json
// @Param uuid path string true "Project UUID"
// @Param request body UpdateProjectRequest true "Fields to update"
// @Success 200 {object} task.Task
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{uuid} [patch]
func (h *ProjectHandler) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	uuid := extractProjectUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	p, err := h.getProjectFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Project not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get project: "+err.Error())
		return
	}

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Apply updates using individual use cases (same pattern as task_handler)
	if req.Title != nil {
		p, err = h.App.SetTaskTitle.Execute(p.ID, *req.Title)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update title: "+err.Error())
			return
		}
	}

	if req.Description != nil {
		p, err = h.App.SetTaskDescription.Execute(p.ID, req.Description)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update description: "+err.Error())
			return
		}
	} else if req.ClearDescription {
		p, err = h.App.SetTaskDescription.Execute(p.ID, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to clear description: "+err.Error())
			return
		}
	}

	if req.AreaName != nil {
		p, err = h.App.SetTaskArea.Execute(p.ID, *req.AreaName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update area: "+err.Error())
			return
		}
	} else if req.ClearArea {
		p, err = h.App.SetTaskArea.Execute(p.ID, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to clear area: "+err.Error())
			return
		}
	}

	if req.PlannedDate != nil {
		p, err = h.App.SetPlannedDate.Execute(p.ID, req.PlannedDate)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update planned date: "+err.Error())
			return
		}
	} else if req.ClearPlannedDate {
		p, err = h.App.SetPlannedDate.Execute(p.ID, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to clear planned date: "+err.Error())
			return
		}
	}

	if req.DueDate != nil {
		p, err = h.App.SetDueDate.Execute(p.ID, req.DueDate)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update due date: "+err.Error())
			return
		}
	} else if req.ClearDueDate {
		p, err = h.App.SetDueDate.Execute(p.ID, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to clear due date: "+err.Error())
			return
		}
	}

	if req.State != nil {
		switch *req.State {
		case "someday":
			p, err = h.App.DeferTask.Execute(p.ID)
		case "active":
			p, err = h.App.ActivateTask.Execute(p.ID)
		default:
			writeError(w, http.StatusBadRequest, "Invalid state: must be 'active' or 'someday'")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update state: "+err.Error())
			return
		}
	}

	if len(req.Tags) > 0 {
		p, err = h.App.SetTags.Execute(p.ID, req.Tags)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update tags: "+err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, p)
}

// handleDeleteProject handles DELETE /api/v1/projects/{uuid}.
// @Summary Delete a project
// @Description Delete a project by its UUID
// @Tags projects
// @Param uuid path string true "Project UUID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{uuid} [delete]
func (h *ProjectHandler) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	uuid := extractProjectUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	p, err := h.getProjectFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Project not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get project: "+err.Error())
		return
	}

	_, err = h.App.DeleteTasks.Execute([]int64{p.ID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete project: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleComplete handles POST /api/v1/projects/{uuid}/complete.
// @Summary Complete a project
// @Description Mark a project as completed
// @Tags projects
// @Produce json
// @Param uuid path string true "Project UUID"
// @Success 200 {object} task.CompleteResult
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{uuid}/complete [post]
func (h *ProjectHandler) handleComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	uuid := extractProjectUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	p, err := h.getProjectFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Project not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get project: "+err.Error())
		return
	}

	results, err := h.App.CompleteTasks.Execute([]int64{p.ID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to complete project: "+err.Error())
		return
	}

	if len(results) > 0 {
		writeJSON(w, http.StatusOK, results[0])
	} else {
		writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
	}
}

// handleUncomplete handles POST /api/v1/projects/{uuid}/uncomplete.
// @Summary Uncomplete a project
// @Description Mark a project as not completed
// @Tags projects
// @Produce json
// @Param uuid path string true "Project UUID"
// @Success 200 {object} task.Task
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{uuid}/uncomplete [post]
func (h *ProjectHandler) handleUncomplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	uuid := extractProjectUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	p, err := h.getProjectFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Project not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get project: "+err.Error())
		return
	}

	results, err := h.App.UncompleteTasks.Execute([]int64{p.ID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to uncomplete project: "+err.Error())
		return
	}

	if len(results) > 0 {
		writeJSON(w, http.StatusOK, results[0])
	} else {
		writeJSON(w, http.StatusOK, p)
	}
}
