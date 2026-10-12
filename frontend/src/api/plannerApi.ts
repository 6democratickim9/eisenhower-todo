import type {
  APDraft,
  ActionPoint,
  DailyCheckpoint,
  DailyCheckpointDraft,
  JiraConfig,
  JiraConfigUpdate,
  JiraWatchItem,
  PlannerSettings,
  PlannerState,
  QueueMode,
  TeamState,
  CodeReviewDraft,
  CodeReviewRequest,
  CodeReviewResponseStatus,
  ScrumDashboard,
} from '../types/models';
import { seedState } from '../data/seed';

export interface PlannerApi {
  getState(): Promise<PlannerState>;
  createAP(draft: APDraft): Promise<ActionPoint>;
  updateAP(id: string, patch: Partial<ActionPoint>): Promise<ActionPoint>;
  deleteAP(id: string): Promise<void>;
  completeAP(id: string): Promise<void>;
  restoreAP(id: string): Promise<void>;
  reorderQueue(ids: string[], mode?: QueueMode): Promise<void>;
  createDailyCheckpoint(draft: DailyCheckpointDraft): Promise<DailyCheckpoint>;
  updateDailyCheckpoint(id: string, patch: Partial<DailyCheckpoint>): Promise<DailyCheckpoint>;
  deleteDailyCheckpoint(id: string): Promise<void>;
  toggleDailyCheckpoint(id: string, dateKey: string): Promise<DailyCheckpoint>;
  saveJiraWatch(items: JiraWatchItem[]): Promise<void>;
  syncJiraWatch(): Promise<JiraWatchItem[]>;
  saveQuickNote(note: string): Promise<void>;
  saveSettings(settings: PlannerSettings): Promise<PlannerSettings>;
  getJiraConfig(): Promise<JiraConfig>;
  saveJiraConfig(settings: JiraConfigUpdate): Promise<JiraConfig>;
  testJira(): Promise<{ok: boolean; message: string}>;
  getTeamState(): Promise<TeamState>;
  createCodeReview(draft: CodeReviewDraft): Promise<CodeReviewRequest>;
  respondCodeReview(id: string, status: CodeReviewResponseStatus): Promise<void>;
  setMdcVersions(id: string, versions: {mdcRcc:string;mdcRgw:string;mdcCommon:string}): Promise<void>;
  archiveCodeReview(id: string): Promise<void>;
  updateCodeReview(id: string, patch: Partial<CodeReviewDraft>): Promise<CodeReviewRequest>;
  getScrumDashboard(date?: string): Promise<ScrumDashboard>;
  saveScrumNote(date: string, note: string): Promise<void>;
  createJiraIssue(input: {projectKey:string;summary:string;description:string;issueType:string;linkedIssue:string}): Promise<{key:string;url:string}>;
}

const STORAGE_KEY = 'eisen-day-react-state-v1';
const JIRA_CONFIG_KEY = 'eisen-day-react-jira-config-v1';
const clone = <T,>(value: T): T => JSON.parse(JSON.stringify(value)) as T;
const wait = (ms = 60) => new Promise((resolve) => setTimeout(resolve, ms));

