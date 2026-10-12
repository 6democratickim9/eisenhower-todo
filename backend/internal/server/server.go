package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"eisenday/internal/planner"
	"eisenday/internal/webui"
)

type API struct{ Store *planner.Store }

func NewHandler(store *planner.Store) http.Handler {
	a := &API{Store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/state", a.getState)
	mux.HandleFunc("POST /api/ap", a.createAP)
	mux.HandleFunc("PUT /api/ap/{id}", a.updateAP)
	mux.HandleFunc("DELETE /api/ap/{id}", a.deleteAP)
	mux.HandleFunc("POST /api/ap/{id}/complete", a.completeAP)
	mux.HandleFunc("POST /api/archive/{id}/restore", a.restoreAP)
	mux.HandleFunc("POST /api/queue", a.reorderQueue)
	mux.HandleFunc("POST /api/daily-checkpoints", a.createDaily)
	mux.HandleFunc("PUT /api/daily-checkpoints/{id}", a.updateDaily)
	mux.HandleFunc("DELETE /api/daily-checkpoints/{id}", a.deleteDaily)
	mux.HandleFunc("POST /api/daily-checkpoints/{id}/toggle", a.toggleDaily)
	mux.HandleFunc("PUT /api/jira/watch", a.saveWatch)
	mux.HandleFunc("POST /api/jira/watch/sync", a.syncWatch)
	mux.HandleFunc("PUT /api/note", a.saveQuickNote)
	mux.HandleFunc("GET /api/jira/config", a.getJiraConfig)
	mux.HandleFunc("PUT /api/jira/config", a.saveJiraConfig)
	mux.HandleFunc("POST /api/jira/test", a.testJira)
	mux.HandleFunc("PUT /api/settings", a.saveSettings)
	mux.HandleFunc("GET /api/info", a.info)
	mux.HandleFunc("GET /api/team/state", a.getTeamState)
	mux.HandleFunc("POST /api/code-review/requests", a.createCodeReview)
	mux.HandleFunc("POST /api/code-review/{id}/response", a.respondCodeReview)
	mux.HandleFunc("PUT /api/code-review/{id}/mdc-version", a.setMdcVersion)
	mux.HandleFunc("PUT /api/code-review/{id}", a.updateCodeReview)
	mux.HandleFunc("POST /api/code-review/{id}/archive", a.archiveCodeReview)
	mux.HandleFunc("GET /api/scrum/dashboard", a.scrumDashboard)
	mux.HandleFunc("PUT /api/scrum/note", a.saveScrumNote)
	mux.HandleFunc("POST /api/jira/create-issue", a.createJiraIssue)
	mux.Handle("/", webui.Handler())
	return middleware(mux)
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (strings.HasPrefix(origin, "http://127.0.0.1:") || strings.HasPrefix(origin, "http://localhost:")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func decode(r *http.Request, v any) error {
	d := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 2<<20))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "not found") {
		status = http.StatusNotFound
	}
	var inUse planner.DataFileInUseError
	if errors.As(err, &inUse) {
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"detail": msg})
}

