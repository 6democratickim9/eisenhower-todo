import * as vscode from "vscode";
import { randomUUID } from "crypto";
import { LocalJsonStorage } from "./storage";
import {
  ActionPoint,
  PlannerData,
  Quadrant
} from "./types";
import {
  baseUrgencyForQuadrant,
  calculateUrgency,
  getQuadrant,
  importanceForQuadrant
} from "./urgency";

export function activate(context: vscode.ExtensionContext) {
  const disposable = vscode.commands.registerCommand("eisenhowerTodo.open", async () => {
    try {
      const root = LocalJsonStorage.resolveWorkspaceRoot();
      const storage = new LocalJsonStorage(root);
      const panel = vscode.window.createWebviewPanel(
        "eisenhowerTodo",
        "Eisenhower Todo",
        vscode.ViewColumn.One,
        { enableScripts: true, retainContextWhenHidden: true }
      );

      let data = await storage.load();

      const render = () => {
        panel.webview.html = getHtml(panel.webview, data, storage.getPath());
      };

      render();

      panel.webview.onDidReceiveMessage(async (message) => {
        switch (message.type) {
          case "create": {
            const now = new Date().toISOString();
            const q = (message.quadrant || "Q2") as Quadrant;
            const ap: ActionPoint = {
              id: randomUUID(),
              title: String(message.title || "").trim(),
              details: cleanOptional(message.details),
              importance: importanceForQuadrant(q),
              baseUrgency: baseUrgencyForQuadrant(q),
              deadline: cleanOptional(message.deadline),
              createdAt: now,
              updatedAt: now,
              queueOrder: nextQueueOrder(data),
              jira: message.jiraUrl
                ? {
                    url: String(message.jiraUrl).trim(),
                    issueKey: cleanOptional(message.jiraIssueKey)
                  }
                : undefined,
              history: [{ at: now, action: "created" }]
            };
            if (!ap.title) return;
            data.actionPoints.push(ap);
            break;
          }

          case "move": {
            const ap = findActive(data, message.id);
            if (!ap) return;
            const q = message.quadrant as Quadrant;
            ap.importance = importanceForQuadrant(q);
            ap.baseUrgency = baseUrgencyForQuadrant(q);
            ap.updatedAt = new Date().toISOString();
            ap.history.push({
              at: ap.updatedAt,
              action: "moved",
              note: q
            });
            break;
          }

          case "complete": {
            const index = data.actionPoints.findIndex((x) => x.id === message.id);
            if (index < 0) return;
            const [ap] = data.actionPoints.splice(index, 1);
            ap.completedAt = new Date().toISOString();
            ap.updatedAt = ap.completedAt;
            ap.history.push({ at: ap.completedAt, action: "completed" });
            data.archive.unshift(ap);
            break;
          }

          case "delete": {
            data.actionPoints = data.actionPoints.filter((x) => x.id !== message.id);
            break;
          }

          case "reorder": {
            const orderedIds = Array.isArray(message.ids) ? message.ids : [];
            orderedIds.forEach((id: string, idx: number) => {
              const ap = findActive(data, id);
              if (ap) ap.queueOrder = idx;
            });
            break;
          }

          case "refresh":
            data = await storage.load();
            break;
        }

        if (message.type !== "refresh") {
          await storage.save(data);
        }
        render();
      });
    } catch (error: any) {
      vscode.window.showErrorMessage(error?.message ?? String(error));
    }
  });

  context.subscriptions.push(disposable);
}

export function deactivate() {}

function findActive(data: PlannerData, id: string): ActionPoint | undefined {
  return data.actionPoints.find((x) => x.id === id);
}

function nextQueueOrder(data: PlannerData): number {
  if (!data.actionPoints.length) return 0;
  return Math.max(...data.actionPoints.map((x) => x.queueOrder ?? 0)) + 1;
}

function cleanOptional(value: unknown): string | undefined {
  const text = typeof value === "string" ? value.trim() : "";
  return text || undefined;
}

