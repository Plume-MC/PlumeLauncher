import { X, FolderOpen, Shield, Wrench, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Badge } from '@/components/ui/badge';

interface InstanceDetailSheetProps {
  isOpen: boolean;
  onClose: () => void;
  instanceName?: string;
  mcVersion?: string;
  loader?: string;
}

export function InstanceDetailSheet({
  isOpen,
  onClose,
  instanceName = 'Instance',
  mcVersion = '1.21',
  loader = 'vanilla',
}: InstanceDetailSheetProps) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/40" onClick={onClose} />

      {/* Sheet */}
      <div className="relative flex h-[520px] w-[420px] max-w-[90vw] flex-col rounded-lg border border-border bg-background shadow-xl">
        {/* Header */}
        <div className="flex items-start justify-between border-b border-border px-5 py-4">
          <div>
            <h2 className="text-sm font-semibold">{instanceName}</h2>
            <p className="mt-0.5 font-mono text-xs text-muted-foreground">
              MC {mcVersion} · <span className="capitalize">{loader}</span>
            </p>
          </div>
          <Button variant="ghost" size="icon-xs" onClick={onClose} aria-label="Close detail">
            <X className="size-4" />
          </Button>
        </div>

        {/* Content */}
        <div className="flex-1 space-y-4 overflow-auto px-5 py-4">
          <div>
            <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
              Status
            </p>
            <Badge variant="outline" className="text-[10px]">Not installed</Badge>
          </div>

          <Separator />

          <div>
            <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
              Overrides
            </p>
            <p className="text-xs text-muted-foreground/50">
              Min RAM, Max RAM, Resolution, Java, JVM args, Window, GPU, Wrapper — inherit from launcher defaults.
            </p>
          </div>

          <Separator />

          <div>
            <p className="mb-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
              State
            </p>
            <p className="text-xs text-muted-foreground/50">Up to date</p>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between border-t border-border px-5 py-3">
          <Button variant="ghost" size="sm" className="text-muted-foreground">
            <FolderOpen className="mr-1.5 size-3.5" />
            Open folder
          </Button>
          <div className="flex items-center gap-1.5">
            <Button variant="ghost" size="icon-sm" aria-label="Verify">
              <Shield className="size-3.5" />
            </Button>
            <Button variant="ghost" size="icon-sm" aria-label="Repair">
              <Wrench className="size-3.5" />
            </Button>
            <Button variant="ghost" size="icon-sm" className="text-destructive" aria-label="Delete">
              <Trash2 className="size-3.5" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
