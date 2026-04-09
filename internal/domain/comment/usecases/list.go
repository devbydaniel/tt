package usecases

import "github.com/devbydaniel/tt/internal/domain/comment"

type ListComments struct {
	Repo *comment.Repository
}

func (l *ListComments) Execute(taskID int64) ([]comment.Comment, error) {
	return l.Repo.ListByTask(taskID)
}
