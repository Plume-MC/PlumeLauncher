import { IconPlayerPlay, IconDownload, IconDotsVertical, IconPlayerStop, IconLoader2 } from '@tabler/icons-react';
import { motion } from 'motion/react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { InstanceState, LoaderType } from '../../../bindings/plumelauncher/internal/instances/models.js';
import vanillaLogo from '@/assets/loaders/vanilla.png';
import fabricLogo from '@/assets/loaders/fabric.png';
import quiltLogo from '@/assets/loaders/quilt.png';

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
  crashExitCode?: number | null;
  onOpenLogs?: () => void;
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

const loaderMeta: Record<string, { src: string; chip: string }> = {
  [LoaderType.LoaderVanilla]: { src: vanillaLogo, chip: 'bg-emerald-500/15 text-emerald-400' },
  [LoaderType.LoaderFabric]: { src: fabricLogo, chip: 'bg-amber-500/15 text-amber-400' },
  [LoaderType.LoaderQuilt]: { src: quiltLogo, chip: 'bg-primary/15 text-primary' },
};

function statusChip(state: InstanceState): string {
  if (state === InstanceState.StateReady) return 'bg-success/15 text-success';
  if (state === InstanceState.StateRunning) return 'bg-info/15 text-info';
  if (
    state === InstanceState.StateDownloading ||
    state === InstanceState.StatePlanning ||
    state === InstanceState.StateVerifying
  ) {
    return 'bg-cyan-500/15 text-cyan-400';
  }
  if (state === InstanceState.StateFailed || state === InstanceState.StateCrashed) {
    return 'bg-destructive/15 text-destructive';
  }
  return 'bg-muted text-muted-foreground';
}

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
  crashExitCode = null,
  onOpenLogs,
}: InstanceCardProps) {
  const actionBusy = busy || actionState !== null;
  const meta = loaderMeta[loader] ?? loaderMeta[LoaderType.LoaderVanilla];
  const progressPct =
    downloadProgress && downloadProgress.totalFiles > 0
      ? Math.min(100, (downloadProgress.fileProgress / downloadProgress.totalFiles) * 100)
      : 0;

  const primaryAction = () => {
    if (state === InstanceState.StateReady || state === InstanceState.StateStopped) {
      return (
        <Button
          size="sm"
          disabled={actionBusy}
          className="h-9 w-full gap-1.5 bg-foreground font-semibold text-background hover:bg-foreground/90"
          onClick={(e) => {
            e.stopPropagation();
            onPlay?.();
            onAction?.('play');
          }}
          aria-label={`Play ${name}`}
        >
          <IconPlayerPlay className="size-3.5" />
          Play
        </Button>
      );
    }
    if (state === InstanceState.StateCrashed) {
      return (
        <div className="flex w-full gap-2">
          <Button
            size="sm"
            variant="secondary"
            disabled={actionBusy}
            className="h-9 flex-1 gap-1.5"
            onClick={(e) => {
              e.stopPropagation();
              onAction?.('play');
            }}
            aria-label={`Play ${name} again`}
          >
            <IconPlayerPlay className="size-3.5" />
            Play again
          </Button>
          {onOpenLogs && (
            <Button
              size="sm"
              variant="ghost"
              className="h-9"
              onClick={(e) => {
                e.stopPropagation();
                onOpenLogs();
              }}
            >
              Logs
            </Button>
          )}
        </div>
      );
    }
    if (state === InstanceState.StateNotInstalled || state === InstanceState.StateFailed) {
      return (
        <Button
          size="sm"
          variant="secondary"
          disabled={actionBusy}
          className="h-9 w-full gap-1.5"
          onClick={(e) => {
            e.stopPropagation();
            onDownload?.();
            onAction?.('install');
          }}
          aria-label={`Install ${name}`}
        >
          <IconDownload className="size-3.5" />
          Install
        </Button>
      );
    }
    if (
      state === InstanceState.StateDownloading ||
      state === InstanceState.StatePlanning ||
      state === InstanceState.StateVerifying
    ) {
      return (
        <Button
          size="sm"
          variant="secondary"
          className="h-9 w-full"
          onClick={(e) => {
            e.stopPropagation();
            onAction?.('cancel');
          }}
          aria-label={`Cancel ${name}`}
        >
          Cancel
        </Button>
      );
    }
    if (state === InstanceState.StateRunning) {
      return (
        <Button
          size="sm"
          variant="secondary"
          disabled={actionBusy}
          className="h-9 w-full gap-1.5"
          onClick={(e) => {
            e.stopPropagation();
            onAction?.('stop');
          }}
          aria-label={`Stop ${name}`}
        >
          {actionState === 'stopping' ? (
            <IconLoader2 className="size-3.5 animate-spin" />
          ) : (
            <IconPlayerStop className="size-3.5" />
          )}
          {actionState === 'stopping' ? 'Stopping...' : 'Stop'}
        </Button>
      );
    }
    if (actionState === 'preparing') {
      return (
        <Button size="sm" variant="secondary" disabled className="h-9 w-full gap-1.5">
          <IconLoader2 className="size-3.5 animate-spin" />
          Launching...
        </Button>
      );
    }
    return null;
  };

  return (
    <motion.article
      layout
      data-slot="instance-card"
      whileHover={{ y: -2 }}
      transition={{ duration: 0.18, ease: [0.16, 1, 0.3, 1] }}
      className="flex flex-col rounded-xl border border-border bg-card/70 p-3.5 transition-colors hover:border-border hover:bg-card"
    >
      <div className="mb-3 flex items-start gap-3">
        <img
          src={meta.src}
          alt=""
          className="size-11 shrink-0 rounded-xl object-cover ring-1 ring-border/60"
        />
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-2">
            <h3 className="truncate text-sm font-semibold text-foreground">{name}</h3>
            <Button
              size="icon-xs"
              variant="ghost"
              className="shrink-0 text-muted-foreground"
              onClick={(event) => {
                event.stopPropagation();
                onOpenDetail?.();
              }}
              aria-label={`Open details for ${name}`}
            >
              <IconDotsVertical className="size-3.5" />
            </Button>
          </div>
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
            <span className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[10px] tabular-nums text-muted-foreground">
              {mcVersion}
            </span>
            <span className={cn('rounded-md px-1.5 py-0.5 text-[10px] font-medium capitalize', meta.chip)}>
              {loader}
            </span>
            <span className={cn('rounded-md px-1.5 py-0.5 text-[10px] font-medium', statusChip(state))}>
              {actionState === 'preparing'
                ? 'Launching'
                : actionState === 'stopping'
                  ? 'Stopping'
                  : stateLabel[state]}
            </span>
          </div>
        </div>
      </div>

      {state === InstanceState.StateDownloading && downloadProgress && downloadProgress.totalFiles > 0 && (
        <div className="mb-3 space-y-1.5">
          <div className="flex justify-between font-mono text-[10px] text-muted-foreground">
            <span>Downloading</span>
            <span>
              {downloadProgress.fileProgress}/{downloadProgress.totalFiles}
            </span>
          </div>
          <div className="h-1 overflow-hidden rounded-full bg-muted">
            <motion.div
              className="h-full rounded-full bg-primary"
              initial={false}
              animate={{ width: `${progressPct}%` }}
              transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
            />
          </div>
        </div>
      )}

      {state === InstanceState.StateCrashed && crashExitCode !== null && (
        <p className="mb-2 font-mono text-[10px] text-destructive">Exit code {crashExitCode}</p>
      )}

      <div className="mt-auto">{primaryAction()}</div>
    </motion.article>
  );
}
