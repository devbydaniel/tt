package task_test

import (
	"testing"
	"time"

	"github.com/devbydaniel/t/internal/domain/area"
	"github.com/devbydaniel/t/internal/domain/project"
	"github.com/devbydaniel/t/internal/domain/task"
	"github.com/devbydaniel/t/internal/testutil"
)

func setupServices(t *testing.T) (*task.Service, *project.Service, *area.Service) {
	t.Helper()
	db := testutil.NewTestDB(t)

	areaRepo := area.NewRepository(db)
	areaSvc := area.NewService(areaRepo)

	projectRepo := project.NewRepository(db)
	projectSvc := project.NewService(projectRepo, areaSvc)

	taskRepo := task.NewRepository(db)
	taskSvc := task.NewService(taskRepo, projectSvc, areaSvc)

	return taskSvc, projectSvc, areaSvc
}

func TestTaskCreate(t *testing.T) {
	taskSvc, _, _ := setupServices(t)

	created, err := taskSvc.Create("Buy groceries", nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.Title != "Buy groceries" {
		t.Errorf("Title = %q, want %q", created.Title, "Buy groceries")
	}
	if created.Status != task.StatusTodo {
		t.Errorf("Status = %q, want %q", created.Status, task.StatusTodo)
	}
	if created.ID == 0 {
		t.Error("ID should be assigned after creation")
	}
	if created.UUID == "" {
		t.Error("UUID should be generated")
	}
	if created.ProjectID != nil {
		t.Error("ProjectID should be nil for standalone task")
	}
	if created.AreaID != nil {
		t.Error("AreaID should be nil for standalone task")
	}
}

func TestTaskComplete(t *testing.T) {
	taskSvc, _, _ := setupServices(t)

	created, _ := taskSvc.Create("Task to complete", nil)

	completed, err := taskSvc.Complete([]int64{created.ID})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(completed) != 1 {
		t.Fatalf("got %d completed tasks, want 1", len(completed))
	}
	if completed[0].Status != task.StatusDone {
		t.Errorf("Status = %q, want %q", completed[0].Status, task.StatusDone)
	}
	if completed[0].CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}

	// Completed task should not appear in list
	tasks, _ := taskSvc.List(nil)
	for _, tk := range tasks {
		if tk.ID == created.ID {
			t.Error("Completed task should not appear in todo list")
		}
	}
}

func TestTaskCompleteNonexistent(t *testing.T) {
	taskSvc, _, _ := setupServices(t)

	_, err := taskSvc.Complete([]int64{999})
	if err == nil {
		t.Error("Complete() should error for nonexistent task")
	}
}

func TestTaskCompleteAlreadyDone(t *testing.T) {
	taskSvc, _, _ := setupServices(t)

	created, _ := taskSvc.Create("Task to complete twice", nil)
	taskSvc.Complete([]int64{created.ID})

	// Try to complete again
	_, err := taskSvc.Complete([]int64{created.ID})
	if err == nil {
		t.Error("Complete() should error when task already done")
	}
}

func TestTaskDelete(t *testing.T) {
	taskSvc, _, _ := setupServices(t)

	created, _ := taskSvc.Create("Task to delete", nil)

	deleted, err := taskSvc.Delete([]int64{created.ID})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if len(deleted) != 1 {
		t.Fatalf("got %d deleted tasks, want 1", len(deleted))
	}
	if deleted[0].ID != created.ID {
		t.Errorf("deleted task ID = %d, want %d", deleted[0].ID, created.ID)
	}

	// Deleted task should not appear anywhere
	tasks, _ := taskSvc.List(nil)
	for _, tk := range tasks {
		if tk.ID == created.ID {
			t.Error("Deleted task should not appear in list")
		}
	}
}

func TestTaskDeleteNonexistent(t *testing.T) {
	taskSvc, _, _ := setupServices(t)

	_, err := taskSvc.Delete([]int64{999})
	if err != task.ErrTaskNotFound {
		t.Errorf("Delete() error = %v, want ErrTaskNotFound", err)
	}
}

