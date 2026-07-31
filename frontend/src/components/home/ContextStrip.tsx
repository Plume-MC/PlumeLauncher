import { User } from 'lucide-react';

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
    <div className="flex items-center gap-3 text-xs text-muted-foreground" data-slot="context-strip">
      <div className="flex items-center gap-1.5">
        <User className="size-3" />
        <span>{accountName}</span>
        <span className="text-muted-foreground/50">({accountType})</span>
      </div>
      <span className="text-muted-foreground/30">·</span>
      <span>{instanceCount} instances</span>
      {liveStatus && (
        <>
          <span className="text-muted-foreground/30">·</span>
          <span className="text-info">{liveStatus}</span>
        </>
      )}
    </div>
  );
}
