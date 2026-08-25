package main

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

//go:embed web/*
var embedded embed.FS

var errDataFileInUse = errors.New("planner data file is already in use")

type HistoryEntry struct {
	At     string `json:"at"`
	Action string `json:"action"`
	Note   string `json:"note,omitempty"`
}

type JiraLink struct {
	IssueKey     string `json:"issueKey,omitempty"`
	URL          string `json:"url"`
	Status       string `json:"status,omitempty"`
	Assignee     string `json:"assignee,omitempty"`
	LastComment  string `json:"lastComment,omitempty"`
	JiraUpdated  string `json:"jiraUpdated,omitempty"`
	LastSyncedAt string `json:"lastSyncedAt,omitempty"`
	SyncError    string `json:"syncError,omitempty"`
}

type ActionPoint struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Details     string         `json:"details,omitempty"`
	Importance  int            `json:"importance"`
	BaseUrgency int            `json:"baseUrgency"`
	Deadline    string         `json:"deadline,omitempty"`
	CreatedAt   string         `json:"createdAt"`
	UpdatedAt   string         `json:"updatedAt"`
	CompletedAt string         `json:"completedAt,omitempty"`
	QueueOrder  int            `json:"queueOrder"`
	Tags        []string       `json:"tags,omitempty"`
	Jira        *JiraLink      `json:"jira,omitempty"`
	History     []HistoryEntry `json:"history"`
}

type PlannerData struct {
	Version      int           `json:"version"`
	Timezone     string        `json:"timezone"`
	QueueMode    string        `json:"queueMode"`
	ActionPoints []ActionPoint `json:"actionPoints"`
	Archive      []ActionPoint `json:"archive"`
}

type JiraSettings struct {
	BaseURL         string `json:"baseUrl"`
	TokenPageURL    string `json:"tokenPageUrl,omitempty"`
	AuthMode        string `json:"authMode"` // bearer | basic
	Username        string `json:"username,omitempty"`
	Token           string `json:"token,omitempty"`
	APIVersion      string `json:"apiVersion"` // 2 | 3
	AutoSyncSeconds int    `json:"autoSyncSeconds"`
}

type AppSettings struct {
	DataPath    string `json:"dataPath"`
	DisplayName string `json:"displayName,omitempty"`
}

type Store struct {
	mu                sync.RWMutex
	path              string
	settingsPath      string
	configPath        string
	needsStorageSetup bool
	data              PlannerData
	jira              JiraSettings
	client            *http.Client
	dataFile          *os.File
	displayName       string
}

