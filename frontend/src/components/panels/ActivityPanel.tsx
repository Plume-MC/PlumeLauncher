import { useState, useEffect, type KeyboardEvent } from 'react';
import { Events } from '@wailsio/runtime';
import { X, Terminal, Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { HomeService } from '../../../bindings/plumelauncher/internal/services/index.js';

type Tab = 'console' | 'downloads';
type DownloadProgress = { instanceId: string; status: string; fileProgress: number; totalFiles: number; byteProgress: number; totalBytes: number; speed: number; eta: number; error?: string };

interface ActivityPanelProps {
  isOpen: boolean;
  onClose: () => void;
  activityCount?: number;
}

export function ActivityPanel({ isOpen, onClose, activityCount = 0 }: ActivityPanelProps) {
  const [activeTab, setActiveTab] = useState<Tab>('downloads');
  const [consoleLines, setConsoleLines] = useState<string[]>([]);
  const [download, setDownload] = useState<DownloadProgress | null>(null);
  const [cancelling, setCancelling] = useState(false);

  useEffect(() => {
    if (!isOpen) return;

    const closeOnEscape = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', closeOnEscape);

    const unsubs = [
       Events.On('log-line', (event: any) => {
          const data = event.data;
          setConsoleLines((prev) => [...prev.slice(-199), `${data?.level === 'error' ? '[ERR] ' : ''}${data?.message ?? ''}`]);
        }),
       Events.On('download-progress', (event: any) => {
          setDownload(event.data as DownloadProgress);
      }),
    ];

    return () => {
      window.removeEventListener('keydown', closeOnEscape);
      for (const unsub of unsubs) unsub();
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const selectTab = (tab: Tab) => setActiveTab(tab);
  const handleTabKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const tabs: Tab[] = ['console', 'downloads'];
    const index = tabs.indexOf(activeTab);
    const next = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : event.key === 'ArrowRight' ? (index + 1) % tabs.length : event.key === 'ArrowLeft' ? (index - 1 + tabs.length) % tabs.length : index;
    if (next === index && !['Home', 'End'].includes(event.key)) return;
    event.preventDefault();
    selectTab(tabs[next]);
    event.currentTarget.querySelector<HTMLButtonElement>(`[data-tab="${tabs[next]}"]`)?.focus();
  };

  const cancelDownload = async () => {
    if (!download?.instanceId) return;
    setCancelling(true);
    try { await HomeService.CancelInstance(download.instanceId); } finally { setCancelling(false); }
  };

  return (
    <div
      className="fixed bottom-4 right-4 z-50 flex w-[560px] max-w-[90vw] flex-col rounded-lg border border-border bg-background shadow-lg"
      style={{ minHeight: 240, maxHeight: '70vh', resize: 'both', overflow: 'hidden' }}
      data-slot="activity-panel"
    >
      {/* Tab bar */}
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
          <div className="flex items-center gap-1" role="tablist" aria-label="Activity views" onKeyDown={handleTabKeyDown}>
              <button
              type="button"
              role="tab"
              aria-selected={activeTab === 'console'}
              aria-controls="activity-console"
              id="activity-tab-console"
              tabIndex={activeTab === 'console' ? 0 : -1}
              data-tab="console"
            className={`flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
              activeTab === 'console'
                ? 'bg-accent text-foreground'
                : 'text-muted-foreground hover:text-foreground'
            }`}
             onClick={() => selectTab('console')}
          >
            <Terminal className="size-3.5" />
            Console
          </button>
           <button
             type="button"
              role="tab"
              aria-selected={activeTab === 'downloads'}
              aria-controls="activity-downloads"
              id="activity-tab-downloads"
              tabIndex={activeTab === 'downloads' ? 0 : -1}
              data-tab="downloads"
            className={`flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
              activeTab === 'downloads'
                ? 'bg-accent text-foreground'
                : 'text-muted-foreground hover:text-foreground'
            }`}
             onClick={() => selectTab('downloads')}
          >
            <Download className="size-3.5" />
            Downloads
             {activityCount > 0 && <Badge variant="outline" className="ml-1 h-4 px-1 text-[9px]">{activityCount}</Badge>}
          </button>
        </div>
        <Button
          variant="ghost"
          size="icon-xs"
          onClick={onClose}
          aria-label="Close activity panel"
        >
          <X className="size-3.5" />
        </Button>
      </div>

      {/* Content */}
      <div id={`activity-${activeTab}`} role="tabpanel" aria-labelledby={`activity-tab-${activeTab}`} className="flex-1 overflow-auto p-3 font-mono text-xs text-muted-foreground">
        {activeTab === 'console' && (
          <div className="space-y-1">
            {consoleLines.length === 0 ? (
              <p className="text-muted-foreground/50">No active console output.</p>
            ) : (
              consoleLines.map((line, i) => <p key={i}>{line}</p>)
            )}
          </div>
        )}
        {activeTab === 'downloads' && (
          <div className="space-y-1">
             {download ? (
              <div className="space-y-2">
                <div className="flex items-center justify-between"><p>{download.status} · {download.fileProgress}/{download.totalFiles} files</p>{download.status === 'downloading' && <Button variant="ghost" size="sm" disabled={cancelling} onClick={() => void cancelDownload()}>{cancelling ? 'Cancelling...' : 'Cancel'}</Button>}</div>
                <progress className="h-1.5 w-full accent-primary" value={download.totalBytes ? download.byteProgress : download.fileProgress} max={download.totalBytes || download.totalFiles} aria-label="Download progress" />
                <p className="text-muted-foreground/70">{download.totalBytes ? `${Math.round(download.byteProgress / 1024 / 1024)} / ${Math.round(download.totalBytes / 1024 / 1024)} MB` : 'Preparing files'}{download.speed > 0 && ` · ${Math.round(download.speed / 1024)} KB/s`}</p>
                {download.error && <p className="text-destructive">{download.error}</p>}
              </div>
             ) : (
              <p className="text-muted-foreground/50">No active downloads.</p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
