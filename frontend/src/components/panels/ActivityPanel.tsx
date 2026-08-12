import { useState, useEffect } from 'react';
import { Events } from '@wailsio/runtime';
import { X, Terminal, Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

type Tab = 'console' | 'downloads';

interface ActivityPanelProps {
  isOpen: boolean;
  onClose: () => void;
  activityCount?: number;
}

export function ActivityPanel({ isOpen, onClose, activityCount = 0 }: ActivityPanelProps) {
  const [activeTab, setActiveTab] = useState<Tab>('downloads');
  const [consoleLines, setConsoleLines] = useState<string[]>([]);
  const [downloadStatus, setDownloadStatus] = useState<string>('');

  useEffect(() => {
    if (!isOpen) return;

    const unsubs = [
      Events.On('log-line', (data: any) => {
        setConsoleLines((prev) => [...prev.slice(-200), data?.message ?? '']);
      }),
      Events.On('download-progress', (data: any) => {
        setDownloadStatus(data?.status ?? '');
      }),
    ];

    return () => {
      for (const unsub of unsubs) unsub();
    };
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed bottom-4 right-4 z-50 flex w-[560px] max-w-[90vw] flex-col rounded-lg border border-border bg-background shadow-lg"
      style={{ minHeight: 240, maxHeight: '70vh', resize: 'both', overflow: 'hidden' }}
      data-slot="activity-panel"
    >
      {/* Tab bar */}
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
         <div className="flex items-center gap-1" role="tablist" aria-label="Activity views">
             <button
             type="button"
             role="tab"
             aria-selected={activeTab === 'console'}
            className={`flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
              activeTab === 'console'
                ? 'bg-accent text-foreground'
                : 'text-muted-foreground hover:text-foreground'
            }`}
            onClick={() => setActiveTab('console')}
          >
            <Terminal className="size-3.5" />
            Console
          </button>
           <button
             type="button"
             role="tab"
             aria-selected={activeTab === 'downloads'}
            className={`flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${
              activeTab === 'downloads'
                ? 'bg-accent text-foreground'
                : 'text-muted-foreground hover:text-foreground'
            }`}
            onClick={() => setActiveTab('downloads')}
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
      <div className="flex-1 overflow-auto p-3 font-mono text-xs text-muted-foreground">
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
            {downloadStatus ? (
              <p>{downloadStatus}</p>
            ) : (
              <p className="text-muted-foreground/50">No active downloads.</p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
