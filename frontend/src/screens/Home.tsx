import { useEffect, useState } from 'react';
import { Events } from '@wailsio/runtime';
import { Plus, Search } from 'lucide-react';
import { motion } from 'motion/react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select';
import { ContextStrip } from '@/components/home/ContextStrip';
import { InstanceCard } from '@/components/home/InstanceCard';
import { InstanceDetailSheet } from '@/components/home/InstanceDetailSheet';
import { CreateInstanceDialog } from '@/components/home/CreateInstanceDialog';
import { DeleteInstanceDialog } from '@/components/home/DeleteInstanceDialog';
import { HomeService, SystemService } from '../../bindings/plumelauncher/internal/services/index.js';
import { toast } from '@/components/ui/toast';
import type { Account } from '../../bindings/plumelauncher/internal/services/models.js';
import type { Instance } from '../../bindings/plumelauncher/internal/instances/models.js';
import { InstanceState, LoaderType } from '../../bindings/plumelauncher/internal/instances/models.js';
import type { DownloadProgressEvent, LaunchStateEvent } from '../../bindings/plumelauncher/internal/services/models.js';
import { useMotionPreference } from '@/components/motion';

interface HomeProps {
  account: Account | null;
  instances: Instance[];
  onRefresh: () => Promise<void>;
}

