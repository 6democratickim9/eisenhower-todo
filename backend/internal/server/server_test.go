package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"eisenday/internal/planner"
)

func newTestStore(t *testing.T) *planner.Store {
	t.Helper()
	d := t.TempDir()
	data := filepath.Join(d, "data", "eisenhower.json")
	if err := os.MkdirAll(filepath.Dir(data), 0o755); err != nil {
		t.Fatal(err)
	}
	p := planner.ResolvedPaths{
		ConfigDir:   d,
		ConfigPath:  filepath.Join(d, "config.json"),
		JiraPath:    filepath.Join(d, "jira-settings.json"),
		DataPath:    data,
		RunningPath: filepath.Join(d, "running.json"),
	}
	s, err := planner.NewStore(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestQuickNotePutRoute(t *testing.T) {
	s := newTestStore(t)
	h := NewHandler(s)

	req := httptest.NewRequest(http.MethodPut, "/api/note", strings.NewReader(`{"note":"hello note"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("PUT /api/note status=%d body=%q", rr.Code, rr.Body.String())
	}
	if got := s.State().QuickNote; got != "hello note" {
		t.Fatalf("quick note=%q", got)
	}
}
