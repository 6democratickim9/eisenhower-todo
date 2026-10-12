package planner

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type TeamMember struct {
	Name         string          `json:"name"`
	ActionPoints []ActionPoint   `json:"actionPoints"`
	Archive      []ActionPoint   `json:"archive,omitempty"`
	JiraWatch    []JiraWatchItem `json:"jiraWatch,omitempty"`
}

type CodeReviewResponse struct {
	RequestID   string `json:"requestId"`
	Reviewer    string `json:"reviewer"`
	Status      string `json:"status"` // done | not_mine
	RespondedAt string `json:"respondedAt"`
}

type CodeReviewRequest struct {
	ID                string               `json:"id"`
	URL               string               `json:"url"`
	Title             string               `json:"title"`
	Description       string               `json:"description,omitempty"`
	Validation        string               `json:"validation"`
	ApprovalMode      string               `json:"approvalMode"` // each | single
	Owner             string               `json:"owner"`
	CreatedAt         string               `json:"createdAt"`
	TargetReviewCount int                  `json:"targetReviewCount"`
	Responses         []CodeReviewResponse `json:"responses,omitempty"`
	DoneCount         int                  `json:"doneCount"`
	SkippedCount      int                  `json:"skippedCount"`
	Complete          bool                 `json:"complete"`
	MyResponse        string               `json:"myResponse,omitempty"`
	MdcRcc            string               `json:"mdcRcc,omitempty"`
	MdcRgw            string               `json:"mdcRgw,omitempty"`
	MdcCommon         string               `json:"mdcCommon,omitempty"`
	ArchivedAt        string               `json:"archivedAt,omitempty"`
}

type TeamState struct {
	Enabled       bool                `json:"enabled"`
	Root          string              `json:"root"`
	CurrentUser   string              `json:"currentUser"`
	Members       []TeamMember        `json:"members"`
	CodeReviews   []CodeReviewRequest `json:"codeReviews"`
	LastScannedAt string              `json:"lastScannedAt"`
}

