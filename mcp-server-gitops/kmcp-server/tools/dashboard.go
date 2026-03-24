package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>K8s Health Dashboard</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f172a; color: #e2e8f0; padding: 16px; }
  h1 { font-size: 20px; margin-bottom: 16px; color: #38bdf8; }
  h2 { font-size: 15px; margin-bottom: 8px; color: #94a3b8; text-transform: uppercase; letter-spacing: 0.05em; }
  .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 16px; }
  .card { background: #1e293b; border-radius: 8px; padding: 14px; border: 1px solid #334155; }
  .card.full { grid-column: 1 / -1; }
  .stat { display: flex; align-items: baseline; gap: 8px; margin-bottom: 4px; }
  .stat-value { font-size: 28px; font-weight: 700; color: #f8fafc; }
  .stat-label { font-size: 13px; color: #64748b; }
  table { width: 100%; border-collapse: collapse; font-size: 13px; }
  th { text-align: left; padding: 6px 8px; color: #64748b; border-bottom: 1px solid #334155; font-weight: 500; }
  td { padding: 6px 8px; border-bottom: 1px solid #1e293b; }
  .badge { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 11px; font-weight: 600; }
  .badge-green { background: #065f46; color: #6ee7b7; }
  .badge-red { background: #7f1d1d; color: #fca5a5; }
  .badge-yellow { background: #713f12; color: #fde047; }
  .loading { color: #64748b; font-style: italic; }
  .error { color: #fca5a5; }
  .refresh-btn { background: #334155; color: #e2e8f0; border: 1px solid #475569; border-radius: 6px; padding: 6px 14px; cursor: pointer; font-size: 13px; float: right; }
  .refresh-btn:hover { background: #475569; }
  .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
  .events-list { max-height: 200px; overflow-y: auto; }
  .event-item { padding: 6px 0; border-bottom: 1px solid #1e293b; font-size: 13px; }
  .event-time { color: #64748b; font-size: 11px; }
</style>
</head>
<body>
<div class="header">
  <h1>Kubernetes Health Dashboard</h1>
  <button class="refresh-btn" onclick="refreshAll()">Refresh</button>
</div>

<div class="grid">
  <div class="card">
    <h2>Nodes</h2>
    <div id="nodes-stats" class="loading">Loading...</div>
    <div id="nodes-table"></div>
  </div>
  <div class="card">
    <h2>Summary</h2>
    <div id="summary" class="loading">Loading...</div>
  </div>
  <div class="card full">
    <h2>Pods — <select id="ns-select" onchange="loadPods(this.value)" style="background:#334155;color:#e2e8f0;border:1px solid #475569;border-radius:4px;padding:2px 6px;font-size:13px;">
      <option value="kagent">kagent</option>
      <option value="agentgateway-system">agentgateway-system</option>
      <option value="flux-system">flux-system</option>
      <option value="default">default</option>
      <option value="kube-system">kube-system</option>
    </select></h2>
    <div id="pods-table" class="loading">Loading...</div>
  </div>
  <div class="card full">
    <h2>Deployments — <span id="deploy-ns">kagent</span></h2>
    <div id="deployments-table" class="loading">Loading...</div>
  </div>
  <div class="card full">
    <h2>Warning Events</h2>
    <div id="events-list" class="events-list loading">Loading...</div>
  </div>
</div>

<script>
// MCP Apps communication via postMessage
let requestId = 0;
const pending = new Map();

function sendMcpRequest(method, params) {
  return new Promise((resolve, reject) => {
    const id = ++requestId;
    pending.set(id, { resolve, reject });
    window.parent.postMessage({ jsonrpc: "2.0", id, method, params }, "*");
    setTimeout(() => {
      if (pending.has(id)) {
        pending.delete(id);
        reject(new Error("Timeout"));
      }
    }, 15000);
  });
}

window.addEventListener("message", (event) => {
  const msg = event.data;
  if (msg && msg.jsonrpc === "2.0" && msg.id && pending.has(msg.id)) {
    const { resolve, reject } = pending.get(msg.id);
    pending.delete(msg.id);
    if (msg.error) reject(new Error(msg.error.message));
    else resolve(msg.result);
  }
});

async function callTool(name, args) {
  try {
    const result = await sendMcpRequest("tools/call", { name, arguments: args });
    if (result && result.content && result.content[0]) {
      return result.content[0].text || "";
    }
    return "";
  } catch (e) {
    return "ERROR: " + e.message;
  }
}

function badge(text, type) {
  return '<span class="badge badge-' + type + '">' + text + '</span>';
}

async function loadNodes() {
  const el = document.getElementById("nodes-stats");
  const tableEl = document.getElementById("nodes-table");
  try {
    const text = await callTool("get_node_status", {});
    if (text.startsWith("ERROR")) { el.innerHTML = '<span class="error">' + text + '</span>'; return; }
    const lines = text.split("\n").filter(l => l.trim() && !l.startsWith("Cluster") && !l.startsWith("---") && !l.startsWith("NAME"));
    const nodes = lines.map(l => {
      const parts = l.trim().split(/\s{2,}/);
      return { name: parts[0], status: parts[1], version: parts[2] };
    }).filter(n => n.name);
    const ready = nodes.filter(n => n.status === "Ready").length;
    el.innerHTML = '<div class="stat"><span class="stat-value">' + ready + '/' + nodes.length + '</span><span class="stat-label">Ready</span></div>';
    tableEl.innerHTML = '<table><tr><th>Name</th><th>Status</th><th>Version</th></tr>' +
      nodes.map(n => '<tr><td>' + n.name + '</td><td>' + badge(n.status, n.status === "Ready" ? "green" : "red") + '</td><td>' + (n.version||"") + '</td></tr>').join("") + '</table>';
    updateSummary("nodes", ready, nodes.length);
  } catch(e) { el.innerHTML = '<span class="error">' + e.message + '</span>'; }
}

let summaryData = { nodes: null, pods: null, deployments: null };
function updateSummary(key, ready, total) {
  summaryData[key] = { ready, total };
  const el = document.getElementById("summary");
  const items = Object.entries(summaryData).filter(([,v]) => v !== null);
  if (items.length === 0) return;
  el.innerHTML = items.map(([k, v]) => {
    const pct = v.total > 0 ? Math.round(v.ready / v.total * 100) : 0;
    const color = pct === 100 ? "green" : pct > 50 ? "yellow" : "red";
    return '<div class="stat"><span class="stat-value">' + v.ready + '/' + v.total + '</span><span class="stat-label">' + k + '</span>' + badge(pct + '%', color) + '</div>';
  }).join("");
}

async function loadPods(ns) {
  const el = document.getElementById("pods-table");
  el.innerHTML = '<span class="loading">Loading...</span>';
  try {
    const text = await callTool("get_pod_status", { namespace: ns });
    if (text.startsWith("ERROR")) { el.innerHTML = '<span class="error">' + text + '</span>'; return; }
    if (text.includes("No pods")) { el.innerHTML = '<em>' + text + '</em>'; updateSummary("pods", 0, 0); return; }
    const lines = text.split("\n").filter(l => l.trim() && !l.startsWith("Pods in") && !l.startsWith("---") && !l.startsWith("NAME"));
    const pods = lines.map(l => {
      const parts = l.trim().split(/\s{2,}/);
      return { name: parts[0], status: parts[1], ready: parts[2], restarts: parts[3] };
    }).filter(p => p.name);
    const running = pods.filter(p => p.status === "Running").length;
    el.innerHTML = '<table><tr><th>Name</th><th>Status</th><th>Ready</th><th>Restarts</th></tr>' +
      pods.map(p => '<tr><td style="font-size:12px">' + p.name + '</td><td>' + badge(p.status, p.status === "Running" ? "green" : "red") + '</td><td>' + (p.ready||"") + '</td><td>' + (p.restarts||"0") + '</td></tr>').join("") + '</table>';
    updateSummary("pods", running, pods.length);
  } catch(e) { el.innerHTML = '<span class="error">' + e.message + '</span>'; }
}

async function loadDeployments(ns) {
  const el = document.getElementById("deployments-table");
  document.getElementById("deploy-ns").textContent = ns;
  el.innerHTML = '<span class="loading">Loading...</span>';
  try {
    const text = await callTool("get_deployment_status", { namespace: ns });
    if (text.startsWith("ERROR")) { el.innerHTML = '<span class="error">' + text + '</span>'; return; }
    if (text.includes("No deployments")) { el.innerHTML = '<em>' + text + '</em>'; updateSummary("deployments", 0, 0); return; }
    const lines = text.split("\n").filter(l => l.trim() && !l.startsWith("Deploy") && !l.startsWith("---") && !l.startsWith("NAME"));
    const deps = lines.map(l => {
      const parts = l.trim().split(/\s{2,}/);
      return { name: parts[0], ready: parts[1], upToDate: parts[2], available: parts[3] };
    }).filter(d => d.name);
    const allReady = deps.filter(d => { const p = (d.ready||"").split("/"); return p[0] === p[1]; }).length;
    el.innerHTML = '<table><tr><th>Name</th><th>Ready</th><th>Up-to-date</th><th>Available</th></tr>' +
      deps.map(d => {
        const p = (d.ready||"").split("/");
        const ok = p[0] === p[1];
        return '<tr><td>' + d.name + '</td><td>' + badge(d.ready||"", ok ? "green" : "red") + '</td><td>' + (d.upToDate||"") + '</td><td>' + (d.available||"") + '</td></tr>';
      }).join("") + '</table>';
    updateSummary("deployments", allReady, deps.length);
  } catch(e) { el.innerHTML = '<span class="error">' + e.message + '</span>'; }
}

async function loadEvents() {
  const el = document.getElementById("events-list");
  try {
    const text = await callTool("get_events", { namespace: "", limit: 10 });
    if (text.startsWith("ERROR")) { el.innerHTML = '<span class="error">' + text + '</span>'; return; }
    if (text.includes("No warning")) { el.innerHTML = '<em style="color:#6ee7b7">No warning events — cluster is healthy</em>'; return; }
    const lines = text.split("\n").filter(l => l.trim() && !l.startsWith("Warning"));
    el.innerHTML = lines.map(l => '<div class="event-item">' + l.replace(/\[([^\]]+)\]/, '<span class="event-time">$1</span>') + '</div>').join("") || '<em>No events</em>';
  } catch(e) { el.innerHTML = '<span class="error">' + e.message + '</span>'; }
}

function refreshAll() {
  const ns = document.getElementById("ns-select").value;
  summaryData = { nodes: null, pods: null, deployments: null };
  loadNodes();
  loadPods(ns);
  loadDeployments(ns);
  loadEvents();
}

// Initial load
refreshAll();
</script>
</body>
</html>`

func (k *K8sTools) DashboardResource(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "ui://k8s-dashboard",
			MIMEType: "text/html;profile=mcp-app",
			Text:     dashboardHTML,
		},
	}, nil
}

func (k *K8sTools) ClusterDashboard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Collect a quick summary to return alongside the UI
	nodes, _ := k.GetNodeStatus(ctx, req)
	summary := "Cluster dashboard loaded."
	if nodes != nil && len(nodes.Content) > 0 {
		if tc, ok := nodes.Content[0].(mcp.TextContent); ok {
			summary = fmt.Sprintf("Dashboard ready.\n\n%s", tc.Text)
		}
	}
	return mcp.NewToolResultText(summary), nil
}
