package project

type Project struct {
	ID     int64
	Name   string
	AreaID *int64
}

// ProjectWithArea includes the area name for display purposes
type ProjectWithArea struct {
	Project
	AreaName *string
}
