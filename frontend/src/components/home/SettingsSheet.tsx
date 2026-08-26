import { useEffect, useState } from 'react';
import { X } from 'lucide-react';
import { Dialog } from '@base-ui/react/dialog';

import { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { JavaRuntimeManager } from '@/components/home/JavaRuntimeManager';
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { SystemService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { LauncherDefaults } from '../../../bindings/plumelauncher/internal/instances/models.js';
import { toast } from '@/components/ui/toast';

type SettingsTab = 'general' | 'java' | 'data' | 'about';

interface SettingsSheetProps {
  isOpen: boolean;
  onClose: () => void;
}

export function SettingsSheet({ isOpen, onClose }: SettingsSheetProps) {
  const [tab, setTab] = useState<SettingsTab>('general');
  const [settings, setSettings] = useState<LauncherDefaults | null>(null);
  const [saved, setSaved] = useState<LauncherDefaults | null>(null);
  const [dataRoot, setDataRoot] = useState('');
  const [appRoot, setAppRoot] = useState('');
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');
  const [dataNotice, setDataNotice] = useState('');
  const [discardOpen, setDiscardOpen] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    setBusy('load');
    setError('');
    setDataNotice('');
    void Promise.all([SystemService.GetSettings(), SystemService.GetDataRoot(), SystemService.GetAppRoot()]).then(([nextSettings, root, app]) => {
      setSettings(nextSettings);
      setSaved(nextSettings);
      setDataRoot(root);
      setAppRoot(app);
    }).catch((err) => setError(err instanceof Error ? err.message : 'Unable to load settings')).finally(() => setBusy(''));
  }, [isOpen]);

  if (!isOpen || !settings) return null;

  const dirty = JSON.stringify(settings) !== JSON.stringify(saved);
  const update = <K extends keyof LauncherDefaults>(key: K, value: LauncherDefaults[K]) => setSettings((current) => current ? { ...current, [key]: value } : current);
  const close = () => dirty ? setDiscardOpen(true) : onClose();

  const save = async () => {
    setBusy('save');
    setError('');
    try {
      await SystemService.UpdateSettings(settings);
      setSaved(settings);
      document.documentElement.classList.toggle('dark', settings.theme !== 'light');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to save settings');
    } finally {
      setBusy('');
    }
  };

  const saveDataRoot = async () => {
    setBusy('data-root');
    setError('');
    setDataNotice('');
    try {
      await SystemService.UpdateDataRoot(dataRoot);
      setDataNotice('Game data root saved. Restart the launcher to apply. Accounts, settings, and logs stay in the app folder.');
      toast.add({ type: 'info', title: 'Restart required', description: 'Game data root will apply after you restart the launcher.' });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to save data root');
    } finally {
      setBusy('');
    }
  };

  return (
    <>
      <Dialog.Root open={isOpen} onOpenChange={(open) => { if (!open) close(); }}>
        <Dialog.Portal>
          <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/40" />
          <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 flex h-[min(680px,90vh)] w-[min(720px,92vw)] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-lg border border-border bg-background shadow-xl">
            <div className="flex items-center justify-between border-b border-border px-5 py-4"><div><Dialog.Title className="text-sm font-semibold">Settings</Dialog.Title>{dirty && <p className="text-xs text-amber-500">Unsaved changes</p>}</div><Button variant="ghost" size="icon-xs" onClick={close} aria-label="Close settings"><X className="size-4" /></Button></div>
            <Tabs value={tab} onValueChange={(value) => setTab(value as SettingsTab)} className="flex min-h-0 flex-1 flex-col">
              <TabsList className="border-b border-border px-5" aria-label="Settings sections"><TabsTrigger value="general">General</TabsTrigger><TabsTrigger value="java">Java</TabsTrigger><TabsTrigger value="data">Data</TabsTrigger><TabsTrigger value="about">About</TabsTrigger></TabsList>
              <div className="min-h-0 flex-1 overflow-auto px-5 py-4 text-sm">
                <TabsContent value="general" className="flex flex-col gap-3"><div className="grid grid-cols-2 gap-3"><label className="flex flex-col gap-1 text-xs">Theme<Select value={settings.theme} onValueChange={(value) => update('theme', value ?? 'dark')}><SelectTrigger aria-label="Theme"><span className="capitalize">{settings.theme}</span></SelectTrigger><SelectContent><SelectItem value="dark">Dark</SelectItem><SelectItem value="light">Light</SelectItem></SelectContent></Select></label><label className="flex flex-col gap-1 text-xs">Close action<Select value={settings.closeAction} onValueChange={(value) => update('closeAction', value ?? 'keep_open')}><SelectTrigger aria-label="Close action"><span>{settings.closeAction}</span></SelectTrigger><SelectContent><SelectItem value="keep_open">Keep open</SelectItem><SelectItem value="minimize">Minimize</SelectItem></SelectContent></Select></label><label className="flex flex-col gap-1 text-xs">Window mode<Select value={settings.windowMode} onValueChange={(value) => update('windowMode', value ?? 'Windowed')}><SelectTrigger aria-label="Window mode"><span>{settings.windowMode}</span></SelectTrigger><SelectContent><SelectItem value="Windowed">Windowed</SelectItem><SelectItem value="Fullscreen">Fullscreen</SelectItem><SelectItem value="Borderless">Borderless</SelectItem></SelectContent></Select></label><label className="flex flex-col gap-1 text-xs">Default width<Input type="number" value={settings.defaultResolutionW} onChange={(event) => update('defaultResolutionW', Number(event.target.value))} /></label><label className="flex flex-col gap-1 text-xs">Default height<Input type="number" value={settings.defaultResolutionH} onChange={(event) => update('defaultResolutionH', Number(event.target.value))} /></label><label className="flex flex-col gap-1 text-xs">Min RAM (MB)<Input type="number" value={settings.defaultMinRamMB} onChange={(event) => update('defaultMinRamMB', Number(event.target.value))} /></label><label className="flex flex-col gap-1 text-xs">Max RAM (MB)<Input type="number" value={settings.defaultMaxRamMB} onChange={(event) => update('defaultMaxRamMB', Number(event.target.value))} /></label></div></TabsContent>
                <TabsContent value="java"><div className="flex flex-col gap-5"><JavaRuntimeManager defaultPath={settings.defaultJavaPath} onDefaultPathChange={(path) => update('defaultJavaPath', path)} onCustomPathAdded={(path) => update('customJavaPaths', (settings.customJavaPaths ?? []).includes(path) ? settings.customJavaPaths : [...(settings.customJavaPaths ?? []), path])} /><div className="grid gap-3 border-t border-border pt-5"><label className="flex flex-col gap-1 text-xs">Default JVM arguments<Input value={settings.defaultJvmArgs} onChange={(event) => update('defaultJvmArgs', event.target.value)} placeholder="Optional JVM arguments" /></label><label className="flex flex-col gap-1 text-xs">Wrapper command<Input value={settings.wrapperCommand} onChange={(event) => update('wrapperCommand', event.target.value)} placeholder="Optional launcher wrapper" /></label><label className="flex flex-col gap-1 text-xs">GPU preference<Select value={settings.gpuPreference} onValueChange={(value) => update('gpuPreference', value ?? 'auto')}><SelectTrigger aria-label="GPU preference"><span className="capitalize">{settings.gpuPreference}</span></SelectTrigger><SelectContent><SelectItem value="auto">Auto</SelectItem><SelectItem value="discrete">Discrete</SelectItem><SelectItem value="integrated">Integrated</SelectItem></SelectContent></Select></label></div></div></TabsContent>
                <TabsContent value="data" className="flex flex-col gap-3">
                  <p className="text-xs text-muted-foreground">Game data root (instances, assets, versions, cache) applies after restart. Existing game data is not moved. Accounts, settings, and logs stay in the app folder.</p>
                  <label className="flex flex-col gap-1 text-xs">Game data root<Input value={dataRoot} onChange={(event) => setDataRoot(event.target.value)} /></label>
                  {appRoot ? <p className="text-[11px] text-muted-foreground">App data (fixed): <span className="font-mono text-foreground/80">{appRoot}</span></p> : null}
                  <div className="flex gap-2">
                    <Button size="sm" disabled={!dataRoot || !!busy} onClick={() => void saveDataRoot()}>{busy === 'data-root' ? 'Saving...' : 'Use after restart'}</Button>
                    <Button variant="secondary" size="sm" disabled={!!busy} onClick={() => void SystemService.OpenLogFolder()}>Open logs</Button>
                  </div>
                  {dataNotice ? <p className="text-xs text-amber-500" role="status">{dataNotice}</p> : null}
                </TabsContent>
                <TabsContent value="about" className="flex flex-col gap-2 text-xs text-muted-foreground"><p className="font-medium text-foreground">Plume Launcher v1.0.0</p><p>Vanilla core with managed Java runtimes.</p><p>Updates are provided through the release page.</p></TabsContent>
                {error && <p role="alert" className="mt-4 text-xs text-destructive">{error}</p>}
              </div>
            </Tabs>
            <div className="flex justify-end border-t border-border px-5 py-3"><Button size="sm" disabled={!dirty || !!busy} onClick={() => void save()}>{busy === 'save' ? 'Saving...' : 'Save settings'}</Button></div>
          </Dialog.Popup>
        </Dialog.Portal>
      </Dialog.Root>
      <AlertDialog open={discardOpen} onOpenChange={setDiscardOpen}><AlertDialogContent><AlertDialogTitle>Discard unsaved settings?</AlertDialogTitle><AlertDialogDescription>Your launcher settings changes will be lost.</AlertDialogDescription><div className="mt-5 flex justify-end gap-2"><Button variant="ghost" onClick={() => setDiscardOpen(false)}>Keep editing</Button><Button variant="destructive" onClick={() => { setDiscardOpen(false); onClose(); }}>Discard</Button></div></AlertDialogContent></AlertDialog>
    </>
  );
}
