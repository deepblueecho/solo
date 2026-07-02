import { readFileSync } from 'node:fs';

const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8');
const assert = (condition, message) => {
  if (!condition) throw new Error(message);
};

const serviceAgent = read('../internal/server/service/agent.go');
const activity = read('lib/agent-activity.ts');
const i18n = read('lib/i18n.ts');
const island = read('components/agents/agent-island.tsx');
const observability = read('components/agents/agent-observability-tab.tsx');
const liveMonitor = read('components/dashboard/live-monitor.tsx');
const islandHook = read('lib/hooks/use-agent-island.ts');

for (const text of ['已接收，正在处理', '仍在运行，暂无可见回复', '仍在运行，暂无新的进度', 'No available daemon to run this agent.']) {
  assert(!serviceAgent.includes(text), `backend should not emit hardcoded product copy: ${text}`);
}

for (const key of ['agentActivityAccepted', 'agentActivityNoVisibleReply', 'agentActivityNoProgress']) {
  assert(i18n.includes(key), `${key} should be translated`);
}

assert(
  activity.includes('ACTIVITY_TEXT_KEYS') &&
    activity.includes('agent.activity.accepted') &&
    activity.includes('agentActivityAccepted'),
  'displayAgentActivity should translate stable activity codes',
);
assert(island.includes('displayAgentActivity('), 'AgentIsland should display translated activity text');
assert(
  !islandHook.includes('setInterval(loadActiveRuns') &&
    islandHook.includes("window.addEventListener('focus', handleFocus)") &&
    islandHook.includes("window.removeEventListener('focus', handleFocus)") &&
    islandHook.includes('if (!isConnected) return;') &&
    islandHook.includes('if (hasConnectedRef.current)') &&
    islandHook.includes('void loadActiveRuns();'),
  'AgentIsland should reconcile active runs on focus and websocket reconnect without polling',
);
assert(observability.includes('displayAgentActivity('), 'observability run lists should display translated activity text');
assert(
  liveMonitor.includes('displayAgentActivity(run.status, run.activity_text, undefined'),
  'timeline titles should translate activity text',
);
assert(
  liveMonitor.includes('const activity = displayAgentActivity(agent.status, agent.activity_text, agent.tool_input_summary)') &&
    liveMonitor.includes('title={activity}') &&
    liveMonitor.includes('group-hover:line-clamp-none') &&
    liveMonitor.includes('group-focus:line-clamp-none'),
  'live monitor agent cards should expose full truncated activity text on hover/focus',
);

console.log('agent activity i18n source checks passed');
