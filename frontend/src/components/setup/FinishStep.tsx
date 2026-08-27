import { useEffect, useState } from 'react';
import { cn } from '@/lib/utils';
import { useReducedMotion } from '@/hooks/useReducedMotion';

interface FinishStepProps {
  onComplete: () => void;
}

export function FinishStep({ onComplete }: FinishStepProps) {
  const reduced = useReducedMotion();
  const [phase, setPhase] = useState<'anim' | 'done'>(reduced ? 'done' : 'anim');

  useEffect(() => {
    if (reduced) { onComplete(); return; }
    const t1 = setTimeout(() => setPhase('done'), 1200);
    const t2 = setTimeout(onComplete, 2000);
    return () => { clearTimeout(t1); clearTimeout(t2); };
  }, [onComplete, reduced]);

  return (
    <div className="flex flex-col items-center justify-center space-y-6 py-8">
      <div
        className={cn(
          'flex size-16 items-center justify-center rounded-full bg-primary/10',
          !reduced && 'transition-all duration-500',
          !reduced && phase === 'done' && 'scale-110'
        )}
      >
        <span className="text-3xl text-primary">✓</span>
      </div>
      <div className="text-center space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">All Set!</h1>
        <p className="text-sm text-muted-foreground">Launching your launcher...</p>
      </div>
      {!reduced && (
        <div className="flex gap-1.5">
          {[0, 150, 300].map((delay) => (
            <div
              key={delay}
              className="size-1.5 rounded-full bg-primary animate-bounce"
              style={{ animationDelay: `${delay}ms` }}
            />
          ))}
        </div>
      )}
    </div>
  );
}
