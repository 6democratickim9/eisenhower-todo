package planner

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type JiraError struct{ Message string }

func (e JiraError) Error() string { return e.Message }

var issueRE = regexp.MustCompile(`(?i)(?:^|/)([A-Z][A-Z0-9_]+-\d+)(?:$|[/?#])`)
var issueExactRE = regexp.MustCompile(`(?i)^[A-Z][A-Z0-9_]+-\d+$`)

func NormalizeBaseURL(raw string) (string, error) {
	v := strings.TrimRight(strings.TrimSpace(raw), "/")
	if v == "" {
		return "", nil
	}
	if !strings.Contains(v, "://") {
		v = "https://" + v
	}
	u, err := url.Parse(v)
	if err != nil || !(u.Scheme == "http" || u.Scheme == "https") || u.Host == "" {
		return "", errors.New("Jira Base URL이 올바르지 않습니다.")
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String(), nil
}
func ExtractIssueKey(v string) string {
	raw := strings.ToUpper(strings.TrimSpace(v))
	if m := issueRE.FindStringSubmatch(raw); len(m) > 1 {
		return m[1]
	}
	if issueExactRE.MatchString(raw) {
		return raw
	}
	return ""
}
func BrowseURL(base, key string) string {
	if base == "" || key == "" {
		return ""
	}
	b, err := NormalizeBaseURL(base)
	if err != nil {
		return ""
	}
	return b + "/browse/" + url.PathEscape(key)
}
func MapWatchStatus(rawStatus, statusCategory string) string {
	c := strings.ToLower(strings.TrimSpace(statusCategory))
	r := strings.ToLower(strings.TrimSpace(rawStatus))
	if c == "done" || c == "complete" || c == "completed" || r == "done" || r == "closed" || r == "resolved" || r == "complete" || r == "completed" {
		return "Done"
	}
	if strings.Contains(r, "block") || strings.Contains(r, "hold") || strings.Contains(r, "imped") {
		return "Blocked"
	}
	if c == "indeterminate" || c == "in progress" || c == "in_progress" || strings.Contains(r, "progress") || strings.Contains(r, "review") || strings.Contains(r, "testing") || strings.Contains(r, "qa") || strings.Contains(r, "develop") {
		return "In Review"
	}
	return "Open"
}

func displayName(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	for _, k := range []string{"displayName", "name", "emailAddress", "accountId"} {
		if x, ok := m[k]; ok && x != nil {
			return fmt.Sprint(x)
		}
	}
	return ""
}
func FlattenComment(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case []any:
		var b strings.Builder
		for _, y := range x {
			b.WriteString(FlattenComment(y))
		}
		return strings.TrimSpace(b.String())
	case map[string]any:
		if x["type"] == "hardBreak" {
			return "\n"
		}
		if t, ok := x["text"].(string); ok {
			return t
		}
		var b strings.Builder
		if arr, ok := x["content"].([]any); ok {
			for _, y := range arr {
				b.WriteString(FlattenComment(y))
			}
		}
		s := strings.TrimSpace(b.String())
		switch x["type"] {
		case "paragraph", "heading", "listItem", "blockquote", "bulletList", "orderedList", "doc":
			if s != "" {
				return s + "\n"
			}
		}
		return s
	default:
		return ""
	}
}

type JiraClient struct{ HTTP *http.Client }