type APInput struct {
	Title       string    `json:"title"`
	Details     string    `json:"details"`
	Quadrant    string    `json:"quadrant"`
	Importance  *int      `json:"importance,omitempty"`
	BaseUrgency *int      `json:"baseUrgency,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Deadline    string    `json:"deadline"`
	Jira        *JiraLink `json:"jira,omitempty"`
}

type QueueRequest struct {
	Mode string   `json:"mode"`
	IDs  []string `json:"ids"`
}

func main() {
	dataPath, settingsPath, configPath, needsStorageSetup, err := resolvePaths()
	if err != nil {
		log.Fatal(err)
	}

	store, err := newStore(dataPath, settingsPath, configPath, needsStorageSetup)
	if err != nil {
		if errors.Is(err, errDataFileInUse) {
			runningURL, runningVersion := readRunningInstanceInfo(configPath)
			if runningVersion == "1.5.1" && runningURL != "" {
				_ = openBrowser(runningURL)
				return
			}
			msg := "같은 JSON을 사용하는 Eisenhower Todo가 이미 실행 중입니다."
			if runningVersion != "" {
				msg += "\n현재 실행 중: v" + runningVersion
			}
			msg += "\n\n기존 버전을 완전히 종료한 뒤 v1.5.1을 다시 실행하세요."
			fmt.Println(msg)
			showAlreadyRunningWarning(msg)
			time.Sleep(800 * time.Millisecond)
			return
		}
		log.Fatal(err)
	}
	defer store.close()

	webFS, err := fs.Sub(embedded, "web")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/state", store.handleState)
	mux.HandleFunc("POST /api/ap", store.handleCreate)
	mux.HandleFunc("PUT /api/ap/{id}", store.handleUpdate)
	mux.HandleFunc("DELETE /api/ap/{id}", store.handleDelete)
	mux.HandleFunc("POST /api/ap/{id}/complete", store.handleComplete)
	mux.HandleFunc("POST /api/archive/{id}/restore", store.handleRestore)
	mux.HandleFunc("POST /api/queue", store.handleQueue)
	mux.HandleFunc("GET /api/jira/settings", store.handleJiraSettingsGet)
	mux.HandleFunc("POST /api/jira/settings", store.handleJiraSettingsSave)
	mux.HandleFunc("POST /api/jira/sync", store.handleJiraSync)
	mux.HandleFunc("POST /api/jira/test", store.handleJiraTest)
	mux.HandleFunc("POST /api/open-url", handleOpenURL)
	mux.HandleFunc("GET /api/storage", store.handleStorageGet)
	mux.HandleFunc("POST /api/storage", store.handleStorageSave)
	mux.HandleFunc("POST /api/storage/pick-folder", store.handlePickFolder)
	mux.HandleFunc("GET /api/profile", store.handleProfileGet)
	mux.HandleFunc("POST /api/profile", store.handleProfileSave)
	mux.HandleFunc("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		store.mu.RLock()
		currentPath := store.path
		needsSetup := store.needsStorageSetup
		displayName := store.displayName
		store.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"dataPath":               currentPath,
			"jiraSettingsPath":       settingsPath,
			"configPath":             configPath,
			"version":                "1.5.1",
			"displayName":            displayName,
			"needsDataLocationSetup": needsSetup,
		})
	})
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	ln, err := listenLocal()
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	appURL := fmt.Sprintf("http://127.0.0.1:%d", addr.Port)
	runningPath := filepath.Join(filepath.Dir(configPath), "running.json")
	_ = writeRunningInstance(runningPath, appURL)
	defer os.Remove(runningPath)
	fmt.Printf("Eisenhower Todo v1.5.1\n%s\nData: %s\nJira settings: %s\n", appURL, dataPath, settingsPath)

	go func() {
		time.Sleep(350 * time.Millisecond)
		_ = openBrowser(appURL)
		store.syncAllJira()
	}()

	go store.autoSyncLoop()

	server := &http.Server{
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func listenLocal() (net.Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:8787")
	if err == nil {
		return ln, nil
	}
	return net.Listen("tcp", "127.0.0.1:0")
}

func resolvePaths() (string, string, string, bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", "", "", false, err
	}
	exeBase := filepath.Dir(exe)

	// v1.4.2+: keep configuration in a stable per-user location so upgrading or
	// moving the executable does not silently switch to a fresh data file.
	userConfig, err := os.UserConfigDir()
	if err != nil {
		return "", "", "", false, fmt.Errorf("cannot resolve user config directory: %w", err)
	}
	appDir := filepath.Join(userConfig, "EisenhowerTodo")
	dataDir := filepath.Join(appDir, "data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", "", "", false, fmt.Errorf("cannot create app data directory %s: %w", dataDir, err)
	}

	configPath := filepath.Join(appDir, "config.json")
	settingsPath := filepath.Join(appDir, "jira-settings.json")
	defaultDataPath := filepath.Join(dataDir, "eisenhower.json")

	// Preferred: stable config created by v1.4.2+.
	if p, ok := readConfiguredDataPath(configPath); ok {
		return p, settingsPath, configPath, false, nil
	}

	// One-time migration from v1.2-v1.4.1 portable layout when the new exe
	// replaces the old exe in the same folder.
	legacyConfig := filepath.Join(exeBase, "eisenhower-config.json")
	if p, ok := readConfiguredDataPath(legacyConfig); ok {
		_ = writeAppSettingsFile(configPath, p, "")

		legacyJira := filepath.Join(exeBase, "data", "jira-settings.json")
		if _, err := os.Stat(settingsPath); errors.Is(err, os.ErrNotExist) {
			if raw, readErr := os.ReadFile(legacyJira); readErr == nil {
				_ = writeBytesAtomic(settingsPath, raw, false)
			}
		}
		return p, settingsPath, configPath, false, nil
	}

	// Migrate the old default portable JSON if present next to this executable.
	legacyDefault := filepath.Join(exeBase, "data", "eisenhower.json")
	if _, err := os.Stat(legacyDefault); err == nil {
		if _, loadErr := loadPlannerFile(legacyDefault); loadErr == nil {
			_ = writeAppSettingsFile(configPath, legacyDefault, "")
			return legacyDefault, settingsPath, configPath, false, nil
		}
	}

	_, defaultErr := os.Stat(defaultDataPath)
	needsSetup := errors.Is(defaultErr, os.ErrNotExist)
	return defaultDataPath, settingsPath, configPath, needsSetup, nil
}

func readConfiguredDataPath(configPath string) (string, bool) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return "", false
	}
	var cfg AppSettings
	if json.Unmarshal(raw, &cfg) != nil || strings.TrimSpace(cfg.DataPath) == "" {
		return "", false
	}
	resolved, err := normalizeDataPath(cfg.DataPath)
	return resolved, err == nil
}

func writeAppSettingsFile(configPath, dataPath, displayName string) error {
	raw, err := json.MarshalIndent(AppSettings{DataPath: dataPath, DisplayName: displayName}, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return writeBytesAtomic(configPath, raw, true)
}

func loadAppSettings(configPath string) (AppSettings, bool) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return AppSettings{}, false
	}
	var cfg AppSettings
	if json.Unmarshal(raw, &cfg) != nil {
		return AppSettings{}, false
	}
	return cfg, true
}

func defaultDisplayName() string {
	for _, key := range []string{"USERNAME", "USER"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return "You"
}

func normalizeDataPath(input string) (string, error) {
	v := strings.TrimSpace(input)
	if v == "" {
		return "", errors.New("JSON 저장 위치를 입력하세요")
	}
	v = os.ExpandEnv(v)
	if !filepath.IsAbs(v) {
		abs, err := filepath.Abs(v)
		if err != nil {
			return "", err
		}
		v = abs
	}
	if info, err := os.Stat(v); err == nil && info.IsDir() {
		v = filepath.Join(v, "eisenhower.json")
	} else if filepath.Ext(v) == "" {
		v = filepath.Join(v, "eisenhower.json")
	}
	if strings.ToLower(filepath.Ext(v)) != ".json" {
		return "", errors.New("저장 파일은 .json 확장자를 사용해야 합니다")
	}
	return filepath.Clean(v), nil
}

func newStore(path, settingsPath, configPath string, needsStorageSetup bool) (*Store, error) {
	cfg, configExists := loadAppSettings(configPath)
	displayName := strings.TrimSpace(cfg.DisplayName)
	if displayName == "" {
		if configExists {
			// v1.4.6 and older showed Minju as a hard-coded greeting.
			// Preserve that for existing users; fresh installs use the OS account name.
			displayName = "Minju"
		} else {
			displayName = defaultDisplayName()
		}
	}

	s := &Store{
		path:              path,
		settingsPath:      settingsPath,
		configPath:        configPath,
		needsStorageSetup: needsStorageSetup,
		displayName:       displayName,
		client: &http.Client{
			Timeout: 12 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	if err := s.loadData(); err != nil {
		return nil, err
	}
	if err := s.loadJiraSettings(); err != nil {
		return nil, err
	}
	dataFile, err := acquireExclusiveDataFile(s.path)
	if err != nil {
		return nil, err
	}
	s.dataFile = dataFile
	return s, nil
}

func (s *Store) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dataFile != nil {
		_ = s.dataFile.Close()
		s.dataFile = nil
	}
}

func (s *Store) loadData() error {
	data, source, err := loadPlannerWithRecovery(s.path)
	if err == nil {
		migrated := false
		if data.Version < 4 {
			data.Version = 4
			migrated = true
		}
		// v1.4.8 is active-only: completed APs are not retained.
		if len(data.Archive) > 0 {
			data.Archive = []ActionPoint{}
			migrated = true
		}
		s.data = data
		if source != s.path || migrated {
			if saveErr := s.saveLocked(); saveErr != nil {
				return fmt.Errorf("could not persist planner migration from %s: %w", source, saveErr)
			}
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	s.data = PlannerData{
		Version:      4,
		Timezone:     "local",
		QueueMode:    "manual",
		ActionPoints: []ActionPoint{},
		Archive:      []ActionPoint{},
	}
	return s.saveLocked()
}

func loadPlannerWithRecovery(path string) (PlannerData, string, error) {
	candidates := []string{path, path + ".bak", path + ".tmp", path + ".old"}
	var firstRealErr error
	foundAny := false

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if firstRealErr == nil {
				firstRealErr = err
			}
			continue
		}
		foundAny = true
		data, err := loadPlannerFile(candidate)
		if err == nil {
			return data, candidate, nil
		}
		if firstRealErr == nil {
			firstRealErr = err
		}
	}

	if firstRealErr != nil {
		return PlannerData{}, "", firstRealErr
	}
	if !foundAny {
		return PlannerData{}, "", os.ErrNotExist
	}
	return PlannerData{}, "", fmt.Errorf("planner JSON and recovery files are invalid for %s", path)
}

func (s *Store) loadJiraSettings() error {
	raw, err := os.ReadFile(s.settingsPath)
	if errors.Is(err, os.ErrNotExist) {
		s.jira = defaultJiraSettings()
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &s.jira); err != nil {
		return fmt.Errorf("invalid Jira settings JSON: %w", err)
	}
	normalizeJiraSettings(&s.jira)
	return nil
}

func defaultJiraSettings() JiraSettings {
	return JiraSettings{AuthMode: "bearer", APIVersion: "2", AutoSyncSeconds: 60}
}

func normalizeJiraSettings(v *JiraSettings) {
	v.BaseURL = normalizeJiraBaseURL(v.BaseURL)
	v.TokenPageURL = strings.TrimSpace(v.TokenPageURL)
	v.Username = strings.TrimSpace(v.Username)
	v.AuthMode = strings.ToLower(strings.TrimSpace(v.AuthMode))
	if v.AuthMode != "basic" {
		v.AuthMode = "bearer"
	}
	if v.APIVersion != "3" {
		v.APIVersion = "2"
	}
	if v.AutoSyncSeconds < 15 {
		v.AutoSyncSeconds = 60
	}
}

func (s *Store) saveLocked() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if s.dataFile != nil {
		return writeExclusiveDataFile(s.dataFile, s.path, raw)
	}
	return writeBytesAtomic(s.path, raw, true)
}

func (s *Store) saveJiraSettingsLocked() error {
	raw, err := json.MarshalIndent(s.jira, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return writeBytesAtomic(s.settingsPath, raw, true)
}

func (s *Store) saveAppSettingsLocked() error {
	return writeAppSettingsFile(s.configPath, s.path, s.displayName)
}

func writeBytesAtomic(path string, raw []byte, keepBackup bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if keepBackup {
		if old, err := os.ReadFile(path); err == nil && len(old) > 0 && json.Valid(old) {
			_ = os.WriteFile(path+".bak", old, 0o600)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func loadPlannerFile(path string) (PlannerData, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return PlannerData{}, err
	}
	var data PlannerData
	if err := json.Unmarshal(raw, &data); err != nil {
		return PlannerData{}, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	if data.Version < 4 {
		data.Version = 4
	}
	if data.QueueMode == "" {
		data.QueueMode = "manual"
	}
	if data.ActionPoints == nil {
		data.ActionPoints = []ActionPoint{}
	}
	if data.Archive == nil {
		data.Archive = []ActionPoint{}
	}
	return data, nil
}

func (s *Store) handleProfileGet(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"displayName": s.displayName})
}

func (s *Store) handleProfileSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DisplayName string `json:"displayName"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	name := strings.TrimSpace(req.DisplayName)
	if name == "" {
		writeError(w, http.StatusBadRequest, "이름을 입력하세요.")
		return
	}
	if len([]rune(name)) > 40 {
		writeError(w, http.StatusBadRequest, "이름은 40자 이하로 입력하세요.")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.displayName
	s.displayName = name
	if err := s.saveAppSettingsLocked(); err != nil {
		s.displayName = old
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"displayName": s.displayName})
}

