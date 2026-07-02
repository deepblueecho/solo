import { readFileSync } from 'node:fs';

const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8');
const assert = (condition, message) => {
  if (!condition) throw new Error(message);
};

const channelView = read('components/dashboard/channel-view.tsx');
const threadPanel = read('components/dashboard/thread-panel.tsx');
const wsTypes = read('lib/ws-types.ts');
const i18n = read('lib/i18n.ts');
const activity = read('lib/agent-activity.ts');

assert(
  wsTypes.includes("type: 'agent.error'") && wsTypes.includes('error?: string'),
  'agent.error websocket type should carry the backend error field',
);
assert(
  i18n.includes('agentRunFailedToast'),
  'i18n should define an agent runtime failure toast message',
);
assert(
  activity.includes('displayAgentErrorReason') && activity.includes('agent.error.no_available_daemon'),
  'agent error reasons should map stable backend codes through i18n',
);
assert(
  channelView.includes("event.type === 'agent.error'") &&
    channelView.includes("event.channel_id === channel.id") &&
    channelView.includes("showToast(t('agentRunFailedToast'") &&
    channelView.includes('displayAgentErrorReason(event.error, event.detail)'),
  'channel view should toast channel-scoped agent.error events',
);
assert(
  threadPanel.includes("event.type === 'agent.error'") &&
    threadPanel.includes('event.thread_id === threadId') &&
    threadPanel.includes("showToast(t('agentRunFailedToast'") &&
    threadPanel.includes('displayAgentErrorReason(event.error, event.detail)'),
  'thread panel should toast thread-scoped agent.error events',
);

console.log('agent error toast source checks passed');
