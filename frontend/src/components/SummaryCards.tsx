import { usePlanner } from '../state/PlannerProvider';
import { dayKey, isToday } from '../utils/planner';

export function SummaryCards(){
  const {state}=usePlanner();
  if(!state)return null;
  const enabled=state.dailyCheckpoints.filter(x=>x.enabled);
  const done=enabled.filter(x=>x.completedDates.includes(dayKey())).length;
  const dueToday=state.actionPoints.filter(x=>isToday(x.deadline)).length;
  return <section className="summary-grid" aria-label="요약">
    <article className="summary-card"><div className="summary-dot coral">⌁</div><div><span>IN PROGRESS</span><strong>{state.actionPoints.length} <small>AP</small></strong><em>현재 진행 중</em></div></article>
    <article className="summary-card"><div className="summary-dot mint">□</div><div><span>DUE TODAY</span><strong>{dueToday} <small>tasks</small></strong><em>오늘 확인 필요</em></div></article>
    <article className="summary-card"><div className="summary-dot lavender">✓</div><div><span>DAILY CHECK</span><strong>{done} <small>/ {enabled.length}</small></strong><em>{enabled.length?Math.round(done/enabled.length*100):0}% complete</em></div></article>
    <article className="summary-card"><div className="summary-dot butter">?</div><div><span>UNCHECKED</span><strong>{Math.max(0,enabled.length-done)} <small>left</small></strong><em>오늘 남은 체크</em></div></article>
  </section>
}