class LocalPlannerApi implements PlannerApi {
  private read(): PlannerState {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      const initial = clone(seedState);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(initial));
      return initial;
    }
    try { return JSON.parse(raw) as PlannerState; }
    catch {
      const fallback = clone(seedState);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(fallback));
      return fallback;
    }
  }
  private write(state: PlannerState) { localStorage.setItem(STORAGE_KEY, JSON.stringify(state)); }
  async getState(){ await wait(); return clone(this.read()); }
  async createAP(draft: APDraft){
    const state=this.read(); const now=new Date().toISOString();
    const ap:ActionPoint={...draft,id:crypto.randomUUID(),queueOrder:state.actionPoints.length,createdAt:now,updatedAt:now};
    state.actionPoints.push(ap); this.write(state); return clone(ap);
  }
  async updateAP(id:string,patch:Partial<ActionPoint>){
    const state=this.read(); const index=state.actionPoints.findIndex(x=>x.id===id);
    if(index<0) throw new Error('AP not found');
    state.actionPoints[index]={...state.actionPoints[index],...patch,id,updatedAt:new Date().toISOString()};
    this.write(state); return clone(state.actionPoints[index]);
  }
  async deleteAP(id:string){const state=this.read();state.actionPoints=state.actionPoints.filter(x=>x.id!==id);this.write(state);}
  async completeAP(id:string){
    const state=this.read(); const index=state.actionPoints.findIndex(x=>x.id===id); if(index<0)return;
    const [ap]=state.actionPoints.splice(index,1); ap.completedAt=new Date().toISOString(); ap.updatedAt=ap.completedAt; state.archive.unshift(ap); this.write(state);
  }
  async restoreAP(id:string){
    const state=this.read();const index=state.archive.findIndex(x=>x.id===id);if(index<0)return;
    const [ap]=state.archive.splice(index,1);delete ap.completedAt;ap.updatedAt=new Date().toISOString();ap.queueOrder=state.actionPoints.length;state.actionPoints.push(ap);this.write(state);
  }
  async reorderQueue(ids:string[],mode?:QueueMode){
    const state=this.read(); const order=new Map(ids.map((id,i)=>[id,i])); state.actionPoints=state.actionPoints.map(ap=>({...ap,queueOrder:order.get(ap.id)??ap.queueOrder})); if(mode)state.queueMode=mode; this.write(state);
  }
  async createDailyCheckpoint(draft:DailyCheckpointDraft){
    const state=this.read();const now=new Date().toISOString();const item:DailyCheckpoint={id:crypto.randomUUID(),title:draft.title,enabled:draft.enabled,order:state.dailyCheckpoints.length,completedDates:[],createdAt:now,updatedAt:now};state.dailyCheckpoints.push(item);this.write(state);return clone(item);
  }
  async updateDailyCheckpoint(id:string,patch:Partial<DailyCheckpoint>){
    const state=this.read();const i=state.dailyCheckpoints.findIndex(x=>x.id===id);if(i<0)throw new Error('Daily checkpoint not found');state.dailyCheckpoints[i]={...state.dailyCheckpoints[i],...patch,id,updatedAt:new Date().toISOString()};this.write(state);return clone(state.dailyCheckpoints[i]);
  }
  async deleteDailyCheckpoint(id:string){const state=this.read();state.dailyCheckpoints=state.dailyCheckpoints.filter(x=>x.id!==id);this.write(state);}
  async toggleDailyCheckpoint(id:string,dateKey:string){
    const state=this.read();const i=state.dailyCheckpoints.findIndex(x=>x.id===id);if(i<0)throw new Error('Daily checkpoint not found');const item=state.dailyCheckpoints[i];if(!item.enabled)throw new Error('Disabled checkpoint');item.completedDates=item.completedDates.includes(dateKey)?item.completedDates.filter(x=>x!==dateKey):[...item.completedDates,dateKey];item.updatedAt=new Date().toISOString();this.write(state);return clone(item);
  }
  async saveJiraWatch(items:JiraWatchItem[]){const state=this.read();state.jiraWatch=clone(items.filter(x=>x.status!=='Done'));this.write(state);}
  async syncJiraWatch(){const state=this.read();const now=new Date().toISOString();state.jiraWatch=state.jiraWatch.filter(x=>x.status!=='Done').map(x=>({...x,lastSyncedAt:now}));this.write(state);await wait(220);return clone(state.jiraWatch);}
  async saveQuickNote(note:string){const state=this.read();state.quickNote=note;this.write(state);}
  async saveSettings(settings:PlannerSettings){const state=this.read();state.settings=clone(settings);this.write(state);return clone(settings);}
  async getJiraConfig(): Promise<JiraConfig>{
    const raw=localStorage.getItem(JIRA_CONFIG_KEY);
    if(raw){try{return JSON.parse(raw) as JiraConfig}catch{/* noop */}}
    const fallback: JiraConfig = {
      baseUrl: this.read().settings.jiraBaseUrl,
      authMode: 'bearer',
      username: '',
      apiVersion: '2',
      tokenConfigured: false,
    };
    return fallback;
  }
  async saveJiraConfig(settings:JiraConfigUpdate){
    const current=await this.getJiraConfig();
    const {token,...safe}=settings;
    const next:JiraConfig={...current,...safe,tokenConfigured:Boolean(token)||current.tokenConfigured};
    localStorage.setItem(JIRA_CONFIG_KEY,JSON.stringify(next));return clone(next);
  }
  async testJira(){await wait(180);return {ok:false,message:'Local mode에서는 실제 Jira 연결 테스트를 하지 않습니다.'};}
  async getTeamState(): Promise<TeamState>{return {enabled:false,root:'',currentUser:this.read().settings.displayName,members:[],codeReviews:[],lastScannedAt:new Date().toISOString()};}
  async createCodeReview(_draft:CodeReviewDraft): Promise<CodeReviewRequest>{throw new Error('Code Review sharing requires the packaged local backend.');}
  async respondCodeReview(_id:string,_status:CodeReviewResponseStatus): Promise<void>{throw new Error('Code Review sharing requires the packaged local backend.');}
  async setMdcVersions(_id:string,_versions:{mdcRcc:string;mdcRgw:string;mdcCommon:string}): Promise<void>{throw new Error('Code Review sharing requires the packaged local backend.');}
  async archiveCodeReview(_id:string): Promise<void>{throw new Error('Code Review sharing requires the packaged local backend.');}
  async updateCodeReview(_id:string,_patch:Partial<CodeReviewDraft>): Promise<CodeReviewRequest>{throw new Error('Code Review sharing requires the packaged local backend.');}
  async getScrumDashboard(_date?:string): Promise<ScrumDashboard>{return {date:new Date().toISOString().slice(0,10),entries:[],note:'',codeReviews:[]};}
  async saveScrumNote(_date:string,_note:string): Promise<void>{throw new Error('Scrum note requires the packaged local backend.');}
  async createJiraIssue(_input:{projectKey:string;summary:string;description:string;issueType:string;linkedIssue:string}): Promise<{key:string;url:string}>{throw new Error('Jira issue creation requires the packaged local backend.');}
}

