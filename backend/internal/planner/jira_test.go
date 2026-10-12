package planner

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestJiraParseAndADFComment(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("auth=%q", got)
		}
		body := `{"fields":{"summary":"Deployment issue","status":{"name":"In Progress","statusCategory":{"key":"indeterminate"}},"assignee":{"displayName":"Minju"},"created":"2026-09-01T00:00:00.000+0000","updated":"2026-09-03T00:00:00.000+0000","comment":{"comments":[{"body":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"staging test done"}]}]}}]}}}`
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	jc := NewJiraClient(client)
	issue, err := jc.FetchIssue(JiraSecretConfig{BaseURL: "https://jira.example.com", AuthMode: "bearer", Token: "secret", APIVersion: "2"}, "ops-123")
	if err != nil {
		t.Fatal(err)
	}
	if issue.IssueKey != "OPS-123" || issue.Assignee != "Minju" || issue.LastComment != "staging test done" {
		t.Fatalf("issue=%+v", issue)
	}
	if MapWatchStatus(issue.RawStatus, issue.StatusCategory) != "In Review" {
		t.Fatalf("status map failed")
	}
}

func TestFlattenComment(t *testing.T) {
	v := map[string]any{"type": "doc", "content": []any{map[string]any{"type": "paragraph", "content": []any{map[string]any{"text": "hello"}}}}}
	if !strings.Contains(FlattenComment(v), "hello") {
		t.Fatal("missing hello")
	}
}
