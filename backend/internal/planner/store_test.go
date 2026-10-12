package planner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func testPaths(t *testing.T) ResolvedPaths {
	t.Helper()
	d := t.TempDir()
	data := filepath.Join(d, "data", "eisenhower.json")
	if err := os.MkdirAll(filepath.Dir(data), 0755); err != nil {
		t.Fatal(err)
	}
	return ResolvedPaths{ConfigDir: d, ConfigPath: filepath.Join(d, "config.json"), JiraPath: filepath.Join(d, "jira-settings.json"), DataPath: data, RunningPath: filepath.Join(d, "running.json")}
}
func intp(v int) *int    { return &v }
func boolp(v bool) *bool { return &v }

func TestStoreAPAndDaily(t *testing.T) {
	p := testPaths(t)
	s, err := NewStore(p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first, err := s.CreateAP(APDraft{Title: "Deploy fix", Details: "check", Importance: intp(88), BaseUrgency: intp(70), Tags: []string{"DevOps", "devops", " SRE "}, Jira: &JiraLink{IssueKey: "ops-123"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Tags) != 2 || first.Jira == nil || first.Jira.IssueKey != "OPS-123" || first.Jira.SyncError == "" {
		t.Fatalf("ap=%+v", first)
	}
	second, _ := s.CreateAP(APDraft{Title: "Docs", Importance: intp(20), BaseUrgency: intp(20)})
	mode := QueueManual
	if err = s.ReorderQueue([]string{second.ID, first.ID}, &mode); err != nil {
		t.Fatal(err)
	}
	st := s.State()
	var priority string
	for _, ap := range st.ActionPoints {
		if ap.QueueOrder == 0 {
			priority = ap.ID
		}
	}
	if priority != second.ID {
		t.Fatalf("manual queue order not respected: %s", priority)
	}
	if err = s.CompleteAP(first.ID); err != nil {
		t.Fatal(err)
	}
	st = s.State()
	if len(st.Archive) == 0 || len(st.JiraWatch) == 0 || st.JiraWatch[0].IssueKey != "OPS-123" {
		t.Fatalf("complete state=%+v", st)
	}
	if err = s.RestoreAP(first.ID); err != nil {
		t.Fatal(err)
	}
	dc, err := s.CreateDaily(DailyCheckpointDraft{Title: "Check batch", Enabled: boolp(true)})
	if err != nil {
		t.Fatal(err)
	}
	dc, err = s.ToggleDaily(dc.ID, "2026-09-03")
	if err != nil || len(dc.CompletedDates) != 1 {
		t.Fatalf("toggle=%+v err=%v", dc, err)
	}
	off := map[string]any{"enabled": false}
	dc, err = s.UpdateDaily(dc.ID, off)
	if err != nil || dc.Enabled {
		t.Fatalf("disable=%+v err=%v", dc, err)
	}
	if _, err = s.ToggleDaily(dc.ID, "2026-09-03"); err == nil {
		t.Fatal("disabled toggle should fail")
	}
	if err = s.SaveQuickNote("deploy 확인 후 Jira 코멘트 남기기"); err != nil {
		t.Fatal(err)
	}
	if got := s.State().QuickNote; got != "deploy 확인 후 Jira 코멘트 남기기" {
		t.Fatalf("quick note=%q", got)
	}
}

func TestLegacyMigrationAndLock(t *testing.T) {
	p := testPaths(t)
	legacy := map[string]any{"version": 5, "queueMode": "manual", "actionPoints": []any{map[string]any{"id": "ap-old", "title": "Old AP", "details": "legacy", "importance": 80, "baseUrgency": 20, "createdAt": "2026-08-01T00:00:00Z", "updatedAt": "2026-08-01T00:00:00Z", "queueOrder": 0, "tags": []string{"Ops"}, "jira": map[string]any{"issueKey": "OPS-1", "status": "In Progress"}}}, "archive": []any{}, "dailyTasks": []any{map[string]any{"id": "daily-1", "title": "Batch check", "createdAt": "2026-08-01T00:00:00Z", "lastCheckedDate": "2026-09-02"}}, "jiraTickets": []any{map[string]any{"apTitle": "Old AP", "trackedAt": "2026-08-01T00:00:00Z", "jira": map[string]any{"issueKey": "OPS-1", "summary": "Old AP jira", "status": "In Progress", "statusCategory": "indeterminate"}}}}
	b, _ := json.Marshal(legacy)
	if err := os.WriteFile(p.DataPath, b, 0600); err != nil {
		t.Fatal(err)
	}
	first, err := NewStore(p)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	st := first.State()
	if st.ActionPoints[0].ID != "ap-old" || st.DailyCheckpoints[0].CompletedDates[0] != "2026-09-02" || st.JiraWatch[0].Status != "In Review" {
		t.Fatalf("migration=%+v", st)
	}
	if _, err = os.Stat(p.DataPath + ".pre-v6.bak"); err != nil {
		t.Fatalf("missing migration backup: %v", err)
	}
	if _, err = NewStore(p); err == nil {
		t.Fatal("second store should be locked")
	}
}

func TestSettingsPathSwitch(t *testing.T) {
	p := testPaths(t)
	s, err := NewStore(p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, _ = s.CreateAP(APDraft{Title: "Persist me", Importance: intp(80), BaseUrgency: intp(30)})
	target := filepath.Join(t.TempDir(), "custom-data")
	saved, err := s.SaveSettings(PlannerSettings{DisplayName: "Minju", Timezone: "local", DataPath: target, AutoSyncMinutes: 15})
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(target, "eisenhower.json")
	if saved.DataPath != expected {
		t.Fatalf("path=%q expected=%q", saved.DataPath, expected)
	}
	if _, err = os.Stat(expected); err != nil {
		t.Fatal(err)
	}
	if len(s.State().ActionPoints) != 1 {
		t.Fatal("data not carried")
	}
}
