'use client';

import { ObservabilityFrame } from '@/components/dashboard/observability-frame';
import { InsightDashboard } from '@/components/dashboard/insight-dashboard';

export default function ObservabilityInsightPage() {
  return (
    <ObservabilityFrame>
      <InsightDashboard />
    </ObservabilityFrame>
  );
}
