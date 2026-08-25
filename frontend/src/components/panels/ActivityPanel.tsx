import { useDeferredValue, useEffect, useState } from 'react';
import { X, Terminal, Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { HomeService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { DownloadProgressEvent } from '../../../bindings/plumelauncher/internal/services/models.js';

type Tab = 'console' | 'downloads';
interface ActivityPanelProps {
  isOpen: boolean;
  onClose: () => void;
  activityCount?: number;
	consoleLines: string[];
	download: DownloadProgressEvent | null;
}

export function ActivityPanel({ isOpen, onClose, activityCount = 0, consoleLines, download }: ActivityPanelProps) {
  const [activeTab, setActiveTab] = useState<Tab>('downloads');
  const [cancelling, setCancelling] = useState(false);
  const deferredConsoleLines = useDeferredValue(consoleLines);

  useEffect(() => {
    if (!isOpen) return;

    const closeOnEscape = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', closeOnEscape);

    return () => {
      window.removeEventListener('keydown', closeOnEscape);
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const cancelDownload = async () => {
    if (!download?.instanceId) return;
    setCancelling(true);
    try { await HomeService.CancelInstance(download.instanceId); } finally { setCancelling(false); }
  };

  return (
    <div
      className="fixed bottom-4 right-4 z-50 flex w-[560px] max-w-[90vw] flex-col rounded-lg border border-border bg-background shadow-lg"
      style={{ minWidth: 320, minHeight: 240, maxHeight: '70vh', resize: 'both', overflow: 'hidden' }}
      data-slot="activity-panel"
    >
      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as Tab)} className="flex flex-1 flex-col">
        <div className="flex items-center justify-between border-b border-border px-3 py-2">
          <TabsList className="border-0" aria-label="Activity views">
            <TabsTrigger value="console"><Terminal />Console</TabsTrigger>
            <TabsTrigger value="downloads"><Download />Downloads{activityCount > 0 && <Badge variant="outline" className="ml-1 h-4 px-1 text-[9px]">{activityCount}</Badge>}</TabsTrigger>
          </TabsList>
        <Button
          variant="ghost"
          size="icon-xs"
          onClick={onClose}
          aria-label="Close activity panel"
        >
          <X className="size-3.5" />
        </Button>
        </div>

        <div className="flex-1 overflow-auto p-3 font-mono text-xs text-muted-foreground">
        <TabsContent value="console">
          <div className="space-y-1">
            {consoleLines.length === 0 ? (
              <p className="text-muted-foreground/50">No active console output.</p>
            ) : (
			  deferredConsoleLines.slice(-200).map((line, i) => <p key={`${consoleLines.length - deferredConsoleLines.length + i}:${line}`} className="[content-visibility:auto]">{line}</p>)
            )}
          </div>
        </TabsContent>
        <TabsContent value="downloads">
          <div className="space-y-1">
             {download ? (
			   <div className="space-y-2" role="status" aria-live="polite">
                <div className="flex items-center justify-between"><p>{download.status} · {download.fileProgress}/{download.totalFiles} files</p>{(download.status === 'downloading' || download.status === 'repairing') && <Button variant="ghost" size="sm" disabled={cancelling} onClick={() => void cancelDownload()}>{cancelling ? 'Cancelling...' : 'Cancel'}</Button>}</div>
                <progress className="h-1.5 w-full accent-primary" value={download.totalBytes ? download.byteProgress : download.fileProgress} max={download.totalBytes || download.totalFiles} aria-label="Download progress" />
                <p className="text-muted-foreground/70">{download.totalBytes ? `${Math.round(download.byteProgress / 1024 / 1024)} / ${Math.round(download.totalBytes / 1024 / 1024)} MB` : 'Preparing files'}{download.speed > 0 && ` · ${Math.round(download.speed / 1024)} KB/s`}</p>
				 {download.error && <p className="text-destructive" role="alert">{download.error}</p>}
              </div>
             ) : (
              <p className="text-muted-foreground/50">No active downloads.</p>
            )}
          </div>
        </TabsContent>
        </div>
      </Tabs>
    </div>
  );
}
