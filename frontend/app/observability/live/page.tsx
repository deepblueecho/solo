'use client';

import { Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import { ObservabilityFrame } from '@/components/dashboard/observability-frame';
import { LiveMonitor } from '@/components/dashboard/live-monitor';
import { Spinner } from '@/components/ui/spinner';

export default function ObservabilityLivePage() {
  return (
    <Suspense fallback={<ObservabilityFallback />}>
      <ObservabilityLiveContent />
    </Suspense>
  );
}

function ObservabilityLiveContent() {
  const searchParams = useSearchParams();
  return (
    <ObservabilityFrame>
      <LiveMonitor selectedRunId={searchParams.get('run_id')} />
    </ObservabilityFrame>
  );
}

function ObservabilityFallback() {
  return (
    <div className="flex h-screen items-center justify-center bg-brutal-cream">
      <Spinner size="md" />
    </div>
  );
}
