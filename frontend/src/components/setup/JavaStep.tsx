import { useEffect, useState } from 'react';
import { Loader2, RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
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
      if (list.length > 0 && !selected) {
        const best = list.find((j) => j.major >= 17) ?? list[0];
        setSelected(best.path);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to scan Java');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { void scan(); }, []);

  const handleNext = async () => {
    if (selected) {
      try {
        const settings = await SystemService.GetSettings();
        await SystemService.UpdateSettings({ ...settings, defaultJavaPath: selected });
      } catch {
        // continue even if save fails; auto-scan will handle it
      }
    }
    onNext();
  };

  return (
    <div className="space-y-6">
      <div className="text-center space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Java Runtime</h1>
        <p className="text-sm text-muted-foreground">Choose which Java to use for Minecraft.</p>
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-muted-foreground">Detected runtimes</span>
          <Button variant="ghost" size="sm" onClick={() => void scan()} disabled={loading}>
            <RefreshCw className={`size-3.5 ${loading ? 'animate-spin' : ''}`} />
          </Button>
        </div>

        {loading ? (
          <div className="flex items-center justify-center gap-2 rounded-lg border border-border p-8 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin" />
            Scanning...
          </div>
        ) : javaList.length === 0 ? (
          <div className="rounded-lg border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
            No Java installations found. You can set one later in Settings.
          </div>
        ) : (
          <div className="space-y-1 max-h-64 overflow-auto rounded-lg border border-border">
            {javaList.map((java) => (
              <button
                key={java.path}
                onClick={() => setSelected(java.path)}
                className={`flex w-full items-center gap-3 px-3 py-2.5 text-left text-sm transition-colors hover:bg-muted ${
                  selected === java.path ? 'bg-primary/10 border-l-2 border-primary' : ''
                }`}
              >
                <div className="flex-1 min-w-0">
                  <p className="truncate font-medium">{java.version}</p>
                  <p className="truncate text-xs text-muted-foreground">{java.path}</p>
                </div>
                {java.source && (
                  <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                    {java.source}
                  </span>
                )}
              </button>
            ))}
          </div>
        )}

        {error && <p className="text-xs text-destructive">{error}</p>}
      </div>

      <div className="flex gap-2">
        <Button variant="outline" onClick={onBack} className="flex-1">Back</Button>
        <Button onClick={handleNext} disabled={loading} className="flex-1">
          {javaList.length === 0 ? 'Skip' : 'Next'}
        </Button>
      </div>
    </div>
  );
}
