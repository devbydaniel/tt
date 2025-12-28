package project

type Project struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	AreaID *int64 `json:"areaId,omitempty"`
}

// ProjectWithArea includes the area name for display purposes
type ProjectWithArea struct {
	Project
	AreaName *string `json:"areaName,omitempty"`
}
