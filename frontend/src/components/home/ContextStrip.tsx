import { cn } from '@/lib/utils';

interface ContextStripProps {
  instanceCount?: number;
  liveStatus?: string;
}

export function ContextStrip({
  instanceCount = 0,
  liveStatus,
}: ContextStripProps) {
  return (
    <div
      className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground"
      data-slot="context-strip"
      aria-live="polite"
    >
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
