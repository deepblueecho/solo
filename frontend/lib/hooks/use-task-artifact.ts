'use client';

import { useCallback, useRef, useState } from 'react';
import { apiClient } from '@/lib/api-client';
import type { TaskArtifact } from '@/lib/types';

const ARTIFACT_POLL_INTERVAL_MS = 1500;
const ARTIFACT_POLL_ATTEMPTS = 200;

export class TaskArtifactStillPendingError extends Error {
  constructor() {
    super('Task artifact is still generating');
    this.name = 'TaskArtifactStillPendingError';
  }
}

const sleep = (ms: number) => new Promise((resolve) => window.setTimeout(resolve, ms));

export function useTaskArtifact() {
  const [generatingTaskIds, setGeneratingTaskIds] = useState<Set<string>>(() => new Set());
  const inFlightByTaskRef = useRef<Map<string, Promise<TaskArtifact>>>(new Map());

  const waitForPublishedArtifact = useCallback(async (taskId: string, mode: 'latest' | 'final', baseline: TaskArtifact): Promise<TaskArtifact> => {
    if (baseline.summary !== 'pending') return baseline;
    for (let attempt = 0; attempt < ARTIFACT_POLL_ATTEMPTS; attempt += 1) {
      await sleep(ARTIFACT_POLL_INTERVAL_MS);
      try {
        const artifact = await apiClient.get<TaskArtifact>(`/api/v1/tasks/${taskId}/artifact/latest?mode=${mode}`);
        if (artifact.summary !== 'pending') {
          return artifact;
        }
      } catch {
        // The agent may not have published yet.
      }
    }
    throw new TaskArtifactStillPendingError();
  }, []);

  const runArtifactMutation = useCallback(async (taskId: string, endpoint: string, mode: 'latest' | 'final'): Promise<TaskArtifact> => {
    const existing = inFlightByTaskRef.current.get(taskId);
    if (existing) {
      return existing;
    }
    setGeneratingTaskIds((prev) => new Set(prev).add(taskId));
    const promise = apiClient.post<TaskArtifact>(endpoint).then((artifact) => waitForPublishedArtifact(taskId, mode, artifact));
    inFlightByTaskRef.current.set(taskId, promise);
    try {
      return await promise;
    } finally {
      inFlightByTaskRef.current.delete(taskId);
      setGeneratingTaskIds((prev) => {
        const next = new Set(prev);
        next.delete(taskId);
        return next;
      });
    }
  }, [waitForPublishedArtifact]);

  const generateArtifact = useCallback(
    (taskId: string): Promise<TaskArtifact> => runArtifactMutation(taskId, `/api/v1/tasks/${taskId}/artifact`, 'latest'),
    [runArtifactMutation],
  );

  const regenerateArtifact = useCallback(
    (taskId: string): Promise<TaskArtifact> => runArtifactMutation(taskId, `/api/v1/tasks/${taskId}/artifact?force=1`, 'latest'),
    [runArtifactMutation],
  );

  const finalizeArtifact = useCallback(
    (taskId: string): Promise<TaskArtifact> => runArtifactMutation(taskId, `/api/v1/tasks/${taskId}/artifact/finalize`, 'final'),
    [runArtifactMutation],
  );

  const fetchArtifactHTML = useCallback((artifact: TaskArtifact): Promise<string> => {
    if (!artifact.updated_at) return apiClient.getText(artifact.url);
    const separator = artifact.url.includes('?') ? '&' : '?';
    return apiClient.getText(`${artifact.url}${separator}v=${encodeURIComponent(artifact.updated_at)}`);
  }, []);

  const listArtifacts = useCallback((taskId: string): Promise<TaskArtifact[]> => {
    return apiClient.get<TaskArtifact[] | null>(`/api/v1/tasks/${taskId}/artifacts`).then((artifacts) => artifacts ?? []);
  }, []);

  const isGeneratingTask = useCallback((taskId: string) => generatingTaskIds.has(taskId), [generatingTaskIds]);

  return {
    generateArtifact,
    regenerateArtifact,
    finalizeArtifact,
    fetchArtifactHTML,
    listArtifacts,
    isGeneratingTask,
  };
}
