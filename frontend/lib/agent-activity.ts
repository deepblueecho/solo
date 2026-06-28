import { t } from '@/lib/i18n';
import type { AgentRunStatus } from '@/lib/hooks/use-agent-island';

const STATUS_ACTIVITY_TEXT = new Set([
  '等待执行',
  '执行中',
  '运行中',
  '思考中',
  '思考中…',
  '生成中',
  '已完成',
  '执行失败',
  '失败',
  'thinking...',
  'generating...',
  'using tool',
  'error',
]);

export function agentRunStatusText(status?: AgentRunStatus): string {
  switch (status) {
  case 'queued':
    return t('runQueued');
  case 'thinking':
    return t('runThinking');
  case 'running':
    return t('runRunning');
  case 'streaming':
    return t('runStreaming');
  case 'waiting_input':
    return t('runWaitingInput');
  case 'waiting_approval':
    return t('runWaitingApproval');
  case 'completed':
    return t('runCompleted');
  case 'failed':
    return t('runFailed');
  case 'cancelled':
    return t('runCancelled');
  case 'timeout':
    return t('runTimeout');
  default:
    return t('agentIdle');
  }
}

export function displayAgentActivity(status: AgentRunStatus | undefined, activityText?: string | null, toolInputSummary?: string | null, fallback?: string): string {
  const summary = toolInputSummary?.trim();
  if (summary) return summary;

  const text = activityText?.trim();
  if (!text) return fallback ?? agentRunStatusText(status);

  if (STATUS_ACTIVITY_TEXT.has(text.toLowerCase()) || STATUS_ACTIVITY_TEXT.has(text)) {
    return fallback ?? agentRunStatusText(status);
  }

  return text;
}