async function request<T>(path:string, options?:RequestInit):Promise<T>{
  // Production uses same-origin URLs so the packaged app can choose any free localhost port.
  const base=import.meta.env.VITE_API_BASE_URL ?? '';
  const response=await fetch(`${base}${path}`,{headers:{'Content-Type':'application/json'},...options});
  if(!response.ok){
    const raw=await response.text();
    try{const body=JSON.parse(raw) as {detail?:string};throw new Error(body.detail||raw||`HTTP ${response.status}`)}
    catch(error){if(error instanceof SyntaxError)throw new Error(raw||`HTTP ${response.status}`);throw error}
  }
  if(response.status===204) return undefined as T;
  return response.json() as Promise<T>;
}

class HttpPlannerApi implements PlannerApi {
  getState(){return request<PlannerState>('/api/state');}
  createAP(draft:APDraft){return request<ActionPoint>('/api/ap',{method:'POST',body:JSON.stringify(draft)});}
  updateAP(id:string,patch:Partial<ActionPoint>){return request<ActionPoint>(`/api/ap/${id}`,{method:'PUT',body:JSON.stringify(patch)});}
  deleteAP(id:string){return request<void>(`/api/ap/${id}`,{method:'DELETE'});}
  completeAP(id:string){return request<void>(`/api/ap/${id}/complete`,{method:'POST'});}
  restoreAP(id:string){return request<void>(`/api/archive/${id}/restore`,{method:'POST'});}
  reorderQueue(ids:string[],mode?:QueueMode){return request<void>('/api/queue',{method:'POST',body:JSON.stringify({ids,mode})});}
  createDailyCheckpoint(draft:DailyCheckpointDraft){return request<DailyCheckpoint>('/api/daily-checkpoints',{method:'POST',body:JSON.stringify(draft)});}
  updateDailyCheckpoint(id:string,patch:Partial<DailyCheckpoint>){return request<DailyCheckpoint>(`/api/daily-checkpoints/${id}`,{method:'PUT',body:JSON.stringify(patch)});}
  deleteDailyCheckpoint(id:string){return request<void>(`/api/daily-checkpoints/${id}`,{method:'DELETE'});}
  toggleDailyCheckpoint(id:string,dateKey:string){return request<DailyCheckpoint>(`/api/daily-checkpoints/${id}/toggle`,{method:'POST',body:JSON.stringify({dateKey})});}
  saveJiraWatch(items:JiraWatchItem[]){return request<void>('/api/jira/watch',{method:'PUT',body:JSON.stringify({items})});}
  syncJiraWatch(){return request<JiraWatchItem[]>('/api/jira/watch/sync',{method:'POST'});}
  async saveQuickNote(note:string){
    try {
      await request<void>('/api/note',{method:'PUT',body:JSON.stringify({note})});
    } catch (error) {
      const message=error instanceof Error?error.message:String(error);
      if(/405|method not allowed/i.test(message)) throw new Error('Note 저장 API를 지원하지 않는 이전 Go 백엔드가 실행 중입니다. 기존 Eisen Day 백엔드를 종료하고 최신 버전을 다시 실행해 주세요.');
      throw error;
    }
  }
  saveSettings(settings:PlannerSettings){return request<PlannerSettings>('/api/settings',{method:'PUT',body:JSON.stringify(settings)});}
  getJiraConfig(){return request<JiraConfig>('/api/jira/config');}
  saveJiraConfig(settings:JiraConfigUpdate){return request<JiraConfig>('/api/jira/config',{method:'PUT',body:JSON.stringify(settings)});}
  testJira(){return request<{ok:boolean;message:string}>('/api/jira/test',{method:'POST'});}
  getTeamState(){return request<TeamState>('/api/team/state');}
  createCodeReview(draft:CodeReviewDraft){return request<CodeReviewRequest>('/api/code-review/requests',{method:'POST',body:JSON.stringify(draft)});}
  respondCodeReview(id:string,status:CodeReviewResponseStatus){return request<void>(`/api/code-review/${id}/response`,{method:'POST',body:JSON.stringify({status})});}
  setMdcVersions(id:string,versions:{mdcRcc:string;mdcRgw:string;mdcCommon:string}){return request<void>(`/api/code-review/${id}/mdc-version`,{method:'PUT',body:JSON.stringify(versions)});}
  archiveCodeReview(id:string){return request<void>(`/api/code-review/${id}/archive`,{method:'POST'});}
  updateCodeReview(id:string,patch:Partial<CodeReviewDraft>){return request<CodeReviewRequest>(`/api/code-review/${id}`,{method:'PUT',body:JSON.stringify(patch)});}
  getScrumDashboard(date?:string){return request<ScrumDashboard>(`/api/scrum/dashboard${date?`?date=${date}`:''}`);}
  saveScrumNote(date:string,note:string){return request<void>('/api/scrum/note',{method:'PUT',body:JSON.stringify({date,note})});}
  createJiraIssue(input:{projectKey:string;summary:string;description:string;issueType:string;linkedIssue:string}){return request<{key:string;url:string}>('/api/jira/create-issue',{method:'POST',body:JSON.stringify(input)});}
}

const mode = import.meta.env.VITE_API_MODE || (import.meta.env.DEV ? 'local' : 'http');
export const plannerApi:PlannerApi = mode === 'local' ? new LocalPlannerApi() : new HttpPlannerApi();
