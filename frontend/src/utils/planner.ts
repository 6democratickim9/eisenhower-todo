import type { ActionPoint, PlannerState, Quadrant } from '../types/models';

export const clamp = (value: number) => Math.max(0, Math.min(100, value));
const lerp = (a: number, b: number, t: number) => a + (b - a) * Math.max(0, Math.min(1, t));

export function deadlinePressure(deadline?: string): number {
  if (!deadline) return 0;
  const hours = (new Date(deadline).getTime() - Date.now()) / 3_600_000;
  if (!Number.isFinite(hours)) return 0;
  if (hours <= 0) return 80;
  if (hours >= 168) return 0;
  if (hours >= 72) return lerp(0, 15, (168 - hours) / 96);
  if (hours >= 24) return lerp(15, 35, (72 - hours) / 48);
  return lerp(35, 70, (24 - hours) / 24);
}

export function effectiveUrgency(ap: ActionPoint): number {
  return Math.round(clamp(ap.baseUrgency + deadlinePressure(ap.deadline)));
}

export function quadrantOf(ap: ActionPoint): Quadrant {
  const important = ap.importance >= 50;
  const urgent = effectiveUrgency(ap) >= 50;
  if (important && urgent) return 'Q1';
  if (important) return 'Q2';
  if (urgent) return 'Q3';
  return 'Q4';
}

export function patchForQuadrant(ap: ActionPoint, target: Quadrant): Partial<ActionPoint> {
  const important = target === 'Q1' || target === 'Q2';
  const urgent = target === 'Q1' || target === 'Q3';
  return {
    importance: important ? Math.max(60, ap.importance) : Math.min(40, ap.importance),
    baseUrgency: urgent ? Math.max(70, ap.baseUrgency) : Math.min(20, ap.baseUrgency),
  };
}

export const isToday = (iso?: string) => {
  if (!iso) return false;
  const a = new Date(iso);
  const b = new Date();
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
};

export const dayKey = (d = new Date()) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

export function orderedQueue(state: PlannerState): ActionPoint[] {
  const list = [...state.actionPoints];
  if (!list.length) return list;
  const oldest = [...list].sort((a,b)=>new Date(a.createdAt).getTime()-new Date(b.createdAt).getTime() || b.importance-a.importance || effectiveUrgency(b)-effectiveUrgency(a) || a.queueOrder-b.queueOrder)[0];
  const rest = list.filter(ap=>ap.id!==oldest.id);
  if (state.queueMode === 'important') {
    rest.sort((a, b) => b.importance - a.importance || effectiveUrgency(b) - effectiveUrgency(a) || a.queueOrder - b.queueOrder);
  } else if (state.queueMode === 'emergency') {
    rest.sort((a, b) => effectiveUrgency(b) - effectiveUrgency(a) || b.importance - a.importance || a.queueOrder - b.queueOrder);
  } else {
    rest.sort((a, b) => a.queueOrder - b.queueOrder);
  }
  return [oldest, ...rest];
}

export function shortDate(iso?: string): string {
  if (!iso) return 'No date';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return 'No date';
  if (isToday(iso)) return 'Today';
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}
