import { useEffect, useState } from 'react';
import { IconX, IconFolderOpen, IconTrash } from '@tabler/icons-react';
import { Dialog } from '@base-ui/react/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select';
import { cn } from '@/lib/utils';
import { InstanceService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { Instance, Settings } from '../../../bindings/plumelauncher/internal/instances/models.js';
import vanillaLogo from '@/assets/loaders/vanilla.png';
import fabricLogo from '@/assets/loaders/fabric.png';
import quiltLogo from '@/assets/loaders/quilt.png';

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

const loaderArtwork: Record<string, string> = {
  vanilla: vanillaLogo,
  fabric: fabricLogo,
  quilt: quiltLogo,
} as const;

const loaderChip: Record<string, string> = {
  vanilla: 'bg-emerald-500/15 text-emerald-400',
  fabric: 'bg-amber-500/15 text-amber-400',
  quilt: 'bg-primary/15 text-primary',
};

const stateTone: Record<string, string> = {
  ready: 'border-success/30 bg-success/10 text-success',
  running: 'border-info/30 bg-info/10 text-info',
  downloading: 'border-cyan-500/30 bg-cyan-500/10 text-cyan-400',
  planning: 'border-cyan-500/30 bg-cyan-500/10 text-cyan-400',
  verifying: 'border-cyan-500/30 bg-cyan-500/10 text-cyan-400',
  failed: 'border-destructive/30 bg-destructive/10 text-destructive',
  crashed: 'border-destructive/30 bg-destructive/10 text-destructive',
};

const stateLabel = (state: string) => state.replaceAll('_', ' ');

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
  const changedCount = Object.keys(draft).filter((key) => draft[key as keyof Draft] !== saved[key as keyof Draft]).length;
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
          <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/55 backdrop-blur-sm transition-[opacity] duration-200 data-ending-style:opacity-0 data-starting-style:opacity-0" />
          <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 flex h-[min(720px,calc(100vh-2rem))] w-[min(640px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-xl border border-border bg-background shadow-xl outline-none transition-[opacity,transform] duration-200 ease-[cubic-bezier(0.16,1,0.3,1)] data-ending-style:scale-95 data-ending-style:opacity-0 data-starting-style:scale-95 data-starting-style:opacity-0">
            <header className="flex items-start justify-between gap-3 border-b border-border px-5 py-4 sm:px-6">
              <div className="flex min-w-0 items-start gap-3">
                <img src={loaderArtwork[instance.loader] ?? vanillaLogo} alt="" className="size-10 shrink-0 rounded-lg ring-1 ring-border/70" />
                <div className="min-w-0">
                  <Dialog.Title className="truncate text-base font-semibold tracking-tight">{instance.name}</Dialog.Title>
                  <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
                    <span className="rounded-md bg-muted px-1.5 py-0.5 font-mono text-[10px] tabular-nums text-muted-foreground">
                      {instance.mcVersion}
                    </span>
                    <span className={cn('rounded-md px-1.5 py-0.5 text-[10px] font-medium capitalize', loaderChip[instance.loader] ?? loaderChip.vanilla)}>
                      {instance.loader}
                    </span>
                    <span className={cn('rounded-md border px-1.5 py-0.5 text-[10px] font-medium capitalize', stateTone[instance.state] ?? 'border-border bg-muted text-muted-foreground')}>
                      {stateLabel(instance.state)}
                    </span>
                  </div>
                </div>
              </div>
              <Button variant="ghost" size="icon-sm" className="shrink-0" onClick={close} aria-label="Close instance details"><IconX className="size-4" /></Button>
            </header>

            <div className="min-h-0 flex-1 overflow-auto">
              <div className="space-y-8 px-5 py-6 sm:px-6">
                <section aria-labelledby="performance-heading" className="space-y-4">
                  <h3 id="performance-heading" className="text-sm font-semibold">Performance</h3>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <Field label="Min RAM" hint="MB"><Input value={draft.minRamMB} onChange={(event) => update('minRamMB', event.target.value)} inputMode="numeric" placeholder="Default" /></Field>
                    <Field label="Max RAM" hint="MB"><Input value={draft.maxRamMB} onChange={(event) => update('maxRamMB', event.target.value)} inputMode="numeric" placeholder="Default" /></Field>
                  </div>
                </section>

                <section aria-labelledby="runtime-heading" className="space-y-4 border-t border-border pt-6">
                  <h3 id="runtime-heading" className="text-sm font-semibold">Runtime</h3>
                  <Field label="Java path"><Input value={draft.javaPath} onChange={(event) => update('javaPath', event.target.value)} placeholder="Default" /></Field>
                </section>

                <section aria-labelledby="display-heading" className="space-y-4 border-t border-border pt-6">
                  <h3 id="display-heading" className="text-sm font-semibold">Display</h3>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <Field label="Width" hint="px"><Input value={draft.resolutionW} onChange={(event) => update('resolutionW', event.target.value)} inputMode="numeric" placeholder="Default" /></Field>
                    <Field label="Height" hint="px"><Input value={draft.resolutionH} onChange={(event) => update('resolutionH', event.target.value)} inputMode="numeric" placeholder="Default" /></Field>
                    <Field label="Window mode">
                      <Select value={draft.windowMode} onValueChange={(value) => update('windowMode', value ?? '')}>
                        <SelectTrigger aria-label="Window mode"><span>{draft.windowMode || 'Default'}</span></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="">Default</SelectItem>
                          <SelectItem value="Windowed">Windowed</SelectItem>
                          <SelectItem value="Fullscreen">Fullscreen</SelectItem>
                          <SelectItem value="Borderless">Borderless</SelectItem>
                        </SelectContent>
                      </Select>
                    </Field>
                    <Field label="GPU">
                      <Select value={draft.gpuPreference} onValueChange={(value) => update('gpuPreference', value ?? '')}>
                        <SelectTrigger aria-label="GPU preference"><span className="capitalize">{draft.gpuPreference || 'Default'}</span></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="">Default</SelectItem>
                          <SelectItem value="auto">Auto</SelectItem>
                          <SelectItem value="discrete">Discrete</SelectItem>
                          <SelectItem value="integrated">Integrated</SelectItem>
                        </SelectContent>
                      </Select>
                    </Field>
                  </div>
                </section>

                <section aria-labelledby="launch-heading" className="space-y-4 border-t border-border pt-6">
                  <h3 id="launch-heading" className="text-sm font-semibold">Launch</h3>
                  <div className="space-y-4">
                    <Field label="JVM arguments"><Input value={draft.jvmArgs} onChange={(event) => update('jvmArgs', event.target.value)} placeholder="Default" /></Field>
                    <Field label="Wrapper command"><Input value={draft.wrapperCommand} onChange={(event) => update('wrapperCommand', event.target.value)} placeholder="Default" /></Field>
                  </div>
                </section>

                {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
              </div>
            </div>

            <footer className="flex flex-wrap items-center justify-between gap-3 border-t border-border bg-background/95 px-5 py-3 sm:px-6">
              <div className="flex items-center gap-1">
                <Button variant="ghost" size="sm" disabled={!!busy} onClick={() => void openFolder()}>
                  <IconFolderOpen className="mr-1.5 size-3.5" />
                  Open folder
                </Button>
                <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" disabled={!!busy} onClick={onDelete}>
                  <IconTrash className="mr-1.5 size-3.5" />
                  Delete
                </Button>
              </div>
              <div className="flex items-center gap-2">
                {dirty ? (
                  <span className="mr-1 text-xs text-muted-foreground">
                    {changedCount} unsaved
                  </span>
                ) : null}
                <Button variant="ghost" size="sm" disabled={!dirty || !!busy} onClick={() => setDraft(saved)}>Revert</Button>
                <Button size="sm" disabled={!dirty || !!busy} onClick={() => void save()}>{busy === 'save' ? 'Saving...' : 'Save'}</Button>
              </div>
            </footer>
          </Dialog.Popup>
        </Dialog.Portal>
      </Dialog.Root>
      <AlertDialog open={discardOpen} onOpenChange={setDiscardOpen}>
        <AlertDialogContent>
          <AlertDialogTitle>Discard unsaved changes?</AlertDialogTitle>
          <AlertDialogDescription>Your instance settings changes will be lost.</AlertDialogDescription>
          <div className="mt-5 flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setDiscardOpen(false)}>Keep editing</Button>
            <Button variant="destructive" onClick={() => { setDiscardOpen(false); onClose(); }}>Discard</Button>
          </div>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <label className="block space-y-1.5">
      <span className="flex items-center justify-between text-xs font-medium">
        <span>{label}</span>
        {hint ? <span className="text-[10px] font-normal text-muted-foreground">{hint}</span> : null}
      </span>
      {children}
    </label>
  );
}
