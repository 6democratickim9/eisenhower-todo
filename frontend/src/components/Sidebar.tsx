import type { CSSProperties } from 'react';
import type { AppView } from '../types/models';
import { ArchiveIcon, CalendarIcon, ChartIcon, CheckIcon, GridIcon, HomeIcon, ListIcon, SettingsIcon, TagIcon } from './Icons';
import { dayKey } from '../utils/planner';
import { usePlanner } from '../state/PlannerProvider';

const nav = [
  ['home','Home',HomeIcon],
  ['queue','Today’s Queue',ListIcon],
  ['all','All AP',GridIcon],
  ['team','Team AP',GridIcon],
  ['scrum','Daily Scrum',CheckIcon],
  ['daily','Daily Check',CheckIcon],
  ['calendar','Calendar',CalendarIcon],
  ['tags','Tags',TagIcon],
  ['statistics','Statistics',ChartIcon],
  ['archive','Archive',ArchiveIcon],
  ['codeReview','Code Review History',ListIcon],
] as const;

export function Sidebar({view,onView,onSettings}:{view:AppView;onView:(view:AppView)=>void;onSettings:()=>void}){
  const {state,teamState}=usePlanner();
  const enabled=state?.dailyCheckpoints.filter(x=>x.enabled) ?? [];
  const done=enabled.filter(x=>x.completedDates.includes(dayKey())).length;
  const progress=enabled.length?Math.round(done/enabled.length*100):0;
  const me=teamState?.currentUser ?? '';
  const reviews=teamState?.codeReviews ?? [];
  const isArchived=(r:typeof reviews[0])=>{
    if(r.archivedAt) return true;
    const hasMdc=!!(r.mdcRcc||r.mdcRgw||r.mdcCommon);
    if(r.complete&&hasMdc) return true;
    return false;
  };
  const activeReviews=reviews.filter(r=>!isArchived(r));
  const requested=activeReviews.filter(r=>r.owner.toLowerCase()===me.toLowerCase());
  const requestedOpen=requested.filter(r=>!r.complete).length;
  const requestedDone=requested.filter(r=>r.complete).length;
  const toReview=activeReviews.filter(r=>r.owner.toLowerCase()!==me.toLowerCase()&&!r.complete&&!r.myResponse).length;

  return <aside className="sidebar">
    <div className="brand-row"><div><div className="brand">EISEN <span>/</span> DAY</div><div className="brand-sub">personal ops dashboard</div></div><button className="icon-btn sidebar-collapse" aria-label="사이드바 접기">«</button></div>
    <nav className="nav-list" aria-label="주요 메뉴">{nav.map(([id,label,Icon])=><button key={id} className={`nav-item ${view===id?'active':''}`} onClick={()=>onView(id)}><Icon className="nav-svg"/><span>{label}</span>{id==='archive'&&state&&state.archive.length>0?<b className="nav-count">{state.archive.length}</b>:null}{id==='codeReview'&&activeReviews.length>0?<b className="nav-count">{activeReviews.length}</b>:null}</button>)}</nav>
    <section className="side-progress">
      <div className="mini-title">Daily Check Progress</div>
      <div className="progress-wrap"><div className="progress-ring" style={{'--progress':progress} as CSSProperties}><strong>{progress}%</strong></div><div><strong className="big-mini">{done} <small>/ {enabled.length}</small></strong><span>complete</span></div></div>
      <div className="week-row">{Array.from({length:7},(_,index)=>{const start=new Date();const mondayOffset=(start.getDay()+6)%7;start.setDate(start.getDate()-mondayOffset+index);const key=dayKey(start);const dayDone=enabled.filter(x=>x.completedDates.includes(key)).length;const complete=enabled.length>0&&dayDone===enabled.length;const labels=['M','T','W','T','F','S','S'];return <span key={key} className={complete?'done':''}>{complete?'✓':'○'}<small>{labels[index]}</small></span>})}</div>
    </section>
    <div className="review-side-stack">
      <button className="review-side-card mine" onClick={()=>onView('codeReview')}>
        <span className="review-side-icon">↗</span><span><b>My review requests</b><small>{requestedOpen} active · {requestedDone} completed</small></span><strong>{requestedOpen}</strong>
      </button>
      <button className={`review-side-card incoming ${toReview>0?'needs-attention':''}`} onClick={()=>onView('codeReview')}>
        <span className="review-side-icon">✓</span><span><b>Reviews to do</b><small>You got code review request</small></span><strong>{toReview}</strong>
      </button>
    </div>
    <button className="settings-card" onClick={onSettings}><SettingsIcon className="settings-svg"/><span><b>Settings</b><small>preferences · Jira · storage</small></span><span>›</span></button>
    <div className="sidebar-footer">local-first · team-share workspace</div>
  </aside>;
}
