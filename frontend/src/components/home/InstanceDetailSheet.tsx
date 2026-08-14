import { useEffect, useState } from 'react';
import { X, FolderOpen, Shield, Wrench, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import { Badge } from '@/components/ui/badge';
import { InstanceService, HomeService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { Instance, Settings } from '../../../bindings/plumelauncher/internal/instances/models.js';

interface InstanceDetailSheetProps {
  isOpen: boolean;
  onClose: () => void;
  instance?: Instance | null;
  onDelete?: () => void;
  onChanged?: () => Promise<void>;
}

type Draft = {
  minRamMB: string;
  maxRamMB: string;
  resolutionW: string;
  resolutionH: string;
  javaPath: string;
  jvmArgs: string;
  windowMode: string;
  gpuPreference: string;
  wrapperCommand: string;
};

const draftFrom = (settings: Settings | undefined): Draft => ({
  minRamMB: settings?.minRamMB?.toString() ?? '',
  maxRamMB: settings?.maxRamMB?.toString() ?? '',
  resolutionW: settings?.resolutionW?.toString() ?? '',
  resolutionH: settings?.resolutionH?.toString() ?? '',
  javaPath: settings?.javaPath ?? '',
  jvmArgs: settings?.jvmArgs ?? '',
  windowMode: settings?.windowMode ?? '',
  gpuPreference: settings?.gpuPreference ?? '',
  wrapperCommand: settings?.wrapperCommand ?? '',
});

const numberOrNull = (value: string) => value.trim() ? Number(value) : null;
const stringOrNull = (value: string) => value.trim() ? value : null;

export function InstanceDetailSheet({ isOpen, onClose, instance, onDelete, onChanged }: InstanceDetailSheetProps) {
  const [draft, setDraft] = useState<Draft>(() => draftFrom(instance?.settings));
  const [saved, setSaved] = useState<Draft>(() => draftFrom(instance?.settings));
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    const next = draftFrom(instance?.settings);
    setDraft(next);
    setSaved(next);
    setError('');
  }, [instance]);

  if (!isOpen || !instance) return null;

  const dirty = JSON.stringify(draft) !== JSON.stringify(saved);
  const update = (key: keyof Draft, value: string) => setDraft((current) => ({ ...current, [key]: value }));
  const close = () => {
    if (dirty && !window.confirm('Discard unsaved instance settings?')) return;
    onClose();
  };

  const save = async () => {
    setBusy('save');
    setError('');
    try {
      await InstanceService.UpdateInstanceSettings(instance.id, {
        minRamMB: numberOrNull(draft.minRamMB), maxRamMB: numberOrNull(draft.maxRamMB),
        resolutionW: numberOrNull(draft.resolutionW), resolutionH: numberOrNull(draft.resolutionH),
        javaPath: stringOrNull(draft.javaPath), jvmArgs: stringOrNull(draft.jvmArgs),
        windowMode: stringOrNull(draft.windowMode), gpuPreference: stringOrNull(draft.gpuPreference),
        wrapperCommand: stringOrNull(draft.wrapperCommand),
      });
      setSaved(draft);
      await onChanged?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to save settings');
    } finally {
      setBusy('');
    }
  };

  const action = async (name: 'folder' | 'verify' | 'repair') => {
    setBusy(name);
    setError('');
    try {
      if (name === 'folder') await InstanceService.OpenInstanceFolder(instance.id);
      if (name === 'verify') await HomeService.VerifyInstance(instance.id);
      if (name === 'repair') await HomeService.RepairInstance(instance.id);
      await onChanged?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : `Unable to ${name} instance`);
    } finally {
      setBusy('');
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" role="dialog" aria-modal="true" aria-labelledby="instance-detail-title">
      <div className="absolute inset-0 bg-black/40" onClick={close} />
      <div className="relative flex h-[min(650px,90vh)] w-[min(520px,92vw)] flex-col rounded-lg border border-border bg-background shadow-xl">
        <div className="flex items-start justify-between border-b border-border px-5 py-4">
          <div>
            <h2 id="instance-detail-title" className="text-sm font-semibold">{instance.name}</h2>
            <p className="mt-0.5 font-mono text-xs text-muted-foreground">MC {instance.mcVersion} · <span className="capitalize">{instance.loader}</span></p>
          </div>
          <Button variant="ghost" size="icon-xs" onClick={close} aria-label="Close detail"><X className="size-4" /></Button>
        </div>
        <div className="flex-1 space-y-4 overflow-auto px-5 py-4">
          <div className="flex items-center justify-between"><Badge variant="outline" className="text-[10px]">{instance.state}</Badge>{dirty && <span className="text-xs font-medium text-amber-500">Unsaved</span>}</div>
          <Separator />
          <fieldset className="space-y-3"><legend className="mb-2 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Overrides</legend>
            <div className="grid grid-cols-2 gap-3">
              <label className="space-y-1 text-xs">Min RAM (MB)<Input value={draft.minRamMB} onChange={(e) => update('minRamMB', e.target.value)} inputMode="numeric" /></label>
              <label className="space-y-1 text-xs">Max RAM (MB)<Input value={draft.maxRamMB} onChange={(e) => update('maxRamMB', e.target.value)} inputMode="numeric" /></label>
              <label className="space-y-1 text-xs">Width<Input value={draft.resolutionW} onChange={(e) => update('resolutionW', e.target.value)} inputMode="numeric" /></label>
              <label className="space-y-1 text-xs">Height<Input value={draft.resolutionH} onChange={(e) => update('resolutionH', e.target.value)} inputMode="numeric" /></label>
            </div>
            <label className="block space-y-1 text-xs">Java path<Input value={draft.javaPath} onChange={(e) => update('javaPath', e.target.value)} placeholder="Inherit launcher default" /></label>
            <label className="block space-y-1 text-xs">JVM arguments<Input value={draft.jvmArgs} onChange={(e) => update('jvmArgs', e.target.value)} placeholder="Inherit launcher default" /></label>
            <div className="grid grid-cols-2 gap-3">
              <label className="space-y-1 text-xs">Window mode<select value={draft.windowMode} onChange={(e) => update('windowMode', e.target.value)} className="h-8 w-full rounded-md border border-border bg-background px-2"><option value="">Inherit</option><option>Windowed</option><option>Fullscreen</option><option>Borderless</option></select></label>
              <label className="space-y-1 text-xs">GPU<select value={draft.gpuPreference} onChange={(e) => update('gpuPreference', e.target.value)} className="h-8 w-full rounded-md border border-border bg-background px-2"><option value="">Inherit</option><option value="auto">Auto</option><option value="discrete">Discrete</option><option value="integrated">Integrated</option></select></label>
            </div>
            <label className="block space-y-1 text-xs">Wrapper command<Input value={draft.wrapperCommand} onChange={(e) => update('wrapperCommand', e.target.value)} placeholder="Inherit launcher default" /></label>
          </fieldset>
          {error && <p role="alert" className="text-xs text-destructive">{error}</p>}
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border px-5 py-3">
          <Button variant="ghost" size="sm" disabled={!!busy} onClick={() => void action('folder')}><FolderOpen className="mr-1.5 size-3.5" />Open folder</Button>
          <div className="flex items-center gap-1.5"><Button variant="ghost" size="icon-sm" disabled={!!busy} onClick={() => void action('verify')} aria-label="Verify"><Shield className="size-3.5" /></Button><Button variant="ghost" size="icon-sm" disabled={!!busy} onClick={() => void action('repair')} aria-label="Repair"><Wrench className="size-3.5" /></Button><Button variant="ghost" size="icon-sm" disabled={!!busy} className="text-destructive" aria-label="Delete" onClick={onDelete}><Trash2 className="size-3.5" /></Button></div>
          <Button size="sm" disabled={!dirty || !!busy} onClick={() => void save()}>{busy === 'save' ? 'Saving...' : 'Save'}</Button>
        </div>
      </div>
    </div>
  );
}