func NewJiraClient(client *http.Client) *JiraClient {
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &JiraClient{HTTP: client}
}
func (c *JiraClient) Configured(cfg JiraSecretConfig) bool {
	return strings.TrimSpace(cfg.BaseURL) != "" && strings.TrimSpace(cfg.Token) != ""
}
func (c *JiraClient) addAuth(req *http.Request, cfg JiraSecretConfig) {
	req.Header.Set("Accept", "application/json")
	if cfg.AuthMode == "basic" {
		raw := base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + cfg.Token))
		req.Header.Set("Authorization", "Basic "+raw)
	} else {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
}
func (c *JiraClient) Test(cfg JiraSecretConfig) (string, error) {
	if !c.Configured(cfg) {
		return "", JiraError{"Jira Base URL과 token을 먼저 설정하세요."}
	}
	base, err := NormalizeBaseURL(cfg.BaseURL)
	if err != nil {
		return "", err
	}
	req, _ := http.NewRequest(http.MethodGet, base+"/rest/api/"+cfg.APIVersion+"/myself", nil)
	c.addAuth(req, cfg)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", JiraError{err.Error()}
	}
	defer resp.Body.Close()
	if err = c.raiseProblem(resp); err != nil {
		return "", err
	}
	var v any
	if err = json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", JiraError{"Jira가 JSON이 아닌 응답을 반환했습니다."}
	}
	name := displayName(v)
	if name != "" {
		return "Jira 연결 성공: " + name, nil
	}
	return "Jira 연결 성공", nil
}
func (c *JiraClient) FetchIssue(cfg JiraSecretConfig, issueKey string) (JiraIssueData, error) {
	key := ExtractIssueKey(issueKey)
	if key == "" {
		return JiraIssueData{}, JiraError{"Jira issue key를 찾을 수 없습니다."}
	}
	if !c.Configured(cfg) {
		return JiraIssueData{}, JiraError{"Jira Base URL과 token을 먼저 설정하세요."}
	}
	base, err := NormalizeBaseURL(cfg.BaseURL)
	if err != nil {
		return JiraIssueData{}, err
	}
	u := base + "/rest/api/" + cfg.APIVersion + "/issue/" + url.PathEscape(key) + "?fields=" + url.QueryEscape("summary,status,assignee,comment,created,updated")
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	c.addAuth(req, cfg)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return JiraIssueData{}, JiraError{err.Error()}
	}
	defer resp.Body.Close()
	if err = c.raiseProblem(resp); err != nil {
		return JiraIssueData{}, err
	}
	var payload map[string]any
	if err = json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return JiraIssueData{}, JiraError{"Jira가 JSON이 아닌 응답을 반환했습니다."}
	}
	fields, _ := payload["fields"].(map[string]any)
	status, _ := fields["status"].(map[string]any)
	cat, _ := status["statusCategory"].(map[string]any)
	commentsObj, _ := fields["comment"].(map[string]any)
	var last string
	if arr, ok := commentsObj["comments"].([]any); ok && len(arr) > 0 {
		if m, ok := arr[len(arr)-1].(map[string]any); ok {
			last = strings.TrimSpace(FlattenComment(m["body"]))
		}
	}
	return JiraIssueData{IssueKey: key, Title: fmt.Sprint(fields["summary"]), RawStatus: fmt.Sprint(status["name"]), StatusCategory: firstNonEmpty(fmt.Sprint(cat["key"]), fmt.Sprint(cat["name"])), Assignee: displayName(fields["assignee"]), LastComment: last, CreatedAt: stringField(fields, "created"), UpdatedAt: stringField(fields, "updated"), URL: BrowseURL(base, key)}, nil
}
func stringField(m map[string]any, k string) string {
	if v, ok := m[k]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}
func firstNonEmpty(v ...string) string {
	for _, x := range v {
		if x != "" && x != "<nil>" {
			return x
		}
	}
	return ""
}
func (c *JiraClient) raiseProblem(resp *http.Response) error {
	status := resp.StatusCode
	ct := strings.ToLower(resp.Header.Get("content-type"))
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	resp.Body = io.NopCloser(strings.NewReader(string(b)))
	body := strings.ToLower(strings.TrimSpace(string(b)))
	if status >= 300 && status < 400 {
		return JiraError{fmt.Sprintf("Jira 인증 리다이렉트 (HTTP %d). Base URL/인증 방식을 확인하세요.", status)}
	}
	if status == 401 {
		return JiraError{"Jira 인증 실패 (HTTP 401). Token 또는 인증 방식을 확인하세요."}
	}
	if status == 403 {
		return JiraError{"Jira 접근 거부 (HTTP 403). Token 권한 또는 이슈 접근 권한을 확인하세요."}
	}
	if status == 404 {
		return JiraError{"Jira API/이슈를 찾을 수 없습니다 (HTTP 404). Base URL, Issue Key, API 버전을 확인하세요."}
	}
	if status < 200 || status >= 300 {
		return JiraError{fmt.Sprintf("Jira HTTP %d", status)}
	}
	if strings.Contains(ct, "text/html") || strings.HasPrefix(body, "<!doctype") || strings.HasPrefix(body, "<html") {
		return JiraError{"Jira가 JSON 대신 HTML을 반환했습니다. SSO/login redirect 가능성이 있습니다."}
	}
	return nil
}

