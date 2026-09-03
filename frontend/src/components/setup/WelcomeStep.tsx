import { Button } from '@/components/ui/button';
import { IconStack2, IconShield, IconTerminal2 } from '@tabler/icons-react';
import plumeMark from '@/assets/plume-mark.webp';

interface WelcomeStepProps {
  onNext: () => void;
}

const FACTS = [
  { icon: IconStack2, text: 'Isolated instances with shared assets' },
  { icon: IconTerminal2, text: 'Exact Java major per Minecraft version' },
  { icon: IconShield, text: 'Offline play and Ely.by sessions in the OS keyring' },
] as const;

export function WelcomeStep({ onNext }: WelcomeStepProps) {
  return (
    <div className="space-y-8">
      <div className="space-y-3 text-center">
        <img src={plumeMark} alt="Plume Launcher" className="mx-auto size-14 rounded-xl" />
        <p className="text-[11px] font-medium uppercase tracking-[0.14em] text-muted-foreground">
          Plume Launcher
        </p>
        <h1 className="text-2xl font-semibold tracking-tight text-foreground">
          Set up once. Launch clean.
        </h1>
        <p className="mx-auto max-w-sm text-sm leading-relaxed text-muted-foreground">
          Configure Java and an account. Everything else lives under one data root.
        </p>
      </div>

      <ul className="space-y-2 rounded-lg border border-border bg-card/40 p-3">
        {FACTS.map(({ icon: Icon, text }) => (
          <li key={text} className="flex items-start gap-3 rounded-md px-2 py-2 text-sm">
            <span className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md border border-border bg-muted/50 text-muted-foreground">
              <Icon className="size-3.5" />
            </span>
            <span className="pt-1 text-left text-foreground/90">{text}</span>
          </li>
        ))}
      </ul>

      <Button
        onClick={onNext}
        className="h-11 w-full bg-foreground font-semibold text-background hover:bg-foreground/90 dark:bg-foreground dark:text-background"
      >
        Continue
      </Button>
    </div>
  );
}
