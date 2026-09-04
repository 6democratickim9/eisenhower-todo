import { useMemo } from 'react';
import { usePlanner } from '../state/PlannerProvider';
import { dayKey, orderedQueue } from '../utils/planner';

export function RightRail({onDailyManage,onJiraManage}:{onDailyManage:()=>void;onJiraManage:()=>void}){
  const {state,toggleDailyCheckpoint,syncJiraWatch}=usePlanner();
  const today=dayKey();
  const queue=useMemo(()=>state?orderedQueue(state).slice(0,3):[],[state]);
  if(!state)return null;
  const daily=state.dailyCheckpoints.filter(x=>x.enabled).sort((a,b)=>a.order-b.order); const done=daily.filter(x=>x.completedDates.includes(today)).length; const pct=daily.length?Math.round(done/daily.length*100):0;
  const counts={Open:0,'In Review':0,Blocked:0,Done:0};state.jiraWatch.forEach(x=>counts[x.status]++);
  return <aside className="right-rail">
    <section className="rail-card"><div className="rail-head"><h2><span className="rail-icon">✓</span> Daily Check</h2><button onClick={onDailyManage}>•••</button></div><div className="rail-progress-line"><b>{done} / {daily.length} complete</b><span>{pct}%</span></div><div className="segmented">{daily.map(x=><i key={x.id} className={x.completedDates.includes(today)?'':'empty'}></i>)}</div><div className="check-list">{daily.slice(0,6).map(x=><label key={x.id}><input type="checkbox" checked={x.completedDates.includes(today)} onChange={()=>void toggleDailyCheckpoint(x.id,today)}/><span>{x.title}</span></label>)}</div><button className="text-link" onClick={onDailyManage}>Manage Daily Check →</button></section>
    <section className="rail-card compact"><div className="rail-head"><h2><span className="rail-icon list">☷</span> Today’s Queue</h2><button>•••</button></div><ol className="queue-list">{queue.map((ap,i)=><li key={ap.id}><span>{i+1}</span><b>{ap.title}</b><time>{ap.deadline?new Date(ap.deadline).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'}):'—'}</time></li>)}</ol></section>
    <section className="rail-card compact"><div className="rail-head"><h2><span className="jira-mark">J</span> My Jira Watch</h2><button onClick={()=>void syncJiraWatch()} title="sync">↻</button></div><div className="jira-metrics"><div><strong>{counts.Open}</strong><small>Open</small></div><div><strong>{counts['In Review']}</strong><small>In Review</small></div><div><strong>{counts.Blocked}</strong><small className="blocked">Blocked</small></div></div><button className="text-link" onClick={onJiraManage}>Open my Jira tickets →</button></section>
  </aside>
}
