// ============================================================================
// TeamsAgentWorkspace — Workspace tab content for an agent on /teams.
// Top bar: agent's workspace path + refresh button.
// Body:   file tree (left, 260px) + file preview with Shiki (right, flex).
// Read-only — no edit, no upload, no delete.
// ============================================================================

'use client';

import { useState, useEffect, useCallback } from 'react';
import { RefreshCw, AlertCircle, FolderOpen, FileText, PanelLeftClose, PanelLeftOpen, Maximize2, Minimize2 } from 'lucide-react';
import { useWorkspaceFiles } from '@/lib/hooks/use-workspace-files';
import { FileTree } from '@/components/workspace/file-tree';
import { FilePreview } from '@/components/workspace/file-preview';
import { Skeleton } from '@/components/ui/skeleton';
import { t } from '@/lib/i18n';
import type { WorkspaceFileNode } from '@/lib/types';

interface TeamsAgentWorkspaceProps {
  agentId: string;
}

function firstFilePath(nodes: WorkspaceFileNode[]): string | null {
  for (const node of nodes) {
    if (node.type === 'file') return node.path;
    const child = firstFilePath(node.children ?? []);
    if (child) return child;
  }
  return null;
}

export function TeamsAgentWorkspace({ agentId }: TeamsAgentWorkspaceProps) {
  const { tree, isLoading, error, loadTree, fetchFileContent } = useWorkspaceFiles(agentId);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState<string | null>(null);
  const [isContentLoading, setIsContentLoading] = useState(false);
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());
  const [filePaneWidth, setFilePaneWidth] = useState(160);
  const [isFilePaneCollapsed, setIsFilePaneCollapsed] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);

  // Initial load: trigger the hook to fetch the root directory
  useEffect(() => {
    void loadTree();
  }, [loadTree, agentId]);

  // Persist expanded paths per agent
  useEffect(() => {
    try {
      const saved = localStorage.getItem(`ws-expand-${agentId}`);
      if (saved) setExpandedPaths(new Set(JSON.parse(saved)));
    } catch { /* ignore */ }
  }, [agentId]);

  useEffect(() => {
    try {
      localStorage.setItem(`ws-expand-${agentId}`, JSON.stringify([...expandedPaths]));
    } catch { /* ignore */ }
  }, [expandedPaths, agentId]);

  const handleSelect = useCallback(
    async (filePath: string, _type?: 'file' | 'directory') => {
      setSelectedPath(filePath);
      setIsContentLoading(true);
      try {
        const content = await fetchFileContent(filePath);
        setFileContent(content);
      } catch {
        setFileContent(null);
      } finally {
        setIsContentLoading(false);
      }
    },
    [fetchFileContent],
  );

  useEffect(() => {
    if (selectedPath || tree.length === 0) return;
    const path = firstFilePath(tree);
    if (path) void handleSelect(path, 'file');
  }, [handleSelect, selectedPath, tree]);

  const handleToggleExpand = useCallback((path: string) => {
    setExpandedPaths((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  }, []);

  const handleLoadDirectory = useCallback(
    (dirPath: string) => {
      loadTree(dirPath);
    },
    [loadTree],
  );

  // ---- Loading (initial) ----
  if (isLoading && tree.length === 0) {
    return (
      <div className="space-y-2 p-4">
        <Skeleton className="h-6 w-3/4 rounded-none" />
        <Skeleton className="h-6 w-1/2 rounded-none" />
        <Skeleton className="h-6 w-2/3 rounded-none" />
      </div>
    );
  }

  // ---- Error ----
  if (error && tree.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <div className="mb-3 flex h-12 w-12 items-center justify-center border-2 border-black bg-brutal-danger-light shadow-brutal-sm">
          <AlertCircle className="h-6 w-6 text-brutal-danger" />
        </div>
        <p className="font-body text-sm text-brutal-danger">{error}</p>
        <button
          type="button"
          onClick={() => loadTree()}
          className="btn-brutal btn-brutal-sm mt-4"
        >
          <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
          Retry
        </button>
      </div>
    );
  }

  // ---- Empty ----
  if (!isLoading && tree.length === 0) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <FolderOpen className="mx-auto h-10 w-10 text-muted-foreground" />
          <p className="mt-2 font-heading text-sm font-bold">Agent workspace has no files yet</p>
          <p className="mt-1 font-mono text-xs text-muted-foreground">
            Files will appear here after running agent tasks
          </p>
        </div>
      </div>
    );
  }

  // ---- Normal: path bar + tree + preview ----
  return (
    <div className={isFullscreen
      ? 'fixed inset-0 z-[80] flex h-screen flex-col border-4 border-black bg-white'
      : 'flex h-full flex-col'
    }>
      {/* Path bar */}
      <div className="flex items-center justify-between border-b-2 border-black bg-white px-3 py-1.5">
        <div className="flex min-w-0 items-center gap-2 font-mono text-[11px] text-muted-foreground">
          <FolderOpen className="h-3.5 w-3.5 flex-shrink-0" />
          <span className="truncate">
            agents/<span className="font-bold text-foreground">{agentId.slice(0, 8)}</span>/workspace
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <button
            type="button"
            onClick={() => setIsFullscreen((fullscreen) => !fullscreen)}
            className="btn-brutal btn-brutal-xs"
            aria-label={isFullscreen ? 'Exit fullscreen workspace' : 'Fullscreen workspace'}
          >
            {isFullscreen ? <Minimize2 className="h-3 w-3" /> : <Maximize2 className="h-3 w-3" />}
          </button>
          <button
            type="button"
            onClick={() => loadTree()}
            className="btn-brutal btn-brutal-xs"
            aria-label={t('workspaceRefreshTree')}
          >
            <RefreshCw className="h-3 w-3" />
          </button>
        </div>
      </div>

      {/* Tree + preview split */}
      <div className="flex flex-1 overflow-hidden">
        <div
          className="relative h-full flex-shrink-0 overflow-hidden border-r-2 border-black bg-white"
          style={{ width: isFilePaneCollapsed ? 34 : filePaneWidth }}
        >
          <div className="flex items-center border-b-2 border-black px-3 py-2">
            {!isFilePaneCollapsed && (
              <span className="font-heading text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                Files
              </span>
            )}
            <button
              type="button"
              onClick={() => setIsFilePaneCollapsed((collapsed) => !collapsed)}
              className="ml-auto flex h-5 w-5 items-center justify-center border-2 border-black bg-white hover:bg-brutal-primary-light"
              aria-label={isFilePaneCollapsed ? 'Expand files' : 'Collapse files'}
            >
              {isFilePaneCollapsed ? <PanelLeftOpen className="h-3 w-3" /> : <PanelLeftClose className="h-3 w-3" />}
            </button>
          </div>
          {!isFilePaneCollapsed && (
            <div className="h-[calc(100%-2.25rem)] overflow-y-auto">
              <FileTree
                tree={tree}
                selectedPath={selectedPath}
                expandedPaths={expandedPaths}
                onSelect={handleSelect}
                onToggleExpand={handleToggleExpand}
                onLoadDirectory={handleLoadDirectory}
              />
            </div>
          )}
          {!isFilePaneCollapsed && (
            <div
              className="absolute right-0 top-0 bottom-0 w-1.5 cursor-col-resize hover:bg-brutal-primary/50"
              onMouseDown={(e) => {
                e.preventDefault();
                const startX = e.clientX;
                const startWidth = filePaneWidth;
                const onMove = (ev: MouseEvent) => {
                  setFilePaneWidth(Math.max(120, Math.min(240, startWidth + ev.clientX - startX)));
                };
                const onUp = () => {
                  document.removeEventListener('mousemove', onMove);
                  document.removeEventListener('mouseup', onUp);
                };
                document.addEventListener('mousemove', onMove);
                document.addEventListener('mouseup', onUp);
              }}
            />
          )}
        </div>
        <div className="h-full flex-1 overflow-y-auto bg-brutal-cream">
          {selectedPath ? (
            <FilePreview
              path={selectedPath}
              content={fileContent}
              isLoading={isContentLoading}
            />
          ) : (
            <div className="flex h-full items-center justify-center">
              <div className="text-center">
                <FileText className="mx-auto h-6 w-6 text-muted-foreground" />
                <p className="mt-2 font-mono text-xs text-muted-foreground">
                  Select a file to preview its content
                </p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
