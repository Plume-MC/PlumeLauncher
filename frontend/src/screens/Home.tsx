import { useState } from 'react';
import { Plus, Search, SlidersHorizontal } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ContextStrip } from '@/components/home/ContextStrip';
import { InstanceCard } from '@/components/home/InstanceCard';
import { InstanceDetailSheet } from '@/components/home/InstanceDetailSheet';
import type { Account } from '../../bindings/plumelauncher/internal/services/models.js';
import type { Instance } from '../../bindings/plumelauncher/internal/instances/models.js';

interface HomeProps {
  account: Account | null;
  instances: Instance[];
  onRefresh: () => Promise<void>;
}

export function Home({ account, instances, onRefresh }: HomeProps) {
  const [search, setSearch] = useState('');
  const [detailOpen, setDetailOpen] = useState(false);

  return (
    <div className="mx-auto flex h-full max-w-5xl flex-col px-5 py-6">
      {/* Header */}
      <div className="mb-4 space-y-2">
        <h1 className="text-lg font-semibold tracking-tight">Instance Library</h1>
         <ContextStrip accountName={account?.displayName || account?.username} accountType={account?.type === 'ely.by' ? 'ely.by' : 'offline'} instanceCount={instances.length} />
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
        <Button variant="ghost" size="icon-sm" aria-label="Filter">
          <SlidersHorizontal className="size-3.5" />
        </Button>
        <Button size="sm" className="gap-1.5">
          <Plus className="size-3.5" />
          New Instance
        </Button>
      </div>

      {/* Card grid */}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {instances.length === 0 && <div className="col-span-full rounded-lg border border-dashed border-border py-16 text-center">
          <p className="text-sm text-muted-foreground">No instances yet.</p>
          <p className="mt-1 text-xs text-muted-foreground/50">
            Click "New Instance" to create one.
          </p>
        </div>}
        {instances.map((instance) => <InstanceCard key={instance.id} name={instance.name} mcVersion={instance.mcVersion} loader={instance.loader} state={instance.state as 'not_installed' | 'ready' | 'running' | 'downloading'} onOpenDetail={() => setDetailOpen(true)} />)}
      </div>

      <InstanceDetailSheet isOpen={detailOpen} onClose={() => setDetailOpen(false)} />
    </div>
  );
}
