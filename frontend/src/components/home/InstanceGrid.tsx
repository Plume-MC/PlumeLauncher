import { IconPlus } from '@tabler/icons-react';
import { motion } from 'motion/react';
import { Button } from '@/components/ui/button';
import { InstanceCard, type InstanceAction } from './InstanceCard';
import type { Instance } from '../../../bindings/plumelauncher/internal/instances/models.js';
import { InstanceState } from '../../../bindings/plumelauncher/internal/instances/models.js';

interface InstanceGridProps {
  instances: Instance[];
  visibleInstances: Instance[];
  reduced: boolean;
  busyId: string;
  downloadInstanceId: string;
  downloadProgress: { fileProgress: number; totalFiles: number } | null;
  runningInstanceId: string;
  launchingInstanceId: string;
  stoppingInstanceId: string;
  stateOverrides: Record<string, InstanceState>;
  crashExitCodes: Record<string, number>;
  onCreate: () => void;
  onAction: (id: string, action: InstanceAction) => void;
  onOpenDetail: (instance: Instance) => void;
  onOpenLogs: () => void;
}

export function InstanceGrid({
  instances,
  visibleInstances,
  reduced,
  busyId,
  downloadInstanceId,
  downloadProgress,
  runningInstanceId,
  launchingInstanceId,
  stoppingInstanceId,
  stateOverrides,
  crashExitCodes,
  onCreate,
  onAction,
  onOpenDetail,
  onOpenLogs,
}: InstanceGridProps) {
  if (instances.length === 0) {
    return (
      <motion.div
        initial={reduced ? false : { opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
        className="flex flex-1 flex-col items-center justify-center rounded-xl border border-border bg-card/30 px-6 py-12 text-center"
      >
        <p className="text-sm font-medium text-foreground">No instances yet</p>
        <p className="mt-1 max-w-sm text-xs text-muted-foreground">
          Create a Vanilla, Fabric, or Quilt install. Files stay isolated under your game data root.
        </p>
        <Button
          size="sm"
          className="mt-5 gap-1.5 bg-foreground font-semibold text-background hover:bg-foreground/90"
          onClick={onCreate}
        >
          <IconPlus className="size-3.5" />
          Create instance
        </Button>
      </motion.div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {visibleInstances.length === 0 ? (
        <div className="col-span-full py-12 text-center text-sm text-muted-foreground">No matching instances.</div>
      ) : (
        visibleInstances.map((instance) => (
          <div key={instance.id}>
            <InstanceCard
              name={instance.name}
              busy={busyId === instance.id}
              actionState={
                instance.id === launchingInstanceId ? 'preparing' : instance.id === stoppingInstanceId ? 'stopping' : null
              }
              crashExitCode={crashExitCodes[instance.id] ?? null}
              onOpenLogs={onOpenLogs}
              mcVersion={instance.mcVersion}
              loader={instance.loader}
              state={
                stateOverrides[instance.id] ?? (instance.id === downloadInstanceId
                  ? InstanceState.StateDownloading
                  : instance.id === runningInstanceId
                    ? InstanceState.StateRunning
                    : instance.state)
              }
              downloadProgress={instance.id === downloadInstanceId ? downloadProgress : null}
              onAction={(action) => onAction(instance.id, action)}
              onOpenDetail={() => onOpenDetail(instance)}
            />
          </div>
        ))
      )}
    </div>
  );
}
