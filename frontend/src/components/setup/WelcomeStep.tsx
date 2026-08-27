import { Button } from '@/components/ui/button';

interface WelcomeStepProps {
  onNext: () => void;
}

export function WelcomeStep({ onNext }: WelcomeStepProps) {
  return (
    <div className="space-y-8 text-center">
      <div className="space-y-3">
        <div className="mx-auto flex size-16 items-center justify-center rounded-2xl bg-primary/10">
          <span className="text-2xl font-bold text-primary">P</span>
        </div>
        <h1 className="text-3xl font-bold tracking-tight">Welcome to Plume</h1>
        <p className="mx-auto max-w-sm text-sm text-muted-foreground">
          A lightweight Minecraft launcher. Let&apos;s get you set up in just a few steps.
        </p>
      </div>
      <Button onClick={onNext} className="w-full">
        Start Setup
      </Button>
    </div>
  );
}
