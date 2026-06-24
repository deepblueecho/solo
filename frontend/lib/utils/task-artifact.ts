import type { Task } from '@/lib/types';

export type TaskArtifactAction = 'hidden' | 'generate' | 'pending' | 'read';

export function getTaskArtifactAction(task: Task | null | undefined, isGenerating = false): TaskArtifactAction {
  if (!task || task.parent_task_id) return 'hidden';
  const status = isGenerating ? 'pending' : (task.artifact_status ?? 'none');
  if (status === 'available') return 'read';
  if (task.status === 'in_review') {
    if (status === 'pending') return 'pending';
    return 'generate';
  }
  return 'hidden';
}

export function taskArtifactActionLabel(action: TaskArtifactAction): string {
  if (action === 'generate') return 'Generate Artifact';
  if (action === 'pending') return 'Generating';
  return 'Artifact';
}
