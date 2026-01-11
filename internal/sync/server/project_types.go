package server

import (
	"time"
)

// CreateProjectRequest represents the request body for creating a project.
// @Description Request body for creating a new project
type CreateProjectRequest struct {
	// Title of the project (required)
	Title string `json:"title" example:"Website Redesign"`
	// Optional description
	Description *string `json:"description,omitempty" example:"Redesign the company website"`
	// Area name to assign the project to
	AreaName *string `json:"areaName,omitempty" example:"Work"`
	// Planned date for the project (RFC3339 format)
	PlannedDate *time.Time `json:"plannedDate,omitempty" example:"2025-01-15T00:00:00Z"`
	// Due date for the project (RFC3339 format)
	DueDate *time.Time `json:"dueDate,omitempty" example:"2025-03-01T00:00:00Z"`
	// Set to true to create project in someday state
	Someday bool `json:"someday,omitempty" example:"false"`
	// Tags to assign to the project
	Tags []string `json:"tags,omitempty" example:"priority,q1"`
}

// UpdateProjectRequest represents the request body for updating a project.
// @Description Request body for updating an existing project (all fields optional)
type UpdateProjectRequest struct {
	// New title for the project
	Title *string `json:"title,omitempty" example:"Updated project title"`
	// New description for the project
	Description *string `json:"description,omitempty" example:"Updated description"`
	// Area name to assign the project to
	AreaName *string `json:"areaName,omitempty" example:"Personal"`
	// New planned date (RFC3339 format)
	PlannedDate *time.Time `json:"plannedDate,omitempty" example:"2025-01-15T00:00:00Z"`
	// New due date (RFC3339 format)
	DueDate *time.Time `json:"dueDate,omitempty" example:"2025-03-01T00:00:00Z"`
	// Project state: "active" or "someday"
	State *string `json:"state,omitempty" example:"active"`
	// Replace all tags with these
	Tags []string `json:"tags,omitempty" example:"priority,q2"`
	// Set to true to clear the description
	ClearDescription bool `json:"clearDescription,omitempty" example:"false"`
	// Set to true to unassign from area
	ClearArea bool `json:"clearArea,omitempty" example:"false"`
	// Set to true to clear the planned date
	ClearPlannedDate bool `json:"clearPlannedDate,omitempty" example:"false"`
	// Set to true to clear the due date
	ClearDueDate bool `json:"clearDueDate,omitempty" example:"false"`
}