func (s *Store) handleStorageGet(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"dataPath":   s.path,
		"needsSetup": s.needsStorageSetup,
	})
}

func (s *Store) handleStorageSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DataPath string `json:"dataPath"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	target, err := normalizeDataPath(req.DataPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if filepath.Clean(target) == filepath.Clean(s.path) {
		s.needsStorageSetup = false
		if err := s.saveAppSettingsLocked(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"dataPath": s.path, "loadedExisting": true})
		return
	}

	loadedExisting := false
	nextData := s.data
	if _, statErr := os.Stat(target); statErr == nil {
		data, loadErr := loadPlannerFile(target)
		if loadErr != nil {
			writeError(w, http.StatusBadRequest, loadErr.Error())
			return
		}
		nextData = data
		loadedExisting = true
	} else if !errors.Is(statErr, os.ErrNotExist) {
		writeError(w, http.StatusBadRequest, statErr.Error())
		return
	}

	newDataFile, lockErr := acquireExclusiveDataFile(target)
	if lockErr != nil {
		if errors.Is(lockErr, errDataFileInUse) {
			writeError(w, http.StatusConflict, "선택한 JSON을 다른 Eisenhower Todo 인스턴스가 사용 중입니다.")
		} else {
			writeError(w, http.StatusInternalServerError, lockErr.Error())
		}
		return
	}

	oldPath := s.path
	oldData := s.data
	oldFile := s.dataFile

	s.path = target
	s.data = nextData
	s.dataFile = newDataFile

	if !loadedExisting {
		if err := s.saveLocked(); err != nil {
			s.path = oldPath
			s.data = oldData
			s.dataFile = oldFile
			_ = newDataFile.Close()
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	s.needsStorageSetup = false
	if err := s.saveAppSettingsLocked(); err != nil {
		s.path = oldPath
		s.data = oldData
		s.dataFile = oldFile
		_ = newDataFile.Close()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if oldFile != nil {
		_ = oldFile.Close()
	}
	writeJSON(w, http.StatusOK, map[string]any{"dataPath": s.path, "loadedExisting": loadedExisting})
}

func (s *Store) handlePickFolder(w http.ResponseWriter, r *http.Request) {
	if runtime.GOOS != "windows" {
		writeError(w, http.StatusNotImplemented, "폴더 선택 버튼은 Windows에서만 지원합니다. 경로를 직접 입력하세요.")
		return
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-STA", "-Command", `[void][System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms'); $d=New-Object System.Windows.Forms.FolderBrowserDialog; $d.Description='Eisenhower Todo JSON 저장 폴더 선택'; if($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK){[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; Write-Output $d.SelectedPath}`)
	out, err := cmd.Output()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "폴더 선택 창을 열지 못했습니다. 경로를 직접 입력하세요.")
		return
	}
	folder := strings.TrimSpace(string(out))
	if folder == "" {
		writeJSON(w, http.StatusOK, map[string]any{"cancelled": true})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"folder": folder, "dataPath": filepath.Join(folder, "eisenhower.json")})
}

