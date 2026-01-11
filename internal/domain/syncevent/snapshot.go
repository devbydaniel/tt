package syncevent

// TaskSnapshotData is the serializable representation of a task for sync.
// Uses UUIDs instead of local IDs for cross-device compatibility.
type TaskSnapshotData struct {
	UUID            string   `json:"uuid"`
	Title           string   `json:"title"`
	Description     *string  `json:"description,omitempty"`
	TaskType        string   `json:"taskType"`
	ParentUUID      *string  `json:"parentUuid,omitempty"`
	AreaUUID        *string  `json:"areaUuid,omitempty"`
	PlannedDate     *string  `json:"plannedDate,omitempty"`
	DueDate         *string  `json:"dueDate,omitempty"`
	State           string   `json:"state"`
	Status          string   `json:"status"`
	CreatedAt       string   `json:"createdAt"`
	CompletedAt     *string  `json:"completedAt,omitempty"`
	RecurType       *string  `json:"recurType,omitempty"`
	RecurRule       *string  `json:"recurRule,omitempty"`
	RecurEnd        *string  `json:"recurEnd,omitempty"`
	RecurPaused     bool     `json:"recurPaused"`
	RecurParentUUID *string  `json:"recurParentUuid,omitempty"`
	Tags            []string `json:"tags,omitempty"`
}

// AreaSnapshotData is the serializable representation of an area for sync.
type AreaSnapshotData struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}
