import { useEffect, useMemo, useState, type DragEvent } from 'react';
import { usePlanner } from '../state/PlannerProvider';
import type { ActionPoint, DailyCheckpoint, JiraWatchItem, QueueMode } from '../types/models';
import { dayKey, effectiveUrgency, orderedQueue, quadrantOf, shortDate } from '../utils/planner';

export function QueueView({onEdit,onComplete}:{onEdit:(ap:ActionPoint)=>void;onComplete?:(id:string)=>Promise<void>}){
  const {state,reorderQueue,completeAP}=usePlanner(); const [dragId,setDragId]=useState<string|null>(null);
  if(!state)return null; const items=orderedQueue(state);
  const drop=(targetId:string)=>{if(!dragId||dragId===targetId)return;const ids=items.map(x=>x.id);const from=ids.indexOf(dragId),to=ids.indexOf(targetId);ids.splice(to,0,ids.splice(from,1)[0]);setDragId(null);void reorderQueue(ids,'manual')};
  return <section className="page-panel"><div className="section-title"><div><span>EXECUTION ORDER</span><h2>Today’s Queue</h2><p>Matrix와 별개로 실제 실행 순서를 정리합니다.</p></div><select value={state.queueMode} onChange={e=>void reorderQueue(items.map(x=>x.id),e.target.value as QueueMode)}><option value="manual">Manual</option><option value="important">Importance</option><option value="emergency">Urgency</option></select></div><div className="queue-board">{items.map((ap,i)=><div key={ap.id} draggable={state.queueMode==='manual'} onDragStart={()=>setDragId(ap.id)} onDragOver={e=>e.preventDefault()} onDrop={()=>drop(ap.id)} className="queue-board-row"><span className="drag-handle">⋮⋮</span><span className="queue-number">{i+1}</span><button className="queue-copy-button" onClick={()=>onEdit(ap)}><b>{ap.title}</b><small>{quadrantOf(ap)} · U{effectiveUrgency(ap)} · I{ap.importance} · {shortDate(ap.deadline)}</small></button><button className="queue-complete" onClick={()=>void (onComplete ?? completeAP)(ap.id)}>✓</button></div>)}</div></section>
}

export function AllAPView({onEdit}:{onEdit:(ap:ActionPoint)=>void}){
  const {state}=usePlanner(); const [query,setQuery]=useState(''); if(!state)return null; const items=state.actionPoints.filter(ap=>(ap.title+' '+ap.tags.join(' ')).toLowerCase().includes(query.toLowerCase()));
  return <section className="page-panel"><div className="section-title"><div><span>ACTIVE AP</span><h2>All AP</h2><p>현재 진행 중인 Action Priority 전체를 봅니다.</p></div><input className="panel-search" placeholder="Filter AP..." value={query} onChange={e=>setQuery(e.target.value)}/></div><div className="ap-table"><div className="table-head"><span>AP</span><span>Quadrant</span><span>Importance</span><span>Urgency</span><span>Deadline</span></div>{items.map(ap=><button key={ap.id} className="table-row" onClick={()=>onEdit(ap)}><span><b>{ap.title}</b><small>{ap.tags.map(t=>`#${t}`).join(' ')}</small></span><span className={`q-chip ${quadrantOf(ap).toLowerCase()}`}>{quadrantOf(ap)}</span><span>{ap.importance}</span><span>{effectiveUrgency(ap)}</span><span>{shortDate(ap.deadline)}</span></button>)}</div></section>
}

export function DailyView({onNew,onEdit}:{onNew:()=>void;onEdit:(item:DailyCheckpoint)=>void}){
  const {state,toggleDailyCheckpoint,updateDailyCheckpoint}=usePlanner(); if(!state)return null; const today=dayKey(); const items=[...state.dailyCheckpoints].sort((a,b)=>a.order-b.order);
  return <section className="page-panel"><div className="section-title"><div><span>RECURRING CHECK</span><h2>Daily Check</h2><p>매일 되살아나는 개인 체크포인트입니다. Disable은 삭제가 아닙니다.</p></div><button className="primary small" onClick={onNew}>＋ Add checkpoint</button></div><div className="daily-manager">{items.map(item=><div className={`daily-manager-row ${!item.enabled?'disabled':''}`} key={item.id}><input type="checkbox" checked={item.completedDates.includes(today)} disabled={!item.enabled} onChange={()=>void toggleDailyCheckpoint(item.id,today)}/><button className="daily-title" onClick={()=>onEdit(item)}><b>{item.title}</b><small>{item.enabled?'Enabled':'Disabled'}</small></button><label className="toggle"><input type="checkbox" checked={item.enabled} onChange={e=>void updateDailyCheckpoint(item.id,{enabled:e.target.checked})}/><span/></label></div>)}</div></section>
}

