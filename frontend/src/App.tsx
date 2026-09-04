import { useState } from 'react';
import { Header } from './components/Header';
import { Matrix } from './components/Matrix';
import { RightRail } from './components/RightRail';
import { Sidebar } from './components/Sidebar';
import { SummaryCards } from './components/SummaryCards';
import { APDrawer, DailyCheckpointDrawer, JiraWatchDrawer, SettingsDrawer } from './components/Drawers';
import { AllAPView, CalendarView, DailyView, QueueView, StatisticsView, TagsView } from './components/Views';
import { usePlanner } from './state/PlannerProvider';
import type { ActionPoint, AppView, DailyCheckpoint } from './types/models';

export default function App(){
  const {state,loading,error}=usePlanner();
  const [view,setView]=useState<AppView>('home');
  const [apDrawer,setApDrawer]=useState<{open:boolean;ap?:ActionPoint}>({open:false});
  const [dailyDrawer,setDailyDrawer]=useState<{open:boolean;item?:DailyCheckpoint}>({open:false});
  const [settingsOpen,setSettingsOpen]=useState(false);
  const [jiraOpen,setJiraOpen]=useState(false);
  const [searchQuery,setSearchQuery]=useState('');
  if(loading||!state)return <div className="loading-screen"><div className="loading-mark">EISEN <span>/</span> DAY</div><p>Loading personal workspace…</p></div>;
  const editAP=(ap:ActionPoint)=>setApDrawer({open:true,ap});
  return <div className="app-shell">
    <Sidebar view={view} onView={setView} onSettings={()=>setSettingsOpen(true)}/>
    <main className="workspace">
      <Header onNewAP={()=>setApDrawer({open:true})} onNewDaily={()=>setDailyDrawer({open:true})} onSearch={setSearchQuery}/>
      <SummaryCards/>
      {error&&<div className="error-banner">{error}</div>}
      {view==='home'&&<div className="dashboard-grid"><Matrix onEdit={editAP} query={searchQuery}/><RightRail onDailyManage={()=>setView('daily')} onJiraManage={()=>setJiraOpen(true)}/></div>}
      {view==='queue'&&<QueueView onEdit={editAP}/>} 
      {view==='all'&&<AllAPView onEdit={editAP}/>} 
      {view==='daily'&&<DailyView onNew={()=>setDailyDrawer({open:true})} onEdit={(item)=>setDailyDrawer({open:true,item})}/>} 
      {view==='calendar'&&<CalendarView onEdit={editAP}/>} 
      {view==='tags'&&<TagsView onEdit={editAP}/>} 
      {view==='statistics'&&<StatisticsView/>}
    </main>
    {apDrawer.open&&<APDrawer ap={apDrawer.ap} onClose={()=>setApDrawer({open:false})}/>} 
    {dailyDrawer.open&&<DailyCheckpointDrawer edit={dailyDrawer.item} onClose={()=>setDailyDrawer({open:false})}/>} 
    {settingsOpen&&<SettingsDrawer onClose={()=>setSettingsOpen(false)}/>} 
    {jiraOpen&&<JiraWatchDrawer onClose={()=>setJiraOpen(false)}/>} 
  </div>;
}
