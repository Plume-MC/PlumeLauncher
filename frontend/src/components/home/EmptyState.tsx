import { IconPlus } from '@tabler/icons-react';
import { motion } from 'motion/react';
import { Button } from '@/components/ui/button';

interface EmptyStateProps {
  reduced: boolean;
  onCreate: () => void;
}

export function EmptyState({ reduced, onCreate }: EmptyStateProps) {
  return (
    <motion.div
      initial={reduced ? false : { opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
      className="flex flex-1 flex-col items-center justify-center rounded-xl border border-dashed border-border bg-card/30 px-6 py-16 text-center"
      data-slot="instance-empty-state"
    >
      <svg
        width="64"
        height="64"
        viewBox="0 0 64 64"
        fill="none"
        aria-hidden="true"
        className="mb-5 text-muted-foreground/70"
      >
        <rect x="6" y="10" width="40" height="40" rx="6" fill="currentColor" opacity="0.18" />
        <rect x="14" y="18" width="40" height="40" rx="6" fill="currentColor" opacity="0.32" />
        <rect x="22" y="26" width="34" height="30" rx="5" stroke="currentColor" strokeWidth="1.5" />
        <path d="M22 38 H56 M30 32 V44" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      </svg>
      <h2 className="text-base font-semibold tracking-tight text-foreground">No instances yet</h2>
      <p className="mt-2 max-w-sm text-sm text-muted-foreground">
        Build your first install (vanilla, fabric, or quilt). Files stay isolated under your data root.
      </p>
      <Button
        size="sm"
        className="mt-5 gap-1.5 bg-foreground font-semibold text-background hover:bg-foreground/90"
        onClick={onCreate}
      >
        <IconPlus className="size-3.5" />
        Create instance
      </Button>
    </motion.div>
  );
}
