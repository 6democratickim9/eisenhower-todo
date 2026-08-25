let state = null;
let dragId = null;
let editingId = null;
let jiraSyncInFlight = false;
let tagFilter = null;
let storagePromptDismissed = false;
let calendarCursor = new Date(new Date().getFullYear(), new Date().getMonth(), 1);

const $ = (s) => document.querySelector(s);
const $$ = (s) => [...document.querySelectorAll(s)];
const on = (selector, event, handler) => {
  const el = typeof selector === 'string' ? $(selector) : selector;
  if (el) el.addEventListener(event, handler);
};

async function api(path, options = {}) {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    ...options,
  });
  if (!res.ok) {
    let message = `${res.status}`;
    try { message = (await res.json()).error || message; } catch {}
    throw new Error(message);
  }
  if (res.status === 204) return null;
  return res.json();
}

async function load() {
  return (async () => {
    state = await api('/api/state');
    const info = await api('/api/info');
    if ($('#dataPath')) $('#dataPath').textContent = info.dataPath || '';
    if ($('#displayName')) $('#displayName').textContent = info.displayName || 'You';
    if ($('#editNameAvatar')) $('#editNameAvatar').textContent = (info.displayName || 'M').trim().charAt(0).toUpperCase() || 'M';
    render();
    if (info.needsDataLocationSetup && !storagePromptDismissed && !$('#storageDialog')?.open) {
      setTimeout(() => openStorageSettings(true), 80);
    }
  })();
}

