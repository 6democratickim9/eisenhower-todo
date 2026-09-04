import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { usePlanner } from '../state/PlannerProvider';
import { plannerApi } from '../api/plannerApi';
import type { APDraft, ActionPoint, DailyCheckpoint, JiraConfig, JiraWatchItem, PlannerSettings } from '../types/models';
import { JiraIssueLink } from './JiraIssueLink';

function DrawerShell({title,subtitle,onClose,children,wide=false}:{title:string;subtitle?:string;onClose:()=>void;children:ReactNode;wide?:boolean}){
  useEffect(()=>{const h=(e:KeyboardEvent)=>{if(e.key==='Escape')onClose()};window.addEventListener('keydown',h);return()=>window.removeEventListener('keydown',h)},[onClose]);
  return <><button className="drawer-backdrop" onClick={onClose} aria-label="닫기"/><aside className={`drawer ${wide?'wide':''}`}><div className="drawer-head"><div><span className="drawer-kicker">EISEN / DAY</span><h2>{title}</h2>{subtitle&&<p>{subtitle}</p>}</div><button className="drawer-close" onClick={onClose}>×</button></div>{children}</aside></>;
}

const emptyDraft:APDraft={title:'',details:'',importance:70,baseUrgency:40,tags:[]};
export function APDrawer({ap,onClose}:{ap?:ActionPoint;onClose:()=>void}){
  const {state,createAP,updateAP,deleteAP,completeAP}=usePlanner();
  const [form,setForm]=useState<APDraft>(()=>ap?{title:ap.title,details:ap.details,importance:ap.importance,baseUrgency:ap.baseUrgency,deadline:ap.deadline,tags:ap.tags,jira:ap.jira}:emptyDraft);
  const save=async()=>{if(!form.title.trim())return; if(ap)await updateAP(ap.id,form); else await createAP(form);onClose()};
  return <DrawerShell title={ap?'Edit AP':'New AP'} subtitle="Action Priority를 실제 데이터로 저장합니다." onClose={onClose}>
    <div className="drawer-body form-stack"><label>Title<input value={form.title} onChange={e=>setForm({...form,title:e.target.value})} autoFocus/></label><label>Details<textarea rows={4} value={form.details} onChange={e=>setForm({...form,details:e.target.value})}/></label><div className="two-col"><label>Importance <b>{form.importance}</b><input type="range" min="0" max="100" value={form.importance} onChange={e=>setForm({...form,importance:Number(e.target.value)})}/></label><label>Base urgency <b>{form.baseUrgency}</b><input type="range" min="0" max="100" value={form.baseUrgency} onChange={e=>setForm({...form,baseUrgency:Number(e.target.value)})}/></label></div><label>Deadline<input type="datetime-local" value={form.deadline?toLocalInput(form.deadline):''} onChange={e=>setForm({...form,deadline:e.target.value?new Date(e.target.value).toISOString():undefined})}/></label><label>Tags<input value={form.tags.join(', ')} onChange={e=>setForm({...form,tags:e.target.value.split(',').map(x=>x.trim()).filter(Boolean)})} placeholder="DevOps, SRE"/></label><label>Jira issue key<div className="jira-field-row"><input value={form.jira?.issueKey||''} onChange={e=>setForm({...form,jira:e.target.value?{...form.jira,issueKey:e.target.value.toUpperCase()}:undefined})} placeholder="OPS-123"/>{form.jira?.issueKey&&<JiraIssueLink issueKey={form.jira.issueKey} url={form.jira.url} baseUrl={state?.settings.jiraBaseUrl} className="jira-open-inline"/>}</div></label></div>
    <div className="drawer-actions">{ap&&<><button className="danger ghost" onClick={async()=>{await deleteAP(ap.id);onClose()}}>Delete</button><button className="success ghost" onClick={async()=>{await completeAP(ap.id);onClose()}}>Complete</button></>}<span/><button className="ghost" onClick={onClose}>Cancel</button><button className="primary" onClick={()=>void save()}>Save AP</button></div>
  </DrawerShell>;
}
function toLocalInput(iso:string){const d=new Date(iso);const offset=d.getTimezoneOffset();return new Date(d.getTime()-offset*60000).toISOString().slice(0,16)}