func (c *JiraClient) AddComment(cfg JiraSecretConfig, issueKey, comment string) error {
	if !c.Configured(cfg) || issueKey == "" || comment == "" {
		return nil
	}
	base, err := NormalizeBaseURL(cfg.BaseURL)
	if err != nil {
		return err
	}
	u := base + "/rest/api/" + cfg.APIVersion + "/issue/" + url.PathEscape(issueKey) + "/comment"
	var body string
	if cfg.APIVersion == "3" {
		body = fmt.Sprintf(`{"body":{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":%q}]}]}}`, comment)
	} else {
		body = fmt.Sprintf(`{"body":%q}`, comment)
	}
	req, _ := http.NewRequest(http.MethodPost, u, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req, cfg)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return JiraError{err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return JiraError{fmt.Sprintf("Jira comment failed: HTTP %d", resp.StatusCode)}
	}
	return nil
}

type CreateIssueInput struct {
	ProjectKey  string `json:"projectKey"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	IssueType   string `json:"issueType"`
	LinkedIssue string `json:"linkedIssue"`
}

func (c *JiraClient) CreateIssue(cfg JiraSecretConfig, input CreateIssueInput) (string, string, error) {
	if !c.Configured(cfg) {
		return "", "", JiraError{"Jira Base URL과 token을 먼저 설정하세요."}
	}
	base, err := NormalizeBaseURL(cfg.BaseURL)
	if err != nil {
		return "", "", err
	}
	issueType := input.IssueType
	if issueType == "" {
		issueType = "Task"
	}
	var descField string
	if cfg.APIVersion == "3" {
		descField = fmt.Sprintf(`"description":{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":%q}]}]}`, input.Description)
	} else {
		descField = fmt.Sprintf(`"description":%q`, input.Description)
	}
	body := fmt.Sprintf(`{"fields":{"project":{"key":%q},"summary":%q,%s,"issuetype":{"name":%q}}}`,
		input.ProjectKey, input.Summary, descField, issueType)
	u := base + "/rest/api/" + cfg.APIVersion + "/issue"
	req, _ := http.NewRequest(http.MethodPost, u, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req, cfg)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", "", JiraError{err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", "", JiraError{fmt.Sprintf("Jira create issue failed: HTTP %d — %s", resp.StatusCode, string(b))}
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", JiraError{"Failed to parse Jira response"}
	}
	key, _ := result["key"].(string)
	issueURL := BrowseURL(base, key)

	// Link to related issue if specified
	if input.LinkedIssue != "" && key != "" {
		linkBody := fmt.Sprintf(`{"type":{"name":"Relates"},"inwardIssue":{"key":%q},"outwardIssue":{"key":%q}}`, key, input.LinkedIssue)
		linkURL := base + "/rest/api/" + cfg.APIVersion + "/issueLink"
		linkReq, _ := http.NewRequest(http.MethodPost, linkURL, strings.NewReader(linkBody))
		linkReq.Header.Set("Content-Type", "application/json")
		c.addAuth(linkReq, cfg)
		linkResp, err := c.HTTP.Do(linkReq)
		if err == nil {
			linkResp.Body.Close()
		}
	}
	return key, issueURL, nil
}
