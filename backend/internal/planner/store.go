package planner

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
)

type Store struct {
	mu         sync.RWMutex
	Paths      ResolvedPaths
	lock       *FileLock
	JiraClient *JiraClient
	Settings   PlannerSettings
	JiraConfig JiraSecretConfig
	Data       PlannerData
}

func NewStore(paths ResolvedPaths) (*Store, error) {
	return NewStoreWithJira(paths, NewJiraClient(nil))
}
func NewStoreWithJira(paths ResolvedPaths, jira *JiraClient) (*Store, error) {
	l := NewFileLock(paths.DataPath)
	if err := l.Acquire(); err != nil {
		return nil, err
	}
	s := &Store{Paths: paths, lock: l, JiraClient: jira}
	var err error
	if s.Settings, err = s.loadSettings(); err != nil {
		s.Close()
		return nil, err
	}
	if s.JiraConfig, err = s.loadJiraConfig(); err != nil {
		s.Close()
		return nil, err
	}
	if s.JiraConfig.BaseURL == "" && s.Settings.JiraBaseURL != "" {
		s.JiraConfig.BaseURL = s.Settings.JiraBaseURL
	}
	if s.Data, err = s.loadData(); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() {
	if s.lock != nil {
		s.lock.Release()
	}
}

func (s *Store) loadSettings() (PlannerSettings, error) {
	raw := safeReadMap(s.Paths.ConfigPath)
	name := DefaultDisplayName()
	timezone := "local"
	jira := ""
	mins := 15
	teamShareRoot := ""
	if raw != nil {
		if v, ok := raw["displayName"].(string); ok && strings.TrimSpace(v) != "" {
			name = v
		}
		if v, ok := raw["timezone"].(string); ok && strings.TrimSpace(v) != "" {
			timezone = v
		}
		if v, ok := raw["jiraBaseUrl"].(string); ok {
			jira = v
		}
		if v, ok := raw["autoSyncMinutes"].(float64); ok && v >= 1 && v <= 1440 {
			mins = int(v)
		}
		if v, ok := raw["teamShareRoot"].(string); ok {
			teamShareRoot = strings.TrimSpace(v)
		}
	}
	commentOnComplete := false
	commentIncludeDetail := false
	commentOnCheckpoint := false
	if raw != nil {
		if v, ok := raw["jiraCommentOnComplete"].(bool); ok {
			commentOnComplete = v
		}
		if v, ok := raw["jiraCommentIncludeDetail"].(bool); ok {
			commentIncludeDetail = v
		}
		if v, ok := raw["jiraCommentOnCheckpoint"].(bool); ok {
			commentOnCheckpoint = v
		}
	}
	return PlannerSettings{DisplayName: name, Timezone: timezone, JiraBaseURL: jira, DataPath: s.Paths.DataPath, AutoSyncMinutes: mins, TeamShareRoot: teamShareRoot, JiraCommentOnComplete: commentOnComplete, JiraCommentIncludeDetail: commentIncludeDetail, JiraCommentOnCheckpoint: commentOnCheckpoint}, nil
}
func (s *Store) saveSettingsLocked() error {
	s.Settings.DataPath = s.Paths.DataPath
	return atomicWriteJSON(s.Paths.ConfigPath, s.Settings, true)
}
func (s *Store) loadJiraConfig() (JiraSecretConfig, error) {
	raw := safeReadMap(s.Paths.JiraPath)
	cfg := JiraSecretConfig{AuthMode: "bearer", APIVersion: "2"}
	if raw != nil {
		cfg.BaseURL = firstString(raw, "baseUrl", "base_url")
		cfg.Token = firstString(raw, "token")
		cfg.Username = firstString(raw, "username")
		if v := strings.ToLower(firstString(raw, "authMode")); v == "basic" {
			cfg.AuthMode = "basic"
		}
		if v := firstString(raw, "apiVersion"); v == "3" {
			cfg.APIVersion = "3"
		}
	}
	if v := os.Getenv("EISEN_JIRA_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("EISEN_JIRA_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("EISEN_JIRA_USERNAME"); v != "" {
		cfg.Username = v
	}
	if v := strings.ToLower(os.Getenv("EISEN_JIRA_AUTH_MODE")); v == "basic" || v == "bearer" {
		cfg.AuthMode = v
	}
	if cfg.BaseURL != "" {
		n, err := NormalizeBaseURL(cfg.BaseURL)
		if err != nil {
			return cfg, err
		}
		cfg.BaseURL = n
	}
	return cfg, nil
}
func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok {
			return v
		}
	}
	return ""
}
func (s *Store) saveJiraConfigLocked() error {
	return atomicWriteJSON(s.Paths.JiraPath, s.JiraConfig, true)
}
func emptyData() PlannerData {
	return PlannerData{Version: 6, ActionPoints: []ActionPoint{}, Archive: []ActionPoint{}, DailyCheckpoints: []DailyCheckpoint{}, JiraWatch: []JiraWatchItem{}, QueueMode: QueueManual}
}