func (s *Store) handleState(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, http.StatusOK, s.data)
}

func (s *Store) handleCreate(w http.ResponseWriter, r *http.Request) {
	var in APInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	importance, baseUrgency := quadrantScores(in.Quadrant)
	if in.Importance != nil {
		importance = clamp(*in.Importance)
	}
	if in.BaseUrgency != nil {
		baseUrgency = clamp(*in.BaseUrgency)
	}

	now := nowISO()
	s.mu.Lock()
	ap := ActionPoint{
		ID:          fmt.Sprintf("ap-%d", time.Now().UnixNano()),
		Title:       title,
		Details:     strings.TrimSpace(in.Details),
		Importance:  importance,
		BaseUrgency: baseUrgency,
		Deadline:    normalizeTime(in.Deadline),
		CreatedAt:   now,
		UpdatedAt:   now,
		QueueOrder:  nextOrder(s.data.ActionPoints),
		Tags:        cleanTags(in.Tags),
		Jira:        cleanJira(in.Jira),
		History:     []HistoryEntry{{At: now, Action: "created"}},
	}
	ap.Jira = s.canonicalizeJiraLocked(ap.Jira)
	s.data.ActionPoints = append(s.data.ActionPoints, ap)
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.mu.Unlock()

	// Any local change triggers a Jira refresh for Jira-linked active APs.
	s.syncAllJira()
	writeJSON(w, http.StatusCreated, ap)
}

