import { useEffect } from 'react';
import { IconCheck } from '@tabler/icons-react';
import { FadeIn, useMotionPreference } from '@/components/motion';

interface FinishStepProps {
  onComplete: () => void;
}

export function FinishStep({ onComplete }: FinishStepProps) {
  const { reduced } = useMotionPreference();

  useEffect(() => {
    const delay = reduced ? 0 : 900;
    const t = setTimeout(onComplete, delay);
    return () => clearTimeout(t);
  }, [onComplete, reduced]);

  return (
    <FadeIn className="flex flex-col items-center justify-center space-y-5 py-6 text-center" direction="up" duration={0.35}>
      <div className="flex size-14 items-center justify-center rounded-full border border-primary/25 bg-primary/10 text-primary">
        <IconCheck className="size-7" stroke={2.25} />
      </div>
      <div className="space-y-1.5">
        <h1 className="text-xl font-semibold tracking-tight">Ready</h1>
        <p className="text-sm text-muted-foreground">Opening the instance library.</p>
      </div>
      <div className="h-0.5 w-24 overflow-hidden rounded-full bg-muted">
        <div
          className="h-full w-full origin-left rounded-full bg-primary"
          style={
            reduced
              ? undefined
              : { animation: 'setup-progress 0.9s cubic-bezier(0.16, 1, 0.3, 1) forwards' }
          }
        />
      </div>
    </FadeIn>
  );
}