func (s *Store) loadData() (PlannerData, error) {
	raw, source, err := loadJSONWithRecovery(s.Paths.DataPath)
	if err != nil {
		return PlannerData{}, err
	}
	if raw == nil {
		d := emptyData()
		if err = atomicWriteJSON(s.Paths.DataPath, d, false); err != nil {
			return d, err
		}
		return d, nil
	}
	d, migrated, err := migrateRaw(raw)
	if err != nil {
		return d, err
	}
	if migrated || source != s.Paths.DataPath {
		if source == s.Paths.DataPath {
			bak := s.Paths.DataPath + ".pre-v6.bak"
			if _, e := os.Stat(bak); os.IsNotExist(e) {
				if b, e2 := os.ReadFile(s.Paths.DataPath); e2 == nil {
					_ = os.WriteFile(bak, b, 0o600)
				}
			}
		}
		if err = atomicWriteJSON(s.Paths.DataPath, d, true); err != nil {
			return d, err
		}
	}
	return d, nil
}

func migrateRaw(raw map[string]any) (PlannerData, bool, error) {
	_, hasDaily := raw["dailyCheckpoints"]
	_, hasWatch := raw["jiraWatch"]
	if hasDaily || hasWatch {
		b, _ := json.Marshal(raw)
		d := emptyData()
		if err := json.Unmarshal(b, &d); err != nil {
			return d, false, fmt.Errorf("Invalid planner data: %w", err)
		}
		m := d.Version != 6
		d.Version = 6
		normalizeDataSlices(&d)
		if err := validatePlannerData(&d); err != nil {
			return d, m, err
		}
		return d, m, nil
	}
	d := emptyData()
	if q, ok := raw["queueMode"].(string); ok && (q == string(QueueManual) || q == string(QueueImportant) || q == string(QueueEmergency)) {
		d.QueueMode = QueueMode(q)
	}
	if v, ok := raw["actionPoints"]; ok {
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &d.ActionPoints)
	}
	if v, ok := raw["archive"]; ok {
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &d.Archive)
	}
	if vals, ok := raw["dailyTasks"].([]any); ok {
		for i, v := range vals {
			m, ok := v.(map[string]any)
			if !ok {
				continue
			}
			id := strAny(m["id"])
			if id == "" {
				id = newID("dc")
			}
			title := strAny(m["title"])
			if title == "" {
				title = "Daily Checkpoint"
			}
			created := strAny(m["createdAt"])
			if created == "" {
				created = NowISO()
			}
			updated := strAny(m["updatedAt"])
			if updated == "" {
				updated = created
			}
			completed := []string{}
			if last := strings.TrimSpace(strAny(m["lastCheckedDate"])); last != "" {
				completed = []string{last}
			}
			disabled, _ := m["disabled"].(bool)
			d.DailyCheckpoints = append(d.DailyCheckpoints, DailyCheckpoint{ID: id, Title: title, Enabled: !disabled, Order: i, CompletedDates: completed, CreatedAt: created, UpdatedAt: updated})
		}
	}
	if vals, ok := raw["jiraTickets"].([]any); ok {
		for _, v := range vals {
			m, ok := v.(map[string]any)
			if !ok {
				continue
			}
			jm, _ := m["jira"].(map[string]any)
			key := ExtractIssueKey(firstNonEmpty(strAny(jm["issueKey"]), strAny(jm["url"])))
			if key == "" {
				continue
			}
			rawStatus := strAny(jm["status"])
			category := strAny(jm["statusCategory"])
			updated := firstNonEmpty(strAny(jm["jiraUpdated"]), strAny(m["trackedAt"]), NowISO())
			d.JiraWatch = append(d.JiraWatch, JiraWatchItem{ID: newID("jw"), IssueKey: key, Title: firstNonEmpty(strAny(jm["summary"]), strAny(m["apTitle"]), key), Status: MapWatchStatus(rawStatus, category), RawStatus: rawStatus, StatusCategory: category, Assignee: strAny(jm["assignee"]), LastComment: strAny(jm["lastComment"]), CreatedAt: firstNonEmpty(strAny(jm["jiraCreated"]), strAny(m["trackedAt"])), UpdatedAt: updated, LastSyncedAt: strAny(jm["lastSyncedAt"]), URL: strAny(jm["url"]), SyncError: strAny(jm["syncError"])})
		}
	}
	normalizeDataSlices(&d)
	if err := validatePlannerData(&d); err != nil {
		return d, true, fmt.Errorf("Legacy planner data migration failed: %w", err)
	}
	return d, true, nil
}
func strAny(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func normalizeDataSlices(d *PlannerData) {
	if d.ActionPoints == nil {
		d.ActionPoints = []ActionPoint{}
	}
	if d.Archive == nil {
		d.Archive = []ActionPoint{}
	}
	if d.DailyCheckpoints == nil {
		d.DailyCheckpoints = []DailyCheckpoint{}
	}
	if d.JiraWatch == nil {
		d.JiraWatch = []JiraWatchItem{}
	}
	if d.QueueMode == "" {
		d.QueueMode = QueueManual
	}
	for i := range d.ActionPoints {
		if d.ActionPoints[i].Tags == nil {
			d.ActionPoints[i].Tags = []string{}
		}
	}
	for i := range d.DailyCheckpoints {
		if d.DailyCheckpoints[i].CompletedDates == nil {
			d.DailyCheckpoints[i].CompletedDates = []string{}
		}
	}
}
func validatePlannerData(d *PlannerData) error {
	for i := range d.ActionPoints {
		if err := validateAP(&d.ActionPoints[i]); err != nil {
			return err
		}
	}
	for i := range d.Archive {
		if err := validateAP(&d.Archive[i]); err != nil {
			return err
		}
	}
	for i := range d.DailyCheckpoints {
		if err := validateDaily(&d.DailyCheckpoints[i]); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) saveDataLocked() error {
	s.Data.Version = 6
	normalizeDataSlices(&s.Data)
	return atomicWriteJSON(s.Paths.DataPath, s.Data, true)
}

func (s *Store) State() PlannerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneState(PlannerState{ActionPoints: s.Data.ActionPoints, Archive: s.Data.Archive, DailyCheckpoints: s.Data.DailyCheckpoints, JiraWatch: s.Data.JiraWatch, QueueMode: s.Data.QueueMode, QuickNote: s.Data.QuickNote, Settings: s.Settings})
}

func (s *Store) SaveQuickNote(note string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len([]rune(note)) > 12000 {
		return errors.New("quick note is too long")
	}
	s.Data.QuickNote = note
	return s.saveDataLocked()
}

func (s *Store) CreateAP(d APDraft) (ActionPoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	imp := 70
	if d.Importance != nil {
		imp = *d.Importance
	}
	urg := 40
	if d.BaseUrgency != nil {
		urg = *d.BaseUrgency
	}
	now := NowISO()
	jira := s.canonicalJira(d.Jira)
	ap := ActionPoint{ID: newID("ap"), Kind: d.Kind, Title: d.Title, Details: d.Details, Importance: imp, BaseUrgency: urg, Deadline: d.Deadline, Tags: d.Tags, Jira: jira, GerritURL: d.GerritURL, QueueOrder: maxQueue(s.Data.ActionPoints) + 1, CreatedAt: now, UpdatedAt: now}
	if err := validateAP(&ap); err != nil {
		return ap, err
	}
	if ap.Jira != nil {
		s.syncLinkBestEffort(&ap)
	}
	s.Data.ActionPoints = append(s.Data.ActionPoints, ap)
	if err := s.saveDataLocked(); err != nil {
		return ap, err
	}
	return cloneAPs([]ActionPoint{ap})[0], nil
}
func maxQueue(list []ActionPoint) int {
	m := -1
	for _, x := range list {
		if x.QueueOrder > m {
			m = x.QueueOrder
		}
	}
	return m
}
func (s *Store) UpdateAP(id string, patch map[string]any) (ActionPoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.findAP(id)
	if idx < 0 {
		return ActionPoint{}, errors.New("AP not found")
	}
	cur := s.Data.ActionPoints[idx]
	b, _ := json.Marshal(cur)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	protected := map[string]bool{"id": true, "createdAt": true, "updatedAt": true, "completedAt": true}
	for k, v := range patch {
		if !protected[k] {
			m[k] = v
		}
	}
	m["id"] = cur.ID
	m["createdAt"] = cur.CreatedAt
	m["updatedAt"] = NowISO()
	delete(m, "completedAt")
	b, _ = json.Marshal(m)
	var up ActionPoint
	if err := json.Unmarshal(b, &up); err != nil {
		return up, err
	}
	if err := validateAP(&up); err != nil {
		return up, err
	}
	up.Jira = s.canonicalJira(up.Jira)
	if up.Jira != nil {
		s.syncLinkBestEffort(&up)
	}
	s.Data.ActionPoints[idx] = up
	if err := s.saveDataLocked(); err != nil {
		return up, err
	}
	return cloneAPs([]ActionPoint{up})[0], nil
}
func (s *Store) DeleteAP(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.findAP(id)
	if idx < 0 {
		return errors.New("AP not found")
	}
	s.Data.ActionPoints = append(s.Data.ActionPoints[:idx], s.Data.ActionPoints[idx+1:]...)
	s.compactQueueLocked()
	return s.saveDataLocked()
}
func (s *Store) CompleteAP(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.findAP(id)
	if idx < 0 {
		return errors.New("AP not found")
	}
	ap := s.Data.ActionPoints[idx]
	s.Data.ActionPoints = append(s.Data.ActionPoints[:idx], s.Data.ActionPoints[idx+1:]...)
	ap.CompletedAt = NowISO()
	ap.UpdatedAt = ap.CompletedAt
	if ap.Jira != nil {
		s.ensureWatchForAPLocked(&ap)
	}
	s.Data.Archive = append([]ActionPoint{ap}, s.Data.Archive...)
	s.compactQueueLocked()
	// Post Jira comment on AP completion
	if s.Settings.JiraCommentOnComplete && ap.Jira != nil && ap.Jira.IssueKey != "" {
		issueKey := ap.Jira.IssueKey
		title := ap.Title
		details := ap.Details
		includeDetail := s.Settings.JiraCommentIncludeDetail
		cfg := s.JiraConfig
		client := s.JiraClient
		log.Printf("[AP-COMPLETE] issueKey=%s title=%q jiraConfigured=%v commentOn=%v", issueKey, title, client.Configured(cfg), s.Settings.JiraCommentOnComplete)
		go func() {
			comment := fmt.Sprintf("[EisenDay] AP completed: %s", title)
			if includeDetail && details != "" {
				comment += "\n\nDetails:\n" + details
			}
			log.Printf("[AP-COMPLETE] posting comment to %s ...", issueKey)
			if err := client.AddComment(cfg, issueKey, comment); err != nil {
				log.Printf("[AP-COMPLETE] jira comment FAILED for %s: %v", issueKey, err)
			} else {
				log.Printf("[AP-COMPLETE] jira comment OK for %s", issueKey)
			}
		}()
	} else {
		log.Printf("[AP-COMPLETE] skipped jira comment: commentOnComplete=%v hasJira=%v issueKey=%q", s.Settings.JiraCommentOnComplete, ap.Jira != nil, func() string { if ap.Jira != nil { return ap.Jira.IssueKey }; return "" }())
	}
	// Auto-create Code Review for commit AP
	if ap.Kind == "commit" && ap.GerritURL != "" {
		go func() {
			draft := CodeReviewDraft{
				URL:          ap.GerritURL,
				Title:        ap.Title,
				Description:  ap.Details,
				Validation:   ap.Details,
				ApprovalMode: "each",
			}
			if _, err := s.CreateCodeReview(draft); err != nil {
				log.Printf("[AP-COMPLETE] auto code review creation FAILED: %v", err)
			} else {
				log.Printf("[AP-COMPLETE] auto code review created for %s", ap.GerritURL)
			}
		}()
	}
	return s.saveDataLocked()
}
func (s *Store) RestoreAP(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, x := range s.Data.Archive {
		if x.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errors.New("Archived AP not found")
	}
	ap := s.Data.Archive[idx]
	s.Data.Archive = append(s.Data.Archive[:idx], s.Data.Archive[idx+1:]...)
	ap.CompletedAt = ""
	ap.UpdatedAt = NowISO()
	ap.QueueOrder = maxQueue(s.Data.ActionPoints) + 1
	s.Data.ActionPoints = append(s.Data.ActionPoints, ap)
	return s.saveDataLocked()
}
func (s *Store) ReorderQueue(ids []string, mode *QueueMode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := map[string]bool{}
	for _, ap := range s.Data.ActionPoints {
		existing[ap.ID] = true
	}
	seen := map[string]bool{}
	requested := []string{}
	for _, id := range ids {
		if existing[id] && !seen[id] {
			seen[id] = true
			requested = append(requested, id)
		}
	}
	sorted := append([]ActionPoint{}, s.Data.ActionPoints...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].QueueOrder < sorted[j].QueueOrder })
	for _, ap := range sorted {
		if !seen[ap.ID] {
			requested = append(requested, ap.ID)
		}
	}
	order := map[string]int{}
	for i, id := range requested {
		order[id] = i
	}
	for i := range s.Data.ActionPoints {
		s.Data.ActionPoints[i].QueueOrder = order[s.Data.ActionPoints[i].ID]
	}
	if mode != nil {
		if *mode != QueueManual && *mode != QueueImportant && *mode != QueueEmergency {
			return errors.New("invalid queue mode")
		}
		s.Data.QueueMode = *mode
	}
	return s.saveDataLocked()
}

