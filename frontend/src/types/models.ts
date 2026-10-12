export type Quadrant = 'Q1' | 'Q2' | 'Q3' | 'Q4';
export type QueueMode = 'manual' | 'important' | 'emergency';
export type AppView = 'home' | 'queue' | 'all' | 'daily' | 'calendar' | 'tags' | 'statistics' | 'archive' | 'team' | 'codeReview' | 'scrum';

export interface JiraLink {
  issueKey: string;
  url?: string;
  summary?: string;
  status?: string;
  statusCategory?: string;
  assignee?: string;
  lastComment?: string;
  jiraCreated?: string;
  jiraUpdated?: string;
  lastSyncedAt?: string;
  syncError?: string;
  paused?: boolean;
  staleDays?: number;
}

export interface ActionPoint {
  id: string;
  kind?: 'ap' | 'meeting' | 'commit';
  title: string;
  details: string;
  importance: number;
  baseUrgency: number;
  deadline?: string;
  tags: string[];
  jira?: JiraLink;
  gerritUrl?: string;
  queueOrder: number;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

export interface DailyCheckpoint {
  id: string;
  title: string;
  enabled: boolean;
  order: number;
  completedDates: string[];
  jiraIssueKey?: string;
  createdAt: string;
  updatedAt: string;
}

export type JiraWatchStatus = 'Open' | 'In Review' | 'Blocked' | 'Done';

export interface JiraWatchItem {
  id: string;
  issueKey: string;
  title: string;
  status: JiraWatchStatus;
  rawStatus?: string;
  statusCategory?: string;
  assignee?: string;
  lastComment?: string;
  createdAt?: string;
  updatedAt: string;
  lastSyncedAt?: string;
  url?: string;
  syncError?: string;
  paused?: boolean;
}

export interface PlannerSettings {
  displayName: string;
  timezone: string;
  jiraBaseUrl: string;
  dataPath: string;
  autoSyncMinutes: number;
  teamShareRoot?: string;
  jiraCommentOnComplete?: boolean;
  jiraCommentIncludeDetail?: boolean;
  jiraCommentOnCheckpoint?: boolean;
}

export interface JiraConfig {
  baseUrl: string;
  authMode: 'bearer' | 'basic';
  username: string;
  apiVersion: '2' | '3';
  tokenConfigured: boolean;
}

export interface JiraConfigUpdate {
  baseUrl?: string;
  authMode?: 'bearer' | 'basic';
  username?: string;
  apiVersion?: '2' | '3';
  token?: string;
}

export interface PlannerState {
  actionPoints: ActionPoint[];
  archive: ActionPoint[];
  dailyCheckpoints: DailyCheckpoint[];
  jiraWatch: JiraWatchItem[];
  queueMode: QueueMode;
  quickNote: string;
  settings: PlannerSettings;
}

export type APDraft = Omit<ActionPoint, 'id' | 'createdAt' | 'updatedAt' | 'queueOrder' | 'completedAt'>;
export type DailyCheckpointDraft = Pick<DailyCheckpoint, 'title' | 'enabled'>;

export interface TeamMember {
  name: string;
  actionPoints: ActionPoint[];
}

export type CodeReviewResponseStatus = 'done' | 'not_mine';
export interface CodeReviewResponse {
  requestId: string;
  reviewer: string;
  status: CodeReviewResponseStatus;
  respondedAt: string;
}

export interface CodeReviewRequest {
  id: string;
  url: string;
  title: string;
  description?: string;
  validation: string;
  approvalMode: 'each' | 'single';
  owner: string;
  createdAt: string;
  targetReviewCount: number;
  responses: CodeReviewResponse[];
  doneCount: number;
  skippedCount: number;
  complete: boolean;
  myResponse?: CodeReviewResponseStatus | '';
  mdcRcc?: string;
  mdcRgw?: string;
  mdcCommon?: string;
  archivedAt?: string;
}

export interface CodeReviewDraft {
  url: string;
  title: string;
  description: string;
  validation: string;
  approvalMode: 'each' | 'single';
}

export interface TeamState {
  enabled: boolean;
  root: string;
  currentUser: string;
  members: TeamMember[];
  codeReviews: CodeReviewRequest[];
  lastScannedAt: string;
}

export interface ScrumDashboardEntry {
  name: string;
  completedToday: ActionPoint[];
  activeJira: JiraLink[];
}

export interface ScrumDashboard {
  date: string;
  entries: ScrumDashboardEntry[];
  note: string;
  codeReviews: CodeReviewRequest[];
}
