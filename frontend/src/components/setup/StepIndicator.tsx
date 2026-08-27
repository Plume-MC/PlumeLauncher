import { cn } from '@/lib/utils';

interface StepIndicatorProps {
  current: number;
  total: number;
}

export function StepIndicator({ current, total }: StepIndicatorProps) {
  return (
    <div className="flex items-center justify-center gap-2 mb-8">
      {Array.from({ length: total }, (_, i) => (
        <div key={i} className="flex items-center gap-2">
          <div
            className={cn(
              'flex size-8 items-center justify-center rounded-full text-xs font-medium transition-all duration-300',
              i + 1 === current
                ? 'bg-primary text-primary-foreground scale-110 shadow-lg shadow-primary/20'
                : i + 1 < current
                  ? 'bg-primary/20 text-primary'
                  : 'bg-muted text-muted-foreground'
            )}
          >
            {i + 1 < current ? '✓' : i + 1}
          </div>
          {i < total - 1 && (
            <div
              className={cn(
                'h-0.5 w-8 rounded-full transition-colors duration-300',
                i + 1 < current ? 'bg-primary/40' : 'bg-muted'
              )}
            />
          )}
        </div>
      ))}
    </div>
  );
}
