'use client';

import { useCallback, useRef, useState } from 'react';
import { apiClient } from '@/lib/api-client';
import type { TaskArtifact } from '@/lib/types';

export class TaskArtifactGenerationInProgressError extends Error {
  constructor() {
    super('Task artifact generation is already in progress for a different task');
    this.name = 'TaskArtifactGenerationInProgressError';
  }
}

export function useTaskArtifact() {
  const [isGenerating, setIsGenerating] = useState(false);
  const isGeneratingRef = useRef(false);
  const inFlightPromiseRef = useRef<Promise<TaskArtifact> | null>(null);
  const inFlightTaskIdRef = useRef<string | null>(null);

  const generateArtifact = useCallback(async (taskId: string): Promise<TaskArtifact> => {
    if (isGeneratingRef.current && inFlightPromiseRef.current) {
      if (inFlightTaskIdRef.current !== taskId) {
        throw new TaskArtifactGenerationInProgressError();
      }
      return inFlightPromiseRef.current;
    }
    isGeneratingRef.current = true;
    inFlightTaskIdRef.current = taskId;
    setIsGenerating(true);
    const promise = apiClient.post<TaskArtifact>(`/api/v1/tasks/${taskId}/artifact`);
    inFlightPromiseRef.current = promise;
    try {
      return await promise;
    } finally {
      isGeneratingRef.current = false;
      inFlightPromiseRef.current = null;
      inFlightTaskIdRef.current = null;
      setIsGenerating(false);
    }
  }, []);

  return { generateArtifact, isGenerating };
}
