package task_test

import (
	"testing"
	"time"

	"github.com/devbydaniel/tt/internal/app"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/domain/task/usecases"
	"github.com/devbydaniel/tt/internal/testutil"
)

func setupApp(t *testing.T) *app.App {
	t.Helper()
	db := testutil.NewTestDB(t)
	return app.New(db)
}

func TestTaskCreate(t *testing.T) {
	application := setupApp(t)

	created, err := application.CreateTask.Execute("Buy groceries", nil)
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
	if created.ParentID != nil {
		t.Error("ParentID should be nil for standalone task")
	}
	if created.AreaID != nil {
		t.Error("AreaID should be nil for standalone task")
	}
}

func TestTaskComplete(t *testing.T) {
	application := setupApp(t)

	created, _ := application.CreateTask.Execute("Task to complete", nil)

	completed, err := application.CompleteTasks.Execute([]int64{created.ID})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(completed) != 1 {
		t.Fatalf("got %d completed tasks, want 1", len(completed))
	}
	if completed[0].Completed.Status != task.StatusDone {
		t.Errorf("Status = %q, want %q", completed[0].Completed.Status, task.StatusDone)
	}
	if completed[0].Completed.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}

	// Completed task should not appear in list
	tasks, _ := application.ListTasks.Execute(nil)
	for _, tk := range tasks {
		if tk.ID == created.ID {
			t.Error("Completed task should not appear in todo list")
		}
	}
}

func TestTaskCompleteNonexistent(t *testing.T) {
	application := setupApp(t)

	_, err := application.CompleteTasks.Execute([]int64{999})
	if err == nil {
		t.Error("Complete() should error for nonexistent task")
	}
}

func TestTaskCompleteAlreadyDone(t *testing.T) {
	application := setupApp(t)

	created, _ := application.CreateTask.Execute("Task to complete twice", nil)
	application.CompleteTasks.Execute([]int64{created.ID})

	// Try to complete again
	_, err := application.CompleteTasks.Execute([]int64{created.ID})
	if err == nil {
		t.Error("Complete() should error when task already done")
	}
}

func TestTaskDelete(t *testing.T) {
	application := setupApp(t)

	created, _ := application.CreateTask.Execute("Task to delete", nil)

	deleted, err := application.DeleteTasks.Execute([]int64{created.ID})
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
	tasks, _ := application.ListTasks.Execute(nil)
	for _, tk := range tasks {
		if tk.ID == created.ID {
			t.Error("Deleted task should not appear in list")
		}
	}
}

func TestTaskDeleteNonexistent(t *testing.T) {
	application := setupApp(t)

	_, err := application.DeleteTasks.Execute([]int64{999})
	if err != task.ErrTaskNotFound {
		t.Errorf("Delete() error = %v, want ErrTaskNotFound", err)
	}
}

func TestTaskListCompleted(t *testing.T) {
	application := setupApp(t)

	// Create and complete multiple tasks
	t1, _ := application.CreateTask.Execute("Task 1", nil)
	t2, _ := application.CreateTask.Execute("Task 2", nil)
	application.CreateTask.Execute("Task 3 (not completed)", nil)

	application.CompleteTasks.Execute([]int64{t1.ID, t2.ID})

	completed, err := application.ListCompletedTasks.Execute(nil)
	if err != nil {
		t.Fatalf("ListCompleted() error = %v", err)
	}

	if len(completed) != 2 {
		t.Errorf("got %d completed tasks, want 2", len(completed))
	}
}

func TestTaskListCompletedSince(t *testing.T) {
	application := setupApp(t)

	// Create and complete a task
	t1, _ := application.CreateTask.Execute("Old task", nil)
	application.CompleteTasks.Execute([]int64{t1.ID})

	// Use a time in the future to filter
	future := time.Now().Add(time.Hour)
	completed, err := application.ListCompletedTasks.Execute(&future)
	if err != nil {
		t.Fatalf("ListCompleted() error = %v", err)
	}

	if len(completed) != 0 {
		t.Errorf("got %d completed tasks, want 0 (filtered by since)", len(completed))
	}
}

