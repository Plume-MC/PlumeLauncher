import { useEffect, useState } from 'react';
import { IconChevronLeft, IconLoader2, IconRefresh } from '@tabler/icons-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { SystemService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { JavaInfo } from '../../../bindings/plumelauncher/internal/java/models.js';

interface JavaStepProps {
  onNext: () => void;
  onBack: () => void;
}

export function JavaStep({ onNext, onBack }: JavaStepProps) {
  const [loading, setLoading] = useState(true);
  const [javaList, setJavaList] = useState<JavaInfo[]>([]);
  const [selected, setSelected] = useState('');
  const [error, setError] = useState('');

  const scan = async () => {
    setLoading(true);
    setError('');
    try {
      const list = (await SystemService.JavaRuntimes()) ?? [];
      setJavaList(list);
      if (list.length > 0) {
        setSelected((current) => {
          if (current && list.some((j) => j.path === current)) return current;
          const best = list.find((j) => j.major >= 17) ?? list[0];
          return best.path;
        });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to scan Java');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void scan();
  }, []);

  const handleNext = async () => {
    if (selected) {
      try {
        const settings = await SystemService.GetSettings();
        await SystemService.UpdateSettings({ ...settings, defaultJavaPath: selected });
      } catch {
        // continue; user can fix in Settings
      }
    }
    onNext();
  };

  return (
    <div className="space-y-6">
      <div className="space-y-1 text-center">
        <h1 className="text-xl font-semibold tracking-tight">Java runtime</h1>
        <p className="text-sm text-muted-foreground">
          Launch uses the exact major required by each Minecraft version.
        </p>
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between gap-2">
          <span className="text-xs font-medium text-muted-foreground">Detected on this machine</span>
          <Button variant="ghost" size="sm" onClick={() => void scan()} disabled={loading} className="h-7 gap-1.5 text-xs">
            <IconRefresh className={cn('size-3', loading && 'animate-spin')} />
            Rescan
          </Button>
        </div>

        {loading ? (
          <div className="flex h-28 items-center justify-center gap-2 rounded-lg border border-border bg-card/30 text-sm text-muted-foreground">
            <IconLoader2 className="size-4 animate-spin" />
            Scanning...
          </div>
        ) : javaList.length === 0 ? (
          <div className="rounded-lg border border-dashed border-border bg-card/20 p-6 text-center text-sm text-muted-foreground">
            No Java found. You can skip and set a path later in Settings.
          </div>
        ) : (
          <div className="max-h-[200px] overflow-auto rounded-lg border border-border bg-card/30">
            {javaList.map((java) => {
              const isSelected = selected === java.path;
              return (
                <button
                  key={java.path}
                  type="button"
                  onClick={() => setSelected(java.path)}
                  className={cn(
                    'flex w-full items-center gap-3 border-b border-border/60 px-3 py-2.5 text-left last:border-0 transition-colors',
                    isSelected ? 'bg-primary/10' : 'hover:bg-muted/50'
                  )}
                >
                  <span
                    className={cn(
                      'size-2 shrink-0 rounded-full',
                      isSelected ? 'bg-primary' : 'bg-muted-foreground/40'
                    )}
                    aria-hidden
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className={cn('text-xs font-semibold', isSelected ? 'text-foreground' : 'text-foreground/80')}>
                        Java {java.version}
                      </span>
                      <span
                        className={cn(
                          'rounded px-1.5 py-0.5 font-mono text-[9px] font-bold',
                          java.major >= 17
                            ? 'bg-success/10 text-success'
                            : java.major >= 11
                              ? 'bg-info/10 text-info'
                              : 'bg-warning/10 text-warning'
                        )}
                      >
                        major {java.major}
                      </span>
                      {java.source ? (
                        <span className="rounded bg-muted px-1.5 py-0.5 text-[9px] font-medium text-muted-foreground">
                          {java.source}
                        </span>
                      ) : null}
                    </div>
                    <p className="mt-0.5 truncate font-mono text-[10px] text-muted-foreground">{java.path}</p>
                  </div>
                </button>
              );
            })}
          </div>
        )}

        {error ? (
          <p role="alert" className="text-xs text-destructive">
            {error}
          </p>
        ) : null}
      </div>

      <div className="flex gap-2">
        <Button variant="outline" onClick={onBack} className="flex-1 gap-1 border-border">
          <IconChevronLeft className="size-4" />
          Back
        </Button>
        <Button
          onClick={() => void handleNext()}
          disabled={loading}
          className="flex-1 bg-foreground font-semibold text-background hover:bg-foreground/90"
        >
          {javaList.length === 0 ? 'Skip' : 'Continue'}
        </Button>
      </div>
    </div>
  );
}