export function DailyCheckpointDrawer({onClose,edit}:{onClose:()=>void;edit?:DailyCheckpoint}){
  const {createDailyCheckpoint,updateDailyCheckpoint,deleteDailyCheckpoint}=usePlanner(); const [title,setTitle]=useState(edit?.title||'');const [enabled,setEnabled]=useState(edit?.enabled??true);
  const save=async()=>{if(!title.trim())return;if(edit)await updateDailyCheckpoint(edit.id,{title,enabled});else await createDailyCheckpoint({title,enabled});onClose()};
  return <DrawerShell title={edit?'Edit Daily Checkpoint':'New Daily Checkpoint'} subtitle="매일 다시 나타나는 개인 반복 체크입니다." onClose={onClose}><div className="drawer-body form-stack"><label>Checkpoint<input value={title} onChange={e=>setTitle(e.target.value)} autoFocus/></label><label className="switch-row"><span><b>Enabled</b><small>끄기는 삭제가 아닙니다.</small></span><input type="checkbox" checked={enabled} onChange={e=>setEnabled(e.target.checked)}/></label></div><div className="drawer-actions">{edit&&<button className="danger ghost" onClick={async()=>{await deleteDailyCheckpoint(edit.id);onClose()}}>Delete</button>}<span/><button className="ghost" onClick={onClose}>Cancel</button><button className="primary" onClick={()=>void save()}>Save</button></div></DrawerShell>
}

export function SettingsDrawer({onClose}:{onClose:()=>void}){
  const {state,saveSettings}=usePlanner();
  const initial=state!.settings;
  const [form,setForm]=useState<PlannerSettings>(initial);
  const [jira,setJira]=useState<JiraConfig>({baseUrl:initial.jiraBaseUrl,authMode:'bearer',username:'',apiVersion:'2',tokenConfigured:false});
  const [token,setToken]=useState('');
  const [jiraMessage,setJiraMessage]=useState('');
  useEffect(()=>{let alive=true;void plannerApi.getJiraConfig().then(value=>{if(alive){setJira(value);setForm(current=>({...current,jiraBaseUrl:value.baseUrl||current.jiraBaseUrl}))}}).catch(e=>{if(alive)setJiraMessage(e instanceof Error?e.message:String(e))});return()=>{alive=false}},[]);
  const save=async()=>{
    await saveSettings(form);
    await plannerApi.saveJiraConfig({baseUrl:form.jiraBaseUrl,authMode:jira.authMode,username:jira.username,apiVersion:jira.apiVersion,...(token.trim()?{token:token.trim()}:{})});
    onClose();
  };
  const test=async()=>{try{setJiraMessage('Testing…');const result=await plannerApi.testJira();setJiraMessage(result.message)}catch(e){setJiraMessage(e instanceof Error?e.message:String(e))}};
  return <DrawerShell title="Settings" subtitle="개인 설정 · Jira · 로컬 JSON 저장소" onClose={onClose}><div className="drawer-body form-stack">
    <label>Display name<input value={form.displayName} onChange={e=>setForm({...form,displayName:e.target.value})}/></label>
    <label>Timezone<input value={form.timezone} onChange={e=>setForm({...form,timezone:e.target.value})}/></label>
    <label>Jira Base URL<input value={form.jiraBaseUrl} onChange={e=>setForm({...form,jiraBaseUrl:e.target.value})} placeholder="https://jira.example.com"/></label>
    <div className="two-col"><label>Jira auth<select value={jira.authMode} onChange={e=>setJira({...jira,authMode:e.target.value as JiraConfig['authMode']})}><option value="bearer">Bearer token</option><option value="basic">Basic</option></select></label><label>REST API<select value={jira.apiVersion} onChange={e=>setJira({...jira,apiVersion:e.target.value as JiraConfig['apiVersion']})}><option value="2">v2</option><option value="3">v3</option></select></label></div>
    {jira.authMode==='basic'&&<label>Jira username<input value={jira.username} onChange={e=>setJira({...jira,username:e.target.value})} placeholder="name@example.com"/></label>}
    <label>Jira token<input type="password" value={token} onChange={e=>setToken(e.target.value)} placeholder={jira.tokenConfigured?'Saved — leave blank to keep current token':'Paste token'}/></label>
    <div className="api-note"><b>Jira connection</b><span>{jira.tokenConfigured?'Token saved':'Token not configured'}</span><small>{jiraMessage||'Token은 Planner state에 노출하지 않고 로컬 jira-settings.json에 별도 저장합니다.'}</small><button className="ghost small" type="button" onClick={()=>void test()}>Test Jira</button></div>
    <label>Local data path<input value={form.dataPath} onChange={e=>setForm({...form,dataPath:e.target.value})}/></label>
    <label>Auto sync minutes<input type="number" min="1" max="1440" value={form.autoSyncMinutes} onChange={e=>setForm({...form,autoSyncMinutes:Number(e.target.value)})}/></label>
    <div className="api-note"><b>Storage mode</b><span>{import.meta.env.VITE_API_MODE==='local'?'Local mock/localStorage':'Python local backend'}</span><small>배포 빌드에서는 FastAPI와 같은 localhost origin을 사용합니다.</small></div>
  </div><div className="drawer-actions"><span/><button className="ghost" onClick={onClose}>Cancel</button><button className="primary" onClick={()=>void save()}>Save settings</button></div></DrawerShell>
}