function esc(value: unknown): string {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function getHtml(webview: vscode.Webview, data: PlannerData, dataPath: string): string {
  const quadrants: Quadrant[] = ["Q1", "Q2", "Q3", "Q4"];
  const titles: Record<Quadrant, string> = {
    Q1: "Urgent + Important",
    Q2: "Not Urgent + Important",
    Q3: "Urgent + Not Important",
    Q4: "Not Urgent + Not Important"
  };

  const active = [...data.actionPoints];
  const byQueue = [...active].sort((a, b) => (a.queueOrder ?? 9999) - (b.queueOrder ?? 9999));

  const card = (ap: ActionPoint) => {
    const urgency = calculateUrgency(ap);
    const details = ap.details ? `<div class="details">${esc(ap.details)}</div>` : "";
    const deadline = ap.deadline ? `<div class="meta">⏰ ${esc(ap.deadline)}</div>` : "";
    const jira = ap.jira?.url
      ? `<div class="meta">🔗 <a href="${esc(ap.jira.url)}">${esc(ap.jira.issueKey || ap.jira.url)}</a></div>`
      : "";

    return `
      <div class="card" draggable="true" data-id="${ap.id}">
        <div class="card-head">
          <strong>${esc(ap.title)}</strong>
          <button class="done" data-id="${ap.id}">✓</button>
        </div>
        ${details}
        ${deadline}
        ${jira}
        <div class="score">Importance ${ap.importance} · Urgency ${urgency}</div>
      </div>
    `;
  };

  const quadrantsHtml = quadrants.map((q) => `
    <section class="quadrant" data-q="${q}">
      <h3>${q} <span>${titles[q]}</span></h3>
      <div class="dropzone" data-q="${q}">
        ${active.filter((ap) => getQuadrant(ap) === q).map(card).join("")}
      </div>
    </section>
  `).join("");

  const queueHtml = byQueue.map((ap) => `
    <div class="queue-item" draggable="true" data-id="${ap.id}">
      <span class="grab">☰</span>
      <span>${esc(ap.title)}</span>
      <small>${getQuadrant(ap)}</small>
    </div>
  `).join("");

  const archiveHtml = data.archive.slice(0, 15).map((ap) => `
    <div class="archive-item">
      <s>${esc(ap.title)}</s>
      <small>${esc(ap.completedAt || "")}</small>
    </div>
  `).join("");

  return `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<style>
  body { font-family: var(--vscode-font-family); padding: 16px; color: var(--vscode-foreground); background: var(--vscode-editor-background); }
  .top { display:flex; justify-content:space-between; gap:12px; align-items:center; margin-bottom:12px; }
  .path { opacity:.65; font-size:11px; word-break:break-all; }
  .new { display:grid; grid-template-columns:2fr 2fr 1fr 1fr 1fr 2fr; gap:8px; margin-bottom:16px; }
  input, textarea, select, button { color:var(--vscode-input-foreground); background:var(--vscode-input-background); border:1px solid var(--vscode-input-border); padding:7px; }
  button { cursor:pointer; }
  .matrix { display:grid; grid-template-columns:1fr 1fr; gap:10px; }
  .quadrant { border:1px solid var(--vscode-panel-border); border-radius:8px; padding:10px; min-height:180px; }
  .quadrant h3 { margin:0 0 8px; }
  .quadrant h3 span { font-weight:normal; opacity:.7; font-size:12px; }
  .dropzone { min-height:135px; }
  .card { border:1px solid var(--vscode-panel-border); background:var(--vscode-sideBar-background); border-radius:6px; padding:9px; margin:7px 0; cursor:grab; }
  .card-head { display:flex; justify-content:space-between; gap:8px; }
  .done { width:30px; }
  .details { white-space:pre-wrap; margin-top:7px; opacity:.9; }
  .meta, .score { margin-top:5px; font-size:11px; opacity:.75; }
  .bottom { display:grid; grid-template-columns:1.3fr .7fr; gap:12px; margin-top:14px; }
  .panel { border:1px solid var(--vscode-panel-border); border-radius:8px; padding:10px; }
  .queue-item { display:grid; grid-template-columns:24px 1fr 35px; gap:6px; align-items:center; border-bottom:1px solid var(--vscode-panel-border); padding:8px 2px; cursor:grab; }
  .archive-item { display:flex; justify-content:space-between; gap:10px; padding:5px 0; opacity:.7; }
  a { color:var(--vscode-textLink-foreground); }
  @media (max-width: 900px) {
    .new { grid-template-columns:1fr; }
    .matrix, .bottom { grid-template-columns:1fr; }
  }
</style>
</head>
<body>
  <div class="top">
    <div>
      <h2>Eisenhower Todo</h2>
      <div class="path">Local JSON: ${esc(dataPath)}</div>
    </div>
    <button id="refresh">Refresh JSON</button>
  </div>

  <div class="new">
    <input id="title" placeholder="Action Point *" />
    <input id="details" placeholder="Details (optional)" />
    <select id="quadrant">
      <option value="Q1">Q1</option>
      <option value="Q2" selected>Q2</option>
      <option value="Q3">Q3</option>
      <option value="Q4">Q4</option>
    </select>
    <input id="deadline" type="datetime-local" />
    <input id="jiraKey" placeholder="Jira key (optional)" />
    <input id="jiraUrl" placeholder="Jira URL (optional)" />
  </div>
  <button id="add">+ Add AP</button>

  <div class="matrix">${quadrantsHtml}</div>

  <div class="bottom">
    <div class="panel">
      <h3>Today's Queue</h3>
      <div id="queue">${queueHtml || "<small>No active APs.</small>"}</div>
    </div>
    <div class="panel">
      <h3>Archive</h3>
      ${archiveHtml || "<small>No completed APs.</small>"}
    </div>
  </div>

<script>
  const vscode = acquireVsCodeApi();

  document.getElementById("add").addEventListener("click", () => {
    const title = document.getElementById("title").value.trim();
    if (!title) return;
    vscode.postMessage({
      type: "create",
      title,
      details: document.getElementById("details").value,
      quadrant: document.getElementById("quadrant").value,
      deadline: document.getElementById("deadline").value,
      jiraIssueKey: document.getElementById("jiraKey").value,
      jiraUrl: document.getElementById("jiraUrl").value
    });
  });

  document.getElementById("refresh").addEventListener("click", () => {
    vscode.postMessage({ type: "refresh" });
  });

  document.querySelectorAll(".done").forEach((el) => {
    el.addEventListener("click", () => {
      vscode.postMessage({ type: "complete", id: el.dataset.id });
    });
  });

  document.querySelectorAll(".card").forEach((el) => {
    el.addEventListener("dragstart", (e) => {
      e.dataTransfer.setData("text/plain", el.dataset.id);
    });
  });

  document.querySelectorAll(".dropzone").forEach((zone) => {
    zone.addEventListener("dragover", (e) => e.preventDefault());
    zone.addEventListener("drop", (e) => {
      e.preventDefault();
      const id = e.dataTransfer.getData("text/plain");
      vscode.postMessage({ type: "move", id, quadrant: zone.dataset.q });
    });
  });

  let queueDragId = null;
  document.querySelectorAll(".queue-item").forEach((el) => {
    el.addEventListener("dragstart", () => queueDragId = el.dataset.id);
    el.addEventListener("dragover", (e) => e.preventDefault());
    el.addEventListener("drop", (e) => {
      e.preventDefault();
      const target = el;
      const dragged = document.querySelector('.queue-item[data-id="' + queueDragId + '"]');
      if (dragged && dragged !== target) {
        target.parentNode.insertBefore(dragged, target);
        const ids = [...document.querySelectorAll(".queue-item")].map(x => x.dataset.id);
        vscode.postMessage({ type: "reorder", ids });
      }
    });
  });
</script>
</body>
</html>`;
}