func (s *Store) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in APInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.mu.Lock()
	i := indexOf(s.data.ActionPoints, id)
	if i < 0 {
		s.mu.Unlock()
		writeError(w, http.StatusNotFound, "AP not found")
		return
	}

	ap := &s.data.ActionPoints[i]
	if title := strings.TrimSpace(in.Title); title != "" {
		ap.Title = title
	}
	ap.Details = strings.TrimSpace(in.Details)
	ap.Deadline = normalizeTime(in.Deadline)
	ap.Jira = mergeJiraLink(ap.Jira, in.Jira)
	ap.Jira = s.canonicalizeJiraLocked(ap.Jira)
	if in.Importance != nil {
		ap.Importance = clamp(*in.Importance)
	}
	if in.BaseUrgency != nil {
		ap.BaseUrgency = clamp(*in.BaseUrgency)
	}
	if in.Tags != nil {
		ap.Tags = cleanTags(in.Tags)
	}
	if in.Quadrant != "" {
		ap.Importance, ap.BaseUrgency = quadrantScores(in.Quadrant)
	}
	ap.UpdatedAt = nowISO()
	note := in.Quadrant
	if note == "" {
		note = fmt.Sprintf("importance=%d, baseUrgency=%d", ap.Importance, ap.BaseUrgency)
	}
	ap.History = append(ap.History, HistoryEntry{At: ap.UpdatedAt, Action: "updated", Note: note})
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := *ap
	s.mu.Unlock()

	s.syncAllJira()
	writeJSON(w, http.StatusOK, result)
}

func (s *Store) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	i := indexOf(s.data.ActionPoints, id)
	if i < 0 {
		s.mu.Unlock()
		writeError(w, http.StatusNotFound, "AP not found")
		return
	}
	s.data.ActionPoints = append(s.data.ActionPoints[:i], s.data.ActionPoints[i+1:]...)
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.mu.Unlock()
	s.syncAllJira()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Store) handleComplete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	i := indexOf(s.data.ActionPoints, id)
	if i < 0 {
		s.mu.Unlock()
		writeError(w, http.StatusNotFound, "AP not found")
		return
	}
	ap := s.data.ActionPoints[i]
	s.data.ActionPoints = append(s.data.ActionPoints[:i], s.data.ActionPoints[i+1:]...)
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.mu.Unlock()
	s.syncAllJira()
	writeJSON(w, http.StatusOK, map[string]any{"id": ap.ID, "removed": true})
}

func (s *Store) handleRestore(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	i := indexOf(s.data.Archive, id)
	if i < 0 {
		s.mu.Unlock()
		writeError(w, http.StatusNotFound, "archived AP not found")
		return
	}
	ap := s.data.Archive[i]
	s.data.Archive = append(s.data.Archive[:i], s.data.Archive[i+1:]...)
	now := nowISO()
	ap.CompletedAt = ""
	ap.UpdatedAt = now
	ap.QueueOrder = nextOrder(s.data.ActionPoints)
	ap.History = append(ap.History, HistoryEntry{At: now, Action: "restored"})
	s.data.ActionPoints = append(s.data.ActionPoints, ap)
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.mu.Unlock()
	s.syncAllJira()
	writeJSON(w, http.StatusOK, ap)
}

func (s *Store) handleQueue(w http.ResponseWriter, r *http.Request) {
	var req QueueRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Mode != "manual" && req.Mode != "important" && req.Mode != "emergency" {
		writeError(w, http.StatusBadRequest, "invalid queue mode")
		return
	}

	s.mu.Lock()
	s.data.QueueMode = req.Mode
	if len(req.IDs) > 0 {
		order := make(map[string]int, len(req.IDs))
		for i, id := range req.IDs {
			order[id] = i
		}
		for i := range s.data.ActionPoints {
			if n, ok := order[s.data.ActionPoints[i].ID]; ok {
				s.data.ActionPoints[i].QueueOrder = n
			}
		}
	}
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	mode := s.data.QueueMode
	s.mu.Unlock()
	s.syncAllJira()
	writeJSON(w, http.StatusOK, map[string]string{"queueMode": mode})
}

func (s *Store) handleJiraSettingsGet(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	safe := s.jira
	if safe.Token != "" {
		safe.Token = "••••••••"
	}
	writeJSON(w, http.StatusOK, safe)
}

