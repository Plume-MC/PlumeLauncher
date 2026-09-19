import { IconPlus } from '@tabler/icons-react';
import { motion } from 'motion/react';
import { InstanceCard, type InstanceAction } from './InstanceCard';
import { EmptyState } from './EmptyState';
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
    return <EmptyState reduced={reduced} onCreate={onCreate} />;
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
      <motion.button
        type="button"
        layout
        whileHover={{ y: -2 }}
        whileTap={{ scale: 0.985 }}
        transition={{ duration: 0.18, ease: [0.16, 1, 0.3, 1] }}
        onClick={onCreate}
        aria-label="Create new instance"
        data-slot="new-instance-tile"
        className="flex min-h-[156px] flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-border/70 bg-transparent text-muted-foreground transition-colors hover:border-border hover:bg-card/40 hover:text-foreground"
      >
        <IconPlus className="size-5" />
        <span className="text-sm font-medium">Create instance</span>
      </motion.button>
    </div>
  );
}
