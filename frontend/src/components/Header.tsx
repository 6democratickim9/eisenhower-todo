import { useEffect, useRef, useState } from 'react';
import { PlusIcon, SearchIcon } from './Icons';
import { usePlanner } from '../state/PlannerProvider';

export function Header({onNewAP,onNewDaily,onSearch}:{onNewAP:()=>void;onNewDaily:()=>void;onSearch:(query:string)=>void}){
  const {state}=usePlanner(); const [open,setOpen]=useState(false); const wrap=useRef<HTMLDivElement>(null);
  useEffect(()=>{const key=(e:KeyboardEvent)=>{if(e.key==='Escape')setOpen(false);if(!open)return;if(e.key.toLowerCase()==='a'){setOpen(false);onNewAP()}if(e.key.toLowerCase()==='d'){setOpen(false);onNewDaily()}};window.addEventListener('keydown',key);return()=>window.removeEventListener('keydown',key)},[open,onNewAP,onNewDaily]);
  useEffect(()=>{const outside=(e:MouseEvent)=>{if(open&&!wrap.current?.contains(e.target as Node))setOpen(false)};window.addEventListener('mousedown',outside);return()=>window.removeEventListener('mousedown',outside)},[open]);
  const hour=new Date().getHours(); const greeting=hour<12?'Good morning':hour<18?'Good afternoon':'Good evening';
  return <header className="topbar">
    <div className="greeting-block"><h1>{greeting}, <span>{state?.settings.displayName || 'Minju'}.</span></h1><p>오늘 필요한 것만 보고, 바로 처리하세요.</p></div>
    <div className="top-actions"><label className="search"><SearchIcon/><input placeholder="Search tasks, tags, projects..." onChange={e=>onSearch(e.target.value)} /></label>
      <div className="add-wrap" ref={wrap}><button className="add-button" onClick={()=>setOpen(v=>!v)}><PlusIcon/> Add something <b>⌄</b></button>
      <div className={`add-menu ${open?'open':''}`} role="dialog" aria-hidden={!open}><img src="/assets/spark.svg" className="decor decor-spark"/><img src="/assets/squiggle.svg" className="decor decor-squiggle"/><div className="add-kicker">✦ QUICK CREATE</div><h3>무엇을 추가하시겠어요?</h3><p>작업 종류를 고르면 오른쪽 편집기로 이어집니다.</p>
        <button className="add-option ap-option" onClick={()=>{setOpen(false);onNewAP()}}><span className="option-icon">!</span><span><b>AP</b><small>한 번 처리하고 끝나는 일반 업무</small></span><kbd>A ↗</kbd></button>
        <button className="add-option daily-option" onClick={()=>{setOpen(false);onNewDaily()}}><span className="option-icon">✓</span><span><b>Daily Checkpoint</b><small>매일 한 번 확인하는 반복 체크</small></span><kbd>D ↗</kbd></button><div className="tip">Tip · A / D 키로 바로 선택</div>
      </div></div>
    </div>
  </header>
}
