import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { plannerApi } from '../api/plannerApi';
import type { APDraft, ActionPoint, DailyCheckpointDraft, JiraWatchItem, PlannerSettings, PlannerState, Quadrant, QueueMode } from '../types/models';
import { patchForQuadrant } from '../utils/planner';

type PlannerContextValue = {
  state: PlannerState | null;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  createAP: (draft: APDraft) => Promise<void>;
  updateAP: (id: string, patch: Partial<ActionPoint>) => Promise<void>;
  deleteAP: (id: string) => Promise<void>;
  completeAP: (id: string) => Promise<void>;
  restoreAP: (id: string) => Promise<void>;
  moveAPToQuadrant: (id: string, q: Quadrant) => Promise<void>;
  reorderQueue: (ids: string[], mode?: QueueMode) => Promise<void>;
  createDailyCheckpoint: (draft: DailyCheckpointDraft) => Promise<void>;
  updateDailyCheckpoint: (id: string, patch: Parameters<typeof plannerApi.updateDailyCheckpoint>[1]) => Promise<void>;
  deleteDailyCheckpoint: (id: string) => Promise<void>;
  toggleDailyCheckpoint: (id: string, dateKey: string) => Promise<void>;
  saveJiraWatch: (items: JiraWatchItem[]) => Promise<void>;
  syncJiraWatch: () => Promise<void>;
  saveSettings: (settings: PlannerSettings) => Promise<void>;
};

const PlannerContext = createContext<PlannerContextValue | null>(null);

export function PlannerProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<PlannerState | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const run = useCallback(async (task: () => Promise<void>) => {
    try { setError(null); await task(); }
    catch (e) { setError(e instanceof Error ? e.message : String(e)); }
  }, []);

  const refresh = useCallback(async () => {
    setLoading(true);
    try { setState(await plannerApi.getState()); setError(null); }
    catch (e) { setError(e instanceof Error ? e.message : String(e)); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { void refresh(); }, [refresh]);

  const actions = useMemo(() => ({
    createAP: (draft: APDraft) => run(async () => { await plannerApi.createAP(draft); await refresh(); }),
    updateAP: (id: string, patch: Partial<ActionPoint>) => run(async () => { await plannerApi.updateAP(id, patch); await refresh(); }),
    deleteAP: (id: string) => run(async () => { await plannerApi.deleteAP(id); await refresh(); }),
    completeAP: (id: string) => run(async () => { await plannerApi.completeAP(id); await refresh(); }),
    restoreAP: (id: string) => run(async () => { await plannerApi.restoreAP(id); await refresh(); }),
    moveAPToQuadrant: (id: string, q: Quadrant) => run(async () => {
      const ap = state?.actionPoints.find((x) => x.id === id);
      if (!ap) return;
      await plannerApi.updateAP(id, patchForQuadrant(ap, q));
      await refresh();
    }),
    reorderQueue: (ids: string[], mode?: QueueMode) => run(async () => { await plannerApi.reorderQueue(ids, mode); await refresh(); }),
    createDailyCheckpoint: (draft: DailyCheckpointDraft) => run(async () => { await plannerApi.createDailyCheckpoint(draft); await refresh(); }),
    updateDailyCheckpoint: (id: string, patch: Parameters<typeof plannerApi.updateDailyCheckpoint>[1]) => run(async () => { await plannerApi.updateDailyCheckpoint(id, patch); await refresh(); }),
    deleteDailyCheckpoint: (id: string) => run(async () => { await plannerApi.deleteDailyCheckpoint(id); await refresh(); }),
    toggleDailyCheckpoint: (id: string, dateKey: string) => run(async () => { await plannerApi.toggleDailyCheckpoint(id, dateKey); await refresh(); }),
    saveJiraWatch: (items: JiraWatchItem[]) => run(async () => { await plannerApi.saveJiraWatch(items); await refresh(); }),
    syncJiraWatch: () => run(async () => { await plannerApi.syncJiraWatch(); await refresh(); }),
    saveSettings: (settings: PlannerSettings) => run(async () => { await plannerApi.saveSettings(settings); await refresh(); }),
  }), [refresh, run, state]);

  return <PlannerContext.Provider value={{ state, loading, error, refresh, ...actions }}>{children}</PlannerContext.Provider>;
}

export function usePlanner() {
  const value = useContext(PlannerContext);
  if (!value) throw new Error('usePlanner must be used inside PlannerProvider');
  return value;
}
