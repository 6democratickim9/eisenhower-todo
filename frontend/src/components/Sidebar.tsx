import type { CSSProperties } from 'react';
import type { AppView } from '../types/models';
import { CalendarIcon, ChartIcon, CheckIcon, GridIcon, HomeIcon, ListIcon, SettingsIcon, TagIcon } from './Icons';
import { dayKey } from '../utils/planner';
import { usePlanner } from '../state/PlannerProvider';

const nav = [
  ['home','Home',HomeIcon],['queue','Today’s Queue',ListIcon],['all','All AP',GridIcon],['daily','Daily Check',CheckIcon],['calendar','Calendar',CalendarIcon],['tags','Tags',TagIcon],['statistics','Statistics',ChartIcon],
] as const;

export function Sidebar({view,onView,onSettings}:{view:AppView;onView:(view:AppView)=>void;onSettings:()=>void}){
  const {state}=usePlanner();
  const enabled=state?.dailyCheckpoints.filter(x=>x.enabled) ?? [];
  const done=enabled.filter(x=>x.completedDates.includes(dayKey())).length;
  const progress=enabled.length?Math.round(done/enabled.length*100):0;
  return <aside className="sidebar">
    <div className="brand-row"><div><div className="brand">EISEN <span>/</span> DAY</div><div className="brand-sub">personal ops dashboard</div></div><button className="icon-btn sidebar-collapse" aria-label="사이드바 접기">«</button></div>
    <nav className="nav-list" aria-label="주요 메뉴">{nav.map(([id,label,Icon])=><button key={id} className={`nav-item ${view===id?'active':''}`} onClick={()=>onView(id)}><Icon className="nav-svg"/><span>{label}</span></button>)}</nav>
    <section className="side-progress">
      <div className="mini-title">Daily Check Progress</div>
      <div className="progress-wrap"><div className="progress-ring" style={{'--progress':progress} as CSSProperties}><strong>{progress}%</strong></div><div><strong className="big-mini">{done} <small>/ {enabled.length}</small></strong><span>complete</span></div></div>
      <div className="week-row">{Array.from({length:7},(_,index)=>{const start=new Date();const mondayOffset=(start.getDay()+6)%7;start.setDate(start.getDate()-mondayOffset+index);const key=dayKey(start);const dayDone=enabled.filter(x=>x.completedDates.includes(key)).length;const complete=enabled.length>0&&dayDone===enabled.length;const labels=['M','T','W','T','F','S','S'];return <span key={key} className={complete?'done':''}>{complete?'✓':'○'}<small>{labels[index]}</small></span>})}</div>
    </section>
    <button className="settings-card" onClick={onSettings}><SettingsIcon className="settings-svg"/><span><b>Settings</b><small>preferences · Jira · storage</small></span><span>›</span></button>
    <div className="sidebar-footer">local-first · private workspace</div>
  </aside>;
}