func (s *Store) handleJiraSettingsSave(w http.ResponseWriter, r *http.Request) {
	var in JiraSettings
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	normalizeJiraSettings(&in)

	s.mu.Lock()
	if in.Token == "" || in.Token == "••••••••" {
		in.Token = s.jira.Token
	}
	s.jira = in
	if err := s.saveJiraSettingsLocked(); err != nil {
		s.mu.Unlock()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.mu.Unlock()
	s.syncAllJira()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Store) handleJiraSync(w http.ResponseWriter, r *http.Request) {
	if !s.jiraConfigured() {
		writeError(w, http.StatusBadRequest, "Jira Base URL과 token을 먼저 설정하세요.")
		return
	}
	s.syncAllJira()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Store) handleJiraTest(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	settings := s.jira
	var linked *JiraLink
	for i := range s.data.ActionPoints {
		if s.data.ActionPoints[i].Jira != nil {
			copy := *s.data.ActionPoints[i].Jira
			linked = &copy
			break
		}
	}
	s.mu.RUnlock()

	if strings.TrimSpace(settings.BaseURL) == "" || strings.TrimSpace(settings.Token) == "" {
		writeError(w, http.StatusBadRequest, "Jira Base URL과 token을 먼저 설정하세요.")
		return
	}

	if linked != nil {
		result := s.fetchJiraIssue(settings, linked)
		if result.SyncError != "" {
			writeError(w, http.StatusBadGateway, result.SyncError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Jira 연결 성공: " + result.IssueKey})
		return
	}

	endpoint := strings.TrimRight(settings.BaseURL, "/") + "/rest/api/" + settings.APIVersion + "/myself"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	addJiraAuth(req, settings)
	resp, err := s.client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		writeError(w, http.StatusBadGateway, "Jira 응답 읽기 실패: "+readErr.Error())
		return
	}
	var payload map[string]any
	if err := decodeJiraJSON(resp, body, &payload); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	name := displayName(payload)
	if name == "" {
		name = asString(payload["name"])
	}
	if name == "" {
		name = asString(payload["emailAddress"])
	}
	message := "Jira 인증 성공"
	if name != "" {
		message += ": " + name
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": message})
}

func (s *Store) jiraConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return strings.TrimSpace(s.jira.BaseURL) != "" && strings.TrimSpace(s.jira.Token) != ""
}

func (s *Store) autoSyncLoop() {
	for {
		s.mu.RLock()
		seconds := s.jira.AutoSyncSeconds
		configured := strings.TrimSpace(s.jira.BaseURL) != "" && strings.TrimSpace(s.jira.Token) != ""
		s.mu.RUnlock()
		if seconds < 15 {
			seconds = 60
		}
		time.Sleep(time.Duration(seconds) * time.Second)
		if configured {
			s.syncAllJira()
		}
	}
}

func (s *Store) syncAllJira() {
	s.mu.RLock()
	settings := s.jira
	if strings.TrimSpace(settings.BaseURL) == "" || strings.TrimSpace(settings.Token) == "" {
		s.mu.RUnlock()
		return
	}
	targets := make([]struct {
		ID   string
		Jira JiraLink
	}, 0)
	for _, ap := range s.data.ActionPoints {
		if ap.Jira != nil {
			targets = append(targets, struct {
				ID   string
				Jira JiraLink
			}{ID: ap.ID, Jira: *ap.Jira})
		}
	}
	s.mu.RUnlock()

	if len(targets) == 0 {
		return
	}

	results := make(map[string]JiraLink, len(targets))
	for _, t := range targets {
		results[t.ID] = s.fetchJiraIssue(settings, &t.Jira)
	}

	s.mu.Lock()
	changed := false
	for i := range s.data.ActionPoints {
		ap := &s.data.ActionPoints[i]
		result, ok := results[ap.ID]
		if !ok || ap.Jira == nil {
			continue
		}
		if note := jiraChangeNote(*ap.Jira, result); note != "" {
			ap.History = append(ap.History, HistoryEntry{At: nowISO(), Action: "jira-change", Note: note})
		}
		ap.Jira = &result
		changed = true
	}
	if changed {
		_ = s.saveLocked()
	}
	s.mu.Unlock()
}

func jiraBrowseURL(baseURL, key, fallback string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	key = strings.TrimSpace(key)
	if baseURL != "" && key != "" {
		return baseURL + "/browse/" + url.PathEscape(key)
	}
	return strings.TrimSpace(fallback)
}

func (s *Store) canonicalizeJiraLocked(j *JiraLink) *JiraLink {
	if j == nil {
		return nil
	}
	clean := cleanJira(j)
	if clean == nil {
		return nil
	}
	key := issueKey(clean)
	clean.IssueKey = key
	clean.URL = jiraBrowseURL(s.jira.BaseURL, key, clean.URL)
	return clean
}

func normalizeJiraBaseURL(raw string) string {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return raw
	}
	path := strings.TrimRight(parsed.Path, "/")
	lower := strings.ToLower(path)
	if i := strings.Index(lower, "/browse/"); i >= 0 {
		path = path[:i]
	}
	if i := strings.Index(strings.ToLower(path), "/rest/api/"); i >= 0 {
		path = path[:i]
	}
	parsed.Path = strings.TrimRight(path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func jiraResponseProblem(resp *http.Response, body []byte) string {
	if resp == nil {
		return "Jira 응답을 받지 못했습니다."
	}
	location := strings.TrimSpace(resp.Header.Get("Location"))
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		if location != "" {
			return fmt.Sprintf("Jira 인증 리다이렉트 (HTTP %d → %s). Token 인증 방식 또는 Jira Base URL을 확인하세요.", resp.StatusCode, location)
		}
		return fmt.Sprintf("Jira 인증 리다이렉트 (HTTP %d). Token 인증 방식 또는 Jira Base URL을 확인하세요.", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return "Jira 인증 실패 (HTTP 401). Token 또는 인증 방식을 확인하세요."
	}
	if resp.StatusCode == http.StatusForbidden {
		return "Jira 접근 거부 (HTTP 403). Token 권한 또는 해당 이슈 접근 권한을 확인하세요."
	}
	if resp.StatusCode == http.StatusNotFound {
		return "Jira API/이슈를 찾을 수 없습니다 (HTTP 404). Base URL, Issue Key, REST API 버전을 확인하세요."
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Sprintf("Jira HTTP %d", resp.StatusCode)
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	trimmed := strings.ToLower(strings.TrimSpace(string(body)))
	if strings.Contains(contentType, "text/html") ||
		strings.HasPrefix(trimmed, "<!doctype") ||
		strings.HasPrefix(trimmed, "<html") ||
		strings.HasPrefix(trimmed, "<") {
		return "Jira가 JSON 대신 HTML을 반환했습니다. 사내 SSO/login 페이지로 연결된 가능성이 큽니다. Bearer/Basic 설정과 Jira Base URL을 확인하세요."
	}
	return ""
}

func decodeJiraJSON(resp *http.Response, body []byte, out any) error {
	if problem := jiraResponseProblem(resp, body); problem != "" {
		return errors.New(problem)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return errors.New("Jira가 빈 응답을 반환했습니다.")
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("Jira JSON 파싱 실패 (Content-Type: %s): %w", resp.Header.Get("Content-Type"), err)
	}
	return nil
}

func (s *Store) fetchJiraIssue(settings JiraSettings, current *JiraLink) JiraLink {
	out := *current
	out.SyncError = ""
	out.LastSyncedAt = nowISO()
	key := issueKey(current)
	if key == "" {
		out.SyncError = "Jira issue key를 찾을 수 없습니다."
		return out
	}
	out.IssueKey = key
	out.URL = jiraBrowseURL(settings.BaseURL, key, out.URL)
	endpoint := strings.TrimRight(settings.BaseURL, "/") + "/rest/api/" + settings.APIVersion + "/issue/" + url.PathEscape(key) + "?fields=status,assignee,comment,updated"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		out.SyncError = err.Error()
		return out
	}
	addJiraAuth(req, settings)
	resp, err := s.client.Do(req)
	if err != nil {
		out.SyncError = err.Error()
		return out
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		out.SyncError = "Jira 응답 읽기 실패: " + readErr.Error()
		return out
	}

	var payload map[string]any
	if err := decodeJiraJSON(resp, body, &payload); err != nil {
		out.SyncError = err.Error()
		return out
	}
	fields, _ := payload["fields"].(map[string]any)
	if status, ok := fields["status"].(map[string]any); ok {
		out.Status = asString(status["name"])
	}
	out.Assignee = displayName(fields["assignee"])
	out.JiraUpdated = asString(fields["updated"])
	if comments, ok := fields["comment"].(map[string]any); ok {
		if list, ok := comments["comments"].([]any); ok && len(list) > 0 {
			if last, ok := list[len(list)-1].(map[string]any); ok {
				out.LastComment = flattenComment(last["body"])
			}
		}
	}
	return out
}

func addJiraAuth(req *http.Request, settings JiraSettings) {
	if settings.AuthMode == "basic" {
		encoded := base64.StdEncoding.EncodeToString([]byte(settings.Username + ":" + settings.Token))
		req.Header.Set("Authorization", "Basic "+encoded)
	} else {
		req.Header.Set("Authorization", "Bearer "+settings.Token)
	}
	req.Header.Set("Accept", "application/json")
}

func issueKey(j *JiraLink) string {
	if j == nil {
		return ""
	}
	if key := strings.TrimSpace(j.IssueKey); key != "" {
		return key
	}
	parsed, err := url.Parse(strings.TrimSpace(j.URL))
	if err == nil {
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		for i := range parts {
			if parts[i] == "browse" && i+1 < len(parts) {
				return strings.TrimSpace(parts[i+1])
			}
		}
		if len(parts) > 0 {
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	return ""
}

func jiraChangeNote(old, next JiraLink) string {
	var changes []string
	if old.Status != "" && next.Status != "" && old.Status != next.Status {
		changes = append(changes, fmt.Sprintf("status: %s → %s", old.Status, next.Status))
	}
	if old.Assignee != next.Assignee && (old.Assignee != "" || next.Assignee != "") {
		changes = append(changes, fmt.Sprintf("assignee: %s → %s", emptyDash(old.Assignee), emptyDash(next.Assignee)))
	}
	if old.LastComment != "" && next.LastComment != "" && old.LastComment != next.LastComment {
		changes = append(changes, "new Jira comment")
	}
	return strings.Join(changes, "; ")
}

func emptyDash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}

func displayName(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range []string{"displayName", "name", "emailAddress", "accountId"} {
		if s := asString(m[key]); s != "" {
			return s
		}
	}
	return ""
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func flattenComment(v any) string {
	if text, ok := v.(string); ok {
		return strings.TrimSpace(text)
	}

	var render func(any) string
	render = func(x any) string {
		switch t := x.(type) {
		case map[string]any:
			typ := asString(t["type"])
			if typ == "hardBreak" {
				return "\n"
			}
			if text, ok := t["text"].(string); ok {
				return text
			}
			content, _ := t["content"].([]any)
			parts := make([]string, 0, len(content))
			for _, child := range content {
				parts = append(parts, render(child))
			}
			joined := strings.Join(parts, "")
			switch typ {
			case "paragraph", "heading", "listItem", "blockquote":
				return strings.TrimSpace(joined) + "\n"
			case "bulletList", "orderedList", "doc":
				return strings.TrimSpace(joined) + "\n"
			default:
				return joined
			}
		case []any:
			parts := make([]string, 0, len(t))
			for _, child := range t {
				parts = append(parts, render(child))
			}
			return strings.Join(parts, "")
		default:
			return ""
		}
	}

	text := strings.ReplaceAll(render(v), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	clean := make([]string, 0, len(lines))
	lastBlank := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if !lastBlank && len(clean) > 0 {
				clean = append(clean, "")
			}
			lastBlank = true
			continue
		}
		clean = append(clean, line)
		lastBlank = false
	}
	return strings.TrimSpace(strings.Join(clean, "\n"))
}

func handleOpenURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	u := strings.TrimSpace(req.URL)
	parsed, err := url.Parse(u)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		writeError(w, http.StatusBadRequest, "invalid URL")
		return
	}
	if err := openBrowser(u); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func quadrantScores(q string) (importance, urgency int) {
	switch strings.ToUpper(q) {
	case "Q1":
		return 80, 80
	case "Q3":
		return 20, 80
	case "Q4":
		return 20, 20
	default:
		return 80, 20
	}
}

func normalizeTime(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UTC().Format(time.RFC3339)
	}
	return v
}

func cleanTags(tags []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tag)
	}
	return out
}

