'use client';

import { useCallback, useRef, useState } from 'react';
import { apiClient } from '@/lib/api-client';
import type { TaskArtifact } from '@/lib/types';

export function useTaskArtifact() {
  const [isGenerating, setIsGenerating] = useState(false);
  const isGeneratingRef = useRef(false);
  const inFlightPromiseRef = useRef<Promise<TaskArtifact> | null>(null);

  const generateArtifact = useCallback(async (taskId: string): Promise<TaskArtifact> => {
    if (isGeneratingRef.current && inFlightPromiseRef.current) {
      return inFlightPromiseRef.current;
    }
    isGeneratingRef.current = true;
    setIsGenerating(true);
    const promise = apiClient.post<TaskArtifact>(`/api/v1/tasks/${taskId}/artifact`);
    inFlightPromiseRef.current = promise;
    try {
      return await promise;
    } finally {
      isGeneratingRef.current = false;
      inFlightPromiseRef.current = null;
      setIsGenerating(false);
    }
  }, []);

  return { generateArtifact, isGenerating };
}
