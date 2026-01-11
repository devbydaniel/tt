package server

import (
	"time"
)

// CreateTaskRequest represents the request body for creating a task.
// @Description Request body for creating a new task
type CreateTaskRequest struct {
	// Title of the task (required)
	Title string `json:"title" example:"Buy groceries"`
	// Optional description
	Description *string `json:"description,omitempty" example:"Get milk, eggs, and bread"`
	// Project name to assign the task to
	ProjectName *string `json:"projectName,omitempty" example:"Shopping"`
	// Area name to assign the task to
	AreaName *string `json:"areaName,omitempty" example:"Personal"`
	// Planned date for the task (RFC3339 format)
	PlannedDate *time.Time `json:"plannedDate,omitempty" example:"2025-01-15T00:00:00Z"`
	// Due date for the task (RFC3339 format)
	DueDate *time.Time `json:"dueDate,omitempty" example:"2025-01-20T00:00:00Z"`
	// Set to true to create task in someday state
	Someday bool `json:"someday,omitempty" example:"false"`
	// Tags to assign to the task
	Tags []string `json:"tags,omitempty" example:"urgent,important"`
	// Recurrence type: "fixed" or "relative"
	RecurType *string `json:"recurType,omitempty" example:"fixed"`
	// Recurrence rule as JSON
	RecurRule *string `json:"recurRule,omitempty" example:"{\"interval\":1,\"unit\":\"week\"}"`
	// Recurrence end date (RFC3339 format)
	RecurEnd *time.Time `json:"recurEnd,omitempty" example:"2025-12-31T00:00:00Z"`
}

// UpdateTaskRequest represents the request body for updating a task.
// @Description Request body for updating an existing task (all fields optional)
type UpdateTaskRequest struct {
	// New title for the task
	Title *string `json:"title,omitempty" example:"Updated task title"`
	// New description for the task
	Description *string `json:"description,omitempty" example:"Updated description"`
	// Project name to assign the task to
	ProjectName *string `json:"projectName,omitempty" example:"Work"`
	// Area name to assign the task to
	AreaName *string `json:"areaName,omitempty" example:"Home"`
	// New planned date (RFC3339 format)
	PlannedDate *time.Time `json:"plannedDate,omitempty" example:"2025-01-15T00:00:00Z"`
	// New due date (RFC3339 format)
	DueDate *time.Time `json:"dueDate,omitempty" example:"2025-01-20T00:00:00Z"`
	// Task state: "active" or "someday"
	State *string `json:"state,omitempty" example:"active"`
	// Replace all tags with these
	Tags []string `json:"tags,omitempty" example:"work,priority"`
	// Set to true to clear the description
	ClearDescription bool `json:"clearDescription,omitempty" example:"false"`
	// Set to true to unassign from project
	ClearProject bool `json:"clearProject,omitempty" example:"false"`
	// Set to true to unassign from area
	ClearArea bool `json:"clearArea,omitempty" example:"false"`
	// Set to true to clear the planned date
	ClearPlannedDate bool `json:"clearPlannedDate,omitempty" example:"false"`
	// Set to true to clear the due date
	ClearDueDate bool `json:"clearDueDate,omitempty" example:"false"`
}

// SetRecurrenceRequest represents the request body for setting task recurrence.
// @Description Request body for setting recurrence on a task
type SetRecurrenceRequest struct {
	// Recurrence type: "fixed" or "relative"
	RecurType string `json:"recurType" example:"fixed"`
	// Recurrence rule as JSON
	RecurRule string `json:"recurRule" example:"{\"interval\":1,\"unit\":\"week\"}"`
	// Optional recurrence end date (RFC3339 format)
	RecurEnd *time.Time `json:"recurEnd,omitempty" example:"2025-12-31T00:00:00Z"`
}

// ErrorResponse represents an error response.
// @Description Error response from the API
type ErrorResponse struct {
	// Error message
	Error string `json:"error" example:"Task not found"`
}

// TaskListResponse represents a list of tasks.
// @Description List of tasks
type TaskListResponse struct {
	// List of tasks
	Tasks []TaskResponse `json:"tasks"`
}

// TaskResponse represents a task in API responses.
// @Description Task object returned by the API
type TaskResponse struct {
	UUID        string     `json:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Title       string     `json:"title" example:"Buy groceries"`
	Description *string    `json:"description,omitempty" example:"Get milk and eggs"`
	TaskType    string     `json:"taskType" example:"task"`
	ParentUUID  *string    `json:"parentUuid,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	AreaID      *int64     `json:"areaId,omitempty" example:"1"`
	PlannedDate *time.Time `json:"plannedDate,omitempty" example:"2025-01-15T00:00:00Z"`
	DueDate     *time.Time `json:"dueDate,omitempty" example:"2025-01-20T00:00:00Z"`
	State       string     `json:"state" example:"active"`
	Status      string     `json:"status" example:"todo"`
	CreatedAt   time.Time  `json:"createdAt" example:"2025-01-01T00:00:00Z"`
	CompletedAt *time.Time `json:"completedAt,omitempty" example:"2025-01-10T00:00:00Z"`
	RecurType   *string    `json:"recurType,omitempty" example:"fixed"`
	RecurRule   *string    `json:"recurRule,omitempty" example:"{\"interval\":1,\"unit\":\"week\"}"`
	RecurEnd    *time.Time `json:"recurEnd,omitempty" example:"2025-12-31T00:00:00Z"`
	RecurPaused bool       `json:"recurPaused,omitempty" example:"false"`
	Tags        []string   `json:"tags,omitempty" example:"work,urgent"`
	ParentName  *string    `json:"parentName,omitempty" example:"Work Project"`
	AreaName    *string    `json:"areaName,omitempty" example:"Personal"`
}