func cleanJira(j *JiraLink) *JiraLink {
	if j == nil {
		return nil
	}
	key := strings.TrimSpace(j.IssueKey)
	rawURL := strings.TrimSpace(j.URL)
	if key == "" && rawURL == "" {
		return nil
	}
	if key == "" {
		key = issueKey(&JiraLink{URL: rawURL})
	}
	return &JiraLink{IssueKey: key, URL: rawURL}
}

func mergeJiraLink(old, in *JiraLink) *JiraLink {
	clean := cleanJira(in)
	if clean == nil {
		return nil
	}
	sameTicket := false
	if old != nil {
		oldKey := issueKey(old)
		newKey := issueKey(clean)
		sameTicket = oldKey != "" && newKey != "" && strings.EqualFold(oldKey, newKey)
		if !sameTicket && strings.TrimSpace(old.URL) != "" && strings.TrimSpace(clean.URL) != "" {
			sameTicket = strings.EqualFold(strings.TrimSpace(old.URL), strings.TrimSpace(clean.URL))
		}
	}
	if sameTicket {
		clean.Status = old.Status
		clean.Assignee = old.Assignee
		clean.LastComment = old.LastComment
		clean.JiraUpdated = old.JiraUpdated
		clean.LastSyncedAt = old.LastSyncedAt
		clean.SyncError = old.SyncError
	}
	return clean
}

