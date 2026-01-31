package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/devbydaniel/tt/internal/app"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// TaskHandler handles task-related HTTP requests.
type TaskHandler struct {
	App *app.App
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(application *app.App) *TaskHandler {
	return &TaskHandler{
		App: application,
	}
}

// extractUUID extracts the UUID from a URL path like /api/v1/tasks/{uuid} or /api/v1/tasks/{uuid}/...
func extractUUID(path string) string {
	// Remove /api/v1/tasks/ prefix
	path = strings.TrimPrefix(path, "/api/v1/tasks/")
	// Get the first segment (UUID)
	if idx := strings.Index(path, "/"); idx != -1 {
		return path[:idx]
	}
	return path
}

// getTaskFromUUID looks up a task by UUID using the use case.
func (h *TaskHandler) getTaskFromUUID(uuid string) (*task.Task, error) {
	return h.App.GetTaskByUUID.Execute(uuid)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

// HandleTasks handles POST /api/v1/tasks (create) and GET /api/v1/tasks (list).
// @Summary Create or list tasks
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Router /api/v1/tasks [post]
// @Router /api/v1/tasks [get]
func (h *TaskHandler) HandleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreateTask(w, r)
	case http.MethodGet:
		h.handleListTasks(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleCreateTask handles POST /api/v1/tasks.
// @Summary Create a new task
// @Description Create a new task with the given parameters
// @Tags tasks
// @Accept json
// @Produce json
// @Param request body CreateTaskRequest true "Task creation parameters"
// @Success 201 {object} task.Task
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/tasks [post]
func (h *TaskHandler) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "Title is required")
		return
	}

	opts := &task.CreateOptions{
		Tags: req.Tags,
	}

	if req.Description != nil {
		opts.Description = *req.Description
	}
	if req.ProjectName != nil {
		opts.ProjectName = *req.ProjectName
	}
	if req.AreaName != nil {
		opts.AreaName = *req.AreaName
	}
	opts.PlannedDate = req.PlannedDate
	opts.DueDate = req.DueDate
	opts.Someday = req.Someday
	opts.RecurType = req.RecurType
	opts.RecurRule = req.RecurRule
	opts.RecurEnd = req.RecurEnd

	t, err := h.App.CreateTask.Execute(req.Title, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create task: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, t)
}

