import { IconPlus, IconSearch } from '@tabler/icons-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select';
import { ContextStrip } from './ContextStrip';
import type { LoaderType } from '../../../bindings/plumelauncher/internal/instances/models.js';

interface InstanceLibraryToolbarProps {
  instanceCount: number;
  liveStatus: string;
  search: string;
  loaderFilter: 'all' | LoaderType;
  sort: 'newest' | 'oldest' | 'name';
  onCreate: () => void;
  onSearchChange: (search: string) => void;
  onLoaderFilterChange: (loaderFilter: 'all' | LoaderType) => void;
  onSortChange: (sort: 'newest' | 'oldest' | 'name') => void;
}

export function InstanceLibraryToolbar({
  instanceCount,
  liveStatus,
  search,
  loaderFilter,
  sort,
  onCreate,
  onSearchChange,
  onLoaderFilterChange,
  onSortChange,
}: InstanceLibraryToolbarProps) {
  return (
    <>
      <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-1.5">
          <h1 className="text-xl font-semibold tracking-tight">Instance Library</h1>
          <ContextStrip
            instanceCount={instanceCount}
            liveStatus={liveStatus}
          />
        </div>
        <Button
          size="sm"
          className="h-9 shrink-0 gap-1.5 bg-foreground font-semibold text-background hover:bg-foreground/90"
          onClick={onCreate}
        >
          <IconPlus className="size-3.5" />
          New instance
        </Button>
      </div>

      <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
        <div className="relative min-w-0 flex-1 sm:min-w-[220px]">
          <IconSearch className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder="Search instances..."
            className="h-9 pl-8 text-xs"
            aria-label="Search instances"
          />
        </div>
        <div className="flex flex-wrap gap-2">
          <Select value={loaderFilter as string} onValueChange={(value) => onLoaderFilterChange((value ?? 'all') as 'all' | LoaderType)}>
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
          <Select value={sort} onValueChange={(value) => onSortChange((value ?? 'newest') as 'newest' | 'oldest' | 'name')}>
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
    </>
  );
}
