import { useEffect, useState } from 'react';
import { X, FolderOpen, Trash2 } from 'lucide-react';
import { Dialog } from '@base-ui/react/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import { Badge } from '@/components/ui/badge';
import { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select';
import { InstanceService } from '../../../bindings/plumelauncher/internal/services/index.js';
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
  const [discardOpen, setDiscardOpen] = useState(false);

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
    if (dirty) {
      setDiscardOpen(true);
      return;
    }
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

  const openFolder = async () => {
    setBusy('folder');
    setError('');
    try {
      await InstanceService.OpenInstanceFolder(instance.id);
      await onChanged?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to open instance folder');
    } finally {
      setBusy('');
    }
  };

  return (
    <>
    <Dialog.Root open={isOpen} onOpenChange={(open) => { if (!open) close(); }}>
      <Dialog.Portal>
      <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/40" />
      <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 flex h-[min(650px,90vh)] w-[min(520px,92vw)] -translate-x-1/2 -translate-y-1/2 flex-col rounded-lg border border-border bg-background shadow-xl">
        <div className="flex items-start justify-between border-b border-border px-5 py-4">
          <div>
            <Dialog.Title className="text-sm font-semibold">{instance.name}</Dialog.Title>
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
               <label className="space-y-1 text-xs">Window mode<Select value={draft.windowMode} onValueChange={(value) => update('windowMode', value ?? '')}><SelectTrigger aria-label="Window mode"><span>{draft.windowMode || 'Inherit'}</span></SelectTrigger><SelectContent><SelectItem value="">Inherit</SelectItem><SelectItem value="Windowed">Windowed</SelectItem><SelectItem value="Fullscreen">Fullscreen</SelectItem><SelectItem value="Borderless">Borderless</SelectItem></SelectContent></Select></label>
               <label className="space-y-1 text-xs">GPU<Select value={draft.gpuPreference} onValueChange={(value) => update('gpuPreference', value ?? '')}><SelectTrigger aria-label="GPU preference"><span>{draft.gpuPreference || 'Inherit'}</span></SelectTrigger><SelectContent><SelectItem value="">Inherit</SelectItem><SelectItem value="auto">Auto</SelectItem><SelectItem value="discrete">Discrete</SelectItem><SelectItem value="integrated">Integrated</SelectItem></SelectContent></Select></label>
            </div>
            <label className="block space-y-1 text-xs">Wrapper command<Input value={draft.wrapperCommand} onChange={(e) => update('wrapperCommand', e.target.value)} placeholder="Inherit launcher default" /></label>
          </fieldset>
          {error && <p role="alert" className="text-xs text-destructive">{error}</p>}
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border px-5 py-3">
          <Button variant="ghost" size="sm" disabled={!!busy} onClick={() => void openFolder()}><FolderOpen className="mr-1.5 size-3.5" />Open folder</Button>
          <Button variant="ghost" size="icon-sm" disabled={!!busy} className="text-destructive" aria-label="Delete" onClick={onDelete}><Trash2 className="size-3.5" /></Button>
          <Button size="sm" disabled={!dirty || !!busy} onClick={() => void save()}>{busy === 'save' ? 'Saving...' : 'Save'}</Button>
        </div>
      </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
    <AlertDialog open={discardOpen} onOpenChange={setDiscardOpen}>
      <AlertDialogContent>
        <AlertDialogTitle>Discard unsaved changes?</AlertDialogTitle>
        <AlertDialogDescription>Your instance settings changes will be lost.</AlertDialogDescription>
        <div className="mt-5 flex justify-end gap-2"><Button variant="ghost" onClick={() => setDiscardOpen(false)}>Keep editing</Button><Button variant="destructive" onClick={() => { setDiscardOpen(false); onClose(); }}>Discard</Button></div>
      </AlertDialogContent>
    </AlertDialog>
    </>
  );
}