// handleListTasks handles GET /api/v1/tasks.
// @Summary List tasks
// @Description List tasks with optional filters
// @Tags tasks
// @Produce json
// @Param project query string false "Filter by project name"
// @Param area query string false "Filter by area name"
// @Param tag query string false "Filter by tag"
// @Param search query string false "Search in task title"
// @Param schedule query string false "Filter by schedule: today, upcoming, anytime, inbox, someday"
// @Param sort query string false "Sort order: field:dir,field:dir (e.g. due:asc,title:desc)"
// @Success 200 {array} task.Task
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/tasks [get]
func (h *TaskHandler) handleListTasks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	opts := &task.ListOptions{
		ProjectName: query.Get("project"),
		AreaName:    query.Get("area"),
		TagName:     query.Get("tag"),
		Search:      query.Get("search"),
		Schedule:    query.Get("schedule"),
	}

	// Parse sort option
	if sortStr := query.Get("sort"); sortStr != "" {
		sortOpts, err := task.ParseSort(sortStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid sort parameter: "+err.Error())
			return
		}
		opts.Sort = sortOpts
	}

	tasks, err := h.App.ListTasks.Execute(opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list tasks: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

// HandleTaskByUUID handles GET/PATCH/DELETE /api/v1/tasks/{uuid}.
// @Summary Get, update, or delete a task
// @Tags tasks
// @Accept json
// @Produce json
// @Param uuid path string true "Task UUID"
// @Security BearerAuth
// @Router /api/v1/tasks/{uuid} [get]
// @Router /api/v1/tasks/{uuid} [patch]
// @Router /api/v1/tasks/{uuid} [delete]
func (h *TaskHandler) HandleTaskByUUID(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Check for sub-routes like /complete, /uncomplete, /recurrence
	if strings.Contains(path, "/complete") {
		h.handleComplete(w, r)
		return
	}
	if strings.Contains(path, "/uncomplete") {
		h.handleUncomplete(w, r)
		return
	}
	if strings.Contains(path, "/recurrence") {
		h.handleRecurrence(w, r)
		return
	}

	// Handle basic CRUD operations
	switch r.Method {
	case http.MethodGet:
		h.handleGetTask(w, r)
	case http.MethodPatch:
		h.handleUpdateTask(w, r)
	case http.MethodDelete:
		h.handleDeleteTask(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGetTask handles GET /api/v1/tasks/{uuid}.
// @Summary Get a task by UUID
// @Description Retrieve a single task by its UUID
// @Tags tasks
// @Produce json
// @Param uuid path string true "Task UUID"
// @Success 200 {object} task.Task
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/tasks/{uuid} [get]
func (h *TaskHandler) handleGetTask(w http.ResponseWriter, r *http.Request) {
	uuid := extractUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	t, err := h.getTaskFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get task: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// handleUpdateTask handles PATCH /api/v1/tasks/{uuid}.
// @Summary Update a task
// @Description Update a task's properties (partial update)
// @Tags tasks
// @Accept json
// @Produce json
// @Param uuid path string true "Task UUID"
// @Param request body UpdateTaskRequest true "Fields to update"
// @Success 200 {object} task.Task
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/tasks/{uuid} [patch]
func (h *TaskHandler) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	uuid := extractUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	// Get the task first to get its ID
	t, err := h.getTaskFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get task: "+err.Error())
		return
	}

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Apply updates using individual use cases
	if req.Title != nil {
		t, err = h.App.SetTaskTitle.Execute(t.ID, *req.Title)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update title: "+err.Error())
			return
		}
	}

	if req.Description != nil {
		t, err = h.App.SetTaskDescription.Execute(t.ID, req.Description)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update description: "+err.Error())
			return
		}
	} else if req.ClearDescription {
		t, err = h.App.SetTaskDescription.Execute(t.ID, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to clear description: "+err.Error())
			return
		}
	}

	if req.ProjectName != nil {
		t, err = h.App.SetTaskProject.Execute(t.ID, *req.ProjectName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update project: "+err.Error())
			return
		}
	} else if req.ClearProject {
		t, err = h.App.SetTaskProject.Execute(t.ID, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to clear project: "+err.Error())
			return
		}
	}

	if req.AreaName != nil {
		t, err = h.App.SetTaskArea.Execute(t.ID, *req.AreaName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update area: "+err.Error())
			return
		}
	} else if req.ClearArea {
		t, err = h.App.SetTaskArea.Execute(t.ID, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to clear area: "+err.Error())
			return
		}
	}

	if req.PlannedDate != nil {
		t, err = h.App.SetPlannedDate.Execute(t.ID, req.PlannedDate)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update planned date: "+err.Error())
			return
		}
	} else if req.ClearPlannedDate {
		t, err = h.App.SetPlannedDate.Execute(t.ID, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to clear planned date: "+err.Error())
			return
		}
	}

	if req.DueDate != nil {
		t, err = h.App.SetDueDate.Execute(t.ID, req.DueDate)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update due date: "+err.Error())
			return
		}
	} else if req.ClearDueDate {
		t, err = h.App.SetDueDate.Execute(t.ID, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to clear due date: "+err.Error())
			return
		}
	}

	if req.State != nil {
		switch *req.State {
		case "someday":
			t, err = h.App.DeferTask.Execute(t.ID)
		case "active":
			t, err = h.App.ActivateTask.Execute(t.ID)
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
		t, err = h.App.SetTags.Execute(t.ID, req.Tags)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to update tags: "+err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, t)
}

// handleDeleteTask handles DELETE /api/v1/tasks/{uuid}.
// @Summary Delete a task
// @Description Delete a task by its UUID
// @Tags tasks
// @Param uuid path string true "Task UUID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/tasks/{uuid} [delete]
func (h *TaskHandler) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	uuid := extractUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	// Get the task first to get its ID
	t, err := h.getTaskFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get task: "+err.Error())
		return
	}

	_, err = h.App.DeleteTasks.Execute([]int64{t.ID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete task: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleComplete handles POST /api/v1/tasks/{uuid}/complete.
// @Summary Complete a task
// @Description Mark a task as completed
// @Tags tasks
// @Produce json
// @Param uuid path string true "Task UUID"
// @Success 200 {object} task.CompleteResult
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/tasks/{uuid}/complete [post]
func (h *TaskHandler) handleComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	uuid := extractUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	t, err := h.getTaskFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get task: "+err.Error())
		return
	}

	results, err := h.App.CompleteTasks.Execute([]int64{t.ID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to complete task: "+err.Error())
		return
	}

	if len(results) > 0 {
		writeJSON(w, http.StatusOK, results[0])
	} else {
		writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
	}
}

// handleUncomplete handles POST /api/v1/tasks/{uuid}/uncomplete.
// @Summary Uncomplete a task
// @Description Mark a task as not completed
// @Tags tasks
// @Produce json
// @Param uuid path string true "Task UUID"
// @Success 200 {object} task.Task
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/tasks/{uuid}/uncomplete [post]
func (h *TaskHandler) handleUncomplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	uuid := extractUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	t, err := h.getTaskFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get task: "+err.Error())
		return
	}

	results, err := h.App.UncompleteTasks.Execute([]int64{t.ID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to uncomplete task: "+err.Error())
		return
	}

	if len(results) > 0 {
		writeJSON(w, http.StatusOK, results[0])
	} else {
		writeJSON(w, http.StatusOK, t)
	}
}

