// ============================================================================
// TeamsAgentProfile — Profile tab content for an agent on /teams.
// Stacks three existing sub-components vertically:
//   - AgentProfileTab  (display name, description, info, status)
//   - AgentRuntimeTab  (model, reasoning, env vars)
//   - AgentSkillsTab   (tools/skills toggle list)
// Each sub-component fetches its own copy of the agent; we accept the
// duplication in exchange for not having to refactor the shared panel.
// v3.3: color lives in the sub-components (status pill, field tags,
// avatar ornament) — no outer tag/header wrapper here.
// ============================================================================

'use client';

import { useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { Trash2 } from 'lucide-react';
import { AgentProfileTab } from '@/components/agents/agent-profile-tab';
import { AgentRuntimeTab } from '@/components/agents/agent-runtime-tab';
import { AgentSkillsTab } from '@/components/agents/agent-skills-tab';
import { BrutalSeparator } from '@/components/ui/brutal-separator';
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogCloseButton,
} from '@/components/ui/dialog';
import { useAgents } from '@/lib/hooks/use-agents';
import { useToast } from '@/components/ui/toast';
import { t } from '@/lib/i18n';

interface TeamsAgentProfileProps {
  agentId: string;
}

export function TeamsAgentProfile({ agentId }: TeamsAgentProfileProps) {
  const router = useRouter();
  const { agents, deleteAgent } = useAgents();
  const { showToast } = useToast();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const agentName = agents.find((a) => a.id === agentId)?.name ?? 'this agent';

  const handleConfirmDelete = useCallback(async () => {
    setDeleting(true);
    try {
      await deleteAgent(agentId);
      showToast(t('agentDeleteSuccess'), 'success');
      const remaining = agents.filter((a) => a.id !== agentId);
      if (remaining.length > 0) {
        router.replace(`/teams?agent=${remaining[0].id}&tab=profile`, { scroll: false });
      } else {
        router.replace('/teams', { scroll: false });
      }
    } catch {
      showToast(t('agentDeleteError'), 'error');
    } finally {
      setDeleting(false);
      setConfirmOpen(false);
    }
  }, [agentId, agents, deleteAgent, router, showToast]);

  return (
    <div className="space-y-6">
      <AgentProfileTab agentId={agentId} />
      <BrutalSeparator />
      <AgentRuntimeTab agentId={agentId} />
      <BrutalSeparator />
      <AgentSkillsTab agentId={agentId} />

      {/* Danger zone — delete agent (soft delete: retains DM history) */}
      <BrutalSeparator />
      <div className="flex justify-end pt-2">
        <button
          type="button"
          onClick={() => setConfirmOpen(true)}
          className="btn-brutal btn-brutal-sm bg-brutal-danger text-white hover:bg-brutal-danger"
        >
          <Trash2 className="mr-1.5 h-3.5 w-3.5" />
          {t('agentDeleteButton')}
        </button>
      </div>

      <Dialog open={confirmOpen} onOpenChange={(open) => !deleting && setConfirmOpen(open)}>
        <DialogHeader>
          <DialogTitle>{t('agentDeleteTitle')}</DialogTitle>
          <DialogCloseButton onClick={() => setConfirmOpen(false)} />
        </DialogHeader>
        <DialogDescription>
          {t('agentDeleteDesc', { name: agentName })}
        </DialogDescription>
        <DialogFooter>
          <button
            type="button"
            onClick={() => setConfirmOpen(false)}
            disabled={deleting}
            className="btn-brutal btn-brutal-sm"
          >
            {t('cancel')}
          </button>
          <button
            type="button"
            onClick={handleConfirmDelete}
            disabled={deleting}
            className="btn-brutal btn-brutal-sm bg-brutal-danger text-white"
          >
            {deleting ? t('deleting') : t('delete')}
          </button>
        </DialogFooter>
      </Dialog>
    </div>
  );
}

function Section({
  header,
  headerColor,
  children,
}: {
  header: string;
  headerColor: string;
  children: React.ReactNode;
}) {
  return (
    <section>
      <span
        className={`inline-block ${headerColor} border-2 border-black px-2 py-0.5 font-heading text-[10px] font-black uppercase tracking-widest text-black shadow-brutal-sm`}
        style={{ transform: 'rotate(-0.6deg)' }}
      >
        ★ {header}
      </span>
      <div className="mt-3">{children}</div>
    </section>
  );
}