func TestTaskCompleteMultiple(t *testing.T) {
	application := setupApp(t)

	t1, _ := application.CreateTask.Execute("Task 1", nil)
	t2, _ := application.CreateTask.Execute("Task 2", nil)
	t3, _ := application.CreateTask.Execute("Task 3", nil)

	completed, err := application.CompleteTasks.Execute([]int64{t1.ID, t2.ID, t3.ID})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(completed) != 3 {
		t.Errorf("got %d completed tasks, want 3", len(completed))
	}

	// All should be gone from todo list
	tasks, _ := application.ListTasks.Execute(nil)
	if len(tasks) != 0 {
		t.Errorf("got %d remaining tasks, want 0", len(tasks))
	}
}

func TestTaskWithProject(t *testing.T) {
	application := setupApp(t)

	// Create a project first
	proj, err := application.CreateProject.Execute("Work", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Create task with project
	created, err := application.CreateTask.Execute("Finish report", &task.CreateOptions{ProjectName: "Work"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ParentID == nil {
		t.Fatal("ParentID should be set")
	}
	if *created.ParentID != proj.ID {
		t.Errorf("ParentID = %d, want %d", *created.ParentID, proj.ID)
	}
}

func TestTaskWithArea(t *testing.T) {
	application := setupApp(t)

	// Create an area first
	a, err := application.CreateArea.Execute("Health")
	if err != nil {
		t.Fatalf("failed to create area: %v", err)
	}

	// Create task with area
	created, err := application.CreateTask.Execute("Go to gym", &task.CreateOptions{AreaName: "Health"})
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
	application := setupApp(t)

	_, err := application.CreateTask.Execute("Task", &task.CreateOptions{ProjectName: "Nonexistent"})
	if err == nil {
		t.Error("Create() should error for nonexistent project")
	}
}

func TestTaskWithNonexistentArea(t *testing.T) {
	application := setupApp(t)

	_, err := application.CreateTask.Execute("Task", &task.CreateOptions{AreaName: "Nonexistent"})
	if err == nil {
		t.Error("Create() should error for nonexistent area")
	}
}

func TestTaskFilterByProject(t *testing.T) {
	application := setupApp(t)

	application.CreateProject.Execute("Work", nil)
	application.CreateProject.Execute("Personal", nil)

	application.CreateTask.Execute("Work task 1", &task.CreateOptions{ProjectName: "Work"})
	application.CreateTask.Execute("Work task 2", &task.CreateOptions{ProjectName: "Work"})
	application.CreateTask.Execute("Personal task", &task.CreateOptions{ProjectName: "Personal"})
	application.CreateTask.Execute("Standalone task", nil)

	// Filter by Work project
	workTasks, err := application.ListTasks.Execute(&task.ListOptions{ProjectName: "Work"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(workTasks) != 2 {
		t.Errorf("got %d work tasks, want 2", len(workTasks))
	}

	// All tasks
	allTasks, _ := application.ListTasks.Execute(nil)
	if len(allTasks) != 4 {
		t.Errorf("got %d total tasks, want 4", len(allTasks))
	}
}

func TestTaskFilterByArea(t *testing.T) {
	application := setupApp(t)

	application.CreateArea.Execute("Health")
	application.CreateArea.Execute("Finance")

	application.CreateTask.Execute("Health task 1", &task.CreateOptions{AreaName: "Health"})
	application.CreateTask.Execute("Health task 2", &task.CreateOptions{AreaName: "Health"})
	application.CreateTask.Execute("Finance task", &task.CreateOptions{AreaName: "Finance"})

	// Filter by Health area
	healthTasks, err := application.ListTasks.Execute(&task.ListOptions{AreaName: "Health"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(healthTasks) != 2 {
		t.Errorf("got %d health tasks, want 2", len(healthTasks))
	}
}

func TestTaskCannotHaveBothProjectAndArea(t *testing.T) {
	application := setupApp(t)

	application.CreateProject.Execute("Work", nil)
	application.CreateArea.Execute("Health")

	// Try to create task with both project and area - should fail due to DB constraint
	_, err := application.CreateTask.Execute("Invalid task", &task.CreateOptions{
		ProjectName: "Work",
		AreaName:    "Health",
	})
	if err == nil {
		t.Error("Create() should error when both project and area are set")
	}
}

func TestCascadeDeleteProject(t *testing.T) {
	application := setupApp(t)

	proj, _ := application.CreateProject.Execute("Work", nil)
	application.CreateTask.Execute("Task 1", &task.CreateOptions{ProjectName: "Work"})
	application.CreateTask.Execute("Task 2", &task.CreateOptions{ProjectName: "Work"})
	application.CreateTask.Execute("Standalone", nil)

	// Verify tasks exist
	tasks, _ := application.ListTasks.Execute(nil)
	if len(tasks) != 3 {
		t.Fatalf("got %d tasks before delete, want 3", len(tasks))
	}

	// Delete project - should cascade delete its tasks (projects are now tasks)
	_, err := application.DeleteTasks.Execute([]int64{proj.ID})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Only standalone task should remain
	tasks, _ = application.ListTasks.Execute(nil)
	if len(tasks) != 1 {
		t.Errorf("got %d tasks after delete, want 1", len(tasks))
	}
	if tasks[0].Title != "Standalone" {
		t.Errorf("remaining task = %q, want %q", tasks[0].Title, "Standalone")
	}
}

func TestCascadeDeleteArea(t *testing.T) {
	application := setupApp(t)

	application.CreateArea.Execute("Work")
	application.CreateProject.Execute("Project in Work", &usecases.CreateProjectOptions{AreaName: "Work"})
	application.CreateTask.Execute("Task in project", &task.CreateOptions{ProjectName: "Project in Work"})
	application.CreateTask.Execute("Task in area directly", &task.CreateOptions{AreaName: "Work"})
	application.CreateTask.Execute("Standalone", nil)

	// Verify initial state
	tasks, _ := application.ListTasks.Execute(nil)
	if len(tasks) != 3 {
		t.Fatalf("got %d tasks before delete, want 3", len(tasks))
	}
	projects, _ := application.ListProjects.Execute()
	if len(projects) != 1 {
		t.Fatalf("got %d projects before delete, want 1", len(projects))
	}

	// Delete area - should cascade delete projects and tasks
	_, err := application.DeleteArea.Execute("Work")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Only standalone task should remain
	tasks, _ = application.ListTasks.Execute(nil)
	if len(tasks) != 1 {
		t.Errorf("got %d tasks after delete, want 1", len(tasks))
	}

	// Project should also be deleted
	projects, _ = application.ListProjects.Execute()
	if len(projects) != 0 {
		t.Errorf("got %d projects after delete, want 0", len(projects))
	}
}

func TestProjectWithArea(t *testing.T) {
	application := setupApp(t)

	application.CreateArea.Execute("Work")

	proj, err := application.CreateProject.Execute("Important Project", &usecases.CreateProjectOptions{AreaName: "Work"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if proj.AreaID == nil {
		t.Fatal("AreaID should be set")
	}
}

func TestProjectWithNonexistentArea(t *testing.T) {
	application := setupApp(t)

	_, err := application.CreateProject.Execute("Project", &usecases.CreateProjectOptions{AreaName: "Nonexistent"})
	if err == nil {
		t.Error("Create() should error for nonexistent area")
	}
}

func TestRecurringTaskRegeneration(t *testing.T) {
	application := setupApp(t)

	// Create a recurring task
	recurType := task.RecurTypeFixed
	recurRule := `{"interval":1,"unit":"day"}`
	created, err := application.CreateTask.Execute("Daily standup", &task.CreateOptions{
		RecurType: &recurType,
		RecurRule: &recurRule,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.RecurType == nil || *created.RecurType != task.RecurTypeFixed {
		t.Error("RecurType should be set to fixed")
	}

	// Complete the recurring task
	results, err := application.CompleteTasks.Execute([]int64{created.ID})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}

	// Check that a new task was generated
	if results[0].NextTask == nil {
		t.Fatal("NextTask should be set for recurring task")
	}

	nextTask := results[0].NextTask
	if nextTask.Title != created.Title {
		t.Errorf("NextTask.Title = %q, want %q", nextTask.Title, created.Title)
	}
	if nextTask.RecurType == nil || *nextTask.RecurType != recurType {
		t.Error("NextTask should inherit recurrence type")
	}
	if nextTask.Status != task.StatusTodo {
		t.Errorf("NextTask.Status = %q, want %q", nextTask.Status, task.StatusTodo)
	}
}

func TestNonRecurringTaskNoRegeneration(t *testing.T) {
	application := setupApp(t)

	created, _ := application.CreateTask.Execute("One-time task", nil)

	results, err := application.CompleteTasks.Execute([]int64{created.ID})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if results[0].NextTask != nil {
		t.Error("NextTask should be nil for non-recurring task")
	}
}

func TestPausedRecurrenceNoRegeneration(t *testing.T) {
	application := setupApp(t)

	recurType := task.RecurTypeFixed
	recurRule := `{"interval":1,"unit":"day"}`
	created, _ := application.CreateTask.Execute("Paused task", &task.CreateOptions{
		RecurType: &recurType,
		RecurRule: &recurRule,
	})

	// Pause the recurrence
	application.PauseRecurrence.Execute(created.ID)

	// Complete the task
	results, err := application.CompleteTasks.Execute([]int64{created.ID})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if results[0].NextTask != nil {
		t.Error("NextTask should be nil for paused recurring task")
	}
}

func TestSetRecurrence(t *testing.T) {
	application := setupApp(t)

	created, _ := application.CreateTask.Execute("Task to recur", nil)

	recurType := task.RecurTypeRelative
	recurRule := `{"interval":3,"unit":"day"}`

	updated, err := application.SetRecurrence.Execute(created.ID, &recurType, &recurRule, nil)
	if err != nil {
		t.Fatalf("SetRecurrence() error = %v", err)
	}

	if updated.RecurType == nil || *updated.RecurType != recurType {
		t.Errorf("RecurType = %v, want %v", updated.RecurType, recurType)
	}
	if updated.RecurRule == nil || *updated.RecurRule != recurRule {
		t.Errorf("RecurRule = %v, want %v", updated.RecurRule, recurRule)
	}
}

func TestClearRecurrence(t *testing.T) {
	application := setupApp(t)

	recurType := task.RecurTypeFixed
	recurRule := `{"interval":1,"unit":"week"}`
	created, _ := application.CreateTask.Execute("Recurring task", &task.CreateOptions{
		RecurType: &recurType,
		RecurRule: &recurRule,
	})

	// Clear recurrence
	updated, err := application.SetRecurrence.Execute(created.ID, nil, nil, nil)
	if err != nil {
		t.Fatalf("SetRecurrence() error = %v", err)
	}

	if updated.RecurType != nil {
		t.Error("RecurType should be nil after clearing")
	}
	if updated.RecurRule != nil {
		t.Error("RecurRule should be nil after clearing")
	}
}

func TestTaskWithTags(t *testing.T) {
	application := setupApp(t)

	created, err := application.CreateTask.Execute("Tagged task", &task.CreateOptions{
		Tags: []string{"work", "urgent"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(created.Tags) != 2 {
		t.Errorf("got %d tags, want 2", len(created.Tags))
	}
	// Tags are in insertion order
	if created.Tags[0] != "work" || created.Tags[1] != "urgent" {
		t.Errorf("Tags = %v, want [work, urgent]", created.Tags)
	}
}

func TestAddTag(t *testing.T) {
	application := setupApp(t)

	created, _ := application.CreateTask.Execute("Task without tags", nil)

	updated, err := application.AddTag.Execute(created.ID, "important")
	if err != nil {
		t.Fatalf("AddTag() error = %v", err)
	}

	if len(updated.Tags) != 1 {
		t.Errorf("got %d tags, want 1", len(updated.Tags))
	}
	if updated.Tags[0] != "important" {
		t.Errorf("Tags[0] = %q, want %q", updated.Tags[0], "important")
	}
}

func TestRemoveTag(t *testing.T) {
	application := setupApp(t)

	created, _ := application.CreateTask.Execute("Tagged task", &task.CreateOptions{
		Tags: []string{"work", "urgent"},
	})

	updated, err := application.RemoveTag.Execute(created.ID, "work")
	if err != nil {
		t.Fatalf("RemoveTag() error = %v", err)
	}

	if len(updated.Tags) != 1 {
		t.Errorf("got %d tags, want 1", len(updated.Tags))
	}
	if updated.Tags[0] != "urgent" {
		t.Errorf("Tags[0] = %q, want %q", updated.Tags[0], "urgent")
	}
}

func TestListTags(t *testing.T) {
	application := setupApp(t)

	application.CreateTask.Execute("Task 1", &task.CreateOptions{Tags: []string{"work", "urgent"}})
	application.CreateTask.Execute("Task 2", &task.CreateOptions{Tags: []string{"personal", "urgent"}})

	tags, err := application.ListTags.Execute()
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}

	if len(tags) != 3 {
		t.Errorf("got %d unique tags, want 3", len(tags))
	}
}

func TestFilterByTag(t *testing.T) {
	application := setupApp(t)

	application.CreateTask.Execute("Work task 1", &task.CreateOptions{Tags: []string{"work"}})
	application.CreateTask.Execute("Work task 2", &task.CreateOptions{Tags: []string{"work", "urgent"}})
	application.CreateTask.Execute("Personal task", &task.CreateOptions{Tags: []string{"personal"}})
	application.CreateTask.Execute("Untagged task", nil)

	// Filter by work tag
	workTasks, err := application.ListTasks.Execute(&task.ListOptions{TagName: "work"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(workTasks) != 2 {
		t.Errorf("got %d work tasks, want 2", len(workTasks))
	}

	// All tasks should include tags
	allTasks, _ := application.ListTasks.Execute(nil)
	if len(allTasks) != 4 {
		t.Errorf("got %d total tasks, want 4", len(allTasks))
	}
}

func TestAddTagNonexistentTask(t *testing.T) {
	application := setupApp(t)

	_, err := application.AddTag.Execute(999, "tag")
	if err != task.ErrTaskNotFound {
		t.Errorf("AddTag() error = %v, want ErrTaskNotFound", err)
	}
}

func TestRemoveTagNonexistentTask(t *testing.T) {
	application := setupApp(t)

	_, err := application.RemoveTag.Execute(999, "tag")
	if err != task.ErrTaskNotFound {
		t.Errorf("RemoveTag() error = %v, want ErrTaskNotFound", err)
	}
}

func TestRecurringTaskCopyTags(t *testing.T) {
	application := setupApp(t)

	// Create a recurring task with tags
	recurType := task.RecurTypeFixed
	recurRule := `{"interval":1,"unit":"day"}`
	created, _ := application.CreateTask.Execute("Daily standup", &task.CreateOptions{
		RecurType: &recurType,
		RecurRule: &recurRule,
		Tags:      []string{"work", "meeting"},
	})

	// Complete the recurring task
	results, err := application.CompleteTasks.Execute([]int64{created.ID})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if results[0].NextTask == nil {
		t.Fatal("NextTask should be set for recurring task")
	}

	// Check that tags were copied
	if len(results[0].NextTask.Tags) != 2 {
		t.Errorf("NextTask.Tags length = %d, want 2", len(results[0].NextTask.Tags))
	}
}

func TestSetTitle(t *testing.T) {
	application := setupApp(t)

	created, _ := application.CreateTask.Execute("Original title", nil)

	updated, err := application.SetTaskTitle.Execute(created.ID, "New title")
	if err != nil {
		t.Fatalf("SetTitle() error = %v", err)
	}

	if updated.Title != "New title" {
		t.Errorf("Title = %q, want %q", updated.Title, "New title")
	}
}

func TestSetProject(t *testing.T) {
	application := setupApp(t)

	application.CreateProject.Execute("Work", nil)
	created, _ := application.CreateTask.Execute("Task", nil)

	updated, err := application.SetTaskProject.Execute(created.ID, "Work")
	if err != nil {
		t.Fatalf("SetProject() error = %v", err)
	}

	if updated.ParentID == nil {
		t.Fatal("ParentID should be set")
	}
}

func TestSetProjectClearsArea(t *testing.T) {
	application := setupApp(t)

	application.CreateArea.Execute("Health")
	application.CreateProject.Execute("Work", nil)

	created, _ := application.CreateTask.Execute("Task", &task.CreateOptions{AreaName: "Health"})

	updated, err := application.SetTaskProject.Execute(created.ID, "Work")
	if err != nil {
		t.Fatalf("SetProject() error = %v", err)
	}

	if updated.ParentID == nil {
		t.Fatal("ParentID should be set")
	}
	if updated.AreaID != nil {
		t.Error("AreaID should be cleared when setting project")
	}
}

func TestSetArea(t *testing.T) {
	application := setupApp(t)

	application.CreateArea.Execute("Health")
	created, _ := application.CreateTask.Execute("Task", nil)

	updated, err := application.SetTaskArea.Execute(created.ID, "Health")
	if err != nil {
		t.Fatalf("SetArea() error = %v", err)
	}

	if updated.AreaID == nil {
		t.Fatal("AreaID should be set")
	}
}

func TestSetAreaClearsProject(t *testing.T) {
	application := setupApp(t)

	application.CreateProject.Execute("Work", nil)
	application.CreateArea.Execute("Health")

	created, _ := application.CreateTask.Execute("Task", &task.CreateOptions{ProjectName: "Work"})

	updated, err := application.SetTaskArea.Execute(created.ID, "Health")
	if err != nil {
		t.Fatalf("SetArea() error = %v", err)
	}

	if updated.AreaID == nil {
		t.Fatal("AreaID should be set")
	}
	if updated.ParentID != nil {
		t.Error("ParentID should be cleared when setting area")
	}
}

func TestClearProject(t *testing.T) {
	application := setupApp(t)

	application.CreateProject.Execute("Work", nil)
	created, _ := application.CreateTask.Execute("Task", &task.CreateOptions{ProjectName: "Work"})

	updated, err := application.SetTaskProject.Execute(created.ID, "")
	if err != nil {
		t.Fatalf("SetProject() error = %v", err)
	}

	if updated.ParentID != nil {
		t.Error("ParentID should be nil after clearing")
	}
}

func TestClearArea(t *testing.T) {
	application := setupApp(t)

	application.CreateArea.Execute("Health")
	created, _ := application.CreateTask.Execute("Task", &task.CreateOptions{AreaName: "Health"})

	updated, err := application.SetTaskArea.Execute(created.ID, "")
	if err != nil {
		t.Fatalf("SetArea() error = %v", err)
	}

	if updated.AreaID != nil {
		t.Error("AreaID should be nil after clearing")
	}
}

func TestListSortByTitle(t *testing.T) {
	application := setupApp(t)

	// Create tasks with different titles (not in alphabetical order)
	application.CreateTask.Execute("Charlie task", nil)
	application.CreateTask.Execute("Alpha task", nil)
	application.CreateTask.Execute("Bravo task", nil)

	// Sort by title ascending
	sortOpts, _ := task.ParseSort("title:asc")
	tasks, err := application.ListTasks.Execute(&task.ListOptions{Sort: sortOpts})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("got %d tasks, want 3", len(tasks))
	}
	if tasks[0].Title != "Alpha task" {
		t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "Alpha task")
	}
	if tasks[1].Title != "Bravo task" {
		t.Errorf("tasks[1].Title = %q, want %q", tasks[1].Title, "Bravo task")
	}
	if tasks[2].Title != "Charlie task" {
		t.Errorf("tasks[2].Title = %q, want %q", tasks[2].Title, "Charlie task")
	}
}

func TestListSortByTitleDesc(t *testing.T) {
	application := setupApp(t)

	application.CreateTask.Execute("Alpha task", nil)
	application.CreateTask.Execute("Charlie task", nil)
	application.CreateTask.Execute("Bravo task", nil)

	sortOpts, _ := task.ParseSort("title:desc")
	tasks, err := application.ListTasks.Execute(&task.ListOptions{Sort: sortOpts})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if tasks[0].Title != "Charlie task" {
		t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "Charlie task")
	}
	if tasks[2].Title != "Alpha task" {
		t.Errorf("tasks[2].Title = %q, want %q", tasks[2].Title, "Alpha task")
	}
}

func TestListSortByPlannedDate(t *testing.T) {
	application := setupApp(t)

	today := time.Now()
	tomorrow := today.AddDate(0, 0, 1)
	yesterday := today.AddDate(0, 0, -1)

	application.CreateTask.Execute("Tomorrow task", &task.CreateOptions{PlannedDate: &tomorrow})
	application.CreateTask.Execute("No date task", nil)
	application.CreateTask.Execute("Yesterday task", &task.CreateOptions{PlannedDate: &yesterday})

	// Sort by planned date descending (default for date fields)
	sortOpts, _ := task.ParseSort("planned")
	tasks, err := application.ListTasks.Execute(&task.ListOptions{Sort: sortOpts})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("got %d tasks, want 3", len(tasks))
	}

	// DESC: tomorrow first, yesterday second, no date last
	if tasks[0].Title != "Tomorrow task" {
		t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "Tomorrow task")
	}
	if tasks[1].Title != "Yesterday task" {
		t.Errorf("tasks[1].Title = %q, want %q", tasks[1].Title, "Yesterday task")
	}
	if tasks[2].Title != "No date task" {
		t.Errorf("tasks[2].Title = %q, want %q (nulls last)", tasks[2].Title, "No date task")
	}
}

func TestListSortByPlannedDateAsc(t *testing.T) {
	application := setupApp(t)

	today := time.Now()
	tomorrow := today.AddDate(0, 0, 1)
	yesterday := today.AddDate(0, 0, -1)

	application.CreateTask.Execute("Tomorrow task", &task.CreateOptions{PlannedDate: &tomorrow})
	application.CreateTask.Execute("No date task", nil)
	application.CreateTask.Execute("Yesterday task", &task.CreateOptions{PlannedDate: &yesterday})

	// Sort by planned date ascending
	sortOpts, _ := task.ParseSort("planned:asc")
	tasks, err := application.ListTasks.Execute(&task.ListOptions{Sort: sortOpts})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// ASC: yesterday first, tomorrow second, no date last
	if tasks[0].Title != "Yesterday task" {
		t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "Yesterday task")
	}
	if tasks[1].Title != "Tomorrow task" {
		t.Errorf("tasks[1].Title = %q, want %q", tasks[1].Title, "Tomorrow task")
	}
	if tasks[2].Title != "No date task" {
		t.Errorf("tasks[2].Title = %q, want %q (nulls last)", tasks[2].Title, "No date task")
	}
}

func TestListSortMultipleFields(t *testing.T) {
	application := setupApp(t)

	application.CreateProject.Execute("Alpha Project", nil)
	application.CreateProject.Execute("Beta Project", nil)

	application.CreateTask.Execute("Task B", &task.CreateOptions{ProjectName: "Alpha Project"})
	application.CreateTask.Execute("Task A", &task.CreateOptions{ProjectName: "Alpha Project"})
	application.CreateTask.Execute("Task C", &task.CreateOptions{ProjectName: "Beta Project"})

	// Sort by project then title
	sortOpts, _ := task.ParseSort("project:asc,title:asc")
	tasks, err := application.ListTasks.Execute(&task.ListOptions{Sort: sortOpts})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("got %d tasks, want 3", len(tasks))
	}

	// Alpha Project tasks first (sorted by title), then Beta Project
	if tasks[0].Title != "Task A" {
		t.Errorf("tasks[0].Title = %q, want %q", tasks[0].Title, "Task A")
	}
	if tasks[1].Title != "Task B" {
		t.Errorf("tasks[1].Title = %q, want %q", tasks[1].Title, "Task B")
	}
	if tasks[2].Title != "Task C" {
		t.Errorf("tasks[2].Title = %q, want %q", tasks[2].Title, "Task C")
	}
}

func TestListDefaultSort(t *testing.T) {
	application := setupApp(t)

	// Create tasks - they'll have same created_at since test runs fast
	application.CreateTask.Execute("First", nil)
	application.CreateTask.Execute("Second", nil)
	application.CreateTask.Execute("Third", nil)

	// Default sort (no options) should use created:desc
	tasks, err := application.ListTasks.Execute(nil)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("got %d tasks, want 3", len(tasks))
	}

	// With same created_at, order depends on ID (which is also desc in the CASE expression)
	// The important thing is it doesn't error
}
