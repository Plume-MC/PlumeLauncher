import { User } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ContextStripProps {
  accountName?: string;
  accountType?: 'offline' | 'ely.by';
  instanceCount?: number;
  liveStatus?: string;
}

export function ContextStrip({
  accountName = 'No account',
  accountType = 'offline',
  instanceCount = 0,
  liveStatus,
}: ContextStripProps) {
  return (
    <div
      className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground"
      data-slot="context-strip"
      aria-live="polite"
    >
      <div className="flex items-center gap-1.5">
        <User className="size-3 shrink-0" />
        <span className="text-foreground/90">{accountName}</span>
        <span className="text-muted-foreground/60">({accountType})</span>
      </div>
      <span className="hidden text-border sm:inline" aria-hidden>
        ·
      </span>
      <span className="font-mono tabular-nums">
        {instanceCount} instance{instanceCount === 1 ? '' : 's'}
      </span>
      {liveStatus ? (
        <>
          <span className="hidden text-border sm:inline" aria-hidden>
            ·
          </span>
          <span className={cn('font-medium text-info')}>{liveStatus}</span>
        </>
      ) : null}
    </div>
  );
}