function nowMs() { return Date.now(); }
function clamp(v) { return Math.max(0, Math.min(100, v)); }
function lerp(a,b,t){ return a + (b-a)*Math.max(0,Math.min(1,t)); }
function deadlinePressure(deadline) {
  if (!deadline) return 0;
  const h = (new Date(deadline).getTime() - nowMs()) / 3600000;
  if (!Number.isFinite(h)) return 0;
  if (h <= 0) return 80;
  if (h >= 168) return 0;
  if (h >= 72) return lerp(0, 15, (168 - h) / 96);
  if (h >= 24) return lerp(15, 35, (72 - h) / 48);
  return lerp(35, 70, (24 - h) / 24);
}
function urgency(ap) { return Math.round(clamp((ap.baseUrgency || 0) + deadlinePressure(ap.deadline))); }
function quadrant(ap) {
  const important = (ap.importance || 0) >= 50;
  const urgent = urgency(ap) >= 50;
  return important ? (urgent ? 'Q1' : 'Q2') : (urgent ? 'Q3' : 'Q4');
}
function isOverdue(ap){ return !!ap.deadline && new Date(ap.deadline).getTime() < nowMs(); }
function esc(v=''){ return String(v).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])); }
function localDateTime(v){ if(!v) return ''; const d=new Date(v); return Number.isNaN(d.getTime()) ? v : d.toLocaleString(); }
function shortDateTime(v){ if(!v) return ''; const d=new Date(v); if(Number.isNaN(d.getTime())) return v; return `${d.getMonth()+1}/${d.getDate()} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`; }
function dueToday(ap){ if(!ap.deadline) return false; const d=new Date(ap.deadline), n=new Date(); return sameDay(d,n); }
function sameDay(a,b){ return a.getFullYear()===b.getFullYear() && a.getMonth()===b.getMonth() && a.getDate()===b.getDate(); }
function dayKey(d){ return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`; }
function parseTags(value){ return [...new Set(String(value||'').split(',').map(x=>x.trim()).filter(Boolean))]; }
function apTags(ap){ return Array.isArray(ap.tags) ? ap.tags : []; }
function matchesTag(ap){ return !tagFilter || apTags(ap).some(t => t.toLowerCase() === tagFilter.toLowerCase()); }

function ageDays(ap){
  const raw = ap.createdAt || ap.updatedAt;
  if(!raw) return 0;
  const d = new Date(raw);
  if(Number.isNaN(d.getTime())) return 0;
  const start = new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  return Math.max(0, Math.floor((today - start) / 86400000));
}
function activeAges(){ return (state?.actionPoints || []).map(ageDays).sort((a,b)=>a-b); }
function oldestAp(){
  const items=[...(state?.actionPoints||[])];
  if(!items.length) return null;
  const maxAge=Math.max(...items.map(ageDays));
  return items.filter(ap=>ageDays(ap)===maxAge).sort((a,b)=>
    (b.importance||0)-(a.importance||0) ||
    urgency(b)-urgency(a) ||
    new Date(a.createdAt||a.updatedAt||0).getTime()-new Date(b.createdAt||b.updatedAt||0).getTime() ||
    (a.queueOrder||0)-(b.queueOrder||0)
  )[0] || null;
}
function ageTone(ap){
  const oldest=oldestAp();
  if(oldest && oldest.id===ap.id) return 'age-red';
  const days=ageDays(ap);
  if(days<=2) return 'age-neutral';
  if(days<=6) return 'age-green';
  return 'age-warm';
}
function ageBadge(ap, compact=false){
  const days = ageDays(ap);
  const created = ap.createdAt || ap.updatedAt || '';
  return `<span class="age-badge ${ageTone(ap)} ${compact?'compact':''}" title="${esc(shortDateTime(created))} 생성">${days}일</span>`;
}
function qClass(q){ return String(q||'').toLowerCase(); }

function render() {
  if (!state) return;
  const active=state.actionPoints || [];
  const ages=active.map(ageDays);
  const queueItems=orderedQueue();
  const overdueCount=active.filter(isOverdue).length;
  $('#activeCount').textContent = active.length;
  $('#dueTodayCount').textContent = active.filter(dueToday).length;
  $('#overdueCount').textContent = overdueCount;
  $('#oldestAgeCount').textContent = `${ages.length ? Math.max(...ages) : 0}일`;
  $('#staleCount').textContent = ages.filter(v=>v>=7).length;
  $('#queueSummaryCount').textContent = queueItems.length;
  $('#queueNavCount').textContent = queueItems.length;
  $('#queuePanelCount').textContent = queueItems.length;
  $('#allApCount').textContent = active.length;
  $('#queueMode').value = state.queueMode || 'manual';

  const visible = active.filter(matchesTag);
  const qCounts={Q1:0,Q2:0,Q3:0,Q4:0};
  $$('.dropzone').forEach(z => z.innerHTML = '');
  for (const ap of visible) {
    const q=quadrant(ap); qCounts[q]++;
    const zone = $(`.dropzone[data-quadrant="${q}"]`);
    zone?.insertAdjacentHTML('beforeend', cardHtml(ap));
  }
  ['Q1','Q2','Q3','Q4'].forEach(q=>{ const el=$(`#${q.toLowerCase()}Count`); if(el) el.textContent=qCounts[q]; });
  for (const zone of $$('.dropzone')) {
    if (!zone.children.length) zone.innerHTML = `<div class="empty">${tagFilter ? `#${esc(tagFilter)} AP가 없습니다.` : 'AP가 없습니다.'}</div>`;
  }

  renderQueue();
  renderCalendar();
  renderTags();
  renderStats();
  bindDynamic();
}

function jiraHtml(ap) {
  const j = ap.jira;
  if (!j?.url) return '';
  const comment = j.lastComment || '';
  const multiline = /[\r\n]/.test(comment);
  const longComment = comment.length > 120 || multiline;
  return `
    <div class="jira-block">
      <div class="jira-row">
        <a class="jira-link" href="${esc(j.url)}" target="_blank" rel="noreferrer">${esc(j.issueKey || 'Jira')}</a>
        ${j.status ? `<span class="jira-pill status">${esc(j.status)}</span>` : ''}
        ${j.assignee ? `<span class="jira-pill">${esc(j.assignee)}</span>` : ''}
      </div>
      ${comment ? `<div class="jira-comment-wrap">
        <div class="jira-comment-head"><span class="jira-comment-label">최근 댓글</span>${longComment ? `<button type="button" class="jira-more-btn" data-id="${esc(ap.id)}">더보기</button>` : ''}</div>
        <div class="jira-comment" data-comment-id="${esc(ap.id)}">${esc(comment)}</div>
      </div>` : ''}
      <div class="jira-foot">
        ${j.lastSyncedAt ? `<span>Sync ${esc(shortDateTime(j.lastSyncedAt))}</span>` : '<span>아직 Sync 안 됨</span>'}
        ${j.syncError ? `<span class="jira-error">⚠ ${esc(j.syncError)}</span>` : ''}
      </div>
    </div>`;
}

