import { ActionPoint, Quadrant } from "./types";

export function calculateUrgency(ap: ActionPoint, now = new Date()): number {
  let urgency = clamp(ap.baseUrgency);

  if (!ap.deadline) return urgency;

  const deadline = new Date(ap.deadline);
  const diffHours = (deadline.getTime() - now.getTime()) / 3_600_000;

  let deadlinePressure = 0;
  if (diffHours <= 0) deadlinePressure = 80;
  else if (diffHours <= 24) deadlinePressure = 50;
  else if (diffHours <= 72) deadlinePressure = 30;
  else if (diffHours <= 168) deadlinePressure = 15;

  return clamp(urgency + deadlinePressure);
}

export function getQuadrant(ap: ActionPoint, now = new Date()): Quadrant {
  const important = ap.importance >= 50;
  const urgent = calculateUrgency(ap, now) >= 50;

  if (important && urgent) return "Q1";
  if (important && !urgent) return "Q2";
  if (!important && urgent) return "Q3";
  return "Q4";
}

export function baseUrgencyForQuadrant(q: Quadrant): number {
  return q === "Q1" || q === "Q3" ? 70 : 20;
}

export function importanceForQuadrant(q: Quadrant): number {
  return q === "Q1" || q === "Q2" ? 80 : 20;
}

function clamp(value: number): number {
  return Math.max(0, Math.min(100, value));
}
