package app

import (
	"github.com/devbydaniel/tt/internal/database"
	"github.com/devbydaniel/tt/internal/domain/area"
	areausecases "github.com/devbydaniel/tt/internal/domain/area/usecases"
	"github.com/devbydaniel/tt/internal/domain/comment"
	commentusecases "github.com/devbydaniel/tt/internal/domain/comment/usecases"
	"github.com/devbydaniel/tt/internal/domain/note"
	noteusecases "github.com/devbydaniel/tt/internal/domain/note/usecases"
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
	GetAreaByUUID *areausecases.GetAreaByUUID
	DeleteArea    *areausecases.DeleteArea
	RenameArea    *areausecases.RenameArea

	// Sync event use cases
	PersistSyncEvent  *synceventusecases.PersistSyncEvent
	PushEvents        *synceventusecases.PushEvents
	SyncEvents        *synceventusecases.SyncEvents
	ResetSync         *synceventusecases.ResetSync
	ListFailedEvents  *synceventusecases.ListFailedEvents
	CleanFailedEvents *synceventusecases.CleanFailedEvents

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
	GetTaskByUUID      *taskusecases.GetTaskByUUID
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
	DeleteTag          *taskusecases.DeleteTag

	// Note use cases (filesystem-backed, no DB)
	ListNotes   *noteusecases.ListNotes
	CreateNote  *noteusecases.CreateNote
	SearchNotes *noteusecases.SearchNotes

	// Comment use cases
	AddComment   *commentusecases.AddComment
	ListComments *commentusecases.ListComments
}

// SyncConfig holds sync configuration.
type SyncConfig struct {
	URL    string
	APIKey string
}