function cardHtml(ap) {
  const u = urgency(ap);
  const overdue = isOverdue(ap);
  const autoUrgent = deadlinePressure(ap.deadline) > 0 && ((ap.baseUrgency < 50 && u >= 50) || overdue);
  const deadline = ap.deadline ? `<span class="badge deadline">${overdue ? 'Overdue · ' : ''}${esc(shortDateTime(ap.deadline))}</span>` : '';
  const auto = autoUrgent ? `<span class="badge">Auto-Urgency</span>` : '';
  const tags = apTags(ap).map(t=>`<button type="button" class="tag-mini tag-filter-link" data-tag="${esc(t)}">#${esc(t)}</button>`).join('');
  return `<div class="ap-card ${u>=50?'urgent':''} ${overdue?'overdue':''}" draggable="true" data-id="${esc(ap.id)}">
    <div class="ap-head"><div class="ap-title-wrap"><div class="ap-title">${esc(ap.title)}</div>${ageBadge(ap,true)}</div><div class="ap-actions">
      <button class="mini-btn history-btn" data-id="${esc(ap.id)}" title="History">↺</button>
      <button class="mini-btn edit-btn" data-id="${esc(ap.id)}">편집</button>
      <button class="mini-btn complete-btn" data-id="${esc(ap.id)}">✓</button>
    </div></div>
    ${ap.details ? `<div class="ap-details">${esc(ap.details)}</div>` : ''}
    <div class="meta-row">${deadline}${auto}${tags}</div>
    ${jiraHtml(ap)}
    <div class="urgency"><div class="urgency-line"><div class="urgency-fill" style="width:${u}%"></div></div><div class="score-text"><span>I${ap.importance}</span><span>U${u}</span></div></div>
  </div>`;
}

function orderedQueue() {
  const items = state.actionPoints.filter(matchesTag);
  const mode = state.queueMode || 'manual';
  if (mode === 'important') items.sort((a,b) => b.importance-a.importance || urgency(b)-urgency(a) || a.queueOrder-b.queueOrder);
  else if (mode === 'emergency') items.sort((a,b) => urgency(b)-urgency(a) || b.importance-a.importance || a.queueOrder-b.queueOrder);
  else items.sort((a,b) => a.queueOrder-b.queueOrder);

  // Execution Priority와 Eisenhower Quadrant는 별개다. 가장 오래된 AP 하나만 항상 Priority 1로 고정한다.
  const oldest=oldestAp();
  if(oldest){
    const i=items.findIndex(x=>x.id===oldest.id);
    if(i>0) items.unshift(items.splice(i,1)[0]);
  }
  return items;
}