// handleRecurrence handles recurrence-related requests.
// PATCH /api/v1/tasks/{uuid}/recurrence - Set recurrence
// POST /api/v1/tasks/{uuid}/recurrence/pause - Pause recurrence
// POST /api/v1/tasks/{uuid}/recurrence/resume - Resume recurrence
func (h *TaskHandler) handleRecurrence(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.HasSuffix(path, "/pause") {
		h.handlePauseRecurrence(w, r)
		return
	}
	if strings.HasSuffix(path, "/resume") {
		h.handleResumeRecurrence(w, r)
		return
	}

	// Handle PATCH /api/v1/tasks/{uuid}/recurrence
	if r.Method == http.MethodPatch {
		h.handleSetRecurrence(w, r)
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// handleSetRecurrence handles PATCH /api/v1/tasks/{uuid}/recurrence.
// @Summary Set task recurrence
// @Description Configure recurrence settings for a task
// @Tags tasks
// @Accept json
// @Produce json
// @Param uuid path string true "Task UUID"
// @Param request body SetRecurrenceRequest true "Recurrence settings"
// @Success 200 {object} task.Task
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/tasks/{uuid}/recurrence [patch]
func (h *TaskHandler) handleSetRecurrence(w http.ResponseWriter, r *http.Request) {
	uuid := extractUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	t, err := h.getTaskFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get task: "+err.Error())
		return
	}

	var req SetRecurrenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.RecurType != "fixed" && req.RecurType != "relative" {
		writeError(w, http.StatusBadRequest, "recurType must be 'fixed' or 'relative'")
		return
	}

	t, err = h.App.SetRecurrence.Execute(t.ID, &req.RecurType, &req.RecurRule, req.RecurEnd)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to set recurrence: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// handlePauseRecurrence handles POST /api/v1/tasks/{uuid}/recurrence/pause.
// @Summary Pause task recurrence
// @Description Pause the recurrence on a recurring task
// @Tags tasks
// @Produce json
// @Param uuid path string true "Task UUID"
// @Success 200 {object} task.Task
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/tasks/{uuid}/recurrence/pause [post]
func (h *TaskHandler) handlePauseRecurrence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	uuid := extractUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	t, err := h.getTaskFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get task: "+err.Error())
		return
	}

	t, err = h.App.PauseRecurrence.Execute(t.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to pause recurrence: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// handleResumeRecurrence handles POST /api/v1/tasks/{uuid}/recurrence/resume.
// @Summary Resume task recurrence
// @Description Resume a paused recurrence on a task
// @Tags tasks
// @Produce json
// @Param uuid path string true "Task UUID"
// @Success 200 {object} task.Task
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/tasks/{uuid}/recurrence/resume [post]
func (h *TaskHandler) handleResumeRecurrence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	uuid := extractUUID(r.URL.Path)
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "UUID is required")
		return
	}

	t, err := h.getTaskFromUUID(uuid)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get task: "+err.Error())
		return
	}

	t, err = h.App.ResumeRecurrence.Execute(t.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to resume recurrence: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// HandleTagsList handles GET /api/v1/tags.
// @Summary List all tags
// @Description List all unique tags in use across tasks
// @Tags tags
// @Produce json
// @Success 200 {array} string
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/tags [get]
func (h *TaskHandler) HandleTagsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	tags, err := h.App.ListTags.Execute()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list tags: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tags)
}

// HandleCompletedTasks handles GET /api/v1/tasks/completed.
// @Summary List completed tasks
// @Description List tasks that have been completed (logbook)
// @Tags tasks
// @Produce json
// @Param since query string false "Only show tasks completed after this date (RFC3339 format)"
// @Success 200 {array} task.Task
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/tasks/completed [get]
func (h *TaskHandler) HandleCompletedTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := r.URL.Query()
	sinceStr := query.Get("since")

	var since *time.Time
	if sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid since parameter: must be RFC3339 format")
			return
		}
		since = &t
	}

	tasks, err := h.App.ListCompletedTasks.Execute(since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list completed tasks: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}
