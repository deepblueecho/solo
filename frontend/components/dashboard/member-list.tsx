// ============================================================================
// MemberList — channel member panel showing users and agents separately
// ============================================================================

'use client';

import { useState } from 'react';
import { Users, Bot, Circle, Plus, User as UserIcon, X } from 'lucide-react';
import { PixelAvatar } from '@/components/ui/pixel-avatar';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogCloseButton,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { t } from '@/lib/i18n';
import type { ChannelMember } from '@/lib/types';

interface MemberListProps {
  users: ChannelMember[];
  agents: ChannelMember[];
  isLoading: boolean;
  onAddAgent: () => void;
  onRemoveAgent?: (memberId: string) => void;
  showHeader?: boolean;
  canAddAgent?: boolean;
}

function MemberItem({ member, onRemove }: { member: ChannelMember; onRemove?: (id: string) => void }) {
  const isAgent = member.member_type === 'agent';
  const [confirming, setConfirming] = useState(false);
  const statusColor = {
    online: 'fill-brutal-success text-brutal-success',
    offline: 'fill-brutal-muted text-brutal-muted',
    thinking: 'fill-brutal-accent text-brutal-accent',
    typing: 'fill-brutal-info text-brutal-info',
  }[member.status] || 'fill-brutal-muted text-brutal-muted';

  const statusLabel = {
    online: t('online'),
    offline: t('offline'),
    thinking: t('thinking'),
    typing: t('typing'),
  }[member.status] || t('offline');

  const handleRemove = () => {
    onRemove?.(member.member_id);
    setConfirming(false);
  };

  return (
    <div className="group flex items-center gap-3 border-2 border-transparent bg-white p-2 transition-all hover:border-black hover:bg-brutal-primary-light hover:shadow-brutal-sm">
      {/* Icon / Avatar */}
      {isAgent ? (
        <PixelAvatar agentId={member.member_id} size="md" />
      ) : (
        <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center border-2 border-black bg-brutal-muted shadow-brutal-sm">
          <span className="font-heading text-[10px] font-bold text-black">
            {member.display_name?.charAt(0)?.toUpperCase() || '?'}
          </span>
        </div>
      )}

      {/* Info */}
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate font-heading text-sm font-bold text-foreground">
            {member.display_name}
          </span>
          <span className="flex-shrink-0 border-2 border-black bg-brutal-primary px-1.5 py-0.5 font-heading text-[10px] font-bold uppercase tracking-wider text-black">
            {isAgent ? t('agent') : t('user')}
          </span>
        </div>
        <div className="mt-0.5 flex items-center gap-1 font-mono text-[11px] text-muted-foreground">
          <Circle className={`h-2 w-2 flex-shrink-0 ${statusColor}`} />
          {statusLabel}
        </div>
      </div>

      {/* Remove */}
      <div className="flex flex-shrink-0 items-center">
        {isAgent && (
          onRemove && (
            <button
              onClick={(e) => { e.stopPropagation(); setConfirming(true); }}
              className="flex-shrink-0 border-2 border-black bg-white px-1.5 py-0.5 opacity-0 group-hover:opacity-100 transition-all shadow-brutal-sm hover:bg-brutal-danger hover:text-black"
              title={t('removeAgent')}
            >
              <X className="h-3 w-3" />
            </button>
          )
        )}
      </div>

      <Dialog open={confirming} onOpenChange={setConfirming}>
        <DialogHeader>
          <DialogTitle>{t('removeAgent')}</DialogTitle>
          <DialogCloseButton onClick={() => setConfirming(false)} />
        </DialogHeader>
        <DialogDescription>
          Remove {member.display_name} from this channel?
        </DialogDescription>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => setConfirming(false)}
          >
            {t('cancel')}
          </Button>
          <Button
            type="button"
            variant="danger"
            onClick={handleRemove}
          >
            {t('confirm')}
          </Button>
        </DialogFooter>
      </Dialog>
    </div>
  );
}

function MemberListSkeleton() {
  return (
    <div className="space-y-2 px-2">
      {[1, 2, 3, 4].map((i) => (
        <div key={i} className="flex items-center gap-2">
          <Skeleton className="h-7 w-7 rounded-none" />
          <Skeleton className={`h-4 ${i % 2 === 0 ? 'w-20' : 'w-16'}`} />
        </div>
      ))}
    </div>
  );
}

export function MemberList({ users, agents, isLoading, onAddAgent, onRemoveAgent, showHeader = true, canAddAgent = true }: MemberListProps) {
  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Section header */}
      {showHeader && (
      <div className="flex items-center justify-between border-b-2 border-black px-4 py-3">
        <div className="flex items-center gap-2 text-sm font-medium text-foreground">
          <Users className="h-4 w-4" />
          <span>{t('members')}</span>
          <span className="text-xs text-muted-foreground">
            {users.length + agents.length}
          </span>
        </div>
        {canAddAgent && (
          <Button
            type="button"
            onClick={onAddAgent}
            variant="success"
            size="icon"
            className="h-7 w-7"
            aria-label={t('addAgentToChannel')}
          >
            <Plus className="h-3.5 w-3.5" />
          </Button>
        )}
      </div>
      )}

      {/* Member list */}
      <div className="flex-1 overflow-y-auto py-2">
        {isLoading ? (
          <MemberListSkeleton />
        ) : (
          <div className="space-y-3">
            {/* Users section */}
            {users.length > 0 && (
              <div>
                <div className="mb-1 flex items-center gap-1.5 px-2">
                  <UserIcon className="h-3 w-3 text-muted-foreground" />
                  <span className="text-[11px] font-bold uppercase tracking-wider text-muted-foreground">
                    {t('membersUsers', { n: users.length })}
                  </span>
                </div>
                {users.map((member) => (
                  <MemberItem key={`user-${member.member_id}`} member={member} />
                ))}
              </div>
            )}

            {/* Agents section */}
            {agents.length > 0 && (
              <div>
                <div className="mb-1 flex items-center gap-1.5 px-2">
                  <Bot className="h-3 w-3 text-muted-foreground" />
                  <span className="text-[11px] font-bold uppercase tracking-wider text-muted-foreground">
                    {t('agent')} ({agents.length})
                  </span>
                </div>
                {agents.map((member) => (
                  <MemberItem key={`agent-${member.member_id}`} member={member} onRemove={onRemoveAgent} />
                ))}
              </div>
            )}

            {/* Empty state */}
            {users.length === 0 && agents.length === 0 && (
              <div className="px-4 py-6 text-center text-sm text-muted-foreground">
                {t('noMembersYet')}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
