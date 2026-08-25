import { useEffect, useState } from 'react';
import { Check, RefreshCw } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { SystemService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { JavaInfo } from '../../../bindings/plumelauncher/internal/java/models.js';

interface JavaRuntimeManagerProps {
  defaultPath: string;
  onDefaultPathChange: (path: string) => void;
  onCustomPathAdded: (path: string) => void;
}

export function JavaRuntimeManager({ defaultPath, onDefaultPathChange, onCustomPathAdded }: JavaRuntimeManagerProps) {
  const [runtimes, setRuntimes] = useState<JavaInfo[]>([]);
  const [customPath, setCustomPath] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const refresh = async () => {
    setBusy(true);
    setError('');
    try {
      setRuntimes(await SystemService.JavaRuntimes() ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to scan Java runtimes');
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => { void refresh(); }, []);

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
    <section className="flex flex-col gap-3" aria-labelledby="java-runtimes-title">
      <div className="flex items-center justify-between gap-2">
        <div><h3 id="java-runtimes-title" className="text-sm font-medium">Java runtimes</h3><p className="text-xs text-muted-foreground">The system default comes from the current PATH.</p></div>
        <Button variant="secondary" size="sm" disabled={busy} onClick={() => void refresh()}><RefreshCw data-icon="inline-start" />{busy ? 'Scanning...' : 'Rescan'}</Button>
      </div>
      <div className="flex flex-col gap-2">
        {runtimes.map((runtime) => <div key={runtime.path} className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2">
          <div className="min-w-0"><div className="flex items-center gap-2"><span className="text-sm font-medium">Java {runtime.version}</span>{runtime.source && <Badge variant="outline">{runtime.source}</Badge>}{runtime.path === defaultPath && <Badge>Selected</Badge>}</div><p className="truncate font-mono text-xs text-muted-foreground">{runtime.path}</p></div>
          <Button variant={runtime.path === defaultPath ? 'secondary' : 'outline'} size="sm" disabled={runtime.path === defaultPath} onClick={() => onDefaultPathChange(runtime.path)}>{runtime.path === defaultPath ? <><Check data-icon="inline-start" />Selected</> : 'Use as default'}</Button>
        </div>)}
        {!busy && runtimes.length === 0 && <p className="rounded-md border border-dashed border-border p-3 text-xs text-muted-foreground">No verified Java runtime found. Add a custom executable path below.</p>}
      </div>
      <div className="flex gap-2"><Input value={customPath} onChange={(event) => setCustomPath(event.target.value)} placeholder="Custom Java executable path" aria-label="Custom Java executable path" /><Button variant="outline" disabled={busy || !customPath.trim()} onClick={() => void addCustom()}>Validate & Add</Button></div>
      {error && <p role="alert" className="text-xs text-destructive">{error}</p>}
    </section>
  );
}
