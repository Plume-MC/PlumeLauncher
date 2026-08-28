import { Play, Download, MoreVertical } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

import { InstanceState, LoaderType } from '../../../bindings/plumelauncher/internal/instances/models.js';


interface InstanceCardProps {
  name: string;
  mcVersion: string;
  loader?: LoaderType;
  state?: InstanceState;
  onPlay?: () => void;
  onDownload?: () => void;
  onOpenDetail?: () => void;
  onAction?: (action: 'play' | 'install' | 'stop' | 'cancel') => void;
  busy?: boolean;
  downloadProgress?: { fileProgress: number; totalFiles: number } | null;
  actionState?: 'preparing' | 'stopping' | null;
}

const stateLabel: Record<InstanceState, string> = {
  [InstanceState.$zero]: 'Unknown',
  [InstanceState.StateNotInstalled]: 'Not installed',
  [InstanceState.StatePlanning]: 'Planning',
  [InstanceState.StateDownloading]: 'Downloading',
  [InstanceState.StateVerifying]: 'Verifying',
  [InstanceState.StateReady]: 'Ready',
  [InstanceState.StateRunning]: 'Running',
  [InstanceState.StateStopped]: 'Stopped',
  [InstanceState.StateCrashed]: 'Crashed',
  [InstanceState.StateFailed]: 'Failed',
};

const stateVariant: Record<InstanceState, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  [InstanceState.$zero]: 'outline',
  [InstanceState.StateNotInstalled]: 'outline',
  [InstanceState.StatePlanning]: 'outline',
  [InstanceState.StateDownloading]: 'outline',
  [InstanceState.StateVerifying]: 'outline',
  [InstanceState.StateReady]: 'default',
  [InstanceState.StateRunning]: 'secondary',
  [InstanceState.StateStopped]: 'secondary',
  [InstanceState.StateCrashed]: 'destructive',
  [InstanceState.StateFailed]: 'destructive',
};

export function InstanceCard({
  name,
  mcVersion,
  loader = LoaderType.LoaderVanilla,
  state = InstanceState.StateNotInstalled,
  onPlay,
  onDownload,
  onOpenDetail,
  onAction,
  busy = false,
  downloadProgress = null,
  actionState = null,
}: InstanceCardProps) {
  const actionBusy = busy || actionState !== null;
  return (
    <Card
      className="hover:border-primary/40"
      data-slot="instance-card"
    >
      <CardHeader className="flex flex-row items-start justify-between gap-2 pb-2">
        <div className="min-w-0 flex-1">
          <CardTitle className="truncate text-sm">{name}</CardTitle>
          <p className="mt-0.5 font-mono text-xs text-muted-foreground">MC {mcVersion}</p>
        </div>
         <Badge variant={stateVariant[state]} className="shrink-0 text-[10px]">
           {stateLabel[state]}
         </Badge>
         <Button size="icon-xs" variant="ghost" onClick={(event) => { event.stopPropagation(); onOpenDetail?.(); }} aria-label={`Open details for ${name}`}>
           <MoreVertical className="size-3.5" />
         </Button>
      </CardHeader>
      <CardContent className="space-y-2 pt-0">
        <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
           {(state === InstanceState.StateReady || state === InstanceState.StateStopped) && (
              <Button size="sm" disabled={actionBusy} onClick={(e) => { e.stopPropagation(); onPlay?.(); onAction?.('play'); }} aria-label={`Play ${name}`}>
              <Play className="size-3" />
              Play
            </Button>
          )}
           {(state === InstanceState.StateNotInstalled || state === InstanceState.StateFailed || state === InstanceState.StateCrashed) && (
              <Button size="sm" variant="secondary" disabled={actionBusy} onClick={(e) => { e.stopPropagation(); onDownload?.(); onAction?.('install'); }} aria-label={`Install ${name}`}>
              <Download className="size-3" />
              Install
            </Button>
          )}
            {(state === InstanceState.StateDownloading || state === InstanceState.StatePlanning || state === InstanceState.StateVerifying) && (
              <Button size="sm" variant="secondary" disabled={actionBusy} onClick={(e) => { e.stopPropagation(); onAction?.('cancel'); }} aria-label={`Cancel ${name}`}>
				  Cancel
				</Button>
            )}
           {state === InstanceState.StateRunning && (
              <Button size="sm" variant="secondary" disabled={actionBusy} onClick={(e) => { e.stopPropagation(); onAction?.('stop'); }} aria-label={`Stop ${name}`}>{actionState === 'stopping' ? 'Stopping...' : 'Stop'}</Button>
            )}
            {actionState === 'preparing' && <Button size="sm" variant="secondary" disabled>Launching...</Button>}
        </div>
        <Badge variant="outline" className="text-[10px] capitalize">{loader}</Badge>
        </div>
        {state === InstanceState.StateDownloading && downloadProgress && downloadProgress.totalFiles > 0 && (
          <div className="space-y-1">
            <progress className="h-1 w-full accent-primary" value={downloadProgress.fileProgress} max={downloadProgress.totalFiles} />
            <p className="text-[10px] text-muted-foreground">{downloadProgress.fileProgress}/{downloadProgress.totalFiles} files</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
