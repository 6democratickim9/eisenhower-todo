package main

import (
	"net/http"
	"testing"
)

func TestJiraResponseProblemHTML(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
	}
	if got := jiraResponseProblem(resp, []byte("<html>login</html>")); got == "" {
		t.Fatal("expected HTML response to be rejected")
	}
}

func TestJiraResponseProblemRedirect(t *testing.T) {
	resp := &http.Response{
		StatusCode: 302,
		Header:     http.Header{"Location": []string{"https://jira.example/login.jsp"}},
	}
	if got := jiraResponseProblem(resp, nil); got == "" {
		t.Fatal("expected redirect response to be diagnosed")
	}
}

func TestNormalizeJiraBaseURL(t *testing.T) {
	got := normalizeJiraBaseURL("https://jira.example/jira/browse/ABC-123")
	if got != "https://jira.example/jira" {
		t.Fatalf("unexpected base url: %s", got)
	}
}
