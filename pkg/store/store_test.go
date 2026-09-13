package store

import (
	"path/filepath"
	"testing"
	"time"
)

func setup(t *testing.T) {
	t.Helper()
	Dir = t.TempDir()
}

func TestSaveAndLoad(t *testing.T) {
	setup(t)

	e := Execution{
		ID:          "exec-1",
		CommandID:   "cmd-abc-123",
		NodeNames:   []string{"node1", "node2"},
		InstanceIDs: []string{"i-aaa", "i-bbb"},
		Command:     "ls -la",
		Timestamp:   time.Now().UTC().Truncate(time.Second),
		Status:      "Running",
	}

	if err := Save(e); err != nil {
		t.Fatal(err)
	}

	execs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(execs) != 1 {
		t.Fatalf("expected 1, got %d", len(execs))
	}
	if execs[0].ID != e.ID || execs[0].CommandID != e.CommandID {
		t.Errorf("data mismatch: %+v", execs[0])
	}
}

func TestUpdateExisting(t *testing.T) {
	setup(t)

	e := Execution{ID: "exec-1", Status: "Running"}
	if err := Save(e); err != nil {
		t.Fatal(err)
	}

	e.Status = "Success"
	if err := Save(e); err != nil {
		t.Fatal(err)
	}

	execs, _ := Load()
	if len(execs) != 1 {
		t.Fatalf("expected 1 after update, got %d", len(execs))
	}
	if execs[0].Status != "Success" {
		t.Errorf("status not updated: %s", execs[0].Status)
	}
}

func TestDelete(t *testing.T) {
	setup(t)

	for _, id := range []string{"exec-1", "exec-2", "exec-3"} {
		if err := Save(Execution{ID: id}); err != nil {
			t.Fatal(err)
		}
	}

	if err := Delete("exec-2"); err != nil {
		t.Fatal(err)
	}

	execs, _ := Load()
	if len(execs) != 2 {
		t.Fatalf("expected 2 after delete, got %d", len(execs))
	}
	for _, e := range execs {
		if e.ID == "exec-2" {
			t.Error("deleted entry still present")
		}
	}
}

func TestOutputSaveLoad(t *testing.T) {
	setup(t)

	content := "hello\nworld\n"
	if err := SaveOutput("exec-1", content); err != nil {
		t.Fatal(err)
	}

	got, err := LoadOutput("exec-1")
	if err != nil {
		t.Fatal(err)
	}
	if got != content {
		t.Errorf("got %q, want %q", got, content)
	}

	if OutputPath("exec-1") != filepath.Join(Dir, "exec-1.txt") {
		t.Errorf("unexpected output path: %s", OutputPath("exec-1"))
	}
}

func TestLoadEmpty(t *testing.T) {
	setup(t)
	execs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(execs) != 0 {
		t.Errorf("expected empty, got %d", len(execs))
	}
}