func (s *Store) CreateDaily(d DailyCheckpointDraft) (DailyCheckpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	enabled := true
	if d.Enabled != nil {
		enabled = *d.Enabled
	}
	now := NowISO()
	item := DailyCheckpoint{ID: newID("dc"), Title: d.Title, Enabled: enabled, Order: maxDailyOrder(s.Data.DailyCheckpoints) + 1, CompletedDates: []string{}, CreatedAt: now, UpdatedAt: now}
	if err := validateDaily(&item); err != nil {
		return item, err
	}
	s.Data.DailyCheckpoints = append(s.Data.DailyCheckpoints, item)
	if err := s.saveDataLocked(); err != nil {
		return item, err
	}
	return item, nil
}
func maxDailyOrder(list []DailyCheckpoint) int {
	m := -1
	for _, x := range list {
		if x.Order > m {
			m = x.Order
		}
	}
	return m
}
func (s *Store) UpdateDaily(id string, patch map[string]any) (DailyCheckpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, x := range s.Data.DailyCheckpoints {
		if x.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return DailyCheckpoint{}, errors.New("Daily checkpoint not found")
	}
	cur := s.Data.DailyCheckpoints[idx]
	b, _ := json.Marshal(cur)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for k, v := range patch {
		if k != "id" && k != "createdAt" && k != "updatedAt" {
			m[k] = v
		}
	}
	m["id"] = cur.ID
	m["createdAt"] = cur.CreatedAt
	m["updatedAt"] = NowISO()
	b, _ = json.Marshal(m)
	var up DailyCheckpoint
	if err := json.Unmarshal(b, &up); err != nil {
		return up, err
	}
	if err := validateDaily(&up); err != nil {
		return up, err
	}
	s.Data.DailyCheckpoints[idx] = up
	if err := s.saveDataLocked(); err != nil {
		return up, err
	}
	return up, nil
}
func (s *Store) DeleteDaily(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	before := len(s.Data.DailyCheckpoints)
	out := s.Data.DailyCheckpoints[:0]
	for _, x := range s.Data.DailyCheckpoints {
		if x.ID != id {
			out = append(out, x)
		}
	}
	s.Data.DailyCheckpoints = out
	if len(out) == before {
		return errors.New("Daily checkpoint not found")
	}
	sort.SliceStable(s.Data.DailyCheckpoints, func(i, j int) bool { return s.Data.DailyCheckpoints[i].Order < s.Data.DailyCheckpoints[j].Order })
	for i := range s.Data.DailyCheckpoints {
		s.Data.DailyCheckpoints[i].Order = i
	}
	return s.saveDataLocked()
}
func (s *Store) ToggleDaily(id, dateKey string) (DailyCheckpoint, error) {
	if err := validateDateKey(dateKey); err != nil {
		return DailyCheckpoint{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Data.DailyCheckpoints {
		item := &s.Data.DailyCheckpoints[i]
		if item.ID != id {
			continue
		}
		if !item.Enabled {
			return DailyCheckpoint{}, errors.New("Disabled Daily Checkpoint는 체크할 수 없습니다.")
		}
		found := -1
		for j, d := range item.CompletedDates {
			if d == dateKey {
				found = j
				break
			}
		}
		if found >= 0 {
			item.CompletedDates = append(item.CompletedDates[:found], item.CompletedDates[found+1:]...)
		} else {
			item.CompletedDates = append(item.CompletedDates, dateKey)
			// Post Jira comment on checkpoint completion
			if s.Settings.JiraCommentOnCheckpoint && item.JiraIssueKey != "" {
				userName := s.Settings.DisplayName
				if userName == "" {
					userName = "user"
				}
				cfg := s.JiraConfig
				client := s.JiraClient
				log.Printf("[CHECKPOINT] issueKey=%s title=%q user=%s jiraConfigured=%v", item.JiraIssueKey, item.Title, userName, client.Configured(cfg))
				go func(issueKey, title, user string) {
					comment := fmt.Sprintf("[EisenDay] Daily checkpoint confirmed by %s: %s", user, title)
					log.Printf("[CHECKPOINT] posting comment to %s ...", issueKey)
					if err := client.AddComment(cfg, issueKey, comment); err != nil {
						log.Printf("[CHECKPOINT] jira comment FAILED for %s: %v", issueKey, err)
					} else {
						log.Printf("[CHECKPOINT] jira comment OK for %s", issueKey)
					}
				}(item.JiraIssueKey, item.Title, userName)
			} else {
				log.Printf("[CHECKPOINT] skipped jira comment: commentOnCheckpoint=%v issueKey=%q", s.Settings.JiraCommentOnCheckpoint, item.JiraIssueKey)
			}
		}
		item.CompletedDates = sortDatesKeepLast400(item.CompletedDates)
		item.UpdatedAt = NowISO()
		if err := s.saveDataLocked(); err != nil {
			return *item, err
		}
		copy := *item
		copy.CompletedDates = append([]string{}, item.CompletedDates...)
		return copy, nil
	}
	return DailyCheckpoint{}, errors.New("Daily checkpoint not found")
}

func (s *Store) SaveJiraWatch(items []JiraWatchItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := map[string]bool{}
	clean := []JiraWatchItem{}
	for _, item := range items {
		key := ExtractIssueKey(item.IssueKey)
		if key == "" || seen[key] || item.Status == "Done" {
			continue
		}
		seen[key] = true
		item.IssueKey = key
		if s.JiraConfig.BaseURL != "" {
			item.URL = BrowseURL(s.JiraConfig.BaseURL, key)
		}
		clean = append(clean, item)
	}
	s.Data.JiraWatch = clean
	return s.saveDataLocked()
}
func (s *Store) SyncJiraWatch() ([]JiraWatchItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []JiraWatchItem{}
	for _, item := range s.Data.JiraWatch {
		synced := s.syncWatchBestEffort(item)
		if synced.Status != "Done" {
			out = append(out, synced)
		}
	}
	s.Data.JiraWatch = out
	for i := range s.Data.ActionPoints {
		if s.Data.ActionPoints[i].Jira != nil {
			s.syncLinkBestEffort(&s.Data.ActionPoints[i])
		}
	}
	if err := s.saveDataLocked(); err != nil {
		return nil, err
	}
	return append([]JiraWatchItem{}, s.Data.JiraWatch...), nil
}

func (s *Store) SaveSettings(ns PlannerSettings) (PlannerSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	target := s.Paths.DataPath
	var err error
	if strings.TrimSpace(ns.DataPath) != "" {
		target, err = normalizeDataPath(ns.DataPath)
		if err != nil {
			return s.Settings, err
		}
	}
	if target != s.Paths.DataPath {
		if err = s.switchDataPathLocked(target); err != nil {
			return s.Settings, err
		}
	}
	name := strings.TrimSpace(ns.DisplayName)
	if name == "" {
		name = DefaultDisplayName()
	}
	tz := strings.TrimSpace(ns.Timezone)
	if tz == "" {
		tz = "local"
	}
	base := ""
	if strings.TrimSpace(ns.JiraBaseURL) != "" {
		base, err = NormalizeBaseURL(ns.JiraBaseURL)
		if err != nil {
			return s.Settings, err
		}
	}
	mins := ns.AutoSyncMinutes
	if mins < 1 || mins > 1440 {
		return s.Settings, errors.New("autoSyncMinutes must be between 1 and 1440")
	}
	s.Settings = PlannerSettings{DisplayName: name, Timezone: tz, JiraBaseURL: base, DataPath: s.Paths.DataPath, AutoSyncMinutes: mins, TeamShareRoot: strings.TrimSpace(ns.TeamShareRoot), JiraCommentOnComplete: ns.JiraCommentOnComplete, JiraCommentIncludeDetail: ns.JiraCommentIncludeDetail, JiraCommentOnCheckpoint: ns.JiraCommentOnCheckpoint}
	if base != s.JiraConfig.BaseURL {
		s.JiraConfig.BaseURL = base
		if err = s.saveJiraConfigLocked(); err != nil {
			return s.Settings, err
		}
	}
	if err = s.saveSettingsLocked(); err != nil {
		return s.Settings, err
	}
	return s.Settings, nil
}
func (s *Store) JiraPublicConfig() JiraConfigPublic {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return JiraConfigPublic{BaseURL: s.JiraConfig.BaseURL, AuthMode: s.JiraConfig.AuthMode, Username: s.JiraConfig.Username, APIVersion: s.JiraConfig.APIVersion, TokenConfigured: strings.TrimSpace(s.JiraConfig.Token) != ""}
}
func (s *Store) SaveJiraConfig(p JiraConfigUpdate) (JiraConfigPublic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.BaseURL != nil {
		base, err := NormalizeBaseURL(*p.BaseURL)
		if err != nil {
			return JiraConfigPublic{}, err
		}
		s.JiraConfig.BaseURL = base
	}
	if p.AuthMode != nil {
		v := strings.ToLower(strings.TrimSpace(*p.AuthMode))
		if v != "bearer" && v != "basic" {
			return JiraConfigPublic{}, errors.New("authMode must be bearer or basic")
		}
		s.JiraConfig.AuthMode = v
	}
	if p.Username != nil {
		s.JiraConfig.Username = strings.TrimSpace(*p.Username)
	}
	if p.APIVersion != nil {
		v := strings.TrimSpace(*p.APIVersion)
		if v != "2" && v != "3" {
			return JiraConfigPublic{}, errors.New("apiVersion must be 2 or 3")
		}
		s.JiraConfig.APIVersion = v
	}
	if p.Token != nil && strings.TrimSpace(*p.Token) != "" {
		s.JiraConfig.Token = strings.TrimSpace(*p.Token)
	}
	s.Settings.JiraBaseURL = s.JiraConfig.BaseURL
	if err := s.saveJiraConfigLocked(); err != nil {
		return JiraConfigPublic{}, err
	}
	if err := s.saveSettingsLocked(); err != nil {
		return JiraConfigPublic{}, err
	}
	return JiraConfigPublic{BaseURL: s.JiraConfig.BaseURL, AuthMode: s.JiraConfig.AuthMode, Username: s.JiraConfig.Username, APIVersion: s.JiraConfig.APIVersion, TokenConfigured: s.JiraConfig.Token != ""}, nil
}
func (s *Store) TestJira() (string, error) {
	s.mu.RLock()
	cfg := s.JiraConfig
	s.mu.RUnlock()
	return s.JiraClient.Test(cfg)
}
func (s *Store) AutoSyncMinutes() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.Settings.AutoSyncMinutes
	if m < 1 {
		m = 15
	}
	return m
}
func (s *Store) HasJiraWatch() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Data.JiraWatch) > 0 && s.JiraClient.Configured(s.JiraConfig)
}