function renderQueue() {
  const el = $('#queue');
  const items = orderedQueue();
  const oldest=oldestAp();
  if (!items.length) { el.innerHTML=`<div class="empty">${tagFilter ? `#${esc(tagFilter)} 큐가 비어 있습니다.` : '진행 중인 AP가 없습니다.'}</div>`; return; }
  el.innerHTML = items.map((ap,index) => `<div class="queue-item" draggable="${state.queueMode==='manual' && !tagFilter}" data-id="${esc(ap.id)}">
    <span class="handle">⋮</span><button class="queue-check queue-complete" data-id="${esc(ap.id)}" title="완료">✓</button>
    <div class="queue-copy"><span class="queue-title">${index===0 && oldest?.id===ap.id?'<b class="priority-one">P1</b> ':''}${esc(ap.title)}</span><div class="queue-submeta">${ageBadge(ap,true)}${ap.jira?.status ? `<small class="queue-jira">${esc(ap.jira.status)}</small>` : ''}</div></div>
    <span class="queue-q ${qClass(quadrant(ap))}">${quadrant(ap)}</span><small class="queue-score">U${urgency(ap)} I${ap.importance}</small>
  </div>`).join('');
}

function renderCalendar() {
  const title = $('#calendarTitle');
  const grid = $('#calendarGrid');
  title.textContent = `${calendarCursor.getFullYear()}년 ${calendarCursor.getMonth()+1}월`;
  const year = calendarCursor.getFullYear(), month = calendarCursor.getMonth();
  const first = new Date(year, month, 1);
  const start = new Date(year, month, 1-first.getDay());
  const dueMap = new Map();
  for (const ap of state.actionPoints.filter(matchesTag)) {
    if (!ap.deadline) continue;
    const d = new Date(ap.deadline);
    if (Number.isNaN(d.getTime())) continue;
    const key=dayKey(d);
    if(!dueMap.has(key)) dueMap.set(key,[]);
    dueMap.get(key).push(ap);
  }
  const today = new Date();
  const cells=[];
  for(let i=0;i<42;i++){
    const d=new Date(start); d.setDate(start.getDate()+i);
    const items=dueMap.get(dayKey(d))||[];
    const dots=items.slice(0,4).map(ap=>`<i class="event-dot ${qClass(quadrant(ap))}" title="${esc(ap.title)}"></i>`).join('');
    const cls=[d.getMonth()!==month?'outside':'',sameDay(d,today)?'today':'',items.length?'has-deadline':''].filter(Boolean).join(' ');
    const tip=items.map(ap=>`${ap.title} (${quadrant(ap)})`).join('\n');
    cells.push(`<div class="calendar-day ${cls}" title="${esc(tip)}"><span>${d.getDate()}</span><div class="day-events">${dots}${items.length>4?`<b>+${items.length-4}</b>`:''}</div></div>`);
  }
  grid.innerHTML=cells.join('');
}

function renderTags() {
  const counts = new Map();
  state.actionPoints.forEach(ap => apTags(ap).forEach(tag => counts.set(tag, (counts.get(tag)||0)+1)));
  const list = [...counts.entries()].sort((a,b)=>b[1]-a[1] || a[0].localeCompare(b[0]));
  $('#clearTagFilterBtn').classList.toggle('hidden', !tagFilter);
  if(!list.length){ $('#tagList').innerHTML='<div class="empty">AP에 태그를 추가하면 여기서 모아볼 수 있습니다.</div>'; return; }
  $('#tagList').innerHTML=list.map(([tag,count],i)=>`<button type="button" class="tag-chip tag-${i%6} ${tagFilter?.toLowerCase()===tag.toLowerCase()?'active':''}" data-tag="${esc(tag)}"><span>${esc(tag)}</span><b>${count}</b></button>`).join('');
}

function renderStats() {
  const counts={Q1:0,Q2:0,Q3:0,Q4:0};
  state.actionPoints.forEach(ap=>counts[quadrant(ap)]++);
  const total=state.actionPoints.length;
  $('#donutTotal').textContent=total;
  const pct = q => total ? counts[q]/total*100 : 0;
  const p1=pct('Q1'), p2=pct('Q2'), p3=pct('Q3'), p4=pct('Q4');
  let a1=p1, a2=a1+p2, a3=a2+p3;
  $('#donut').style.background = total ? `conic-gradient(var(--q1) 0 ${a1}%, var(--q2) ${a1}% ${a2}%, var(--q3) ${a2}% ${a3}%, var(--q4) ${a3}% 100%)` : '#e5e7dc';
  $('#statsLegend').innerHTML=['Q1','Q2','Q3','Q4'].map(q=>`<div><i class="legend-dot ${qClass(q)}"></i><span>${q}</span><b>${counts[q]}</b><small>${total?Math.round(counts[q]/total*100):0}%</small></div>`).join('');

  const agingBands=[
    {label:'0–2일',count:0,cls:'age-neutral'},
    {label:'3–6일',count:0,cls:'age-green'},
    {label:'7–13일',count:0,cls:'age-warm'},
    {label:'14일+',count:0,cls:'age-red'},
  ];
  state.actionPoints.forEach(ap=>{
    const d=ageDays(ap);
    if(d<=2) agingBands[0].count++;
    else if(d<=6) agingBands[1].count++;
    else if(d<=13) agingBands[2].count++;
    else agingBands[3].count++;
  });
  const maxAgeCount=Math.max(1,...agingBands.map(x=>x.count));
  $('#agingStats').innerHTML=agingBands.map(x=>`<div class="aging-row"><span class="age-dot ${x.cls}"></span><b>${x.label}</b><div class="aging-bar"><i class="${x.cls}" style="width:${(x.count/maxAgeCount)*100}%"></i></div><strong>${x.count}</strong></div>`).join('');
}

function bindDynamic() {
  $$('.ap-card').forEach(card => card.addEventListener('dragstart', e => { dragId=card.dataset.id; e.dataTransfer.setData('text/plain',dragId); }));
  $$('.edit-btn').forEach(b => b.addEventListener('click', e => { e.stopPropagation(); openEdit(b.dataset.id); }));
  $$('.history-btn').forEach(b => b.addEventListener('click', e => { e.stopPropagation(); openHistory(b.dataset.id); }));
  $$('.jira-more-btn').forEach(b => b.addEventListener('click', e => {
    e.stopPropagation();
    const comment = $(`.jira-comment[data-comment-id="${CSS.escape(b.dataset.id)}"]`);
    if(!comment) return;
    const expanded = comment.classList.toggle('expanded');
    b.textContent = expanded ? '접기' : '더보기';
  }));
  $$('.tag-filter-link,.tag-chip').forEach(b => b.addEventListener('click', e => { e.stopPropagation(); tagFilter=b.dataset.tag; render(); }));
  $$('.complete-btn,.queue-complete').forEach(b => b.addEventListener('click', async e => { e.stopPropagation(); await api(`/api/ap/${b.dataset.id}/complete`,{method:'POST'}); toast('완료 · 목록에서 제거했습니다.'); await load(); }));
  $$('.queue-item').forEach(item => {
    item.addEventListener('dragstart', e => { if(state.queueMode!=='manual' || tagFilter){e.preventDefault();return;} dragId=item.dataset.id; });
    item.addEventListener('dragover', e => { if(state.queueMode==='manual' && !tagFilter) e.preventDefault(); });
    item.addEventListener('drop', async e => {
      if(state.queueMode!=='manual' || tagFilter) return;
      e.preventDefault();
      const targetId=item.dataset.id;
      if(!dragId || dragId===targetId) return;
      const ids=orderedQueue().map(x=>x.id);
      const from=ids.indexOf(dragId), to=ids.indexOf(targetId);
      if(from<0||to<0)return;
      ids.splice(to,0,ids.splice(from,1)[0]);
      await api('/api/queue',{method:'POST',body:JSON.stringify({mode:'manual',ids})});
      await load();
    });
  });
}

function updateScorePreview(){
  const importance = Number($('#importance').value || 0);
  const baseUrgency = Number($('#baseUrgency').value || 0);
  const rawDeadline = $('#deadline').value;
  const deadline = rawDeadline ? new Date(rawDeadline).toISOString() : '';
  const preview = { importance, baseUrgency, deadline };
  const effectiveUrgency = urgency(preview);
  const q=quadrant(preview);
  $('#importanceValue').textContent = importance;
  $('#baseUrgencyValue').textContent = baseUrgency;
  $('#quadrantPreview').textContent = q;
  $('#urgencyPreview').textContent = `I${importance} · U${effectiveUrgency}`;
  $$('.q-pick').forEach(btn=>btn.classList.toggle('selected',btn.dataset.qSelect===q));
}

function resetApForm(){
  editingId=null;
  $('#apForm').reset();
  $('#apId').value='';
  $('#importance').value='80';
  $('#baseUrgency').value='20';
  $('#editorTitle').textContent='AP 추가';
  $('#editorSubtitle').textContent='새로운 작업(AP)을 입력하세요.';
  $('#deleteBtn').classList.add('hidden');
  $('#resetApFormBtn').classList.add('hidden');
  const submit=$('.editor-submit'); if(submit) submit.textContent='＋ AP 추가하기';
  updateScorePreview();
}

function openEdit(id){
  const ap=state.actionPoints.find(x=>x.id===id); if(!ap)return;
  editingId=id;
  $('#editorTitle').textContent='AP 편집';
  $('#editorSubtitle').textContent='수정 후 저장하면 기존 AP에 반영됩니다.';
  $('#deleteBtn').classList.remove('hidden');
  $('#resetApFormBtn').classList.remove('hidden');
  const submit=$('.editor-submit'); if(submit) submit.textContent='변경사항 저장';
  $('#title').value=ap.title;
  $('#details').value=ap.details||'';
  $('#importance').value=ap.importance;
  $('#baseUrgency').value=ap.baseUrgency;
  $('#deadline').value=toLocalInput(ap.deadline);
  $('#tagsInput').value=apTags(ap).join(', ');
  $('#jiraKey').value=ap.jira?.issueKey||'';
  updateScorePreview();
  $('#apEditorPanel').scrollTo({top:0,behavior:'smooth'});
  setTimeout(()=>$('#title').focus(),180);
}

function applyQuadrantPreset(q, focusTitle=false){
  const presets={Q1:[80,80],Q2:[80,20],Q3:[20,80],Q4:[20,20]};
  const pair=presets[q]||presets.Q2;
  $('#importance').value=String(pair[0]);
  $('#baseUrgency').value=String(pair[1]);
  updateScorePreview();
  if(focusTitle) setTimeout(()=>$('#title')?.focus(),80);
}
function startNewApInQuadrant(q){
  resetApForm();
  applyQuadrantPreset(q,true);
  $('#apEditorPanel')?.scrollTo({top:0,behavior:'smooth'});
}

function toLocalInput(v){ if(!v)return''; const d=new Date(v); if(Number.isNaN(d.getTime()))return''; const p=n=>String(n).padStart(2,'0'); return `${d.getFullYear()}-${p(d.getMonth()+1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`; }
function openHistory(id){
  const ap=state.actionPoints.find(x=>x.id===id); if(!ap)return;
  $('#historyTitle').textContent=`History · ${ap.title}`;
  const h=[...(ap.history||[])].reverse();
  $('#historyBody').innerHTML=h.length?h.map(x=>`<div class="history-row"><b>${esc(x.action)}</b>${x.note?` · ${esc(x.note)}`:''}<small>${esc(localDateTime(x.at))}</small></div>`).join(''):'<div class="empty">History가 없습니다.</div>';
  $('#historyDialog').showModal();
}
function toast(msg){ const t=$('#toast'); t.textContent=msg; t.classList.add('show'); clearTimeout(toast.timer); toast.timer=setTimeout(()=>t.classList.remove('show'),1800); }

async function openProfileSettings(){
  const p = await api('/api/profile');
  $('#profileDisplayName').value = p.displayName || '';
  $('#profileMessage').textContent = '';
  $('#profileDialog').showModal();
  setTimeout(()=>$('#profileDisplayName')?.focus(),80);
}
async function saveProfileSettings(){
  const displayName = $('#profileDisplayName').value.trim();
  if(!displayName) throw new Error('이름을 입력하세요.');
  const res = await api('/api/profile',{method:'POST',body:JSON.stringify({displayName})});
  if($('#displayName')) $('#displayName').textContent = res.displayName;
  $('#profileDialog').close();
  toast(`이름을 ${res.displayName}(으)로 저장했습니다.`);
}

async function openStorageSettings(firstRun=false){
  const info = await api('/api/storage');
  $('#storageDataPath').value = info.dataPath || '';
  $('#storageMessage').textContent = firstRun ? '처음 사용할 JSON 저장 위치를 확인해 주세요. 닫으면 기본 위치를 계속 사용합니다.' : '';
  $('#storageDialog').showModal();
}
async function saveStorageSettings(){
  const dataPath = $('#storageDataPath').value.trim();
  if(!dataPath) throw new Error('JSON 저장 위치를 입력하세요.');
  const res = await api('/api/storage', {method:'POST', body:JSON.stringify({dataPath})});
  storagePromptDismissed=true;
  $('#storageMessage').textContent = res.loadedExisting ? '기존 JSON을 불러왔습니다.' : '새 위치에 저장했습니다.';
  $('#storageDialog').close();
  toast('JSON 저장 위치를 적용했습니다.');
  await load();
}

async function openJiraSettings(){
  const s = await api('/api/jira/settings');
  $('#jiraBaseUrl').value = s.baseUrl || '';
  $('#jiraTokenPageUrl').value = s.tokenPageUrl || '';
  $('#jiraAuthMode').value = s.authMode || 'bearer';
  $('#jiraApiVersion').value = s.apiVersion || '2';
  $('#jiraUsername').value = s.username || '';
  $('#jiraToken').value = s.token || '';
  $('#jiraAutoSyncSeconds').value = s.autoSyncSeconds || 60;
  $('#jiraMessage').textContent = '';
  $('#jiraDialog').showModal();
}
function jiraSettingsPayload(){
  return {
    baseUrl: $('#jiraBaseUrl').value.trim(), tokenPageUrl: $('#jiraTokenPageUrl').value.trim(), authMode: $('#jiraAuthMode').value,
    username: $('#jiraUsername').value.trim(), token: $('#jiraToken').value.trim(), apiVersion: $('#jiraApiVersion').value,
    autoSyncSeconds: Number($('#jiraAutoSyncSeconds').value || 60),
  };
}
async function saveJiraSettings(){
  await api('/api/jira/settings', {method:'POST', body:JSON.stringify(jiraSettingsPayload())});
  $('#jiraMessage').textContent = '저장 완료 · Jira Sync 완료';
  await load();
}
async function syncJira(silent=false){
  if(jiraSyncInFlight) return;
  jiraSyncInFlight=true;
  try { await api('/api/jira/sync',{method:'POST',body:'{}'}); await load(); if(!silent) toast('Jira를 동기화했습니다.'); }
  catch(err){ if(!silent) alert(`Jira Sync 실패: ${err.message}`); }
  finally { jiraSyncInFlight=false; }
}

// Matrix drag & drop
$$('.dropzone').forEach(zone => {
  zone.addEventListener('dragover', e => { e.preventDefault(); zone.classList.add('drag-over'); });
  zone.addEventListener('dragleave', () => zone.classList.remove('drag-over'));
  zone.addEventListener('drop', async e => {
    e.preventDefault(); zone.classList.remove('drag-over');
    const id=e.dataTransfer.getData('text/plain')||dragId;
    const ap=state.actionPoints.find(x=>x.id===id); if(!ap)return;
    await api(`/api/ap/${id}`,{method:'PUT',body:JSON.stringify({title:ap.title,details:ap.details||'',quadrant:zone.dataset.quadrant,deadline:ap.deadline||'',jira:ap.jira?{issueKey:ap.jira.issueKey||'',url:ap.jira.url||''}:null})});
    await load();
  });
});

// Static controls. Every binding is null-safe so one missing control cannot break dialog close handlers.
$$('.side-link[data-scroll-target]').forEach(b => b.addEventListener('click', () => { $$('.side-link').forEach(x=>x.classList.remove('active')); b.classList.add('active'); document.getElementById(b.dataset.scrollTarget)?.scrollIntoView({behavior:'smooth',block:'start'}); }));
on('#sideProfileSettingsBtn','click',openProfileSettings);
on('#editNameBtn','click',openProfileSettings);
on('#sideJiraSettingsBtn','click',openJiraSettings);
on('#sideStorageSettingsBtn','click',()=>openStorageSettings(false));
on('#reloadBtn','click',load);
on('#jiraSyncBtn','click',()=>syncJira(false));
on('#queueMode','change',async e=>{ await api('/api/queue',{method:'POST',body:JSON.stringify({mode:e.target.value,ids:[]})}); await load(); });
on('#calendarPrevBtn','click',()=>{ calendarCursor=new Date(calendarCursor.getFullYear(),calendarCursor.getMonth()-1,1); renderCalendar(); });
on('#calendarNextBtn','click',()=>{ calendarCursor=new Date(calendarCursor.getFullYear(),calendarCursor.getMonth()+1,1); renderCalendar(); });
on('#clearTagFilterBtn','click',()=>{ tagFilter=null; render(); });
on('#importance','input',updateScorePreview);
on('#baseUrgency','input',updateScorePreview);
on('#deadline','input',updateScorePreview);
$$('.q-pick').forEach(b=>b.addEventListener('click',()=>applyQuadrantPreset(b.dataset.qSelect,true)));
$$('.quadrant-add').forEach(b=>b.addEventListener('click',()=>startNewApInQuadrant(b.dataset.qAdd)));
on('#editNameAvatar','click',openProfileSettings);
on('#resetApFormBtn','click',resetApForm);

on('#apForm','submit',async e=>{
  e.preventDefault();
  const deadlineValue=$('#deadline').value;
  const payload={
    title:$('#title').value.trim(), details:$('#details').value.trim(), importance:Number($('#importance').value), baseUrgency:Number($('#baseUrgency').value),
    deadline:deadlineValue?new Date(deadlineValue).toISOString():'', tags:parseTags($('#tagsInput').value),
    jira:$('#jiraKey').value.trim()?{issueKey:$('#jiraKey').value.trim(),url:''}:null,
  };
  if(!payload.title)return;
  const wasEditing=!!editingId;
  if(editingId) await api(`/api/ap/${editingId}`,{method:'PUT',body:JSON.stringify(payload)});
  else await api('/api/ap',{method:'POST',body:JSON.stringify(payload)});
  resetApForm(); toast(wasEditing?'수정했습니다.':'AP를 추가했습니다.'); await load();
});
on('#deleteBtn','click',async()=>{
  if(!editingId)return;
  if(!confirm('이 AP를 삭제할까요? 완료 Archive에는 남지 않습니다.'))return;
  await api(`/api/ap/${editingId}`,{method:'DELETE'}); resetApForm(); toast('삭제했습니다.'); await load();
});

// Profile dialog
on('#profileCloseBtn','click',()=>$('#profileDialog')?.close());
on('#profileCancelBtn','click',()=>$('#profileDialog')?.close());
on('#profileForm','submit',async e=>{
  e.preventDefault();
  try{ await saveProfileSettings(); }
  catch(err){ $('#profileMessage').textContent=`저장 실패: ${err.message}`; }
});

// Jira dialog
on('#jiraCloseBtn','click',()=>$('#jiraDialog')?.close());
on('#jiraCancelBtn','click',()=>$('#jiraDialog')?.close());
on('#jiraForm','submit',async e=>{ e.preventDefault(); try{ await saveJiraSettings(); $('#jiraDialog').close(); toast('Jira 설정을 저장했습니다.'); }catch(err){ $('#jiraMessage').textContent=`저장 실패: ${err.message}`; }});
on('#openTokenPageBtn','click',async()=>{ const url=$('#jiraTokenPageUrl').value.trim(); if(!url)return alert('Token 발급 페이지 URL을 입력하세요.'); try{await api('/api/open-url',{method:'POST',body:JSON.stringify({url})});}catch(err){alert(err.message);} });
on('#jiraTestBtn','click',async()=>{ try{ await api('/api/jira/settings',{method:'POST',body:JSON.stringify(jiraSettingsPayload())}); const res=await api('/api/jira/test',{method:'POST',body:'{}'}); $('#jiraMessage').textContent=res.message||'Jira 연결 성공'; await load(); }catch(err){ $('#jiraMessage').textContent=`연결 실패: ${err.message}`; }});

// Storage dialog
on('#storageCloseBtn','click',()=>{ storagePromptDismissed=true; $('#storageDialog')?.close(); });
on('#storageCancelBtn','click',()=>{ storagePromptDismissed=true; $('#storageDialog')?.close(); });
on('#storageForm','submit',async e=>{ e.preventDefault(); try{await saveStorageSettings();}catch(err){$('#storageMessage').textContent=`저장 실패: ${err.message}`;} });
on('#pickStorageFolderBtn','click',async()=>{ $('#storageMessage').textContent='폴더 선택 창을 여는 중...'; try{const res=await api('/api/storage/pick-folder',{method:'POST',body:'{}'}); if(res.dataPath)$('#storageDataPath').value=res.dataPath; $('#storageMessage').textContent=res.cancelled?'폴더 선택을 취소했습니다.':'';}catch(err){$('#storageMessage').textContent=`${err.message} 직접 경로를 입력해도 됩니다.`;} });

on('#closeHistoryBtn','click',()=>$('#historyDialog')?.close());

// Clicking the backdrop closes all settings/history dialogs as well.
['#profileDialog','#jiraDialog','#storageDialog','#historyDialog'].forEach(sel=>on(sel,'click',e=>{ if(e.target===e.currentTarget){ if(sel==='#storageDialog') storagePromptDismissed=true; e.currentTarget.close(); }}));

resetApForm();
load().catch(err=>{ console.error(err); alert(`로드 실패: ${err.message}`); });
setInterval(()=>{ if(state) render(); },60_000);


async function refreshRunningVersionBadge(){
  try{
    const info = await api('/api/info');
    const el = document.getElementById('appVersion');
    if(el && info?.version) el.textContent = `v${info.version}`;
    if($('#displayName') && info?.displayName) $('#displayName').textContent = info.displayName;
  }catch(_){}
}
refreshRunningVersionBadge();
