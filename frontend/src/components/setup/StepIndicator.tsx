import { IconBox, IconCoffee, IconUser, IconCircleCheck } from '@tabler/icons-react';
import { cn } from '@/lib/utils';

const STEPS = [
  { id: 1, label: 'Welcome', icon: IconBox },
  { id: 2, label: 'Java', icon: IconCoffee },
  { id: 3, label: 'Account', icon: IconUser },
  { id: 4, label: 'Ready', icon: IconCircleCheck },
] as const;

interface StepIndicatorProps {
  current: number;
}

export function StepIndicator({ current }: StepIndicatorProps) {
  return (
    <div className="mb-8 flex flex-col gap-3" aria-label="Setup progress">
      <div className="flex items-start justify-between gap-1">
        {STEPS.map((s, i) => {
          const Icon = s.icon;
          const active = current === s.id;
          const done = current > s.id;
          return (
            <div key={s.id} className="flex flex-1 items-center gap-1">
              <div className="flex min-w-0 flex-1 flex-col items-center gap-1.5">
                <div
                  className={cn(
                    'flex size-9 items-center justify-center rounded-full border transition-colors duration-200',
                    active && 'border-primary bg-primary text-primary-foreground',
                    done && 'border-primary/30 bg-primary/15 text-primary',
                    !active && !done && 'border-border bg-muted/40 text-muted-foreground'
                  )}
                  aria-current={active ? 'step' : undefined}
                >
                  {done ? <IconCircleCheck className="size-4" /> : <Icon className="size-4" />}
                </div>
                <span
                  className={cn(
                    'max-w-full truncate text-[10px] font-medium',
                    active ? 'text-foreground' : 'text-muted-foreground'
                  )}
                >
                  {s.label}
                </span>
              </div>
              {i < STEPS.length - 1 && (
                <div
                  className={cn(
                    'mb-5 h-px min-w-3 flex-1 rounded-full',
                    current > s.id ? 'bg-primary/50' : 'bg-border'
                  )}
                  aria-hidden
                />
              )}
            </div>
          );
        })}
      </div>
      <div className="h-0.5 overflow-hidden rounded-full bg-muted">
        <div
          className="h-full rounded-full bg-primary transition-[width] duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
          style={{ width: `${(current / STEPS.length) * 100}%` }}
        />
      </div>
    </div>
  );
}