export function JiraWatchDrawer({onClose}:{onClose:()=>void}){
  const {state,saveJiraWatch,syncJiraWatch}=usePlanner(); const [items,setItems]=useState<JiraWatchItem[]>(()=>state!.jiraWatch); const [key,setKey]=useState(''); const [title,setTitle]=useState('');
  useEffect(()=>setItems(state!.jiraWatch),[state]);
  const counts=useMemo(()=>items.reduce<Record<string,number>>((a,x)=>(a[x.status]=(a[x.status]||0)+1,a),{}),[items]);
  const add=()=>{if(!key.trim()||!title.trim())return;setItems(v=>[...v,{id:crypto.randomUUID(),issueKey:key.toUpperCase(),title,status:'Open',updatedAt:new Date().toISOString()}]);setKey('');setTitle('')};
  return <DrawerShell title="My Jira Watch" subtitle="내가 추적할 Jira 티켓만 모아봅니다." onClose={onClose} wide><div className="drawer-body"><div className="jira-summary-strip"><span>Open <b>{counts.Open||0}</b></span><span>In Review <b>{counts['In Review']||0}</b></span><span>Blocked <b>{counts.Blocked||0}</b></span></div><div className="inline-add"><input placeholder="OPS-123" value={key} onChange={e=>setKey(e.target.value)}/><input placeholder="Ticket title" value={title} onChange={e=>setTitle(e.target.value)}/><button onClick={add}>Add</button></div><div className="jira-watch-list">{items.map(item=><div key={item.id} className="jira-watch-row"><div><JiraIssueLink issueKey={item.issueKey} url={item.url} baseUrl={state!.settings.jiraBaseUrl} className="jira-watch-key"/><span>{item.title}</span></div><select value={item.status} onChange={e=>setItems(v=>v.map(x=>x.id===item.id?{...x,status:e.target.value as JiraWatchItem['status']}:x))}><option>Open</option><option>In Review</option><option>Blocked</option><option>Done</option></select><button onClick={()=>setItems(v=>v.filter(x=>x.id!==item.id))}>×</button></div>)}</div></div><div className="drawer-actions"><button className="ghost" onClick={()=>void syncJiraWatch()}>Sync now</button><span/><button className="ghost" onClick={onClose}>Cancel</button><button className="primary" onClick={async()=>{await saveJiraWatch(items);onClose()}}>Save watch</button></div></DrawerShell>
}