type CodeReviewDraft struct {
	URL          string `json:"url"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Validation   string `json:"validation"`
	ApprovalMode string `json:"approvalMode"` // each | single
}

func inferUserDirAndTeamRoot(dataPath, configuredRoot string) (userDir, root, username string) {
	clean := filepath.Clean(dataPath)
	dir := filepath.Dir(clean)
	if strings.EqualFold(filepath.Base(dir), "data") {
		userDir = filepath.Dir(dir)
	} else {
		userDir = dir
	}
	username = filepath.Base(userDir)
	if strings.TrimSpace(configuredRoot) != "" {
		root = filepath.Clean(os.ExpandEnv(configuredRoot))
		if !filepath.IsAbs(root) {
			if abs, err := filepath.Abs(root); err == nil {
				root = abs
			}
		}
	} else {
		root = filepath.Dir(userDir)
	}
	return
}

func readPlannerDataReadOnly(path string) (PlannerData, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return PlannerData{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return PlannerData{}, err
	}
	d, _, err := migrateRaw(raw)
	return d, err
}

func findMemberDataFile(userDir string) string {
	candidates := []string{
		filepath.Join(userDir, "data", "eisenhower.json"),
		filepath.Join(userDir, "eisenhower.json"),
		filepath.Join(userDir, "data", "planner.json"),
		filepath.Join(userDir, "planner.json"),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func (s *Store) TeamState() TeamState {
	s.mu.RLock()
	dataPath := s.Paths.DataPath
	teamRoot := s.Settings.TeamShareRoot
	s.mu.RUnlock()
	userDir, root, currentUser := inferUserDirAndTeamRoot(dataPath, teamRoot)
	result := TeamState{Enabled: false, Root: root, CurrentUser: currentUser, Members: []TeamMember{}, CodeReviews: []CodeReviewRequest{}, LastScannedAt: NowISO()}
	if currentUser == "" || root == "" {
		return result
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return result
	}
	result.Enabled = true
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		memberDir := filepath.Join(root, entry.Name())
		p := findMemberDataFile(memberDir)
		if p == "" {
			continue
		}
		d, err := readPlannerDataReadOnly(p)
		if err != nil {
			continue
		}
		result.Members = append(result.Members, TeamMember{Name: entry.Name(), ActionPoints: d.ActionPoints, Archive: d.Archive, JiraWatch: d.JiraWatch})
	}
	sort.Slice(result.Members, func(i, j int) bool {
		return strings.ToLower(result.Members[i].Name) < strings.ToLower(result.Members[j].Name)
	})
	result.CodeReviews = scanCodeReviews(root, currentUser)
	_ = userDir
	return result
}

func scanCodeReviews(root, currentUser string) []CodeReviewRequest {
	requests := []CodeReviewRequest{}
	entries, err := os.ReadDir(root)
	if err != nil {
		return requests
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		pattern := filepath.Join(root, entry.Name(), "code-review", "requests", "*.json")
		files, _ := filepath.Glob(pattern)
		for _, p := range files {
			b, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			var req CodeReviewRequest
			if json.Unmarshal(b, &req) != nil || req.ID == "" {
				continue
			}
			if req.Owner == "" {
				req.Owner = entry.Name()
			}
			if req.TargetReviewCount < 1 {
				req.TargetReviewCount = 2
			}
			req.Responses = scanResponses(root, req.ID)
			req.DoneCount = 0
			req.SkippedCount = 0
			req.MyResponse = ""
			for _, resp := range req.Responses {
				if strings.EqualFold(resp.Status, "done") {
					req.DoneCount++
				}
				if strings.EqualFold(resp.Status, "not_mine") {
					req.SkippedCount++
				}
				if strings.EqualFold(resp.Reviewer, currentUser) {
					req.MyResponse = resp.Status
				}
			}
			if req.ApprovalMode == "single" {
				req.Complete = req.DoneCount >= 1
			} else {
				req.Complete = req.DoneCount >= req.TargetReviewCount
			}
			requests = append(requests, req)
		}
	}
	sort.Slice(requests, func(i, j int) bool { return requests[i].CreatedAt > requests[j].CreatedAt })
	return requests
}

func scanResponses(root, requestID string) []CodeReviewResponse {
	out := []CodeReviewResponse{}
	entries, err := os.ReadDir(root)
	if err != nil {
		return out
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		p := filepath.Join(root, entry.Name(), "code-review", "responses", requestID+".json")
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var resp CodeReviewResponse
		if json.Unmarshal(b, &resp) == nil && resp.RequestID == requestID {
			out = append(out, resp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RespondedAt < out[j].RespondedAt })
	return out
}

func validateReviewURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("코드리뷰 URL은 http/https 전체 주소여야 합니다.")
	}
	return u.String(), nil
}

func (s *Store) CreateCodeReview(d CodeReviewDraft) (CodeReviewRequest, error) {
	s.mu.RLock()
	dataPath := s.Paths.DataPath
	teamRoot := s.Settings.TeamShareRoot
	s.mu.RUnlock()
	userDir, root, owner := inferUserDirAndTeamRoot(dataPath, teamRoot)
	if _, err := os.Stat(root); err != nil {
		return CodeReviewRequest{}, fmt.Errorf("team SharePoint root를 읽을 수 없습니다: %w", err)
	}
	reviewURL, err := validateReviewURL(d.URL)
	if err != nil {
		return CodeReviewRequest{}, err
	}
	title := strings.TrimSpace(d.Title)
	if title == "" {
		return CodeReviewRequest{}, errors.New("리뷰 제목을 입력하세요.")
	}
	validation := strings.TrimSpace(d.Validation)
	if validation == "" {
		return CodeReviewRequest{}, errors.New("검증 내용을 입력하세요.")
	}
	description := strings.TrimSpace(d.Description)
	approvalMode := strings.TrimSpace(d.ApprovalMode)
	if approvalMode != "each" && approvalMode != "single" {
		approvalMode = "each"
	}
	req := CodeReviewRequest{ID: newID("cr"), URL: reviewURL, Title: title, Description: description, Validation: validation, ApprovalMode: approvalMode, Owner: owner, CreatedAt: NowISO(), TargetReviewCount: 2, Responses: []CodeReviewResponse{}}
	dir := filepath.Join(userDir, "code-review", "requests")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return req, err
	}
	if err := atomicWriteJSON(filepath.Join(dir, req.ID+".json"), req, false); err != nil {
		return req, err
	}
	return req, nil
}

func (s *Store) RespondCodeReview(requestID, status string) error {
	status = strings.TrimSpace(strings.ToLower(status))
	if status != "done" && status != "not_mine" {
		return errors.New("status must be done or not_mine")
	}
	s.mu.RLock()
	dataPath := s.Paths.DataPath
	teamRoot := s.Settings.TeamShareRoot
	s.mu.RUnlock()
	userDir, root, reviewer := inferUserDirAndTeamRoot(dataPath, teamRoot)
	requests := scanCodeReviews(root, reviewer)
	var found *CodeReviewRequest
	for i := range requests {
		if requests[i].ID == requestID {
			found = &requests[i]
			break
		}
	}
	if found == nil {
		return errors.New("code review request not found")
	}
	if strings.EqualFold(found.Owner, reviewer) {
		return errors.New("요청자는 자신의 코드리뷰 요청을 완료 처리할 수 없습니다.")
	}
	resp := CodeReviewResponse{RequestID: requestID, Reviewer: reviewer, Status: status, RespondedAt: time.Now().UTC().Format(time.RFC3339)}
	dir := filepath.Join(userDir, "code-review", "responses")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return atomicWriteJSON(filepath.Join(dir, requestID+".json"), resp, true)
}


func (s *Store) UpdateCodeReview(requestID string, patch map[string]any) (CodeReviewRequest, error) {
	s.mu.RLock()
	dataPath := s.Paths.DataPath
	teamRoot := s.Settings.TeamShareRoot
	s.mu.RUnlock()
	userDir, _, currentUser := inferUserDirAndTeamRoot(dataPath, teamRoot)
	p := filepath.Join(userDir, "code-review", "requests", requestID+".json")
	b, err := os.ReadFile(p)
	if err != nil {
		return CodeReviewRequest{}, fmt.Errorf("code review request not found: %w", err)
	}
	var req CodeReviewRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return CodeReviewRequest{}, err
	}
	if !strings.EqualFold(req.Owner, currentUser) {
		return CodeReviewRequest{}, errors.New("본인이 생성한 코드리뷰만 수정할 수 있습니다.")
	}
	if v, ok := patch["url"].(string); ok {
		u, err := validateReviewURL(v)
		if err != nil {
			return CodeReviewRequest{}, err
		}
		req.URL = u
	}
	if v, ok := patch["title"].(string); ok {
		v = strings.TrimSpace(v)
		if v == "" {
			return CodeReviewRequest{}, errors.New("리뷰 제목을 입력하세요.")
		}
		req.Title = v
	}
	if v, ok := patch["description"].(string); ok {
		req.Description = strings.TrimSpace(v)
	}
	if v, ok := patch["validation"].(string); ok {
		v = strings.TrimSpace(v)
		if v == "" {
			return CodeReviewRequest{}, errors.New("검증 내용을 입력하세요.")
		}
		req.Validation = v
	}
	if v, ok := patch["approvalMode"].(string); ok {
		if v == "each" || v == "single" {
			req.ApprovalMode = v
		}
	}
	if err := atomicWriteJSON(p, req, true); err != nil {
		return CodeReviewRequest{}, err
	}
	return req, nil
}
type MdcVersions struct {
	MdcRcc    string `json:"mdcRcc"`
	MdcRgw    string `json:"mdcRgw"`
	MdcCommon string `json:"mdcCommon"`
}

func (s *Store) SetMdcVersions(requestID string, versions MdcVersions) error {
	s.mu.RLock()
	dataPath := s.Paths.DataPath
	teamRoot := s.Settings.TeamShareRoot
	s.mu.RUnlock()
	userDir, root, currentUser := inferUserDirAndTeamRoot(dataPath, teamRoot)
	_ = root
	p := filepath.Join(userDir, "code-review", "requests", requestID+".json")
	b, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("code review request not found: %w", err)
	}
	var req CodeReviewRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	if !strings.EqualFold(req.Owner, currentUser) {
		return errors.New("MDC 버전은 요청자만 기입할 수 있습니다.")
	}
	req.MdcRcc = strings.TrimSpace(versions.MdcRcc)
	req.MdcRgw = strings.TrimSpace(versions.MdcRgw)
	req.MdcCommon = strings.TrimSpace(versions.MdcCommon)
	return atomicWriteJSON(p, req, true)
}

func (s *Store) ArchiveCodeReview(requestID string) error {
	s.mu.RLock()
	dataPath := s.Paths.DataPath
	teamRoot := s.Settings.TeamShareRoot
	s.mu.RUnlock()
	_, root, currentUser := inferUserDirAndTeamRoot(dataPath, teamRoot)
	// 요청 파일 찾기: 모든 멤버 디렉토리에서 탐색
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("team root 읽기 실패: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		p := filepath.Join(root, entry.Name(), "code-review", "requests", requestID+".json")
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var req CodeReviewRequest
		if json.Unmarshal(b, &req) != nil || req.ID != requestID {
			continue
		}
		// 소유자이거나 complete된 리뷰에 done 응답한 리뷰어만 아카이브 가능
		isOwner := strings.EqualFold(req.Owner, currentUser)
		if !isOwner {
			// 리뷰어인지 확인 — 해당 사용자의 response 파일 존재 여부
			respPath := filepath.Join(root, currentUser, "code-review", "responses", requestID+".json")
			rb, rerr := os.ReadFile(respPath)
			if rerr != nil {
				return errors.New("본인이 생성했거나 리뷰 참여한 코드리뷰만 아카이브할 수 있습니다.")
			}
			var resp CodeReviewResponse
			if json.Unmarshal(rb, &resp) != nil || resp.Status != "done" {
				return errors.New("리뷰 완료(done)한 코드리뷰만 아카이브할 수 있습니다.")
			}
		}
		req.ArchivedAt = NowISO()
		return atomicWriteJSON(p, req, true)
	}
	return errors.New("code review request not found")
}

// --- Daily Scrum ---

type ScrumDashboardEntry struct {
	Name           string        `json:"name"`
	CompletedToday []ActionPoint `json:"completedToday"`
	ActiveJira     []JiraLink    `json:"activeJira"`
}

type ScrumDashboard struct {
	Date        string                `json:"date"`
	Entries     []ScrumDashboardEntry `json:"entries"`
	Note        string                `json:"note"`
	CodeReviews []CodeReviewRequest   `json:"codeReviews"`
}

func (s *Store) ScrumDashboard(dateStr string) ScrumDashboard {
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	s.mu.RLock()
	dataPath := s.Paths.DataPath
	teamRoot := s.Settings.TeamShareRoot
	s.mu.RUnlock()
	_, root, _ := inferUserDirAndTeamRoot(dataPath, teamRoot)
	dash := ScrumDashboard{Date: dateStr, Entries: []ScrumDashboardEntry{}, Note: readScrumNote(root, dateStr), CodeReviews: []CodeReviewRequest{}}
	if root == "" {
		return dash
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return dash
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		p := findMemberDataFile(filepath.Join(root, entry.Name()))
		if p == "" {
			continue
		}
		d, err := readPlannerDataReadOnly(p)
		if err != nil {
			continue
		}
		var completed []ActionPoint
		for _, ap := range d.Archive {
			if ap.CompletedAt != "" && strings.HasPrefix(ap.CompletedAt, dateStr) {
				completed = append(completed, ap)
			}
		}
		var activeJira []JiraLink
		// JiraWatch에 등록된 티켓만 표시
		now := time.Now()
		for _, jw := range d.JiraWatch {
			if jw.IssueKey != "" && !strings.EqualFold(jw.Status, "Done") {
				staleDays := 0
				if jw.UpdatedAt != "" {
					if t, err := time.Parse(time.RFC3339, jw.UpdatedAt); err == nil {
						days := int(now.Sub(t).Hours() / 24)
						if days >= 5 {
							staleDays = days
						}
					}
				}
				activeJira = append(activeJira, JiraLink{
					IssueKey:       jw.IssueKey,
					URL:            jw.URL,
					Summary:        jw.Title,
					Status:         jw.RawStatus,
					StatusCategory: jw.StatusCategory,
					Assignee:       jw.Assignee,
					Paused:         jw.Paused,
					StaleDays:      staleDays,
				})
			}
		}
		if completed == nil {
			completed = []ActionPoint{}
		}
		if activeJira == nil {
			activeJira = []JiraLink{}
		}
		dash.Entries = append(dash.Entries, ScrumDashboardEntry{Name: entry.Name(), CompletedToday: completed, ActiveJira: activeJira})
	}
	sort.Slice(dash.Entries, func(i, j int) bool {
		return strings.ToLower(dash.Entries[i].Name) < strings.ToLower(dash.Entries[j].Name)
	})
	// complete된 코드리뷰를 대시보드에 포함
	_, currentUser := func() (string, string) {
		ud, _, u := inferUserDirAndTeamRoot(dataPath, teamRoot)
		_ = ud
		return root, u
	}()
	allReviews := scanCodeReviews(root, currentUser)
	for _, r := range allReviews {
		if !r.Complete {
			continue
		}
		// complete 시점 = 마지막 done 응답 날짜
		completedDate := ""
		for _, resp := range r.Responses {
			if strings.EqualFold(resp.Status, "done") && resp.RespondedAt > completedDate {
				completedDate = resp.RespondedAt
			}
		}
		// 해당 날짜에 complete된 리뷰만 표시
		if !strings.HasPrefix(completedDate, dateStr) {
			continue
		}
		dash.CodeReviews = append(dash.CodeReviews, r)
	}
	return dash
}

func scrumNotePath(root, dateStr string) string {
	return filepath.Join(root, "_scrum", dateStr+".txt")
}

func readScrumNote(root, dateStr string) string {
	if root == "" {
		return ""
	}
	b, err := os.ReadFile(scrumNotePath(root, dateStr))
	if err != nil {
		return ""
	}
	return string(b)
}

func (s *Store) SaveScrumNote(dateStr, note string) error {
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	s.mu.RLock()
	dataPath := s.Paths.DataPath
	teamRoot := s.Settings.TeamShareRoot
	s.mu.RUnlock()
	_, root, _ := inferUserDirAndTeamRoot(dataPath, teamRoot)
	if root == "" {
		return errors.New("team root not configured")
	}
	p := scrumNotePath(root, dateStr)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(note), 0o644)
}
