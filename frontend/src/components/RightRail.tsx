import { useEffect, useMemo, useState } from 'react';
import { usePlanner } from '../state/PlannerProvider';
import { dayKey, orderedQueue } from '../utils/planner';

export function RightRail({onDailyManage,onJiraManage,onQueueManage,onCalendarManage}:{onDailyManage:()=>void;onJiraManage:()=>void;onQueueManage:()=>void;onCalendarManage:()=>void}){
  const {state,toggleDailyCheckpoint,syncJiraWatch,saveQuickNote}=usePlanner();
  const today=dayKey();
  const queue=useMemo(()=>state?orderedQueue(state).slice(0,3):[],[state]);
  const meetings=useMemo(()=>{
    if(!state)return [];
    const now=new Date();
    const meetingPattern=/(meeting|미팅|회의|standup|sync|모임|면접|interview)/i;
    return state.actionPoints
      .filter(ap=>ap.deadline&&new Date(ap.deadline).toDateString()===now.toDateString())
      .filter(ap=>ap.kind==='meeting'||meetingPattern.test(`${ap.title} ${ap.details} ${ap.tags.join(' ')}`))
      .sort((a,b)=>new Date(a.deadline!).getTime()-new Date(b.deadline!).getTime());
  },[state]);
  const [note,setNote]=useState('');
  const [saved,setSaved]=useState(true);

  useEffect(()=>{
    if(state){setNote(state.quickNote ?? '');setSaved(true);}
  },[state?.quickNote]);

  if(!state)return null;
  const daily=state.dailyCheckpoints.filter(x=>x.enabled).sort((a,b)=>a.order-b.order);
  const done=daily.filter(x=>x.completedDates.includes(today)).length;
  const pct=daily.length?Math.round(done/daily.length*100):0;
  const counts={Open:0,'In Review':0,Blocked:0,Done:0};
  state.jiraWatch.forEach(x=>counts[x.status]++);

  const persistNote=async()=>{
    if(note === (state.quickNote ?? '')){setSaved(true);return;}
    await saveQuickNote(note);
  };

  return <aside className="right-rail">
    <section className="rail-card"><div className="rail-head"><h2><span className="rail-icon">✓</span> Daily Check</h2><button onClick={onDailyManage}>•••</button></div><div className="rail-progress-line"><b>{done} / {daily.length} complete</b><span>{pct}%</span></div><div className="segmented">{daily.map(x=><i key={x.id} className={x.completedDates.includes(today)?'':'empty'}></i>)}</div><div className="check-list">{daily.slice(0,6).map(x=><label key={x.id}><input type="checkbox" checked={x.completedDates.includes(today)} onChange={()=>void toggleDailyCheckpoint(x.id,today)}/><span>{x.title}</span></label>)}</div><button className="text-link" onClick={onDailyManage}>Manage Daily Check →</button></section>
    <section className="rail-card compact queue-rail-card" role="button" tabIndex={0} onClick={onQueueManage} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();onQueueManage()}}}><div className="rail-head"><h2><span className="rail-icon list">☷</span> Today’s Queue</h2><button onClick={e=>{e.stopPropagation();onQueueManage()}} title="우선순위 조정">•••</button></div><ol className="queue-list">{queue.map((ap,i)=><li key={ap.id}><span>{i+1}</span><b>{ap.title}</b><time>{ap.deadline?new Date(ap.deadline).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'}):'—'}</time></li>)}</ol><div className="queue-manage-hint">Click to adjust priority →</div></section>
    <section className="rail-card compact jira-card"><div className="rail-head"><h2><span className="jira-mark">J</span> Jira Watch</h2><button onClick={()=>void syncJiraWatch()} title="sync">↻</button></div><div className="jira-metrics"><div><strong>{counts.Open}</strong><small>Open</small></div><div><strong>{counts['In Review']}</strong><small>In Review</small></div><div><strong>{counts.Blocked}</strong><small className="blocked">Blocked</small></div></div><button className="text-link" onClick={onJiraManage}>Open Jira Watch →</button></section>
    <section className="rail-card compact note-card">
      <div className="rail-head"><h2><span className="note-mark">✎</span> Note</h2><span className={`note-save-state ${saved?'saved':''}`}>{saved?'Saved':'Unsaved'}</span></div>
      <textarea
        value={note}
        maxLength={12000}
        placeholder="메모를 남겨두세요…"
        onChange={(e)=>{setNote(e.target.value);setSaved(false)}}
        onBlur={()=>void persistNote()}
        onKeyDown={(e)=>{if((e.ctrlKey||e.metaKey)&&e.key==='Enter'){e.preventDefault();void persistNote();}}}
      />
      <div className="note-footer"><small>{note.length.toLocaleString()} / 12,000</small><button onClick={()=>void persistNote()} disabled={saved}>Save note</button></div>
    </section>
    <section className="meeting-bar" role="button" tabIndex={0} onClick={onCalendarManage} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();onCalendarManage()}}}>
      <div className="meeting-bar-icon">◷</div>
      <div className="meeting-bar-copy"><b>오늘 미팅</b>{meetings.length?<span>{meetings.slice(0,2).map(m=>`${new Date(m.deadline!).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'})} ${m.title}`).join(' · ')}</span>:<span>등록된 미팅이 없습니다.</span>}</div>
      <strong>{meetings.length}</strong>
      <span className="meeting-bar-arrow">→</span>
    </section>
  </aside>
}
