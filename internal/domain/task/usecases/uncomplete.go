package usecases

import "github.com/devbydaniel/tt/internal/domain/task"

type UncompleteTasks struct {
	Repo *task.Repository
}

func (u *UncompleteTasks) Execute(ids []int64) ([]task.Task, error) {
	var tasks []task.Task

	for _, id := range ids {
		if err := u.Repo.Uncomplete(id); err != nil {
			return tasks, err
		}
		t, err := u.Repo.GetByID(id)
		if err != nil {
			return tasks, err
		}
		tasks = append(tasks, *t)
	}

	return tasks, nil
}
