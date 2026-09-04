export type Quadrant = 'Q1' | 'Q2' | 'Q3' | 'Q4';
export type QueueMode = 'manual' | 'important' | 'emergency';
export type AppView = 'home' | 'queue' | 'all' | 'daily' | 'calendar' | 'tags' | 'statistics';

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
}

export interface ActionPoint {
  id: string;
  title: string;
  details: string;
  importance: number;
  baseUrgency: number;
  deadline?: string;
  tags: string[];
  jira?: JiraLink;
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
}

export interface PlannerSettings {
  displayName: string;
  timezone: string;
  jiraBaseUrl: string;
  dataPath: string;
  autoSyncMinutes: number;
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
  settings: PlannerSettings;
}

export type APDraft = Omit<ActionPoint, 'id' | 'createdAt' | 'updatedAt' | 'queueOrder' | 'completedAt'>;
export type DailyCheckpointDraft = Pick<DailyCheckpoint, 'title' | 'enabled'>;
