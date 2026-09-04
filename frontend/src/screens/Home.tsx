import { lazy, Suspense, useEffect, useRef, useState } from 'react';
import { Events } from '@wailsio/runtime';
import { HomeService, SystemService } from '../../bindings/plumelauncher/internal/services/index.js';
import { toast } from '@/components/ui/toast';
import type { Account } from '../../bindings/plumelauncher/internal/services/models.js';
import type { Instance } from '../../bindings/plumelauncher/internal/instances/models.js';
import { InstanceState, LoaderType } from '../../bindings/plumelauncher/internal/instances/models.js';
import type { DownloadProgressEvent, LaunchStateEvent } from '../../bindings/plumelauncher/internal/services/models.js';
import { useMotionPreference } from '@/components/motion';

const InstanceLibraryToolbar = lazy(() => import('@/components/home/InstanceLibraryToolbar').then((module) => ({ default: module.InstanceLibraryToolbar })));
const InstanceGrid = lazy(() => import('@/components/home/InstanceGrid').then((module) => ({ default: module.InstanceGrid })));
const InstanceDetailSheet = lazy(() => import('@/components/home/InstanceDetailSheet').then((module) => ({ default: module.InstanceDetailSheet })));
const CreateInstanceDialog = lazy(() => import('@/components/home/CreateInstanceDialog').then((module) => ({ default: module.CreateInstanceDialog })));
const DeleteInstanceDialog = lazy(() => import('@/components/home/DeleteInstanceDialog').then((module) => ({ default: module.DeleteInstanceDialog })));

function LoadingOverlay() {
  return <div className="sr-only" role="status">Loading...</div>;
}

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
  const onRefreshRef = useRef(onRefresh);
  onRefreshRef.current = onRefresh;

  useEffect(() => {
    let frame = 0;
    let nextDownload: DownloadProgressEvent | null = null;
    let nextLaunch: LaunchStateEvent | null = null;
    const flush = () => {
      frame = 0;
      if (nextDownload) {
        const data = nextDownload;
        nextDownload = null;
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
          void onRefreshRef.current();
        }
      }
      if (nextLaunch) {
        const data = nextLaunch;
        nextLaunch = null;
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
          void onRefreshRef.current();
        }
      }
    };
    const schedule = () => {
      if (!frame) frame = requestAnimationFrame(flush);
    };
    const flushNow = () => {
      if (frame) cancelAnimationFrame(frame);
      flush();
    };
    const unsubs = [
      Events.On('download-progress', (event) => {
        const data: DownloadProgressEvent = event.data;
        nextDownload = data;
        if (['completed', 'failed', 'cancelled'].includes(data.status)) flushNow();
        else schedule();
      }),
      Events.On('launch-state', (event) => {
        const data: LaunchStateEvent = event.data;
        nextLaunch = data;
        if (['stopped', 'failed', 'crashed'].includes(data.state)) flushNow();
        else schedule();
      }),
    ];
    return () => {
      if (frame) cancelAnimationFrame(frame);
      unsubs.forEach((unsubscribe) => unsubscribe());
    };
  }, []);

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
      <Suspense fallback={<LoadingOverlay />}>
        <InstanceLibraryToolbar
          accountName={account?.displayName || account?.username}
          accountType={account?.type === 'ely.by' ? 'ely.by' : 'offline'}
          instanceCount={instances.length}
          liveStatus={liveStatus}
          search={search}
          loaderFilter={loaderFilter}
          sort={sort}
          onCreate={() => setCreateOpen(true)}
          onSearchChange={setSearch}
          onLoaderFilterChange={setLoaderFilter}
          onSortChange={setSort}
        />
        <InstanceGrid
          instances={instances}
          visibleInstances={visibleInstances}
          reduced={reduced}
          busyId={busyId}
          downloadInstanceId={downloadInstanceId}
          downloadProgress={downloadProgress}
          runningInstanceId={runningInstanceId}
          launchingInstanceId={launchingInstanceId}
          stoppingInstanceId={stoppingInstanceId}
          crashExitCodes={crashExitCodes}
          onCreate={() => setCreateOpen(true)}
          onAction={(id, action) => void runAction(id, action)}
          onOpenDetail={(instance) => {
            setDetailInstance(instance);
            setDetailOpen(true);
          }}
          onOpenLogs={() => void SystemService.OpenLogFolder()}
        />
      </Suspense>

      <Suspense fallback={<LoadingOverlay />}>
        {detailOpen && <InstanceDetailSheet
          isOpen={detailOpen}
          onClose={() => setDetailOpen(false)}
          instance={detailInstance}
          onChanged={onRefresh}
          onDelete={() => {
            setDetailOpen(false);
            setDeleteId(detailInstance?.id ?? null);
          }}
        />}
        {createOpen && <CreateInstanceDialog open={createOpen} onOpenChange={setCreateOpen} onCreated={onRefresh} />}
        {deleteId && <DeleteInstanceDialog
          instanceId={deleteId}
          instanceName={detailInstance?.name}
          onOpenChange={(open) => {
            if (!open) setDeleteId(null);
          }}
          onDeleted={onRefresh}
        />}
      </Suspense>
    </div>
  );
}
