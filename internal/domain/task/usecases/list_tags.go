package usecases

import "github.com/devbydaniel/tt/internal/domain/task"

type ListTags struct {
	Repo *task.Repository
}

func (l *ListTags) Execute() ([]string, error) {
	return l.Repo.ListTags()
}
