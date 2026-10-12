import { useEffect, useState } from 'react';
import { Header } from './components/Header';
import { Matrix } from './components/Matrix';
import { RightRail } from './components/RightRail';
import { Sidebar } from './components/Sidebar';
import { SummaryCards } from './components/SummaryCards';
import { APDrawer, DailyCheckpointDrawer, JiraWatchDrawer, SettingsDrawer } from './components/Drawers';
import { AllAPView, ArchiveView, CalendarView, CodeReviewView, DailyView, QueueView, ScrumView, StatisticsView, TagsView, TeamView } from './components/Views';
import { usePlanner } from './state/PlannerProvider';
import type { ActionPoint, AppView, DailyCheckpoint } from './types/models';

export default function App(){
  const {state,loading,error,completeAP,restoreAP,teamState}=usePlanner();
  const [view,setView]=useState<AppView>('home');
  const [apDrawer,setApDrawer]=useState<{open:boolean;ap?:ActionPoint}>({open:false});
  const [dailyDrawer,setDailyDrawer]=useState<{open:boolean;item?:DailyCheckpoint}>({open:false});
  const [settingsOpen,setSettingsOpen]=useState(false);
  const [jiraOpen,setJiraOpen]=useState(false);
  const [searchQuery,setSearchQuery]=useState('');
  const [completionToast,setCompletionToast]=useState<{id:string;title:string}|null>(null);
  const [dismissedCompleteReviews,setDismissedCompleteReviews]=useState<Set<string>>(()=>{try{return new Set(JSON.parse(localStorage.getItem('eisen-day-dismissed-review-complete')||'[]') as string[])}catch{return new Set()}});
  useEffect(()=>{if(!completionToast)return;const timer=window.setTimeout(()=>setCompletionToast(null),8000);return()=>window.clearTimeout(timer)},[completionToast]);
  if(loading||!state)return <div className="loading-screen"><div className="loading-mark">EISEN <span>/</span> DAY</div><p>Loading personal workspace…</p></div>;
  const editAP=(ap:ActionPoint)=>setApDrawer({open:true,ap});
  const completeWithUndo=async(id:string)=>{
    const ap=state.actionPoints.find(x=>x.id===id);
    if(!ap)return;
    await completeAP(id);
    setCompletionToast({id,title:ap.title});
  };
  const undoCompletion=async()=>{
    if(!completionToast)return;
    const target=completionToast;
    setCompletionToast(null);
    try{await restoreAP(target.id)}catch{setCompletionToast(target)}
  };
  const currentUser=teamState?.currentUser??'';
  const incomingReview=teamState?.codeReviews.find(r=>r.owner.toLowerCase()!==currentUser.toLowerCase()&&!r.complete&&!r.myResponse&&!(r.mdcRcc||r.mdcRgw||r.mdcCommon));
  const completedOwnReview=teamState?.codeReviews.find(r=>r.owner.toLowerCase()===currentUser.toLowerCase()&&r.complete&&!dismissedCompleteReviews.has(r.id)&&!(r.mdcRcc||r.mdcRgw||r.mdcCommon));
  const dismissCompletedReview=(id:string)=>{const next=new Set(dismissedCompleteReviews);next.add(id);setDismissedCompleteReviews(next);localStorage.setItem('eisen-day-dismissed-review-complete',JSON.stringify([...next]));};
  return <div className="app-shell">
    <Sidebar view={view} onView={setView} onSettings={()=>setSettingsOpen(true)}/>
    <main className="workspace">
      <Header onNewAP={()=>setApDrawer({open:true})} onNewDaily={()=>setDailyDrawer({open:true})} onNewCommit={()=>setApDrawer({open:true,ap:{kind:'commit'} as ActionPoint})} onSearch={setSearchQuery}/>
      {incomingReview&&<div className="review-alert incoming" role="status"><span className="review-alert-mark">!</span><div><b>Code review request from {incomingReview.owner}</b><small>{incomingReview.title}</small></div><button onClick={()=>window.open(incomingReview.url,'_blank','noopener,noreferrer')}>확인 · Open review ↗</button><button className="review-alert-history" onClick={()=>setView('codeReview')}>내역</button></div>}
      {completedOwnReview&&<div className="review-alert completed" role="status"><span className="review-alert-mark">✓</span><div><b>Code review completed</b><small>{completedOwnReview.title} · {completedOwnReview.doneCount}/{completedOwnReview.targetReviewCount} reviewers</small></div><button onClick={()=>setView('codeReview')}>확인</button><button className="review-alert-history" onClick={()=>dismissCompletedReview(completedOwnReview.id)}>닫기</button></div>}
      <SummaryCards/>
      {error&&<div className="error-banner">{error}</div>}
      {view==='home'&&<div className="dashboard-grid"><Matrix onEdit={editAP} onComplete={completeWithUndo} query={searchQuery}/><RightRail onDailyManage={()=>setView('daily')} onJiraManage={()=>setJiraOpen(true)} onQueueManage={()=>setView('queue')} onCalendarManage={()=>setView('calendar')}/></div>}
      {view==='queue'&&<QueueView onEdit={editAP} onComplete={completeWithUndo}/>}
      {view==='all'&&<AllAPView onEdit={editAP}/>}
      {view==='daily'&&<DailyView onNew={()=>setDailyDrawer({open:true})} onEdit={(item)=>setDailyDrawer({open:true,item})}/>}
      {view==='calendar'&&<CalendarView onEdit={editAP}/>}
      {view==='tags'&&<TagsView onEdit={editAP}/>}
      {view==='statistics'&&<StatisticsView/>}
      {view==='archive'&&<ArchiveView/>}
      {view==='team'&&<TeamView/>}
      {view==='codeReview'&&<CodeReviewView/>}
      {view==='scrum'&&<ScrumView/>}
    </main>
    {apDrawer.open&&<APDrawer ap={apDrawer.ap} onClose={()=>setApDrawer({open:false})}/>}
    {dailyDrawer.open&&<DailyCheckpointDrawer edit={dailyDrawer.item} onClose={()=>setDailyDrawer({open:false})}/>}
    {settingsOpen&&<SettingsDrawer onClose={()=>setSettingsOpen(false)}/>}
    {jiraOpen&&<JiraWatchDrawer onClose={()=>setJiraOpen(false)}/>}
    {completionToast&&<div className="completion-toast" role="status" aria-live="polite"><span className="toast-check">✓</span><span className="toast-copy"><b>AP가 완료되어 목록에서 제거되었습니다.</b><small title={completionToast.title}>{completionToast.title}</small></span><button className="toast-undo" onClick={()=>void undoCompletion()}>복구하기</button><button className="toast-close" aria-label="알림 닫기" onClick={()=>setCompletionToast(null)}>×</button></div>}
  </div>;
}
