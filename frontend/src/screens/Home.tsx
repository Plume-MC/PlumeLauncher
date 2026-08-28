import { useEffect, useState } from 'react';
import { Events } from '@wailsio/runtime';
import { Plus, Search, UserRound } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select';
import { ContextStrip } from '@/components/home/ContextStrip';
import { InstanceCard } from '@/components/home/InstanceCard';
import { InstanceDetailSheet } from '@/components/home/InstanceDetailSheet';
import { CreateInstanceDialog } from '@/components/home/CreateInstanceDialog';
import { DeleteInstanceDialog } from '@/components/home/DeleteInstanceDialog';
import { AccountDialog } from '@/components/home/AccountDialog';
import { HomeService } from '../../bindings/plumelauncher/internal/services/index.js';
import { toast } from '@/components/ui/toast';
import type { Account } from '../../bindings/plumelauncher/internal/services/models.js';
import type { Instance } from '../../bindings/plumelauncher/internal/instances/models.js';
import { InstanceState, LoaderType } from '../../bindings/plumelauncher/internal/instances/models.js';
import type { DownloadProgressEvent, LaunchStateEvent } from '../../bindings/plumelauncher/internal/services/models.js';

interface HomeProps {
  account: Account | null;
  accounts: Account[];
  instances: Instance[];
  onRefresh: () => Promise<void>;
}