export function CalendarView({onEdit}:{onEdit:(ap:ActionPoint)=>void}){
  const {state}=usePlanner(); const [cursor,setCursor]=useState(()=>new Date(new Date().getFullYear(),new Date().getMonth(),1)); if(!state)return null;
  const data=useMemo(()=>{const year=cursor.getFullYear(),month=cursor.getMonth(),first=new Date(year,month,1),start=new Date(year,month,1-first.getDay());return Array.from({length:42},(_,i)=>{const d=new Date(start);d.setDate(start.getDate()+i);const items=state.actionPoints.filter(ap=>ap.deadline&&sameDay(new Date(ap.deadline),d));return{d,items,out:d.getMonth()!==month}})},[cursor,state.actionPoints]);
  const move=(delta:number)=>setCursor(new Date(cursor.getFullYear(),cursor.getMonth()+delta,1));
  return <section className="page-panel"><div className="section-title"><div><span>DEADLINE VIEW</span><h2>Calendar</h2><p>AP deadline을 월 단위로 확인합니다.</p></div><div className="month-nav"><button onClick={()=>move(-1)}>‹</button><b>{cursor.toLocaleDateString(undefined,{year:'numeric',month:'long'})}</b><button onClick={()=>move(1)}>›</button></div></div><div className="calendar-week"><span>Sun</span><span>Mon</span><span>Tue</span><span>Wed</span><span>Thu</span><span>Fri</span><span>Sat</span></div><div className="calendar-grid">{data.map(({d,items,out})=><div key={d.toISOString()} className={`calendar-cell ${out?'outside':''} ${sameDay(d,new Date())?'today':''}`}><b>{d.getDate()}</b><div>{items.slice(0,3).map(ap=><button key={ap.id} className={`calendar-event ${quadrantOf(ap).toLowerCase()}`} onClick={()=>onEdit(ap)}>{ap.title}</button>)}{items.length>3&&<small>+{items.length-3} more</small>}</div></div>)}</div></section>
}
function sameDay(a:Date,b:Date){return a.getFullYear()===b.getFullYear()&&a.getMonth()===b.getMonth()&&a.getDate()===b.getDate()}

