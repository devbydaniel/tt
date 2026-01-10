package app

import (
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/area"
	areausecases "github.com/devbydaniel/tt/internal/domain/area/usecases"
	"github.com/devbydaniel/tt/internal/domain/syncevent"
	synceventusecases "github.com/devbydaniel/tt/internal/domain/syncevent/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
	taskusecases "github.com/devbydaniel/tt/internal/domain/task/usecases"
)

type App struct {
	// Area use cases
	CreateArea    *areausecases.CreateArea
	ListAreas     *areausecases.ListAreas
	GetAreaByName *areausecases.GetAreaByName
	GetAreaByID   *areausecases.GetAreaByID
	DeleteArea    *areausecases.DeleteArea
	RenameArea    *areausecases.RenameArea

	// Sync event use cases
	PersistSyncEvent *synceventusecases.PersistSyncEvent
	PushEvents       *synceventusecases.PushEvents

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

// SyncConfig holds sync configuration.
type SyncConfig struct {
	URL    string
	APIKey string
}

func New(db *database.DB, clientID string, syncCfg *SyncConfig) *App {
	// Create repositories
	areaRepo := area.NewRepository(db)
	taskRepo := task.NewRepository(db)

	// Create area use cases (no cross-domain dependencies)
	createArea := &areausecases.CreateArea{Repo: areaRepo}
	listAreas := &areausecases.ListAreas{Repo: areaRepo}
	getAreaByName := &areausecases.GetAreaByName{Repo: areaRepo}
	getAreaByID := &areausecases.GetAreaByID{Repo: areaRepo}
	deleteArea := &areausecases.DeleteArea{Repo: areaRepo}
	renameArea := &areausecases.RenameArea{Repo: areaRepo}

	// Create project use cases (projects are now tasks with task_type='project')
	getProjectByName := &taskusecases.GetProjectByName{Repo: taskRepo}
	getTask := &taskusecases.GetTask{Repo: taskRepo}

	// Create sync event persister (nil if sync is disabled)
	var syncPersister taskusecases.SyncEventPersister
	var syncEventRepo *syncevent.Repository
	if clientID != "" {
		syncEventRepo = syncevent.NewRepository(db)
		syncPersister = &synceventusecases.PersistSyncEvent{
			Repo:       syncEventRepo,
			TaskLookup: getTask,
			AreaLookup: getAreaByID,
		}
	}

	// Create push events use case (nil if sync server not configured)
	var pushEvents *synceventusecases.PushEvents
	if syncCfg != nil && syncCfg.URL != "" && syncCfg.APIKey != "" && clientID != "" {
		if syncEventRepo == nil {
			syncEventRepo = syncevent.NewRepository(db)
		}
		pushEvents = &synceventusecases.PushEvents{
			Repo:     syncEventRepo,
			Client:   syncevent.NewClient(syncCfg.URL, syncCfg.APIKey),
			ClientID: clientID,
		}
	}

	createProject := &taskusecases.CreateProject{
		Repo:          taskRepo,
		AreaLookup:    getAreaByName,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	listProjects := &taskusecases.ListProjects{Repo: taskRepo}
	listAllProjects := &taskusecases.ListAllProjects{Repo: taskRepo}
	listProjectsWithArea := &taskusecases.ListProjectsWithArea{Repo: taskRepo}

	// Create task use cases
	createTask := &taskusecases.CreateTask{
		Repo:          taskRepo,
		ProjectLookup: getProjectByName,
		AreaLookup:    getAreaByName,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	listTasks := &taskusecases.ListTasks{
		Repo:          taskRepo,
		ProjectLookup: getProjectByName,
		AreaLookup:    getAreaByName,
	}
	completeTasks := &taskusecases.CompleteTasks{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	uncompleteTasks := &taskusecases.UncompleteTasks{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	deleteTasks := &taskusecases.DeleteTasks{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	listCompletedTasks := &taskusecases.ListCompletedTasks{Repo: taskRepo}
	deferTask := &taskusecases.DeferTask{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	activateTask := &taskusecases.ActivateTask{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	setPlannedDate := &taskusecases.SetPlannedDate{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	setDueDate := &taskusecases.SetDueDate{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	setTaskProject := &taskusecases.SetTaskProject{
		Repo:          taskRepo,
		ProjectLookup: getProjectByName,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	setTaskArea := &taskusecases.SetTaskArea{
		Repo:          taskRepo,
		AreaLookup:    getAreaByName,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	setTaskTitle := &taskusecases.SetTaskTitle{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	setTaskDescription := &taskusecases.SetTaskDescription{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	setRecurrence := &taskusecases.SetRecurrence{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	pauseRecurrence := &taskusecases.PauseRecurrence{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	resumeRecurrence := &taskusecases.ResumeRecurrence{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	setRecurrenceEnd := &taskusecases.SetRecurrenceEnd{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	addTag := &taskusecases.AddTag{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	removeTag := &taskusecases.RemoveTag{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}
	listTagsUC := &taskusecases.ListTags{Repo: taskRepo}
	setTags := &taskusecases.SetTags{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
	}

	// Expose sync persister for direct access (nil if sync disabled)
	var persistSyncEvent *synceventusecases.PersistSyncEvent
	if p, ok := syncPersister.(*synceventusecases.PersistSyncEvent); ok {
		persistSyncEvent = p
	}

	return &App{
		// Area
		CreateArea:    createArea,
		ListAreas:     listAreas,
		GetAreaByName: getAreaByName,
		GetAreaByID:   getAreaByID,
		DeleteArea:    deleteArea,
		RenameArea:    renameArea,

		// Sync event
		PersistSyncEvent: persistSyncEvent,
		PushEvents:       pushEvents,

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