func (a *API) getState(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, a.Store.State()) }
func (a *API) saveQuickNote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Note string `json:"note"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	if err := a.Store.SaveQuickNote(body.Note); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (a *API) createAP(w http.ResponseWriter, r *http.Request) {
	var d planner.APDraft
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(w, err)
		return
	}
	v, err := a.Store.CreateAP(d)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 201, v)
}
func (a *API) updateAP(w http.ResponseWriter, r *http.Request) {
	var p map[string]any
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, err)
		return
	}
	v, err := a.Store.UpdateAP(r.PathValue("id"), p)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, v)
}
func (a *API) deleteAP(w http.ResponseWriter, r *http.Request) {
	if err := a.Store.DeleteAP(r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (a *API) completeAP(w http.ResponseWriter, r *http.Request) {
	if err := a.Store.CompleteAP(r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (a *API) restoreAP(w http.ResponseWriter, r *http.Request) {
	if err := a.Store.RestoreAP(r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (a *API) reorderQueue(w http.ResponseWriter, r *http.Request) {
	var q struct {
		IDs  []string           `json:"ids"`
		Mode *planner.QueueMode `json:"mode,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		writeError(w, err)
		return
	}
	if err := a.Store.ReorderQueue(q.IDs, q.Mode); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (a *API) createDaily(w http.ResponseWriter, r *http.Request) {
	var d planner.DailyCheckpointDraft
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(w, err)
		return
	}
	v, err := a.Store.CreateDaily(d)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 201, v)
}
func (a *API) updateDaily(w http.ResponseWriter, r *http.Request) {
	var p map[string]any
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, err)
		return
	}
	v, err := a.Store.UpdateDaily(r.PathValue("id"), p)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, v)
}
func (a *API) deleteDaily(w http.ResponseWriter, r *http.Request) {
	if err := a.Store.DeleteDaily(r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (a *API) toggleDaily(w http.ResponseWriter, r *http.Request) {
	var p struct {
		DateKey string `json:"dateKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, err)
		return
	}
	v, err := a.Store.ToggleDaily(r.PathValue("id"), p.DateKey)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, v)
}
func (a *API) saveWatch(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Items []planner.JiraWatchItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, err)
		return
	}
	if err := a.Store.SaveJiraWatch(p.Items); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (a *API) syncWatch(w http.ResponseWriter, r *http.Request) {
	v, err := a.Store.SyncJiraWatch()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, v)
}
func (a *API) getJiraConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, a.Store.JiraPublicConfig())
}
func (a *API) saveJiraConfig(w http.ResponseWriter, r *http.Request) {
	var p planner.JiraConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, err)
		return
	}
	v, err := a.Store.SaveJiraConfig(p)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, v)
}
func (a *API) testJira(w http.ResponseWriter, r *http.Request) {
	msg, err := a.Store.TestJira()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "message": msg})
}
func (a *API) saveSettings(w http.ResponseWriter, r *http.Request) {
	var p planner.PlannerSettings
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, err)
		return
	}
	v, err := a.Store.SaveSettings(p)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (a *API) getTeamState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, a.Store.TeamState())
}
func (a *API) createCodeReview(w http.ResponseWriter, r *http.Request) {
	var d planner.CodeReviewDraft
	if err := decode(r, &d); err != nil {
		writeError(w, err)
		return
	}
	v, err := a.Store.CreateCodeReview(d)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 201, v)
}
func (a *API) respondCodeReview(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status string `json:"status"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	if err := a.Store.RespondCodeReview(r.PathValue("id"), body.Status); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}


func (a *API) updateCodeReview(w http.ResponseWriter, r *http.Request) {
	var patch map[string]any
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, err)
		return
	}
	v, err := a.Store.UpdateCodeReview(r.PathValue("id"), patch)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, v)
}
func (a *API) setMdcVersion(w http.ResponseWriter, r *http.Request) {
	var body planner.MdcVersions
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	if err := a.Store.SetMdcVersions(r.PathValue("id"), body); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) archiveCodeReview(w http.ResponseWriter, r *http.Request) {
	if err := a.Store.ArchiveCodeReview(r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) createJiraIssue(w http.ResponseWriter, r *http.Request) {
	var input planner.CreateIssueInput
	if err := decode(r, &input); err != nil {
		writeError(w, err)
		return
	}
	key, issueURL, err := a.Store.JiraClient.CreateIssue(a.Store.JiraConfig, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 201, map[string]string{"key": key, "url": issueURL})
}
func (a *API) scrumDashboard(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	writeJSON(w, 200, a.Store.ScrumDashboard(date))
}

func (a *API) saveScrumNote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Date string `json:"date"`
		Note string `json:"note"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, err)
		return
	}
	if err := a.Store.SaveScrumNote(body.Date, body.Note); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) info(w http.ResponseWriter, r *http.Request) {
	s := a.Store.State()
	writeJSON(w, 200, map[string]any{"version": "2.1.0-go", "backend": "go", "dataPath": s.Settings.DataPath, "configPath": a.Store.Paths.ConfigPath, "jiraSettingsPath": a.Store.Paths.JiraPath})
}

func StartAutoSync(store *planner.Store, stop <-chan struct{}) {
	go func() {
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-stop:
			return
		}
		for {
			mins := store.AutoSyncMinutes()
			if store.HasJiraWatch() {
				if _, err := store.SyncJiraWatch(); err != nil {
					log.Printf("jira auto-sync: %v", err)
				}
			}
			timer.Reset(time.Duration(mins) * time.Minute)
			select {
			case <-timer.C:
			case <-stop:
				return
			}
		}
	}()
}
