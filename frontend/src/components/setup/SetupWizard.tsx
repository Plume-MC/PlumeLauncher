import { useState, useCallback } from 'react';
import { StepIndicator } from './StepIndicator';
import { WelcomeStep } from './WelcomeStep';
import { JavaStep } from './JavaStep';
import { AccountStep } from './AccountStep';
import { FinishStep } from './FinishStep';
import { PageTransition } from '@/components/motion';

interface SetupWizardProps {
  onComplete: () => Promise<void> | void;
}

const TOTAL_STEPS = 4;

export function SetupWizard({ onComplete }: SetupWizardProps) {
  const [step, setStep] = useState(1);
  const [dir, setDir] = useState<1 | -1>(1);

  const next = useCallback(() => {
    setDir(1);
    setStep((s) => Math.min(s + 1, TOTAL_STEPS));
  }, []);

  const back = useCallback(() => {
    setDir(-1);
    setStep((s) => Math.max(s - 1, 1));
  }, []);

  return (
    <main className="flex min-h-[100dvh] flex-col bg-background text-foreground select-none">
      <div className="flex flex-1 flex-col items-center justify-center px-5 py-10">
        <div className="w-full max-w-md">
          <StepIndicator current={step} />
          <PageTransition stepKey={step} direction={dir} className="min-h-[320px]">
            {step === 1 && <WelcomeStep onNext={next} />}
            {step === 2 && <JavaStep onNext={next} onBack={back} />}
            {step === 3 && <AccountStep onNext={next} onBack={back} />}
            {step === 4 && <FinishStep onComplete={onComplete} />}
          </PageTransition>
        </div>
      </div>
    </main>
  );
}