export function TagsView({onEdit}:{onEdit:(ap:ActionPoint)=>void}){
  const {state}=usePlanner(); const [tag,setTag]=useState<string|null>(null); if(!state)return null; const counts=new Map<string,number>();state.actionPoints.forEach(ap=>ap.tags.forEach(t=>counts.set(t,(counts.get(t)||0)+1)));const tags=[...counts.entries()].sort((a,b)=>b[1]-a[1]);const items=tag?state.actionPoints.filter(ap=>ap.tags.includes(tag)):state.actionPoints;
  return <section className="page-panel"><div className="section-title"><div><span>CONTEXT FILTER</span><h2>Tags</h2><p>업무 컨텍스트별 AP를 빠르게 좁혀봅니다.</p></div>{tag&&<button className="ghost small" onClick={()=>setTag(null)}>Clear #{tag}</button>}</div><div className="tag-cloud">{tags.map(([name,count])=><button key={name} className={tag===name?'active':''} onClick={()=>setTag(name)}>#{name}<b>{count}</b></button>)}</div><div className="tag-results">{items.map(ap=><button key={ap.id} onClick={()=>onEdit(ap)}><span className={`q-chip ${quadrantOf(ap).toLowerCase()}`}>{quadrantOf(ap)}</span><b>{ap.title}</b><small>{ap.tags.join(' · ')}</small></button>)}</div></section>
}

export function StatisticsView(){
  const {state}=usePlanner(); if(!state)return null; const qCounts={Q1:0,Q2:0,Q3:0,Q4:0};state.actionPoints.forEach(ap=>qCounts[quadrantOf(ap)]++);const total=Math.max(1,state.actionPoints.length);const completed=state.archive.length;const enabled=state.dailyCheckpoints.filter(x=>x.enabled);const dailyDone=enabled.filter(x=>x.completedDates.includes(dayKey())).length;
  return <section className="page-panel"><div className="section-title"><div><span>PERSONAL SIGNALS</span><h2>Statistics</h2><p>성과 과시가 아니라, 내가 어디에 시간을 쓰고 있는지 보는 화면입니다.</p></div></div><div className="stat-grid"><article><span>Active AP</span><strong>{state.actionPoints.length}</strong><small>current workload</small></article><article><span>Completed</span><strong>{completed}</strong><small>archive total</small></article><article><span>Daily Check</span><strong>{dailyDone}/{enabled.length}</strong><small>today</small></article><article><span>Jira Watch</span><strong>{state.jiraWatch.length}</strong><small>tracked by me</small></article></div><div className="chart-card"><h3>Quadrant distribution</h3>{(Object.keys(qCounts) as (keyof typeof qCounts)[]).map(q=><div className="bar-row" key={q}><span>{q}</span><div><i className={q.toLowerCase()} style={{width:`${qCounts[q]/total*100}%`}}/></div><b>{qCounts[q]}</b></div>)}</div></section>
}

export function JiraView({items,onChange,onSync}:{items:JiraWatchItem[];onChange:(items:JiraWatchItem[])=>void;onSync:()=>void}){
  return <section className="page-panel"><div className="section-title"><div><span>PERSONAL WATCHLIST</span><h2>My Jira Watch</h2><p>Done이 아닌 내가 볼 티켓만 추적합니다.</p></div><button className="ghost small" onClick={onSync}>↻ Sync</button></div><div className="jira-full-list">{items.map(item=><div key={item.id}><b>{item.issueKey}</b><span>{item.title}</span><select value={item.status} onChange={e=>onChange(items.map(x=>x.id===item.id?{...x,status:e.target.value as JiraWatchItem['status']}:x))}><option>Open</option><option>In Review</option><option>Blocked</option><option>Done</option></select></div>)}</div></section>
}


export function ArchiveView(){
  const {state,restoreAP}=usePlanner();
  const [query,setQuery]=useState('');
  const [range,setRange]=useState<'all'|'7'|'30'>('all');
  const [restoring,setRestoring]=useState<string|null>(null);
  const [detail,setDetail]=useState<import('../types/models').ActionPoint|null>(null);
  if(!state)return null;

  const now=Date.now();
  const items=[...state.archive]
    .filter(ap=>{
      const hay=(ap.title+' '+ap.details+' '+ap.tags.join(' ')+' '+(ap.jira?.issueKey||'')).toLowerCase();
      if(query.trim()&&!hay.includes(query.trim().toLowerCase()))return false;
      if(range==='all'||!ap.completedAt)return true;
      const ageDays=(now-new Date(ap.completedAt).getTime())/86400000;
      return ageDays<=Number(range);
    })
    .sort((a,b)=>(b.completedAt||'').localeCompare(a.completedAt||''));

  const formatDateTime=(value?:string)=>{
    if(!value)return '—';
    const d=new Date(value);
    if(Number.isNaN(d.getTime()))return '—';
    return new Intl.DateTimeFormat(undefined,{year:'numeric',month:'short',day:'numeric',hour:'2-digit',minute:'2-digit'}).format(d);
  };
  const duration=(ap:ActionPoint)=>{
    if(!ap.completedAt)return '—';
    const ms=new Date(ap.completedAt).getTime()-new Date(ap.createdAt).getTime();
    if(!Number.isFinite(ms)||ms<0)return '—';
    const days=Math.max(0,Math.floor(ms/86400000));
    if(days===0){const hours=Math.max(1,Math.floor(ms/3600000));return `${hours}h`;}
    return `${days}d`;
  };
  const restore=async(id:string)=>{
    if(restoring)return;
    setRestoring(id);
    try{await restoreAP(id)}finally{setRestoring(null)}
  };

  return <section className="page-panel archive-page">
    <div className="section-title archive-heading">
      <div><span>COMPLETED WORK</span><h2>Archive</h2><p>완료한 AP를 확인하고, 잘못 완료한 항목은 언제든 다시 복구할 수 있습니다.</p></div>
      <div className="archive-count"><strong>{state.archive.length}</strong><small>completed AP</small></div>
    </div>
    <div className="archive-toolbar">
      <input className="panel-search archive-search" placeholder="Search completed AP, tag, Jira key..." value={query} onChange={e=>setQuery(e.target.value)}/>
      <select value={range} onChange={e=>setRange(e.target.value as 'all'|'7'|'30')} aria-label="완료 기간 필터">
        <option value="all">All time</option><option value="7">Last 7 days</option><option value="30">Last 30 days</option>
      </select>
    </div>
    {items.length===0?<div className="archive-empty"><b>완료된 AP가 없습니다.</b><span>체크해서 완료한 AP가 여기에 쌓입니다.</span></div>:
    <div className="archive-table">
      <div className="archive-table-head"><span>AP</span><span>Created</span><span>Completed</span><span>Duration</span><span></span></div>
      {items.map(ap=><div className="archive-row" key={ap.id} onClick={()=>setDetail(ap)} style={{cursor:'pointer'}}>
        <span className="archive-ap"><i className="archive-check">✓</i><span><b>{ap.title}</b><small>{[ap.jira?.issueKey,...ap.tags].filter(Boolean).map(x=>`#${x}`).join(' ')||'No tags'}</small></span></span>
        <span>{formatDateTime(ap.createdAt)}</span><span>{formatDateTime(ap.completedAt)}</span><span>{duration(ap)}</span>
        <button className="archive-restore" disabled={restoring===ap.id} onClick={(e)=>{e.stopPropagation();void restore(ap.id)}}>{restoring===ap.id?'Restoring…':'↶ Restore'}</button>
      </div>)}
    </div>}
    {detail&&<div className="archive-detail-overlay" onClick={()=>setDetail(null)}><div className="archive-detail-card" onClick={e=>e.stopPropagation()}><div className="archive-detail-head"><b>{detail.title}</b><button onClick={()=>setDetail(null)}>×</button></div><div className="archive-detail-body">{detail.jira?.issueKey&&<div className="archive-detail-field"><span>Jira</span><span>{detail.jira.issueKey}{detail.jira.url&&<> — <a href={detail.jira.url} target="_blank" rel="noopener noreferrer">Open ↗</a></>}</span></div>}{detail.tags.length>0&&<div className="archive-detail-field"><span>Tags</span><span>{detail.tags.map(t=>`#${t}`).join(' ')}</span></div>}<div className="archive-detail-field"><span>Importance</span><span>{detail.importance}</span></div><div className="archive-detail-field"><span>Urgency</span><span>{detail.baseUrgency}</span></div>{detail.deadline&&<div className="archive-detail-field"><span>Deadline</span><span>{new Date(detail.deadline).toLocaleString()}</span></div>}<div className="archive-detail-field"><span>Created</span><span>{formatDateTime(detail.createdAt)}</span></div><div className="archive-detail-field"><span>Completed</span><span>{formatDateTime(detail.completedAt)}</span></div>{detail.details&&<div className="archive-detail-details"><span>Details</span><pre>{detail.details}</pre></div>}</div></div></div>}
  </section>;
}

export function TeamTodosView(){
  const {teamState}=usePlanner();
  const [member,setMember]=useState<string>('all');
  if(!teamState)return <section className="page-panel"><div className="archive-empty"><b>Team workspace를 확인하는 중입니다.</b></div></section>;
  if(!teamState.enabled)return <section className="page-panel"><div className="section-title"><div><span>SHAREPOINT TEAM</span><h2>Team Todos</h2><p>Settings에서 SharePoint 동기화 폴더를 데이터 경로로 지정하면 자동으로 팀원을 찾습니다.</p></div></div><div className="archive-empty"><b>공유 폴더를 찾지 못했습니다.</b><span>예: SharePoint (local) / USER_NAME / data / eisenhower.json</span></div></section>;
  const members=teamState.members.filter(x=>x.name!==teamState.currentUser);
  const visible=member==='all'?members:members.filter(x=>x.name===member);
  const all=visible.flatMap(m=>m.actionPoints.map(ap=>({ap,owner:m.name})));
  const grouped={Q1:[] as typeof all,Q2:[] as typeof all,Q3:[] as typeof all,Q4:[] as typeof all};
  all.forEach(item=>grouped[quadrantOf(item.ap)].push(item));
  return <section className="page-panel team-page">
    <div className="section-title"><div><span>SHAREPOINT TEAM</span><h2>Team Todos</h2><p>팀원의 active AP를 읽기 전용으로 모아서 같은 4분면 기준으로 봅니다.</p></div><select value={member} onChange={e=>setMember(e.target.value)}><option value="all">All team members</option>{members.map(m=><option key={m.name} value={m.name}>{m.name}</option>)}</select></div>
    <div className="team-meta"><span><b>{teamState.currentUser}</b> as me</span><span>{members.length} teammates</span><span className="team-root" title={teamState.root}>{teamState.root}</span></div>
    <div className="team-matrix">{(['Q1','Q2','Q3','Q4'] as const).map(q=><section key={q} className={`team-quadrant ${q.toLowerCase()}`}><header><b>{q}</b><span>{q==='Q1'?'DO FIRST':q==='Q2'?'SCHEDULE':q==='Q3'?'DELEGATE':'BACKLOG'}</span><i>{grouped[q].length}</i></header><div className="team-task-list">{grouped[q].length?grouped[q].map(({ap,owner})=><article key={`${owner}-${ap.id}`}><span className="team-owner">{owner}</span><b>{ap.title}</b><small>{ap.jira?.issueKey||ap.tags.slice(0,2).join(' · ')||'No tag'} · U{effectiveUrgency(ap)} I{ap.importance}</small></article>):<div className="team-empty">No AP</div>}</div></section>)}</div>
  </section>;
}

export function TeamView(){
  const {teamState,refreshTeam}=usePlanner();
  const [selected,setSelected]=useState('');
  const members=teamState?.members ?? [];
  const current=selected || members[0]?.name || '';
  const member=members.find(m=>m.name===current);
  const qs=['Q1','Q2','Q3','Q4'] as const;
  return <section className="page-panel team-page">
    <div className="section-title"><div><span>SHAREPOINT TEAM VIEW</span><h2>Team AP</h2><p>팀 공유 폴더에서 각 구성원의 활성 AP를 읽기 전용으로 표시합니다.</p></div><div className="team-actions"><select value={current} onChange={e=>setSelected(e.target.value)}>{members.map(m=><option key={m.name} value={m.name}>{m.name}{m.name===teamState?.currentUser?' (me)':''}</option>)}</select><button className="ghost small" onClick={()=>void refreshTeam()}>↻ Refresh</button></div></div>
    {!teamState?.enabled?<div className="team-empty"><b>Team SharePoint root가 설정되지 않았습니다.</b><span>Settings에서 팀 공유 루트를 지정해 주세요.</span></div>:!member?<div className="team-empty"><b>읽을 수 있는 팀원 데이터가 없습니다.</b><span>{teamState.root}</span></div>:<>
      <div className="team-member-head"><span className="team-avatar">{member.name.slice(0,1).toUpperCase()}</span><div><b>{member.name}</b><small>{member.actionPoints.length} active AP · read only</small></div></div>
      <div className="team-matrix">{qs.map(q=>{const items=member.actionPoints.filter(ap=>quadrantOf(ap)===q);return <section key={q} className={`team-quadrant ${q.toLowerCase()}`}><div className="team-q-head"><b>{q}</b><span>{q==='Q1'?'DO FIRST':q==='Q2'?'SCHEDULE':q==='Q3'?'DELEGATE':'BACKLOG'}</span><strong>{items.length}</strong></div><div className="team-task-list">{items.length?items.map(ap=><article key={ap.id}><div><b>{ap.title}</b><small>{ap.jira?.issueKey||ap.tags.slice(0,2).join(' · ')||'No tag'}</small></div><span>{shortDate(ap.deadline)}</span></article>):<div className="team-q-empty">No AP</div>}</div></section>})}</div>
    </>}
  </section>;
}

export function CodeReviewView(){
  const {teamState,createCodeReview,respondCodeReview,setMdcVersions,updateCodeReview,archiveCodeReview,refreshTeam}=usePlanner();
  const [form,setForm]=useState({url:'',title:'',description:'',validation:'',approvalMode:'each' as 'each'|'single'});
  const [filter,setFilter]=useState<'active'|'all'|'mine'|'todo'|'archived'>('active');
  const [submitting,setSubmitting]=useState(false);
  const [mdcEditing,setMdcEditing]=useState<string|null>(null);
  const [mdcDraft,setMdcDraft]=useState({mdcRcc:'',mdcRgw:'',mdcCommon:''});
  const [editing,setEditing]=useState<string|null>(null);
  const [editForm,setEditForm]=useState({url:'',title:'',description:'',validation:'',approvalMode:'each' as 'each'|'single'});
  const me=teamState?.currentUser ?? '';
  const reviews=teamState?.codeReviews ?? [];
  const isArchived=(r:typeof reviews[0])=>{
    if(r.archivedAt) return true;
    const hasMdc=!!(r.mdcRcc||r.mdcRgw||r.mdcCommon);
    if(r.complete&&hasMdc) return true;
    return false;
  };
  const visible=reviews.filter(r=>{
    if(filter==='archived')return isArchived(r);
    if(filter==='active')return !isArchived(r);
    if(filter==='mine')return r.owner.toLowerCase()===me.toLowerCase()&&!isArchived(r);
    if(filter==='todo')return r.owner.toLowerCase()!==me.toLowerCase()&&!r.complete&&!r.myResponse;
    return true; // 'all'
  });
  const submit=async()=>{if(!form.url.trim()||!form.title.trim()||!form.validation.trim())return;setSubmitting(true);try{await createCodeReview(form);setForm({url:'',title:'',description:'',validation:'',approvalMode:'each'})}finally{setSubmitting(false)}};
  const open=(url:string)=>window.open(url,'_blank','noopener,noreferrer');
  const submitMdc=async(id:string)=>{if(!mdcDraft.mdcRcc.trim()&&!mdcDraft.mdcRgw.trim()&&!mdcDraft.mdcCommon.trim())return;try{await setMdcVersions(id,mdcDraft);setMdcEditing(null);setMdcDraft({mdcRcc:'',mdcRgw:'',mdcCommon:''})}catch{}};
  const startEdit=(r:typeof reviews[0])=>{setEditing(r.id);setEditForm({url:r.url,title:r.title,description:r.description||'',validation:r.validation,approvalMode:r.approvalMode||'each'})};
  const submitEdit=async(id:string)=>{if(!editForm.title.trim()||!editForm.validation.trim())return;setSubmitting(true);try{await updateCodeReview(id,editForm);setEditing(null)}finally{setSubmitting(false)}};
  return <section className="page-panel code-review-page">
    <div className="section-title"><div><span>TEAM REVIEW WORKFLOW</span><h2>Code Review</h2><p>SharePoint 동기화 폴더를 통해 팀원에게 코드리뷰를 요청하고 응답을 수집합니다.</p></div><button className="ghost small" onClick={()=>void refreshTeam()}>↻ Refresh</button></div>
    <div className="review-compose"><div className="review-compose-head"><div><b>Request code review</b><small>Owner: {me||'SharePoint user folder'}</small></div><select className="approval-mode-select" value={form.approvalMode} onChange={e=>setForm({...form,approvalMode:e.target.value as 'each'|'single'})}><option value="each">각자 CR+1</option><option value="single">CR+2 한명</option></select></div><div className="review-form-grid"><label>Review URL<input value={form.url} onChange={e=>setForm({...form,url:e.target.value})} placeholder="https://git.example.com/.../merge_requests/123"/></label><label>Review title<input value={form.title} onChange={e=>setForm({...form,title:e.target.value})} placeholder="RCC secretkey migration review"/></label><label className="review-description">변경 내용<textarea rows={2} value={form.description} onChange={e=>setForm({...form,description:e.target.value})} placeholder="어떤 코드를 왜 변경했는지 간단히"/></label><label className="review-validation">검증 포인트<textarea rows={2} value={form.validation} onChange={e=>setForm({...form,validation:e.target.value})} placeholder="확인해야 할 항목, 영향 범위"/></label><button className="primary review-submit" disabled={submitting} onClick={()=>void submit()}>{submitting?'Submitting…':'Submit request'}</button></div></div>
    <div className="review-history-head"><div><button className={filter==='active'?'active':''} onClick={()=>setFilter('active')}>Active</button><button className={filter==='all'?'active':''} onClick={()=>setFilter('all')}>All</button><button className={filter==='mine'?'active':''} onClick={()=>setFilter('mine')}>My requests</button><button className={filter==='todo'?'active':''} onClick={()=>setFilter('todo')}>To review</button><button className={filter==='archived'?'active':''} onClick={()=>setFilter('archived')}>Archived</button></div><span>{visible.length} request{visible.length===1?'':'s'}</span></div>
    <div className="review-history">{visible.map(r=>{const mine=r.owner.toLowerCase()===me.toLowerCase();const pending=!mine&&!r.complete&&!r.myResponse;const isEditing=editing===r.id;const hasMdc=!!(r.mdcRcc||r.mdcRgw||r.mdcCommon);return <article key={r.id} className={`review-history-card ${pending?'pending':''} ${r.complete?'complete':''}`}><div className="review-status-mark">{r.complete?'✓':pending?'!':r.doneCount}</div>{isEditing?<div className="review-edit-form"><label>URL<input value={editForm.url} onChange={e=>setEditForm({...editForm,url:e.target.value})}/></label><label>Title<input value={editForm.title} onChange={e=>setEditForm({...editForm,title:e.target.value})}/></label><label>변경 내용<textarea rows={2} value={editForm.description} onChange={e=>setEditForm({...editForm,description:e.target.value})}/></label><label>검증 포인트<textarea rows={2} value={editForm.validation} onChange={e=>setEditForm({...editForm,validation:e.target.value})}/></label><div className="review-edit-actions"><select value={editForm.approvalMode} onChange={e=>setEditForm({...editForm,approvalMode:e.target.value as 'each'|'single'})}><option value="each">각자 CR+1</option><option value="single">CR+2 한명</option></select><button className="success small" disabled={submitting} onClick={()=>void submitEdit(r.id)}>저장</button><button className="ghost small" onClick={()=>setEditing(null)}>취소</button></div></div>:<><div className="review-history-copy"><div className="review-title-line"><b>{r.title}</b><span>{r.owner}</span><span className="approval-mode-badge">{r.approvalMode==='single'?'CR+2 한명':'각자 CR+1'}</span></div>{r.description&&<p className="review-description-text">{r.description}</p>}<p>{r.validation}</p><small>{new Date(r.createdAt).toLocaleString()} · approvals {r.doneCount}/{r.approvalMode==='single'?1:r.targetReviewCount}{r.skippedCount?` · not mine ${r.skippedCount}`:''}</small>{hasMdc&&<div className="mdc-version-tags"><span className="mdc-version-tag">RCC-MDC {r.mdcRcc||'—'}</span><span className="mdc-version-tag">RGW-MDC {r.mdcRgw||'—'}</span><span className="mdc-version-tag">COMMON-MDC {r.mdcCommon||'—'}</span></div>}{mine&&r.complete&&!hasMdc&&<div className="mdc-version-input">{mdcEditing===r.id?<div className="mdc-3-inputs"><label>RCC-MDC (CXS2010264_1)<input className="mdc-input" value={mdcDraft.mdcRcc} onChange={e=>setMdcDraft({...mdcDraft,mdcRcc:e.target.value})} placeholder="R134A05"/></label><label>RGW-MDC (CXS2010276_1)<input className="mdc-input" value={mdcDraft.mdcRgw} onChange={e=>setMdcDraft({...mdcDraft,mdcRgw:e.target.value})} placeholder="R134A05"/></label><label>COMMON-MDC (CXS2010434_1)<input className="mdc-input" value={mdcDraft.mdcCommon} onChange={e=>setMdcDraft({...mdcDraft,mdcCommon:e.target.value})} placeholder="R73A16"/></label><div className="mdc-btn-row"><button className="success small" onClick={()=>void submitMdc(r.id)}>저장</button><button className="ghost small" onClick={()=>{setMdcEditing(null);setMdcDraft({mdcRcc:'',mdcRgw:'',mdcCommon:''})}}>취소</button></div></div>:<button className="ghost small mdc-add-btn" onClick={()=>{setMdcEditing(r.id);setMdcDraft({mdcRcc:r.mdcRcc||'',mdcRgw:r.mdcRgw||'',mdcCommon:r.mdcCommon||''})}}>+ MDC 버전 기입</button>}</div>}</div><div className="review-card-actions"><button className="ghost small" onClick={()=>open(r.url)}>Open link ↗</button>{mine&&<button className="ghost small" onClick={()=>startEdit(r)}>✎ 수정</button>}{pending&&<><button className="success small" onClick={()=>void respondCodeReview(r.id,'done')}>Reviewed ✓</button><button className="ghost small" onClick={()=>void respondCodeReview(r.id,'not_mine')}>Not my area</button></>}{r.complete&&!r.archivedAt&&<button className="ghost small" onClick={()=>void archiveCodeReview(r.id)}>📦 Archive</button>}{mine&&<span className={`review-owner-state ${r.complete?'done':''}`}>{r.complete?'Review complete':'Waiting for reviews'}</span>}{!mine&&r.myResponse&&<span className="review-owner-state done">{r.myResponse==='done'?'Reviewed':'Not my area'}</span>}{r.archivedAt&&<span className="review-owner-state done">Archived</span>}</div></>}</article>})}{visible.length===0&&<div className="team-empty"><b>Code review request가 없습니다.</b><span>새 요청을 만들거나 SharePoint 동기화를 확인해 주세요.</span></div>}</div>
  </section>;
}

export function ScrumView(){
  const [date,setDate]=useState(()=>new Date().toISOString().slice(0,10));
  const [dash,setDash]=useState<import('../types/models').ScrumDashboard|null>(null);
  const [note,setNote]=useState('');
  const [saving,setSaving]=useState(false);
  const [noteTimer,setNoteTimer]=useState<ReturnType<typeof setTimeout>|null>(null);
  const {plannerApi:_}=({} as any); // we import directly
  const load=async(d:string)=>{
    try{
      const {plannerApi}=await import('../api/plannerApi');
      const result=await plannerApi.getScrumDashboard(d);
      setDash(result);setNote(result.note);
    }catch(e){console.warn('scrum load failed',e)}
  };
  useEffect(()=>{void load(date)},[date]);
  const moveDate=(delta:number)=>{const d=new Date(date);d.setDate(d.getDate()+delta);setDate(d.toISOString().slice(0,10))};
  const today=new Date().toISOString().slice(0,10);
  const saveNote=async(text:string)=>{
    setNote(text);
    if(noteTimer)clearTimeout(noteTimer);
    const timer=setTimeout(async()=>{
      setSaving(true);
      try{const {plannerApi}=await import('../api/plannerApi');await plannerApi.saveScrumNote(date,text)}catch{}finally{setSaving(false)}
    },800);
    setNoteTimer(timer);
  };
  const completedReviews=dash?.codeReviews ?? [];
  return <section className="page-panel scrum-page">
    <div className="section-title"><div><span>TEAM STANDUP</span><h2>Daily Scrum</h2><p>팀원별 오늘 완료한 AP + 진행중인 Jira 티켓 + 코드리뷰 현황을 한눈에 봅니다.</p></div><div className="scrum-date-nav"><button onClick={()=>moveDate(-1)}>‹</button><b>{date}{date===today?' (오늘)':''}</b><button onClick={()=>moveDate(1)} disabled={date>=today}>›</button></div></div>
    {!dash?<div className="team-empty"><b>Loading…</b></div>:<>
      <div className="scrum-grid">{dash.entries.map(entry=><article key={entry.name} className="scrum-member-card">
        <div className="scrum-member-head"><span className="team-avatar">{entry.name.slice(0,1).toUpperCase()}</span><div><b>{entry.name}</b><small>{entry.completedToday.length} completed · {entry.activeJira.length} Jira</small></div></div>
        {entry.completedToday.length>0&&<div className="scrum-section"><div className="scrum-section-label">✓ Completed today</div>{entry.completedToday.map(ap=><div key={ap.id} className="scrum-ap-row"><span className="scrum-check">✓</span><span>{ap.title}</span>{ap.jira?.issueKey&&<small className="scrum-jira-key">{ap.jira.issueKey}</small>}</div>)}</div>}
        {entry.activeJira.length>0&&<div className="scrum-section"><div className="scrum-section-label">🎫 Active Jira</div>{entry.activeJira.map((j,i)=><div key={`${j.issueKey}-${i}`} className={`scrum-jira-row${j.paused?' scrum-jira-paused':''}${(j.staleDays&&j.staleDays>=5)?' scrum-jira-stale':''}`}>{j.url?<a href={j.url} target="_blank" rel="noopener noreferrer" className="scrum-jira-link">{j.issueKey}</a>:<span className="scrum-jira-link">{j.issueKey}</span>}<span className="scrum-jira-summary">{j.summary||j.issueKey}</span>{j.paused&&<span className="scrum-tag-paused">⏸ 중단</span>}{(j.staleDays&&j.staleDays>=5)?<span className="scrum-tag-stale">⚠ {j.staleDays}일 정체</span>:null}{j.status&&<span className="scrum-jira-status">{j.status}</span>}</div>)}</div>}
        {entry.completedToday.length===0&&entry.activeJira.length===0&&<div className="scrum-empty-member">기록 없음</div>}
      </article>)}</div>
      {completedReviews.length>0&&<div className="scrum-reviews-section"><div className="scrum-section-label">📝 Completed Code Reviews</div><div className="scrum-reviews-list">{completedReviews.map(r=>{const hasMdc=!!(r.mdcRcc||r.mdcRgw||r.mdcCommon);return <article key={r.id} className="scrum-review-card"><div className="scrum-review-head"><span className="scrum-review-check">✓</span><div><b>{r.title}</b><small>{r.owner} · approvals {r.doneCount}/{r.approvalMode==='single'?1:r.targetReviewCount}{r.skippedCount?` · not mine ${r.skippedCount}`:''}</small></div></div>{hasMdc?<div className="mdc-version-tags"><span className="mdc-version-tag">RCC-MDC {r.mdcRcc||'—'}</span><span className="mdc-version-tag">RGW-MDC {r.mdcRgw||'—'}</span><span className="mdc-version-tag">COMMON-MDC {r.mdcCommon||'—'}</span></div>:<div className="scrum-mdc-pending">MDC 미입력</div>}</article>})}</div></div>}
      <div className="scrum-note-section"><div className="scrum-note-head"><b>Scrum Note</b><small>{saving?'저장 중…':'공유 메모 — 팀 전체가 볼 수 있습니다'}</small></div><textarea className="scrum-note-textarea" rows={4} value={note} onChange={e=>void saveNote(e.target.value)} placeholder="오늘 스크럼 공유 사항을 적어주세요…"/></div>
    </>}
  </section>;
}
