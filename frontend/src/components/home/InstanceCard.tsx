import { Play, Download, MoreVertical } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useReducedMotion } from '@/hooks/useReducedMotion';

type InstanceState = 'not_installed' | 'ready' | 'running' | 'downloading' | 'failed' | 'stopped' | 'crashed';

interface InstanceCardProps {
  name: string;
  mcVersion: string;
  loader?: 'vanilla' | 'fabric' | 'quilt';
  state?: InstanceState;
  onPlay?: () => void;
  onDownload?: () => void;
  onOpenDetail?: () => void;
  onAction?: (action: 'play' | 'install' | 'stop' | 'repair') => void;
  busy?: boolean;
}

const stateLabel: Record<InstanceState, string> = {
  not_installed: 'Not installed',
  ready: 'Ready',
  running: 'Running',
  downloading: 'Downloading',
};

const stateVariant: Record<InstanceState, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  not_installed: 'outline',
  ready: 'default',
  running: 'secondary',
  downloading: 'outline',
};

export function InstanceCard({
  name,
  mcVersion,
  loader = 'vanilla',
  state = 'not_installed',
  onPlay,
  onDownload,
  onOpenDetail,
  onAction,
  busy = false,
}: InstanceCardProps) {
  const reducedMotion = useReducedMotion();
  return (
    <Card
      className="cursor-pointer transition-colors hover:border-primary/40"
      onClick={onOpenDetail}
      onKeyDown={(event) => { if ((event.key === 'Enter' || event.key === ' ') && onOpenDetail) { event.preventDefault(); onOpenDetail(); } }}
      tabIndex={0}
      role="button"
      aria-label={`Open details for ${name}`}
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
      </CardHeader>
      <CardContent className="flex items-center justify-between pt-0">
        <div className="flex items-center gap-1.5">
           {(state === 'ready' || state === 'stopped') && (
             <Button size="sm" disabled={busy} onClick={(e) => { e.stopPropagation(); onPlay?.(); onAction?.('play'); }} aria-label={`Play ${name}`}>
              <Play className="size-3" />
              Play
            </Button>
          )}
           {(state === 'not_installed' || state === 'failed' || state === 'crashed') && (
             <Button size="sm" variant="secondary" disabled={busy} onClick={(e) => { e.stopPropagation(); onDownload?.(); onAction?.(state === 'not_installed' ? 'install' : 'repair'); }} aria-label={`${state === 'not_installed' ? 'Install' : 'Repair'} ${name}`}>
              <Download className="size-3" />
              Install
            </Button>
          )}
           {state === 'downloading' && (
            <Badge variant="outline" className="text-[10px]">
               <Download className={`mr-1 size-3 ${reducedMotion ? '' : 'animate-pulse'}`} />
              Downloading
            </Badge>
          )}
           {state === 'running' && (
             <Button size="sm" variant="secondary" disabled={busy} onClick={(e) => { e.stopPropagation(); onAction?.('stop'); }} aria-label={`Stop ${name}`}>Stop</Button>
           )}
        </div>
        <Badge variant="outline" className="text-[10px] capitalize">{loader}</Badge>
      </CardContent>
    </Card>
  );
}