func New(db *database.DB, clientID string, syncCfg *SyncConfig, notesDir string) *App {
	// Create repositories
	areaRepo := area.NewRepository(db)
	taskRepo := task.NewRepository(db)
	commentRepo := comment.NewRepository(db)
	noteRepo := note.NewRepository(notesDir)

	// Create read-only area use cases first (needed by sync persister)
	listAreas := &areausecases.ListAreas{Repo: areaRepo}
	getAreaByName := &areausecases.GetAreaByName{Repo: areaRepo}
	getAreaByID := &areausecases.GetAreaByID{Repo: areaRepo}
	getAreaByUUID := &areausecases.GetAreaByUUID{Repo: areaRepo}

	// Create project use cases (projects are now tasks with task_type='project')
	getProjectByName := &taskusecases.GetProjectByName{Repo: taskRepo}
	getTask := &taskusecases.GetTask{Repo: taskRepo}
	getTaskByUUID := &taskusecases.GetTaskByUUID{Repo: taskRepo}

	// Create sync event persister (nil if sync is disabled)
	var syncPersister taskusecases.SyncEventPersister
	var areaSyncPersister areausecases.SyncEventPersister
	var syncEventRepo *syncevent.Repository
	var resetSync *synceventusecases.ResetSync
	var listFailedEvents *synceventusecases.ListFailedEvents
	var cleanFailedEvents *synceventusecases.CleanFailedEvents
	if clientID != "" {
		syncEventRepo = syncevent.NewRepository(db)
		persistSyncEventUC := &synceventusecases.PersistSyncEvent{
			Repo:       syncEventRepo,
			TaskLookup: getTask,
			AreaLookup: getAreaByID,
		}
		syncPersister = persistSyncEventUC
		areaSyncPersister = persistSyncEventUC

		// ResetSync needs the sync client to reset the server.
		// Client is created below if syncCfg is available; we set it after.
		resetSync = &synceventusecases.ResetSync{
			Repo:          syncEventRepo,
			TaskLister:    &taskusecases.ListAllTasks{Repo: taskRepo},
			AreaLister:    listAreas,
			SyncPersister: persistSyncEventUC,
			ClientID:      clientID,
			DB:            db,
		}
		listFailedEvents = &synceventusecases.ListFailedEvents{Repo: syncEventRepo}
		cleanFailedEvents = &synceventusecases.CleanFailedEvents{Repo: syncEventRepo}
	}

	// Create area use cases with sync support
	createArea := &areausecases.CreateArea{
		Repo:          areaRepo,
		SyncPersister: areaSyncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	renameArea := &areausecases.RenameArea{
		Repo:          areaRepo,
		SyncPersister: areaSyncPersister,
		ClientID:      clientID,
		DB:            db,
	}

	// Create deleteTasks first (needed by deleteArea)
	deleteTasks := &taskusecases.DeleteTasks{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}

	// Create deleteArea with task deletion support
	deleteArea := &areausecases.DeleteArea{
		Repo:          areaRepo,
		TaskLister:    taskRepo,
		TaskDeleter:   deleteTasks,
		SyncPersister: areaSyncPersister,
		ClientID:      clientID,
		DB:            db,
	}

	// Create push events use case (nil if sync server not configured)
	var pushEvents *synceventusecases.PushEvents
	var syncEvents *synceventusecases.SyncEvents
	if syncCfg != nil && syncCfg.URL != "" && syncCfg.APIKey != "" && clientID != "" {
		if syncEventRepo == nil {
			syncEventRepo = syncevent.NewRepository(db)
		}
		syncClient := syncevent.NewClient(syncCfg.URL, syncCfg.APIKey)
		pushEvents = &synceventusecases.PushEvents{
			Repo:     syncEventRepo,
			Client:   syncClient,
			ClientID: clientID,
		}
		// Create the applier for applying remote entity states
		applier := &synceventusecases.ApplyEntityStates{
			TaskUpserter:     taskRepo,
			TaskByUUIDLookup: taskRepo,
			AreaByUUIDLookup: areaRepo,
			AreaUpserter:     areaRepo,
			PendingRepo:      syncEventRepo,
		}
		syncEvents = &synceventusecases.SyncEvents{
			Repo:     syncEventRepo,
			Client:   syncClient,
			ClientID: clientID,
			Applier:  applier,
		}
		// Wire the sync client into ResetSync so it can reset the server
		if resetSync != nil {
			resetSync.Client = syncClient
		}
	}

	createProject := &taskusecases.CreateProject{
		Repo:          taskRepo,
		AreaLookup:    getAreaByName,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
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
		DB:            db,
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
		DB:            db,
	}
	uncompleteTasks := &taskusecases.UncompleteTasks{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	listCompletedTasks := &taskusecases.ListCompletedTasks{Repo: taskRepo}
	deferTask := &taskusecases.DeferTask{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	activateTask := &taskusecases.ActivateTask{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	setPlannedDate := &taskusecases.SetPlannedDate{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	setDueDate := &taskusecases.SetDueDate{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	setTaskProject := &taskusecases.SetTaskProject{
		Repo:          taskRepo,
		ProjectLookup: getProjectByName,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	setTaskArea := &taskusecases.SetTaskArea{
		Repo:          taskRepo,
		AreaLookup:    getAreaByName,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	setTaskTitle := &taskusecases.SetTaskTitle{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	setTaskDescription := &taskusecases.SetTaskDescription{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	setRecurrence := &taskusecases.SetRecurrence{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	pauseRecurrence := &taskusecases.PauseRecurrence{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	resumeRecurrence := &taskusecases.ResumeRecurrence{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	setRecurrenceEnd := &taskusecases.SetRecurrenceEnd{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	addTag := &taskusecases.AddTag{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	removeTag := &taskusecases.RemoveTag{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	listTagsUC := &taskusecases.ListTags{Repo: taskRepo}
	setTags := &taskusecases.SetTags{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}
	deleteTag := &taskusecases.DeleteTag{
		Repo:          taskRepo,
		SyncPersister: syncPersister,
		ClientID:      clientID,
		DB:            db,
	}

	// Expose sync persister for direct access (nil if sync disabled)
	var persistSyncEventExposed *synceventusecases.PersistSyncEvent
	if p, ok := syncPersister.(*synceventusecases.PersistSyncEvent); ok {
		persistSyncEventExposed = p
	}

	return &App{
		// Area
		CreateArea:    createArea,
		ListAreas:     listAreas,
		GetAreaByName: getAreaByName,
		GetAreaByID:   getAreaByID,
		GetAreaByUUID: getAreaByUUID,
		DeleteArea:    deleteArea,
		RenameArea:    renameArea,

		// Sync event
		PersistSyncEvent:  persistSyncEventExposed,
		PushEvents:        pushEvents,
		SyncEvents:        syncEvents,
		ResetSync:         resetSync,
		ListFailedEvents:  listFailedEvents,
		CleanFailedEvents: cleanFailedEvents,

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
		GetTaskByUUID:      getTaskByUUID,
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
		DeleteTag:          deleteTag,

		// Notes
		ListNotes:   &noteusecases.ListNotes{Repo: noteRepo},
		CreateNote:  &noteusecases.CreateNote{Repo: noteRepo},
		SearchNotes: &noteusecases.SearchNotes{Repo: noteRepo},

		// Comments
		AddComment: &commentusecases.AddComment{
			Repo:       commentRepo,
			TaskLookup: getTask,
		},
		ListComments: &commentusecases.ListComments{Repo: commentRepo},
	}
}