export function Home({ account, accounts, instances, onRefresh }: HomeProps) {
  const [search, setSearch] = useState('');
  const [loaderFilter, setLoaderFilter] = useState<'all' | LoaderType>('all');
  const [sort, setSort] = useState<'newest' | 'oldest' | 'name'>('newest');
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailInstance, setDetailInstance] = useState<Instance | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [busyId, setBusyId] = useState('');
   const [accountsOpen, setAccountsOpen] = useState(false);
   const [liveStatus, setLiveStatus] = useState('');
	const [downloadInstanceId, setDownloadInstanceId] = useState('');
	const [downloadProgress, setDownloadProgress] = useState<{ fileProgress: number; totalFiles: number } | null>(null);
	const [runningInstanceId, setRunningInstanceId] = useState('');
	const [launchingInstanceId, setLaunchingInstanceId] = useState('');
	const [stoppingInstanceId, setStoppingInstanceId] = useState('');

	useEffect(() => {
	  const unsubs = [
		Events.On('download-progress', (event) => {
		  const data: DownloadProgressEvent = event.data;
		  setLiveStatus(data.status === 'downloading' ? `Downloading ${data.fileProgress}/${data.totalFiles}` : data.status === 'repairing' ? 'Repairing instance' : data.status === 'cancelled' ? 'Download cancelled' : data.status === 'failed' ? 'Download failed' : '');
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
		  setLiveStatus(data.state === 'preparing' ? 'Preparing Minecraft' : data.state === 'running' ? 'Minecraft is running' : data.state === 'stopping' ? 'Stopping Minecraft' : data.state === 'crashed' ? 'Minecraft crashed' : data.state === 'failed' ? 'Launch failed' : '');
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
			void onRefresh();
		  }
		}),
	  ];
	  return () => unsubs.forEach((unsubscribe) => unsubscribe());
	}, [onRefresh]);

  const visibleInstances = [...instances]
    .filter((instance) => loaderFilter === 'all' || instance.loader === loaderFilter)
    .filter((instance) => instance.name.toLowerCase().includes(search.toLowerCase()))
    .sort((a, b) => sort === 'name' ? a.name.localeCompare(b.name) : sort === 'oldest' ? a.createdAt.localeCompare(b.createdAt) : b.createdAt.localeCompare(a.createdAt));

  const runAction = async (id: string, action: 'play' | 'install' | 'stop' | 'cancel') => {
    if (busyId) return;
    setBusyId(id)
    try {
      if (action === 'play') await HomeService.LaunchInstance(id)
      if (action === 'install') await HomeService.InstallInstance(id)
      if (action === 'cancel') await HomeService.CancelInstance(id)
      if (action === 'stop') await HomeService.StopInstance(id)
		  toast.add({ type: 'success', title: action === 'cancel' ? 'Cancellation requested' : action === 'play' ? 'Launch started' : `${action === 'install' ? 'Install' : 'Stop'} requested` })
    } catch (error) {
      if (error instanceof Error && /cancelled|canceled/i.test(error.message)) {
        toast.add({ type: 'info', title: action === 'play' ? 'Launch cancelled' : 'Install cancelled', description: 'The instance was returned to its previous state.' })
        return
      }
      toast.add({ type: 'error', title: 'Action failed', description: error instanceof Error ? error.message : 'Please check the launcher logs.', priority: 'high' })
    } finally {
      await onRefresh()
      setBusyId('')
    }
  }

  return (
    <div className="mx-auto flex h-full max-w-5xl flex-col px-5 py-6">
      {/* Header */}
      <div className="mb-4 space-y-2">
        <h1 className="text-lg font-semibold tracking-tight">Instance Library</h1>
          <ContextStrip accountName={account?.displayName || account?.username} accountType={account?.type === 'ely.by' ? 'ely.by' : 'offline'} instanceCount={instances.length} liveStatus={liveStatus} />
      </div>

      {/* Toolbar */}
      <div className="mb-4 flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search instances..."
            className="h-8 pl-8 text-xs"
            aria-label="Search instances"
          />
        </div>
          <Select value={loaderFilter as string} onValueChange={(value) => setLoaderFilter((value ?? 'all') as 'all' | LoaderType)}>
            <SelectTrigger aria-label="Filter by loader"><span>{loaderFilter === 'all' ? 'All loaders' : loaderFilter}</span></SelectTrigger>
            <SelectContent><SelectItem value="all">All loaders</SelectItem><SelectItem value="vanilla">Vanilla</SelectItem><SelectItem value="fabric">Fabric</SelectItem><SelectItem value="quilt">Quilt</SelectItem></SelectContent>
          </Select>
          <Select value={sort} onValueChange={(value) => setSort((value ?? 'newest') as typeof sort)}>
            <SelectTrigger aria-label="Sort instances"><span className="capitalize">{sort}</span></SelectTrigger>
            <SelectContent><SelectItem value="newest">Newest</SelectItem><SelectItem value="oldest">Oldest</SelectItem><SelectItem value="name">Name</SelectItem></SelectContent>
          </Select>
         <Button size="sm" className="gap-1.5" onClick={() => setCreateOpen(true)}>
          <Plus className="size-3.5" />
          New Instance
         </Button>
         <Button size="sm" variant="ghost" className="gap-1.5" onClick={() => setAccountsOpen(true)}><UserRound className="size-3.5" />Accounts</Button>
      </div>

      {/* Card grid */}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {instances.length === 0 && <div className="col-span-full flex flex-col items-center gap-4 rounded-lg border border-dashed border-border py-16 text-center">
          <div className="flex size-14 items-center justify-center rounded-2xl bg-primary/10">
            <Plus className="size-6 text-primary" />
          </div>
          <div>
            <p className="text-sm font-medium">Your library is empty</p>
            <p className="mt-1 text-xs text-muted-foreground">Create your first instance to start playing.</p>
          </div>
          <Button size="sm" onClick={() => setCreateOpen(true)}>
            <Plus className="mr-1.5 size-3.5" />
            Create Instance
          </Button>
        </div>}
        {instances.length > 0 && visibleInstances.length === 0 && <div className="col-span-full py-16 text-center text-sm text-muted-foreground">No matching instances.</div>}
		 {visibleInstances.map((instance) => <InstanceCard key={instance.id} name={instance.name} busy={busyId === instance.id} actionState={instance.id === launchingInstanceId ? 'preparing' : instance.id === stoppingInstanceId ? 'stopping' : null} mcVersion={instance.mcVersion} loader={instance.loader} state={instance.id === downloadInstanceId ? InstanceState.StateDownloading : instance.id === runningInstanceId ? InstanceState.StateRunning : instance.state} downloadProgress={instance.id === downloadInstanceId ? downloadProgress : null} onAction={(action) => void runAction(instance.id, action)} onOpenDetail={() => { setDetailInstance(instance); setDetailOpen(true); }} />)}
      </div>

       <InstanceDetailSheet isOpen={detailOpen} onClose={() => setDetailOpen(false)} instance={detailInstance} onChanged={onRefresh} onDelete={() => { setDetailOpen(false); setDeleteId(detailInstance?.id ?? null); }} />
      <CreateInstanceDialog open={createOpen} onOpenChange={setCreateOpen} onCreated={onRefresh} />
       <DeleteInstanceDialog instanceId={deleteId} instanceName={detailInstance?.name} onOpenChange={(open) => { if (!open) setDeleteId(null); }} onDeleted={onRefresh} />
       <AccountDialog open={accountsOpen} onOpenChange={setAccountsOpen} accounts={accounts} onChanged={onRefresh} />
    </div>
  );
}