func (s *Store) switchDataPathLocked(target string) error {
	if err := os.MkdirAll(filepathDir(target), 0o755); err != nil {
		return err
	}
	nl := NewFileLock(target)
	if err := nl.Acquire(); err != nil {
		return err
	}
	var nd PlannerData
	if _, err := os.Stat(target); err == nil {
		raw, _, err := loadJSONWithRecovery(target)
		if err != nil {
			nl.Release()
			return err
		}
		if raw == nil {
			nl.Release()
			return errors.New("선택한 JSON 파일을 읽을 수 없습니다.")
		}
		nd, migrated, err := migrateRaw(raw)
		if err != nil {
			nl.Release()
			return err
		}
		if migrated {
			if err = atomicWriteJSON(target, nd, true); err != nil {
				nl.Release()
				return err
			}
		}
	} else if os.IsNotExist(err) {
		nd = s.Data
		if err = atomicWriteJSON(target, nd, false); err != nil {
			nl.Release()
			return err
		}
	} else {
		nl.Release()
		return err
	}
	old := s.lock
	s.lock = nl
	s.Paths.DataPath = target
	s.Data = nd
	s.Settings.DataPath = target
	if err := s.saveSettingsLocked(); err != nil {
		s.lock = old
		nl.Release()
		return err
	}
	old.Release()
	return nil
}
func filepathDir(p string) string {
	idx := strings.LastIndexAny(p, "/\\")
	if idx < 0 {
		return "."
	}
	return p[:idx]
}