func TestTaskListCompleted(t *testing.T) {
	taskSvc, _, _ := setupServices(t)

	// Create and complete multiple tasks
	t1, _ := taskSvc.Create("Task 1", nil)
	t2, _ := taskSvc.Create("Task 2", nil)
	taskSvc.Create("Task 3 (not completed)", nil)

	taskSvc.Complete([]int64{t1.ID, t2.ID})

	completed, err := taskSvc.ListCompleted(nil)
	if err != nil {
		t.Fatalf("ListCompleted() error = %v", err)
	}

	if len(completed) != 2 {
		t.Errorf("got %d completed tasks, want 2", len(completed))
	}
}

func TestTaskListCompletedSince(t *testing.T) {
	taskSvc, _, _ := setupServices(t)

	// Create and complete a task
	t1, _ := taskSvc.Create("Old task", nil)
	taskSvc.Complete([]int64{t1.ID})

	// Use a time in the future to filter
	future := time.Now().Add(time.Hour)
	completed, err := taskSvc.ListCompleted(&future)
	if err != nil {
		t.Fatalf("ListCompleted() error = %v", err)
	}

	if len(completed) != 0 {
		t.Errorf("got %d completed tasks, want 0 (filtered by since)", len(completed))
	}
}

func TestTaskCompleteMultiple(t *testing.T) {
	taskSvc, _, _ := setupServices(t)

	t1, _ := taskSvc.Create("Task 1", nil)
	t2, _ := taskSvc.Create("Task 2", nil)
	t3, _ := taskSvc.Create("Task 3", nil)

	completed, err := taskSvc.Complete([]int64{t1.ID, t2.ID, t3.ID})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(completed) != 3 {
		t.Errorf("got %d completed tasks, want 3", len(completed))
	}

	// All should be gone from todo list
	tasks, _ := taskSvc.List(nil)
	if len(tasks) != 0 {
		t.Errorf("got %d remaining tasks, want 0", len(tasks))
	}
}

