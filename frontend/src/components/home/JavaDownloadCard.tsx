import { useEffect, useState } from 'react';
import { IconDownload, IconLoader2, IconTrash, IconX } from '@tabler/icons-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { SystemService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { JavaDownloadProgressEvent } from '../../../bindings/plumelauncher/internal/services/models.js';

interface JavaDownloadCardProps {
  major: number;
  recommended?: boolean;
  installed?: boolean;
  onDownloadComplete?: (major?: number) => void;
}

const VERSION_LABELS: Record<number, string> = {
  8: 'Java 8 (Legacy)',
  17: 'Java 17 (LTS)',
  21: 'Java 21 (LTS)',
  25: 'Java 25 (Latest)',
};

const LTS_MAJORS = new Set([8, 17, 21]);

export function JavaDownloadCard({ major, recommended, installed, onDownloadComplete }: JavaDownloadCardProps) {
  const [status, setStatus] = useState<'idle' | 'downloading' | 'extracting' | 'completed' | 'failed' | 'cancelled'>(
    installed ? 'completed' : 'idle'
  );
  const [progress, setProgress] = useState({ bytesRead: 0, totalBytes: 0 });
  const [error, setError] = useState('');

  useEffect(() => {
    if (installed) {
      setStatus('completed');
    }
  }, [installed]);

  useEffect(() => {
    let unsub: (() => void) | undefined;

    const setup = async () => {
      const { Events } = await import('@wailsio/runtime');
      unsub = Events.On('java-download-progress', (event: { data: JavaDownloadProgressEvent }) => {
        const data = event.data;
        if (data.major !== major) return;

        if (data.status === 'downloading') {
          setStatus('downloading');
          setProgress({ bytesRead: data.bytesRead, totalBytes: data.totalBytes });
        } else if (data.status === 'extracting') {
          setStatus('extracting');
        } else if (data.status === 'completed') {
          setStatus('completed');
          setProgress({ bytesRead: data.totalBytes, totalBytes: data.totalBytes });
          onDownloadComplete?.(major);
        } else if (data.status === 'failed') {
          setStatus('failed');
          setError(data.error || 'Download failed');
        } else if (data.status === 'cancelled') {
          setStatus('cancelled');
        }
      });
    };

    void setup();
    return () => { unsub?.(); };
  }, [major, onDownloadComplete]);

  const handleDownload = async () => {
    setError('');
    setStatus('downloading');
    try {
      await SystemService.DownloadJava(major);
    } catch (err) {
      setStatus('failed');
      setError(err instanceof Error ? err.message : 'Download failed');
    }
  };

  const handleCancel = async () => {
    try {
      await SystemService.CancelJavaDownload(major);
      setStatus('cancelled');
    } catch {
      // ignore
    }
  };

  const handleDelete = async () => {
    try {
      await SystemService.DeleteManagedRuntime(major);
      setStatus('idle');
      setProgress({ bytesRead: 0, totalBytes: 0 });
      onDownloadComplete?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed');
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const mb = bytes / (1024 * 1024);
    return `${Math.round(mb)} MB`;
  };

  const pct = progress.totalBytes > 0 ? Math.round((progress.bytesRead / progress.totalBytes) * 100) : 0;

  return (
    <div className={cn(
      'flex flex-col gap-2 rounded-lg border p-3 transition-colors',
      status === 'completed' ? 'border-success/30 bg-success/5' : 'border-border bg-card/30'
    )}>
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="text-xs font-semibold text-foreground">
              Java {major}
            </span>
            {LTS_MAJORS.has(major) && (
              <Badge variant="outline" className="text-[9px]">LTS</Badge>
            )}
            {recommended && (
              <Badge className="text-[9px] bg-primary/20 text-primary border-primary/30">Recommended</Badge>
            )}
          </div>
          <p className="mt-0.5 text-[10px] text-muted-foreground">
            {VERSION_LABELS[major] || `Java ${major}`}
          </p>
        </div>

        {status === 'idle' && (
          <Button variant="outline" size="sm" onClick={() => void handleDownload()} className="shrink-0 gap-1 h-7 text-[11px]">
            <IconDownload className="size-3" />
            Download
          </Button>
        )}

        {status === 'downloading' && (
          <Button variant="outline" size="sm" onClick={() => void handleCancel()} className="shrink-0 gap-1 h-7 text-[11px] text-destructive border-destructive/30">
            <IconX className="size-3" />
            Cancel
          </Button>
        )}

        {status === 'completed' && installed && (
          <Button variant="ghost" size="sm" onClick={() => void handleDelete()} className="shrink-0 gap-1 h-7 text-[11px] text-destructive">
            <IconTrash className="size-3" />
          </Button>
        )}
      </div>

      {status === 'downloading' && progress.totalBytes > 0 && (
        <div className="space-y-1">
          <div className="h-1.5 overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-cyan transition-[width] duration-200"
              style={{ width: `${pct}%` }}
            />
          </div>
          <p className="text-[10px] font-mono text-muted-foreground">
            {formatBytes(progress.bytesRead)} / {formatBytes(progress.totalBytes)} ({pct}%)
          </p>
        </div>
      )}

      {status === 'extracting' && (
        <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
          <IconLoader2 className="size-3 animate-spin" />
          Extracting...
        </div>
      )}

      {status === 'completed' && !installed && (
        <div className="flex items-center gap-1.5 text-[10px] text-success">
          <IconDownload className="size-3" />
          Installed
        </div>
      )}

      {error && (
        <p role="alert" className="text-[10px] text-destructive">{error}</p>
      )}
    </div>
  );
}
