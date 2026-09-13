import { useCallback, useEffect, useRef, useState } from 'react';
import { IconCheck, IconRefresh } from '@tabler/icons-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { JavaDownloadCard } from '@/components/home/JavaDownloadCard';
import { SystemService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { JavaInfo } from '../../../bindings/plumelauncher/internal/java/models.js';

interface JavaRuntimeManagerProps {
  defaultPath: string;
  onDefaultPathChange: (path: string) => void;
  onCustomPathAdded: (path: string) => void;
}

const SUPPORTED_MAJORS = [25, 21, 17, 8];

export function JavaRuntimeManager({ defaultPath, onDefaultPathChange, onCustomPathAdded }: JavaRuntimeManagerProps) {
  const [runtimes, setRuntimes] = useState<JavaInfo[]>([]);
  const [customPath, setCustomPath] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [managedInstalled, setManagedInstalled] = useState<Record<number, boolean>>({});

  const refresh = async () => {
    setBusy(true);
    setError('');
    try {
      setRuntimes(await SystemService.JavaRuntimes() ?? []);
      const managed = (await SystemService.ListManagedRuntimes()) ?? [];
      const installed: Record<number, boolean> = {};
      for (const rt of managed) {
        installed[rt.major] = rt.installed;
      }
      setManagedInstalled(installed);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to scan Java runtimes');
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => { void refresh(); }, []);

  const refreshRef = useRef(refresh);
  refreshRef.current = refresh;

  const handleDownloadComplete = useCallback(() => {
    void refreshRef.current();
  }, []);

  const addCustom = async () => {
    if (!customPath.trim()) return;
    setBusy(true);
    setError('');
    try {
      const runtime = await SystemService.AddCustomJava(customPath.trim());
      if (runtime) {
        onCustomPathAdded(runtime.path);
        setCustomPath('');
        await refresh();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Java validation failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4">
      <section className="rounded-lg bg-card/40 p-4" aria-labelledby="java-runtimes-title">
        <div className="flex items-center justify-between gap-2">
          <div>
            <h3 id="java-runtimes-title" className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Java runtimes</h3>
            <p className="mt-1 text-[11px] text-muted-foreground">The system default comes from the current PATH.</p>
          </div>
          <Button variant="secondary" size="sm" disabled={busy} onClick={() => void refresh()} className="gap-1.5">
            <IconRefresh className={cn('size-3', busy && 'animate-spin')} />
            {busy ? 'Scanning...' : 'Rescan'}
          </Button>
        </div>

        <div className="mt-3 flex flex-col gap-1.5 rounded-lg border border-border bg-card/30">
          {runtimes.map((runtime) => {
            const isSelected = runtime.path === defaultPath;
            return (
              <div
                key={runtime.path}
                className={cn(
                  'flex items-center gap-3 border-b border-border/50 px-3 py-2.5 last:border-0',
                  isSelected && 'bg-primary/5'
                )}
              >
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-1.5">
                    <span className={cn('text-xs font-semibold', isSelected ? 'text-foreground' : 'text-foreground/80')}>
                      Java {runtime.version}
                    </span>
                    {runtime.source && <Badge variant="outline" className="text-[9px]">{runtime.source}</Badge>}
                    {isSelected && <Badge className="text-[9px]">Selected</Badge>}
                  </div>
                  <p className="mt-0.5 truncate font-mono text-[10px] text-muted-foreground">{runtime.path}</p>
                </div>
                <Button
                  variant={isSelected ? 'ghost' : 'outline'}
                  size="sm"
                  disabled={isSelected}
                  onClick={() => onDefaultPathChange(runtime.path)}
                  className="shrink-0 gap-1"
                >
                  {isSelected ? <><IconCheck className="size-3" /> Selected</> : 'Use as default'}
                </Button>
              </div>
            );
          })}
          {runtimes.length === 0 && !busy && (
            <p className="p-4 text-center text-xs text-muted-foreground">No verified Java runtime found. Add a custom executable path below or download one.</p>
          )}
        </div>

        <div className="mt-3 flex gap-2">
          <Input
            value={customPath}
            onChange={(event) => setCustomPath(event.target.value)}
            placeholder="Custom Java executable path"
            aria-label="Custom Java executable path"
            className="h-8"
          />
          <Button variant="outline" size="sm" disabled={busy || !customPath.trim()} onClick={() => void addCustom()}>
            Validate & Add
          </Button>
        </div>
        {error && <p role="alert" className="text-xs text-destructive">{error}</p>}
      </section>

      <section className="rounded-lg bg-card/40 p-4" aria-labelledby="java-download-title">
        <div>
          <h3 id="java-download-title" className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Download JDK (Temurin)</h3>
          <p className="mt-1 text-[11px] text-muted-foreground">Download official Eclipse Temurin JDK runtimes.</p>
        </div>
        <div className="mt-3 grid grid-cols-2 gap-2">
          {SUPPORTED_MAJORS.map((major) => (
            <JavaDownloadCard
              key={major}
              major={major}
              recommended={major === 25}
              installed={managedInstalled[major] ?? false}
              onDownloadComplete={handleDownloadComplete}
            />
          ))}
        </div>
      </section>
    </div>
  );
}
