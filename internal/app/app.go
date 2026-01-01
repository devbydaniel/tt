package app

import (
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/area"
	areausecases "github.com/devbydaniel/tt/internal/domain/area/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
	taskusecases "github.com/devbydaniel/tt/internal/domain/task/usecases"
)

type App struct {
	// Area use cases
	CreateArea    *areausecases.CreateArea
	ListAreas     *areausecases.ListAreas
	GetAreaByName *areausecases.GetAreaByName
	DeleteArea    *areausecases.DeleteArea
	RenameArea    *areausecases.RenameArea

	// Project use cases (projects are now tasks with task_type='project')
	CreateProject        *taskusecases.CreateProject
	ListProjects         *taskusecases.ListProjects
	ListAllProjects      *taskusecases.ListAllProjects
	ListProjectsWithArea *taskusecases.ListProjectsWithArea
	GetProjectByName     *taskusecases.GetProjectByName

	// Task use cases
	CreateTask         *taskusecases.CreateTask
	ListTasks          *taskusecases.ListTasks
	GetTask            *taskusecases.GetTask
	CompleteTasks      *taskusecases.CompleteTasks
	UncompleteTasks    *taskusecases.UncompleteTasks
	DeleteTasks        *taskusecases.DeleteTasks
	ListCompletedTasks *taskusecases.ListCompletedTasks
	DeferTask          *taskusecases.DeferTask
	ActivateTask       *taskusecases.ActivateTask
	SetPlannedDate     *taskusecases.SetPlannedDate
	SetDueDate         *taskusecases.SetDueDate
	SetTaskProject     *taskusecases.SetTaskProject
	SetTaskArea        *taskusecases.SetTaskArea
	SetTaskTitle       *taskusecases.SetTaskTitle
	SetTaskDescription *taskusecases.SetTaskDescription
	SetRecurrence      *taskusecases.SetRecurrence
	PauseRecurrence    *taskusecases.PauseRecurrence
	ResumeRecurrence   *taskusecases.ResumeRecurrence
	SetRecurrenceEnd   *taskusecases.SetRecurrenceEnd
	AddTag             *taskusecases.AddTag
	RemoveTag          *taskusecases.RemoveTag
	ListTags           *taskusecases.ListTags
	SetTags            *taskusecases.SetTags
}

func New(db *database.DB) *App {
	// Create repositories
	areaRepo := area.NewRepository(db)
	taskRepo := task.NewRepository(db)

	// Create area use cases (no cross-domain dependencies)
	createArea := &areausecases.CreateArea{Repo: areaRepo}
	listAreas := &areausecases.ListAreas{Repo: areaRepo}
	getAreaByName := &areausecases.GetAreaByName{Repo: areaRepo}
	deleteArea := &areausecases.DeleteArea{Repo: areaRepo}
	renameArea := &areausecases.RenameArea{Repo: areaRepo}

	// Create project use cases (projects are now tasks with task_type='project')
	getProjectByName := &taskusecases.GetProjectByName{Repo: taskRepo}
	createProject := &taskusecases.CreateProject{
		Repo:       taskRepo,
		AreaLookup: getAreaByName,
	}
	listProjects := &taskusecases.ListProjects{Repo: taskRepo}
	listAllProjects := &taskusecases.ListAllProjects{Repo: taskRepo}
	listProjectsWithArea := &taskusecases.ListProjectsWithArea{Repo: taskRepo}

	// Create task use cases
	createTask := &taskusecases.CreateTask{
		Repo:          taskRepo,
		ProjectLookup: getProjectByName,
		AreaLookup:    getAreaByName,
	}
	listTasks := &taskusecases.ListTasks{
		Repo:          taskRepo,
		ProjectLookup: getProjectByName,
		AreaLookup:    getAreaByName,
	}
	getTask := &taskusecases.GetTask{Repo: taskRepo}
	completeTasks := &taskusecases.CompleteTasks{Repo: taskRepo}
	uncompleteTasks := &taskusecases.UncompleteTasks{Repo: taskRepo}
	deleteTasks := &taskusecases.DeleteTasks{Repo: taskRepo}
	listCompletedTasks := &taskusecases.ListCompletedTasks{Repo: taskRepo}
	deferTask := &taskusecases.DeferTask{Repo: taskRepo}
	activateTask := &taskusecases.ActivateTask{Repo: taskRepo}
	setPlannedDate := &taskusecases.SetPlannedDate{Repo: taskRepo}
	setDueDate := &taskusecases.SetDueDate{Repo: taskRepo}
	setTaskProject := &taskusecases.SetTaskProject{
		Repo:          taskRepo,
		ProjectLookup: getProjectByName,
	}
	setTaskArea := &taskusecases.SetTaskArea{
		Repo:       taskRepo,
		AreaLookup: getAreaByName,
	}
	setTaskTitle := &taskusecases.SetTaskTitle{Repo: taskRepo}
	setTaskDescription := &taskusecases.SetTaskDescription{Repo: taskRepo}
	setRecurrence := &taskusecases.SetRecurrence{Repo: taskRepo}
	pauseRecurrence := &taskusecases.PauseRecurrence{Repo: taskRepo}
	resumeRecurrence := &taskusecases.ResumeRecurrence{Repo: taskRepo}
	setRecurrenceEnd := &taskusecases.SetRecurrenceEnd{Repo: taskRepo}
	addTag := &taskusecases.AddTag{Repo: taskRepo}
	removeTag := &taskusecases.RemoveTag{Repo: taskRepo}
	listTagsUC := &taskusecases.ListTags{Repo: taskRepo}
	setTags := &taskusecases.SetTags{Repo: taskRepo}

	return &App{
		// Area
		CreateArea:    createArea,
		ListAreas:     listAreas,
		GetAreaByName: getAreaByName,
		DeleteArea:    deleteArea,
		RenameArea:    renameArea,

		// Project (tasks with task_type='project')
		CreateProject:        createProject,
		ListProjects:         listProjects,
		ListAllProjects:      listAllProjects,
		ListProjectsWithArea: listProjectsWithArea,
		GetProjectByName:     getProjectByName,

		// Task
		CreateTask:         createTask,
		ListTasks:          listTasks,
		GetTask:            getTask,
		CompleteTasks:      completeTasks,
		UncompleteTasks:    uncompleteTasks,
		DeleteTasks:        deleteTasks,
		ListCompletedTasks: listCompletedTasks,
		DeferTask:          deferTask,
		ActivateTask:       activateTask,
		SetPlannedDate:     setPlannedDate,
		SetDueDate:         setDueDate,
		SetTaskProject:     setTaskProject,
		SetTaskArea:        setTaskArea,
		SetTaskTitle:       setTaskTitle,
		SetTaskDescription: setTaskDescription,
		SetRecurrence:      setRecurrence,
		PauseRecurrence:    pauseRecurrence,
		ResumeRecurrence:   resumeRecurrence,
		SetRecurrenceEnd:   setRecurrenceEnd,
		AddTag:             addTag,
		RemoveTag:          removeTag,
		ListTags:           listTagsUC,
		SetTags:            setTags,
	}
}
