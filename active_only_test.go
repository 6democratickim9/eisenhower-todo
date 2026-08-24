package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestActiveOnlyMigrationClearsArchive(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "planner.json")
	configPath := filepath.Join(dir, "config.json")
	jiraPath := filepath.Join(dir, "jira.json")
	old := PlannerData{Version: 3, QueueMode: "manual", ActionPoints: []ActionPoint{{ID: "active-1", Title: "keep", CreatedAt: "2026-08-20T00:00:00Z"}}, Archive: []ActionPoint{{ID: "done-1", Title: "remove", CompletedAt: "2026-08-21T00:00:00Z"}}}
	raw, _ := json.Marshal(old)
	if err := os.WriteFile(dataPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := newStore(dataPath, jiraPath, configPath, false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	if len(s.data.ActionPoints) != 1 {
		t.Fatalf("active AP lost: %#v", s.data.ActionPoints)
	}
	if len(s.data.Archive) != 0 {
		t.Fatalf("archive not cleared: %#v", s.data.Archive)
	}
	if s.data.Version != 4 {
		t.Fatalf("expected version 4, got %d", s.data.Version)
	}
}
