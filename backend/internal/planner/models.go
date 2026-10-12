package planner

type QueueMode string

const (
	QueueManual    QueueMode = "manual"
	QueueImportant QueueMode = "important"
	QueueEmergency QueueMode = "emergency"
)

type JiraLink struct {
	IssueKey       string `json:"issueKey"`
	URL            string `json:"url,omitempty"`
	Summary        string `json:"summary,omitempty"`
	Status         string `json:"status,omitempty"`
	StatusCategory string `json:"statusCategory,omitempty"`
	Assignee       string `json:"assignee,omitempty"`
	LastComment    string `json:"lastComment,omitempty"`
	JiraCreated    string `json:"jiraCreated,omitempty"`
	JiraUpdated    string `json:"jiraUpdated,omitempty"`
	LastSyncedAt   string `json:"lastSyncedAt,omitempty"`
	SyncError      string `json:"syncError,omitempty"`
	Paused         bool   `json:"paused,omitempty"`
	StaleDays      int    `json:"staleDays,omitempty"`
}

type ActionPoint struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind,omitempty"` // ap | meeting | commit
	Title       string    `json:"title"`
	Details     string    `json:"details"`
	Importance  int       `json:"importance"`
	BaseUrgency int       `json:"baseUrgency"`
	Deadline    string    `json:"deadline,omitempty"`
	Tags        []string  `json:"tags"`
	Jira        *JiraLink `json:"jira,omitempty"`
	GerritURL   string    `json:"gerritUrl,omitempty"`
	QueueOrder  int       `json:"queueOrder"`
	CreatedAt   string    `json:"createdAt"`
	UpdatedAt   string    `json:"updatedAt"`
	CompletedAt string    `json:"completedAt,omitempty"`
}

type APDraft struct {
	Title       string    `json:"title"`
	Kind        string    `json:"kind,omitempty"`
	Details     string    `json:"details"`
	Importance  *int      `json:"importance"`
	BaseUrgency *int      `json:"baseUrgency"`
	Deadline    string    `json:"deadline,omitempty"`
	Tags        []string  `json:"tags"`
	GerritURL   string    `json:"gerritUrl,omitempty"`
	Jira        *JiraLink `json:"jira,omitempty"`
}

type DailyCheckpoint struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Enabled        bool     `json:"enabled"`
	Order          int      `json:"order"`
	CompletedDates []string `json:"completedDates"`
	JiraIssueKey   string   `json:"jiraIssueKey,omitempty"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}

type DailyCheckpointDraft struct {
	Title   string `json:"title"`
	Enabled *bool  `json:"enabled"`
}

type JiraWatchItem struct {
	ID             string `json:"id"`
	IssueKey       string `json:"issueKey"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	RawStatus      string `json:"rawStatus,omitempty"`
	StatusCategory string `json:"statusCategory,omitempty"`
	Assignee       string `json:"assignee,omitempty"`
	LastComment    string `json:"lastComment,omitempty"`
	CreatedAt      string `json:"createdAt,omitempty"`
	UpdatedAt      string `json:"updatedAt"`
	LastSyncedAt   string `json:"lastSyncedAt,omitempty"`
	URL            string `json:"url,omitempty"`
	SyncError      string `json:"syncError,omitempty"`
	Paused         bool   `json:"paused,omitempty"`
}

type PlannerSettings struct {
	DisplayName              string `json:"displayName"`
	Timezone                 string `json:"timezone"`
	JiraBaseURL              string `json:"jiraBaseUrl"`
	DataPath                 string `json:"dataPath"`
	AutoSyncMinutes          int    `json:"autoSyncMinutes"`
	TeamShareRoot            string `json:"teamShareRoot,omitempty"`
	JiraCommentOnComplete    bool   `json:"jiraCommentOnComplete"`
	JiraCommentIncludeDetail bool   `json:"jiraCommentIncludeDetail"`
	JiraCommentOnCheckpoint  bool   `json:"jiraCommentOnCheckpoint"`
}

type PlannerData struct {
	Version          int               `json:"version"`
	ActionPoints     []ActionPoint     `json:"actionPoints"`
	Archive          []ActionPoint     `json:"archive"`
	DailyCheckpoints []DailyCheckpoint `json:"dailyCheckpoints"`
	JiraWatch        []JiraWatchItem   `json:"jiraWatch"`
	QueueMode        QueueMode         `json:"queueMode"`
	QuickNote        string            `json:"quickNote,omitempty"`
}

type PlannerState struct {
	ActionPoints     []ActionPoint     `json:"actionPoints"`
	Archive          []ActionPoint     `json:"archive"`
	DailyCheckpoints []DailyCheckpoint `json:"dailyCheckpoints"`
	JiraWatch        []JiraWatchItem   `json:"jiraWatch"`
	QueueMode        QueueMode         `json:"queueMode"`
	QuickNote        string            `json:"quickNote"`
	Settings         PlannerSettings   `json:"settings"`
}

type JiraConfigPublic struct {
	BaseURL         string `json:"baseUrl"`
	AuthMode        string `json:"authMode"`
	Username        string `json:"username"`
	APIVersion      string `json:"apiVersion"`
	TokenConfigured bool   `json:"tokenConfigured"`
}

type JiraConfigUpdate struct {
	BaseURL    *string `json:"baseUrl,omitempty"`
	AuthMode   *string `json:"authMode,omitempty"`
	Username   *string `json:"username,omitempty"`
	APIVersion *string `json:"apiVersion,omitempty"`
	Token      *string `json:"token,omitempty"`
}

type JiraSecretConfig struct {
	BaseURL    string `json:"baseUrl"`
	AuthMode   string `json:"authMode"`
	Username   string `json:"username"`
	Token      string `json:"token"`
	APIVersion string `json:"apiVersion"`
}

type JiraIssueData struct {
	IssueKey       string
	Title          string
	RawStatus      string
	StatusCategory string
	Assignee       string
	LastComment    string
	CreatedAt      string
	UpdatedAt      string
	URL            string
}
