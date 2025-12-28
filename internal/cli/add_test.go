package cli_test

import (
	"bytes"
	"testing"

	"github.com/devbydaniel/tt/config"
	"github.com/devbydaniel/tt/internal/cli"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/project"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/testutil"
)

func setupCLI(t *testing.T) *cli.Dependencies {
	t.Helper()
	db := testutil.NewTestDB(t)

	areaRepo := area.NewRepository(db)
	areaSvc := area.NewService(areaRepo)

	projectRepo := project.NewRepository(db)
	projectSvc := project.NewService(projectRepo, areaSvc)

	taskRepo := task.NewRepository(db)
	taskSvc := task.NewService(taskRepo, projectSvc, areaSvc)

	return &cli.Dependencies{
		TaskService:    taskSvc,
		AreaService:    areaSvc,
		ProjectService: projectSvc,
		Config:         &config.Config{},
	}
}

func TestAddWithPlannedShorthand(t *testing.T) {
	deps := setupCLI(t)

	// Create an area first
	_, err := deps.AreaService.Create("work")
	if err != nil {
		t.Fatalf("failed to create area: %v", err)
	}

	// Test using -P shorthand for --planned with -a for area
	cmd := cli.NewRootCmd(deps)
	cmd.SetArgs([]string{"add", "Test Task", "-a", "work", "-P", "today"})

	var out bytes.Buffer
	cmd.SetOut(&out)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Verify task was created with correct area
	tasks, err := deps.TaskService.List(nil)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "Test Task" {
		t.Errorf("title = %q, want %q", tasks[0].Title, "Test Task")
	}
	if tasks[0].AreaID == nil {
		t.Error("AreaID should be set")
	}
	if tasks[0].PlannedDate == nil {
		t.Error("PlannedDate should be set")
	}
	if tasks[0].ProjectID != nil {
		t.Error("ProjectID should be nil when only area is specified")
	}
}

func TestAddWithAreaAndPlannedLongFlag(t *testing.T) {
	deps := setupCLI(t)

	// Create an area first
	_, err := deps.AreaService.Create("personal")
	if err != nil {
		t.Fatalf("failed to create area: %v", err)
	}

	// Test using --planned long flag with -a for area
	cmd := cli.NewRootCmd(deps)
	cmd.SetArgs([]string{"add", "Another Task", "-a", "personal", "--planned", "tomorrow"})

	var out bytes.Buffer
	cmd.SetOut(&out)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Verify task was created
	tasks, err := deps.TaskService.List(nil)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].AreaID == nil {
		t.Error("AreaID should be set")
	}
	if tasks[0].PlannedDate == nil {
		t.Error("PlannedDate should be set")
	}
}

func TestAddCannotSpecifyBothProjectAndArea(t *testing.T) {
	deps := setupCLI(t)

	// Create both project and area
	_, err := deps.AreaService.Create("health")
	if err != nil {
		t.Fatalf("failed to create area: %v", err)
	}
	_, err = deps.ProjectService.Create("work", "")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Try to add task with both -p and -a explicitly
	cmd := cli.NewRootCmd(deps)
	cmd.SetArgs([]string{"add", "Invalid Task", "-p", "work", "-a", "health"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err = cmd.Execute()
	if err == nil {
		t.Error("expected error when specifying both project and area")
	}
}

func TestAddWithTodayShorthand(t *testing.T) {
	deps := setupCLI(t)

	// Create an area first
	_, err := deps.AreaService.Create("work")
	if err != nil {
		t.Fatalf("failed to create area: %v", err)
	}

	// Test using -T shorthand for --today with -a for area
	cmd := cli.NewRootCmd(deps)
	cmd.SetArgs([]string{"add", "Today Task", "-a", "work", "-T"})

	var out bytes.Buffer
	cmd.SetOut(&out)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Verify task was created with planned date set to today
	tasks, err := deps.TaskService.List(nil)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].PlannedDate == nil {
		t.Error("PlannedDate should be set when using --today")
	}
	if tasks[0].AreaID == nil {
		t.Error("AreaID should be set")
	}
}

func TestAddCannotSpecifyBothTodayAndPlanned(t *testing.T) {
	deps := setupCLI(t)

	cmd := cli.NewRootCmd(deps)
	cmd.SetArgs([]string{"add", "Task", "-T", "-P", "tomorrow"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when specifying both --today and --planned")
	}
}
