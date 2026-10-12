import { useState, type DragEvent, type KeyboardEvent, type MouseEvent } from 'react';
import { usePlanner } from '../state/PlannerProvider';
import type { ActionPoint, Quadrant } from '../types/models';
import { effectiveUrgency, quadrantOf, shortDate } from '../utils/planner';

const meta:Record<Quadrant,{title:string;sub:string}>={Q1:{title:'DO FIRST',sub:'Important & Urgent'},Q2:{title:'SCHEDULE',sub:'Important & Not Urgent'},Q3:{title:'DELEGATE',sub:'Not Important & Urgent'},Q4:{title:'BACKLOG',sub:'Not Important & Not Urgent'}};

function TaskCard({ap,onEdit,onComplete}:{ap:ActionPoint;onEdit:(ap:ActionPoint)=>void;onComplete:(id:string)=>Promise<void>}){
  const q=quadrantOf(ap); const urgency=effectiveUrgency(ap);
  const [completing,setCompleting]=useState(false);
  const openFromKeyboard=(e:KeyboardEvent<HTMLDivElement>)=>{
    if(e.key==='Enter'||e.key===' '){e.preventDefault();onEdit(ap)}
  };
  const complete=async(e:MouseEvent<HTMLButtonElement>)=>{
    e.preventDefault();
    e.stopPropagation();
    if(completing)return;
    setCompleting(true);
    await onComplete(ap.id);
    // If completion failed, PlannerProvider keeps the AP in the active list.
    // Reset the visual state so the user can try again.
    setCompleting(false);
  };
  return <div
    className={`task-card ${completing?'completing':''}`}
    role="button"
    tabIndex={0}
    draggable={!completing}
    aria-label={`Edit ${ap.title}`}
    onDragStart={(e)=>{if(completing){e.preventDefault();return}e.dataTransfer.setData('text/ap-id',ap.id);e.dataTransfer.effectAllowed='move'}}
    onClick={()=>{if(!completing)onEdit(ap)}}
    onKeyDown={openFromKeyboard}
  >
    <button
      type="button"
      className={`task-state task-complete ${q==='Q1'?'coral-ring':q==='Q2'?'mint-ring':q==='Q3'?'lavender-ring':'butter-ring'} ${completing?'checked':''}`}
      aria-label={`Complete ${ap.title}`}
      title="Complete AP"
      disabled={completing}
      onMouseDown={(e)=>e.stopPropagation()}
      onClick={complete}
    >{completing?'✓':q==='Q1'?'!':q==='Q3'?'◷':''}</button>
    <span className="task-main"><b>{ap.title}</b><small>{ap.jira?.issueKey?<a href={ap.jira.url||`#${ap.jira.issueKey}`} target="_blank" rel="noopener noreferrer" className="task-jira-link" onClick={e=>e.stopPropagation()}>{ap.jira.issueKey}</a>:<i>{ap.id.slice(0,8).toUpperCase()}</i>}{ap.tags.slice(0,1).map(t=><i key={t}>{t}</i>)}</small></span>
    <span className={`task-date ${shortDate(ap.deadline)==='Today'?'hot':''}`}>{shortDate(ap.deadline)}<small className="score-hint">U{urgency} I{ap.importance}</small></span>
  </div>
}

function QuadrantCard({q,items,onEdit,onComplete,onDrop}:{q:Quadrant;items:ActionPoint[];onEdit:(ap:ActionPoint)=>void;onComplete:(id:string)=>Promise<void>;onDrop:(id:string,q:Quadrant)=>void}){
  const [over,setOver]=useState(false);
  const drop=(e:DragEvent)=>{e.preventDefault();setOver(false);const id=e.dataTransfer.getData('text/ap-id');if(id)onDrop(id,q)};
  return <article className={`quadrant ${q.toLowerCase()} ${over?'drag-over':''}`} onDragOver={(e)=>{e.preventDefault();setOver(true)}} onDragLeave={()=>setOver(false)} onDrop={drop}>
    <div className="q-head"><div><span className="q-badge">{q}</span><div><b>{meta[q].title}</b><small>{meta[q].sub}</small></div></div><span className="q-count">{items.length}</span></div>
    <div className="task-list">{items.length?items.map(ap=><TaskCard key={ap.id} ap={ap} onEdit={onEdit} onComplete={onComplete}/>):<div className="empty-card">Drop AP here</div>}</div>
    {q!=='Q4'&&<img src="/assets/dots.svg" className={`quad-decor ${q==='Q1'?'dots-tr':q==='Q2'?'dots-br':'dots-bl'}`} alt=""/>}{q==='Q4'&&<img src="/assets/squiggle.svg" className="quad-decor squiggle-br" alt=""/>}
  </article>
}

export function Matrix({onEdit,onComplete,query=''}:{onEdit:(ap:ActionPoint)=>void;onComplete?:(id:string)=>Promise<void>;query?:string}){
  const {state,moveAPToQuadrant,completeAP}=usePlanner();
  if(!state)return null;
  const needle=query.trim().toLowerCase();
  const visible=needle?state.actionPoints.filter(ap=>(ap.title+' '+ap.details+' '+ap.tags.join(' ')+' '+(ap.jira?.issueKey||'')).toLowerCase().includes(needle)):state.actionPoints;
  const grouped=(q:Quadrant)=>visible.filter(ap=>quadrantOf(ap)===q);
  return <section className="matrix-area">
    <div className="matrix-axis axis-important">IMPORTANT</div><div className="matrix-axis axis-not-important">NOT IMPORTANT</div>
    {(['Q1','Q2','Q3','Q4'] as Quadrant[]).map(q=><QuadrantCard key={q} q={q} items={grouped(q)} onEdit={onEdit} onComplete={onComplete ?? completeAP} onDrop={(id,target)=>void moveAPToQuadrant(id,target)}/>) }
    <div className="matrix-axis axis-urgent">URGENT</div><div className="matrix-axis axis-not-urgent">NOT URGENT</div>
  </section>
}
