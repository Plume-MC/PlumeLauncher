import { useState, useCallback } from 'react';
import { StepIndicator } from './StepIndicator';
import { WelcomeStep } from './WelcomeStep';
import { JavaStep } from './JavaStep';
import { AccountStep } from './AccountStep';
import { FinishStep } from './FinishStep';

interface SetupWizardProps {
  onComplete: () => void;
}

const TOTAL_STEPS = 4;

export function SetupWizard({ onComplete }: SetupWizardProps) {
  const [step, setStep] = useState(1);

  const next = useCallback(() => setStep((s) => Math.min(s + 1, TOTAL_STEPS)), []);
  const back = useCallback(() => setStep((s) => Math.max(s - 1, 1)), []);

  return (
    <main className="grid min-h-[100dvh] place-items-center bg-background px-5 text-foreground">
      <section className="w-full max-w-md space-y-2 rounded-lg border border-border bg-card p-6 shadow-sm">
        <StepIndicator current={step} total={TOTAL_STEPS} />
        {step === 1 && <WelcomeStep onNext={next} />}
        {step === 2 && <JavaStep onNext={next} onBack={back} />}
        {step === 3 && <AccountStep onNext={next} onBack={back} />}
        {step === 4 && <FinishStep onComplete={onComplete} />}
      </section>
    </main>
  );
}
