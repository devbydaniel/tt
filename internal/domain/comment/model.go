package comment

import "time"

type Comment struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"taskId"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}