export function Home({ account, instances, onRefresh }: HomeProps) {
  const { reduced } = useMotionPreference();
  const [search, setSearch] = useState('');
  const [loaderFilter, setLoaderFilter] = useState<'all' | LoaderType>('all');
  const [sort, setSort] = useState<'newest' | 'oldest' | 'name'>('newest');
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailInstance, setDetailInstance] = useState<Instance | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [busyId, setBusyId] = useState('');
  const [liveStatus, setLiveStatus] = useState('');
  const [downloadInstanceId, setDownloadInstanceId] = useState('');
  const [downloadProgress, setDownloadProgress] = useState<{ fileProgress: number; totalFiles: number } | null>(null);
  const [runningInstanceId, setRunningInstanceId] = useState('');
  const [launchingInstanceId, setLaunchingInstanceId] = useState('');
  const [stoppingInstanceId, setStoppingInstanceId] = useState('');
  const [crashExitCodes, setCrashExitCodes] = useState<Record<string, number>>({});

  useEffect(() => {
    const unsubs = [
      Events.On('download-progress', (event) => {
        const data: DownloadProgressEvent = event.data;
        setLiveStatus(
          data.status === 'downloading'
            ? `Downloading ${data.fileProgress}/${data.totalFiles}`
            : data.status === 'repairing'
              ? 'Repairing instance'
              : data.status === 'cancelled'
                ? 'Download cancelled'
                : data.status === 'failed'
                  ? 'Download failed'
                  : ''
        );
        if (data.status === 'downloading' || data.status === 'repairing') {
          setDownloadInstanceId(data.instanceId);
          setDownloadProgress({ fileProgress: data.fileProgress, totalFiles: data.totalFiles });
        } else if (data.status === 'completed' || data.status === 'cancelled' || data.status === 'failed') {
          setDownloadInstanceId('');
          setDownloadProgress(null);
          void onRefresh();
        }
      }),
      Events.On('launch-state', (event) => {
        const data: LaunchStateEvent = event.data;
        setLiveStatus(
          data.state === 'preparing'
            ? 'Preparing Minecraft'
            : data.state === 'running'
              ? 'Minecraft is running'
              : data.state === 'stopping'
                ? 'Stopping Minecraft'
                : data.state === 'crashed'
                  ? 'Minecraft crashed'
                  : data.state === 'failed'
                    ? 'Launch failed'
                    : ''
        );
        if (data.state === 'preparing') {
          setLaunchingInstanceId(data.instanceId);
        } else if (data.state === 'stopping') {
          setStoppingInstanceId(data.instanceId);
        } else if (data.state === 'running') {
          setLaunchingInstanceId('');
          setStoppingInstanceId('');
          setRunningInstanceId(data.instanceId);
        } else if (data.state === 'stopped' || data.state === 'crashed' || data.state === 'failed') {
          setLaunchingInstanceId('');
          setStoppingInstanceId('');
          setRunningInstanceId('');
          if (data.state === 'crashed' && data.exitCode !== undefined && data.exitCode !== null) {
            setCrashExitCodes((current) => ({ ...current, [data.instanceId]: data.exitCode as number }));
          }
          void onRefresh();
        }
      }),
    ];
    return () => unsubs.forEach((unsubscribe) => unsubscribe());
  }, [onRefresh]);

  const visibleInstances = [...instances]
    .filter((instance) => loaderFilter === 'all' || instance.loader === loaderFilter)
    .filter((instance) => instance.name.toLowerCase().includes(search.toLowerCase()))
    .sort((a, b) =>
      sort === 'name'
        ? a.name.localeCompare(b.name)
        : sort === 'oldest'
          ? String(a.createdAt).localeCompare(String(b.createdAt))
          : String(b.createdAt).localeCompare(String(a.createdAt))
    );

  const runAction = async (id: string, action: 'play' | 'install' | 'stop' | 'cancel') => {
    // Cancel must work while install/play holds the card busy.
    if (busyId && action !== 'cancel') return;

    if (action === 'install') {
      setBusyId(id);
      toast.add({ type: 'success', title: 'Install requested' });
      void HomeService.InstallInstance(id)
        .catch((error) => {
          if (error instanceof Error && /cancelled|canceled/i.test(error.message)) {
            toast.add({
              type: 'info',
              title: 'Install cancelled',
              description: 'The instance was returned to its previous state.',
            });
            return;
          }
          toast.add({
            type: 'error',
            title: 'Action failed',
            description: error instanceof Error ? error.message : 'Please check the launcher logs.',
            priority: 'high',
          });
        })
        .finally(() => {
          void onRefresh();
          setBusyId((current) => (current === id ? '' : current));
        });
      // Clear busy quickly so Cancel stays clickable; download progress uses events.
      setTimeout(() => setBusyId((current) => (current === id ? '' : current)), 400);
      return;
    }

    if (action === 'cancel') {
      try {
        await HomeService.CancelInstance(id);
        toast.add({ type: 'success', title: 'Cancellation requested' });
      } catch (error) {
        toast.add({
          type: 'error',
          title: 'Cancel failed',
          description: error instanceof Error ? error.message : 'Please check the launcher logs.',
          priority: 'high',
        });
      } finally {
        setBusyId('');
        void onRefresh();
      }
      return;
    }

    setBusyId(id);
    try {
      if (action === 'play') await HomeService.LaunchInstance(id);
      if (action === 'stop') await HomeService.StopInstance(id);
      toast.add({
        type: 'success',
        title: action === 'play' ? 'Launch started' : 'Stop requested',
      });
    } catch (error) {
      if (error instanceof Error && /cancelled|canceled/i.test(error.message)) {
        toast.add({
          type: 'info',
          title: 'Launch cancelled',
          description: 'The instance was returned to its previous state.',
        });
        return;
      }
      toast.add({
        type: 'error',
        title: 'Action failed',
        description: error instanceof Error ? error.message : 'Please check the launcher logs.',
        priority: 'high',
      });
    } finally {
      await onRefresh();
      setBusyId('');
    }
  };

  return (
    <div className="mx-auto flex h-full w-full max-w-7xl flex-col px-4 py-5 sm:px-6 lg:px-8">
      <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-1.5">
          <h1 className="text-xl font-semibold tracking-tight">Instance Library</h1>
          <ContextStrip
            accountName={account?.displayName || account?.username}
            accountType={account?.type === 'ely.by' ? 'ely.by' : 'offline'}
            instanceCount={instances.length}
            liveStatus={liveStatus}
          />
        </div>
        <Button
          size="sm"
          className="h-9 shrink-0 gap-1.5 bg-foreground font-semibold text-background hover:bg-foreground/90"
          onClick={() => setCreateOpen(true)}
        >
          <Plus className="size-3.5" />
          New instance
        </Button>
      </div>

      <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
        <div className="relative min-w-0 flex-1 sm:min-w-[220px]">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search instances..."
            className="h-9 pl-8 text-xs"
            aria-label="Search instances"
          />
        </div>
        <div className="flex flex-wrap gap-2">
          <Select
            value={loaderFilter as string}
            onValueChange={(value) => setLoaderFilter((value ?? 'all') as 'all' | LoaderType)}
          >
            <SelectTrigger aria-label="Filter by loader" className="h-9 w-[9.5rem] shrink-0">
              <span className="truncate capitalize">{loaderFilter === 'all' ? 'All loaders' : loaderFilter}</span>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All loaders</SelectItem>
              <SelectItem value="vanilla">Vanilla</SelectItem>
              <SelectItem value="fabric">Fabric</SelectItem>
              <SelectItem value="quilt">Quilt</SelectItem>
            </SelectContent>
          </Select>
          <Select value={sort} onValueChange={(value) => setSort((value ?? 'newest') as typeof sort)}>
            <SelectTrigger aria-label="Sort instances" className="h-9 w-[7.5rem] shrink-0">
              <span className="truncate capitalize">{sort === 'name' ? 'Name' : sort}</span>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="newest">Newest</SelectItem>
              <SelectItem value="oldest">Oldest</SelectItem>
              <SelectItem value="name">Name</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {instances.length === 0 ? (
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
            onClick={() => setCreateOpen(true)}
          >
            <Plus className="size-3.5" />
            Create instance
          </Button>
        </motion.div>
      ) : (
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
                    instance.id === launchingInstanceId
                      ? 'preparing'
                      : instance.id === stoppingInstanceId
                        ? 'stopping'
                        : null
                  }
                  crashExitCode={crashExitCodes[instance.id] ?? null}
                  onOpenLogs={() => void SystemService.OpenLogFolder()}
                  mcVersion={instance.mcVersion}
                  loader={instance.loader}
                  state={
                    instance.id === downloadInstanceId
                      ? InstanceState.StateDownloading
                      : instance.id === runningInstanceId
                        ? InstanceState.StateRunning
                        : instance.state
                  }
                  downloadProgress={instance.id === downloadInstanceId ? downloadProgress : null}
                  onAction={(action) => void runAction(instance.id, action)}
                  onOpenDetail={() => {
                    setDetailInstance(instance);
                    setDetailOpen(true);
                  }}
                />
              </div>
            ))
          )}
        </div>
      )}

      <InstanceDetailSheet
        isOpen={detailOpen}
        onClose={() => setDetailOpen(false)}
        instance={detailInstance}
        onChanged={onRefresh}
        onDelete={() => {
          setDetailOpen(false);
          setDeleteId(detailInstance?.id ?? null);
        }}
      />
      <CreateInstanceDialog open={createOpen} onOpenChange={setCreateOpen} onCreated={onRefresh} />
      <DeleteInstanceDialog
        instanceId={deleteId}
        instanceName={detailInstance?.name}
        onOpenChange={(open) => {
          if (!open) setDeleteId(null);
        }}
        onDeleted={onRefresh}
      />
    </div>
  );
}
