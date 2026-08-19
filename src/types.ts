export type Quadrant = "Q1" | "Q2" | "Q3" | "Q4";

export interface JiraLink {
  issueKey?: string;
  url: string;
}

export interface HistoryEntry {
  at: string;
  action: string;
  note?: string;
}

export interface ActionPoint {
  id: string;
  title: string;
  details?: string;
  importance: number;
  baseUrgency: number;
  deadline?: string;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
  queueOrder?: number;
  jira?: JiraLink;
  history: HistoryEntry[];
}

export interface PlannerData {
  version: 1;
  timezone: string;
  actionPoints: ActionPoint[];
  archive: ActionPoint[];
}
