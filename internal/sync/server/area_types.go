package server

// CreateAreaRequest represents the request body for creating an area.
// @Description Request body for creating a new area
type CreateAreaRequest struct {
	// Name of the area (required)
	Name string `json:"name" example:"Personal"`
}

// UpdateAreaRequest represents the request body for updating an area.
// @Description Request body for renaming an area
type UpdateAreaRequest struct {
	// New name for the area (required)
	Name string `json:"name" example:"Work"`
}