func (s *Store) findAP(id string) int {
	for i, x := range s.Data.ActionPoints {
		if x.ID == id {
			return i
		}
	}
	return -1
}
func (s *Store) compactQueueLocked() {
	sorted := make([]*ActionPoint, 0, len(s.Data.ActionPoints))
	for i := range s.Data.ActionPoints {
		sorted = append(sorted, &s.Data.ActionPoints[i])
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].QueueOrder == sorted[j].QueueOrder {
			return sorted[i].CreatedAt < sorted[j].CreatedAt
		}
		return sorted[i].QueueOrder < sorted[j].QueueOrder
	})
	for i, ap := range sorted {
		ap.QueueOrder = i
	}
}
func (s *Store) canonicalJira(j *JiraLink) *JiraLink {
	if j == nil {
		return nil
	}
	x := *j
	key := ExtractIssueKey(firstNonEmpty(x.IssueKey, x.URL))
	if key == "" {
		return nil
	}
	x.IssueKey = key
	if s.JiraConfig.BaseURL != "" {
		x.URL = BrowseURL(s.JiraConfig.BaseURL, key)
	}
	return &x
}
func (s *Store) applyIssueToLink(link *JiraLink, issue JiraIssueData) {
	link.IssueKey = issue.IssueKey
	link.URL = issue.URL
	link.Summary = issue.Title
	link.Status = issue.RawStatus
	link.StatusCategory = issue.StatusCategory
	link.Assignee = issue.Assignee
	link.LastComment = issue.LastComment
	link.JiraCreated = issue.CreatedAt
	link.JiraUpdated = issue.UpdatedAt
	link.LastSyncedAt = NowISO()
	link.SyncError = ""
}
func (s *Store) syncLinkBestEffort(ap *ActionPoint) {
	if ap.Jira == nil {
		return
	}
	ap.Jira.LastSyncedAt = NowISO()
	if !s.JiraClient.Configured(s.JiraConfig) {
		ap.Jira.SyncError = "Jira 인증 설정이 필요합니다."
		return
	}
	issue, err := s.JiraClient.FetchIssue(s.JiraConfig, ap.Jira.IssueKey)
	if err != nil {
		ap.Jira.SyncError = err.Error()
		return
	}
	s.applyIssueToLink(ap.Jira, issue)
}
func (s *Store) syncWatchBestEffort(item JiraWatchItem) JiraWatchItem {
	item.LastSyncedAt = NowISO()
	if s.JiraConfig.BaseURL != "" {
		item.URL = BrowseURL(s.JiraConfig.BaseURL, item.IssueKey)
	}
	if !s.JiraClient.Configured(s.JiraConfig) {
		item.SyncError = "Jira 인증 설정이 필요합니다."
		return item
	}
	issue, err := s.JiraClient.FetchIssue(s.JiraConfig, item.IssueKey)
	if err != nil {
		item.SyncError = err.Error()
		return item
	}
	item.Title = firstNonEmpty(issue.Title, item.Title)
	item.RawStatus = issue.RawStatus
	item.StatusCategory = issue.StatusCategory
	item.Status = MapWatchStatus(issue.RawStatus, issue.StatusCategory)
	item.Assignee = issue.Assignee
	item.LastComment = issue.LastComment
	item.CreatedAt = firstNonEmpty(issue.CreatedAt, item.CreatedAt)
	item.UpdatedAt = firstNonEmpty(issue.UpdatedAt, NowISO())
	item.URL = issue.URL
	item.SyncError = ""
	return item
}
func (s *Store) ensureWatchForAPLocked(ap *ActionPoint) {
	if ap.Jira == nil {
		return
	}
	key := ExtractIssueKey(ap.Jira.IssueKey)
	if key == "" || strings.EqualFold(ap.Jira.StatusCategory, "done") {
		return
	}
	for i := range s.Data.JiraWatch {
		if strings.EqualFold(s.Data.JiraWatch[i].IssueKey, key) {
			if ap.Jira.Summary != "" {
				s.Data.JiraWatch[i].Title = ap.Jira.Summary
			}
			return
		}
	}
	s.Data.JiraWatch = append(s.Data.JiraWatch, JiraWatchItem{ID: newID("jw"), IssueKey: key, Title: firstNonEmpty(ap.Jira.Summary, ap.Title), Status: MapWatchStatus(ap.Jira.Status, ap.Jira.StatusCategory), RawStatus: ap.Jira.Status, StatusCategory: ap.Jira.StatusCategory, Assignee: ap.Jira.Assignee, LastComment: ap.Jira.LastComment, CreatedAt: firstNonEmpty(ap.Jira.JiraCreated, ap.CreatedAt), UpdatedAt: firstNonEmpty(ap.Jira.JiraUpdated, ap.UpdatedAt), LastSyncedAt: ap.Jira.LastSyncedAt, URL: ap.Jira.URL, SyncError: ap.Jira.SyncError})
}
