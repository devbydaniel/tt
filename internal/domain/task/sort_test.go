package task

import (
	"testing"
)

func TestParseSort(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []SortOption
		wantErr bool
	}{
		{
			name:  "empty string returns default",
			input: "",
			want:  []SortOption{{Field: SortByID, Direction: SortAsc}},
		},
		{
			name:  "single field with default direction (text)",
			input: "title",
			want:  []SortOption{{Field: SortByTitle, Direction: SortAsc}},
		},
		{
			name:  "single field with default direction (date)",
			input: "created",
			want:  []SortOption{{Field: SortByCreated, Direction: SortDesc}},
		},
		{
			name:  "single field with explicit asc",
			input: "title:asc",
			want:  []SortOption{{Field: SortByTitle, Direction: SortAsc}},
		},
		{
			name:  "single field with explicit desc",
			input: "title:desc",
			want:  []SortOption{{Field: SortByTitle, Direction: SortDesc}},
		},
		{
			name:  "multiple fields",
			input: "due,title",
			want: []SortOption{
				{Field: SortByDue, Direction: SortDesc},
				{Field: SortByTitle, Direction: SortAsc},
			},
		},
		{
			name:  "multiple fields with directions",
			input: "due:asc,title:desc",
			want: []SortOption{
				{Field: SortByDue, Direction: SortAsc},
				{Field: SortByTitle, Direction: SortDesc},
			},
		},
		{
			name:  "all valid fields",
			input: "id,title,planned,due,created,project,area",
			want: []SortOption{
				{Field: SortByID, Direction: SortAsc},
				{Field: SortByTitle, Direction: SortAsc},
				{Field: SortByPlanned, Direction: SortDesc},
				{Field: SortByDue, Direction: SortDesc},
				{Field: SortByCreated, Direction: SortDesc},
				{Field: SortByProject, Direction: SortAsc},
				{Field: SortByArea, Direction: SortAsc},
			},
		},
		{
			name:  "case insensitive field",
			input: "TITLE",
			want:  []SortOption{{Field: SortByTitle, Direction: SortAsc}},
		},
		{
			name:  "case insensitive direction",
			input: "title:DESC",
			want:  []SortOption{{Field: SortByTitle, Direction: SortDesc}},
		},
		{
			name:  "whitespace handling",
			input: " due , title:asc ",
			want: []SortOption{
				{Field: SortByDue, Direction: SortDesc},
				{Field: SortByTitle, Direction: SortAsc},
			},
		},
		{
			name:    "invalid field",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "invalid direction",
			input:   "title:invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("ParseSort() got %d options, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i].Field != tt.want[i].Field {
					t.Errorf("ParseSort()[%d].Field = %v, want %v", i, got[i].Field, tt.want[i].Field)
				}
				if got[i].Direction != tt.want[i].Direction {
					t.Errorf("ParseSort()[%d].Direction = %v, want %v", i, got[i].Direction, tt.want[i].Direction)
				}
			}
		})
	}
}

func TestValidSortFields(t *testing.T) {
	fields := ValidSortFields()
	expected := []string{"id", "title", "planned", "due", "created", "project", "area"}

	if len(fields) != len(expected) {
		t.Errorf("ValidSortFields() returned %d fields, want %d", len(fields), len(expected))
	}

	for i, f := range expected {
		if fields[i] != f {
			t.Errorf("ValidSortFields()[%d] = %q, want %q", i, fields[i], f)
		}
	}
}

func TestDefaultSort(t *testing.T) {
	got := DefaultSort()
	if len(got) != 1 {
		t.Fatalf("DefaultSort() returned %d options, want 1", len(got))
	}
	if got[0].Field != SortByID {
		t.Errorf("DefaultSort()[0].Field = %v, want %v", got[0].Field, SortByID)
	}
	if got[0].Direction != SortAsc {
		t.Errorf("DefaultSort()[0].Direction = %v, want %v", got[0].Direction, SortAsc)
	}
}
