import type { PlannerState } from '../types/models';
import { dayKey } from '../utils/planner';

const now = new Date();
const isoPlusDays = (days: number, hour = 18) => {
  const d = new Date(now);
  d.setDate(d.getDate() + days);
  d.setHours(hour, 0, 0, 0);
  return d.toISOString();
};
const created = (daysAgo: number) => {
  const d = new Date(now);
  d.setDate(d.getDate() - daysAgo);
  return d.toISOString();
};

export const seedState: PlannerState = {
  actionPoints: [
    { id:'ap-123', title:'Fix prod deploy route', details:'배포 헬스 체크와 라우팅 전환 조건 확인', importance:88, baseUrgency:82, deadline:isoPlusDays(0,21), tags:['DevOps'], jira:{issueKey:'OPS-123',status:'In Progress'}, queueOrder:0, createdAt:created(5), updatedAt:created(0) },
    { id:'ap-124', title:'Validate payment alerts', details:'결제 알림 임계치와 Slack routing 검증', importance:84, baseUrgency:76, deadline:isoPlusDays(0,22), tags:['Payments'], queueOrder:1, createdAt:created(3), updatedAt:created(0) },
    { id:'ap-201', title:'Refactor Jenkins shared lib', details:'중복 pipeline helper 정리', importance:76, baseUrgency:24, deadline:isoPlusDays(3), tags:['Platform'], jira:{issueKey:'PLAT-201',status:'Open'}, queueOrder:2, createdAt:created(7), updatedAt:created(1) },
    { id:'ap-202', title:'Write DR runbook', details:'DR 전환 절차 및 검증 항목 문서화', importance:72, baseUrgency:18, deadline:isoPlusDays(6), tags:['SRE'], queueOrder:3, createdAt:created(4), updatedAt:created(1) },
    { id:'ap-301', title:'Check access request', details:'권한 요청 및 만료일 검토', importance:34, baseUrgency:74, deadline:isoPlusDays(0,20), tags:['IT Ops'], queueOrder:4, createdAt:created(1), updatedAt:created(0) },
    { id:'ap-302', title:'Review vendor notice', details:'벤더 유지보수 공지 검토', importance:28, baseUrgency:66, deadline:isoPlusDays(1), tags:['Legal'], queueOrder:5, createdAt:created(2), updatedAt:created(0) },
    { id:'ap-401', title:'Clean old dashboard', details:'사용하지 않는 로컬 dashboard archive', importance:22, baseUrgency:18, tags:['Backlog'], queueOrder:6, createdAt:created(9), updatedAt:created(3) },
    { id:'ap-402', title:'Archive old notes', details:'오래된 운영 메모 정리', importance:18, baseUrgency:12, tags:['Backlog'], queueOrder:7, createdAt:created(12), updatedAt:created(4) },
  ],
  archive: [],
  dailyCheckpoints: [
    {id:'dc-1',title:'Stand-up & plan the day',enabled:true,order:0,completedDates:[dayKey()],createdAt:created(30),updatedAt:created(1)},
    {id:'dc-2',title:'Inbox zero (< 10 items)',enabled:true,order:1,completedDates:[dayKey()],createdAt:created(30),updatedAt:created(1)},
    {id:'dc-3',title:'Clear 3 AP items',enabled:true,order:2,completedDates:[dayKey()],createdAt:created(30),updatedAt:created(1)},
    {id:'dc-4',title:'Write 1 line of docs',enabled:true,order:3,completedDates:[dayKey()],createdAt:created(30),updatedAt:created(1)},
    {id:'dc-5',title:'Read something useful',enabled:true,order:4,completedDates:[],createdAt:created(30),updatedAt:created(1)},
    {id:'dc-6',title:'End-of-day reflection',enabled:true,order:5,completedDates:[],createdAt:created(30),updatedAt:created(1)},
  ],
  jiraWatch: [
    {id:'jw-1',issueKey:'OPS-91',title:'Deploy health routing follow-up',status:'Open',updatedAt:created(0)},
    {id:'jw-2',issueKey:'PLAT-144',title:'Jenkins migration follow-up',status:'In Review',updatedAt:created(0)},
    {id:'jw-3',issueKey:'SEC-32',title:'Access policy review',status:'Blocked',updatedAt:created(1)},
    {id:'jw-4',issueKey:'OPS-84',title:'Nginx reload investigation',status:'Open',updatedAt:created(2)},
  ],
  queueMode: 'manual',
  settings: {
    displayName:'Minju',
    timezone:Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Seoul',
    jiraBaseUrl:'',
    dataPath:'local browser storage',
    autoSyncMinutes:15,
  },
};