func nextOrder(items []ActionPoint) int {
	max := -1
	for _, ap := range items {
		if ap.QueueOrder > max {
			max = ap.QueueOrder
		}
	}
	return max + 1
}

func indexOf(items []ActionPoint, id string) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	return d.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func writeRunningInstance(path, appURL string) error {
	payload := map[string]any{
		"url":     appURL,
		"pid":     os.Getpid(),
		"version": "1.5.1",
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return writeBytesAtomic(path, raw, false)
}

func readRunningInstanceInfo(configPath string) (string, string) {
	path := filepath.Join(filepath.Dir(configPath), "running.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	var payload struct {
		URL     string `json:"url"`
		Version string `json:"version"`
	}
	if json.Unmarshal(raw, &payload) != nil {
		return "", ""
	}
	u := strings.TrimSpace(payload.URL)
	if !strings.HasPrefix(u, "http://127.0.0.1:") {
		return "", strings.TrimSpace(payload.Version)
	}

	client := &http.Client{Timeout: 350 * time.Millisecond}
	resp, err := client.Get(u + "/api/info")
	if err != nil {
		return "", strings.TrimSpace(payload.Version)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", strings.TrimSpace(payload.Version)
	}

	var info struct {
		Version string `json:"version"`
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if json.Unmarshal(body, &info) == nil && strings.TrimSpace(info.Version) != "" {
		return u, strings.TrimSpace(info.Version)
	}
	return u, strings.TrimSpace(payload.Version)
}

func openBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}