func TestTaskWithProject(t *testing.T) {
	taskSvc, projectSvc, _ := setupServices(t)

	// Create a project first
	proj, err := projectSvc.Create("Work", "")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Create task with project
	created, err := taskSvc.Create("Finish report", &task.CreateOptions{ProjectName: "Work"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ProjectID == nil {
		t.Fatal("ProjectID should be set")
	}
	if *created.ProjectID != proj.ID {
		t.Errorf("ProjectID = %d, want %d", *created.ProjectID, proj.ID)
	}
}

func TestTaskWithArea(t *testing.T) {
	taskSvc, _, areaSvc := setupServices(t)

	// Create an area first
	a, err := areaSvc.Create("Health")
	if err != nil {
		t.Fatalf("failed to create area: %v", err)
	}

	// Create task with area
	created, err := taskSvc.Create("Go to gym", &task.CreateOptions{AreaName: "Health"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.AreaID == nil {
		t.Fatal("AreaID should be set")
	}
	if *created.AreaID != a.ID {
		t.Errorf("AreaID = %d, want %d", *created.AreaID, a.ID)
	}
}

func TestTaskWithNonexistentProject(t *testing.T) {
	taskSvc, _, _ := setupServices(t)

	_, err := taskSvc.Create("Task", &task.CreateOptions{ProjectName: "Nonexistent"})
	if err == nil {
		t.Error("Create() should error for nonexistent project")
	}
}

func TestTaskWithNonexistentArea(t *testing.T) {
	taskSvc, _, _ := setupServices(t)

	_, err := taskSvc.Create("Task", &task.CreateOptions{AreaName: "Nonexistent"})
	if err == nil {
		t.Error("Create() should error for nonexistent area")
	}
}

func TestTaskFilterByProject(t *testing.T) {
	taskSvc, projectSvc, _ := setupServices(t)

	projectSvc.Create("Work", "")
	projectSvc.Create("Personal", "")

	taskSvc.Create("Work task 1", &task.CreateOptions{ProjectName: "Work"})
	taskSvc.Create("Work task 2", &task.CreateOptions{ProjectName: "Work"})
	taskSvc.Create("Personal task", &task.CreateOptions{ProjectName: "Personal"})
	taskSvc.Create("Standalone task", nil)

	// Filter by Work project
	workTasks, err := taskSvc.List(&task.ListOptions{ProjectName: "Work"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(workTasks) != 2 {
		t.Errorf("got %d work tasks, want 2", len(workTasks))
	}

	// All tasks
	allTasks, _ := taskSvc.List(nil)
	if len(allTasks) != 4 {
		t.Errorf("got %d total tasks, want 4", len(allTasks))
	}
}

func TestTaskFilterByArea(t *testing.T) {
	taskSvc, _, areaSvc := setupServices(t)

	areaSvc.Create("Health")
	areaSvc.Create("Finance")

	taskSvc.Create("Health task 1", &task.CreateOptions{AreaName: "Health"})
	taskSvc.Create("Health task 2", &task.CreateOptions{AreaName: "Health"})
	taskSvc.Create("Finance task", &task.CreateOptions{AreaName: "Finance"})

	// Filter by Health area
	healthTasks, err := taskSvc.List(&task.ListOptions{AreaName: "Health"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(healthTasks) != 2 {
		t.Errorf("got %d health tasks, want 2", len(healthTasks))
	}
}

func TestTaskCannotHaveBothProjectAndArea(t *testing.T) {
	taskSvc, projectSvc, areaSvc := setupServices(t)

	projectSvc.Create("Work", "")
	areaSvc.Create("Health")

	// Try to create task with both project and area - should fail due to DB constraint
	_, err := taskSvc.Create("Invalid task", &task.CreateOptions{
		ProjectName: "Work",
		AreaName:    "Health",
	})
	if err == nil {
		t.Error("Create() should error when both project and area are set")
	}
}

func TestCascadeDeleteProject(t *testing.T) {
	taskSvc, projectSvc, _ := setupServices(t)

	projectSvc.Create("Work", "")
	taskSvc.Create("Task 1", &task.CreateOptions{ProjectName: "Work"})
	taskSvc.Create("Task 2", &task.CreateOptions{ProjectName: "Work"})
	taskSvc.Create("Standalone", nil)

	// Verify tasks exist
	tasks, _ := taskSvc.List(nil)
	if len(tasks) != 3 {
		t.Fatalf("got %d tasks before delete, want 3", len(tasks))
	}

	// Delete project - should cascade delete its tasks
	_, err := projectSvc.Delete("Work")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Only standalone task should remain
	tasks, _ = taskSvc.List(nil)
	if len(tasks) != 1 {
		t.Errorf("got %d tasks after delete, want 1", len(tasks))
	}
	if tasks[0].Title != "Standalone" {
		t.Errorf("remaining task = %q, want %q", tasks[0].Title, "Standalone")
	}
}

func TestCascadeDeleteArea(t *testing.T) {
	taskSvc, projectSvc, areaSvc := setupServices(t)

	areaSvc.Create("Work")
	projectSvc.Create("Project in Work", "Work")
	taskSvc.Create("Task in project", &task.CreateOptions{ProjectName: "Project in Work"})
	taskSvc.Create("Task in area directly", &task.CreateOptions{AreaName: "Work"})
	taskSvc.Create("Standalone", nil)

	// Verify initial state
	tasks, _ := taskSvc.List(nil)
	if len(tasks) != 3 {
		t.Fatalf("got %d tasks before delete, want 3", len(tasks))
	}
	projects, _ := projectSvc.List()
	if len(projects) != 1 {
		t.Fatalf("got %d projects before delete, want 1", len(projects))
	}

	// Delete area - should cascade delete projects and tasks
	_, err := areaSvc.Delete("Work")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Only standalone task should remain
	tasks, _ = taskSvc.List(nil)
	if len(tasks) != 1 {
		t.Errorf("got %d tasks after delete, want 1", len(tasks))
	}

	// Project should also be deleted
	projects, _ = projectSvc.List()
	if len(projects) != 0 {
		t.Errorf("got %d projects after delete, want 0", len(projects))
	}
}

func TestProjectWithArea(t *testing.T) {
	_, projectSvc, areaSvc := setupServices(t)

	areaSvc.Create("Work")

	proj, err := projectSvc.Create("Important Project", "Work")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if proj.AreaID == nil {
		t.Fatal("AreaID should be set")
	}
}

func TestProjectWithNonexistentArea(t *testing.T) {
	_, projectSvc, _ := setupServices(t)

	_, err := projectSvc.Create("Project", "Nonexistent")
	if err == nil {
		t.Error("Create() should error for nonexistent area")
	}
}
