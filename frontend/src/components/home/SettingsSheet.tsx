import { useEffect, useState } from 'react';
import { X, RefreshCw } from 'lucide-react';
import { Dialog } from '@base-ui/react/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { SystemService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { LauncherDefaults } from '../../../bindings/plumelauncher/internal/instances/models.js';
import type { JavaInfo } from '../../../bindings/plumelauncher/internal/java/models.js';

type SettingsTab = 'general' | 'java' | 'data' | 'about';

interface SettingsSheetProps { isOpen: boolean; onClose: () => void; }

export function SettingsSheet({ isOpen, onClose }: SettingsSheetProps) {
  const [tab, setTab] = useState<SettingsTab>('general');
  const [settings, setSettings] = useState<LauncherDefaults | null>(null);
  const [java, setJava] = useState<JavaInfo[]>([]);
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');
  const [saved, setSaved] = useState<LauncherDefaults | null>(null);
  const [dataRoot, setDataRoot] = useState('');
  const [discardOpen, setDiscardOpen] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    setBusy('load');
    setError('');
    Promise.all([SystemService.GetSettings(), SystemService.GetDataRoot()]).then(([value, root]) => { setSettings(value); setSaved(value); setDataRoot(root); }).catch((err) => setError(err instanceof Error ? err.message : 'Unable to load settings')).finally(() => setBusy(''));
  }, [isOpen]);

  if (!isOpen || !settings) return null;

  const dirty = JSON.stringify(settings) !== JSON.stringify(saved);
  const update = <K extends keyof LauncherDefaults>(key: K, value: LauncherDefaults[K]) => setSettings((current) => current ? { ...current, [key]: value } : current);
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
      await SystemService.UpdateSettings(settings);
      setSaved(settings);
      document.documentElement.classList.toggle('dark', settings.theme !== 'light');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to save settings');
    } finally {
      setBusy('');
    }
  };
  const scanJava = async () => {
    setBusy('java');
    try { setJava((await SystemService.ScanJava()) ?? []); } catch (err) { setError(err instanceof Error ? err.message : 'Unable to scan Java'); } finally { setBusy(''); }
  };
  const openLogs = async () => {
    setBusy('logs');
    try { await SystemService.OpenLogFolder(); } catch (err) { setError(err instanceof Error ? err.message : 'Unable to open logs'); } finally { setBusy(''); }
  };
  const saveDataRoot = async () => {
    setBusy('data-root');
    setError('');
    try { await SystemService.UpdateDataRoot(dataRoot); } catch (err) { setError(err instanceof Error ? err.message : 'Unable to save data root'); } finally { setBusy(''); }
  };

  const tabs: SettingsTab[] = ['general', 'java', 'data', 'about'];
  return (
    <>
    <Dialog.Root open={isOpen} onOpenChange={(open) => { if (!open) close(); }}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/40" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 flex h-[min(590px,90vh)] w-[min(620px,92vw)] -translate-x-1/2 -translate-y-1/2 flex-col rounded-lg border border-border bg-background shadow-xl">
        <div className="flex items-center justify-between border-b border-border px-5 py-4"><div><Dialog.Title className="text-sm font-semibold">Settings</Dialog.Title>{dirty && <p className="text-xs text-amber-500">Unsaved changes</p>}</div><Button variant="ghost" size="icon-xs" onClick={close} aria-label="Close settings"><X className="size-4" /></Button></div>
         <Tabs value={tab} onValueChange={(value) => setTab(value as SettingsTab)}><TabsList className="px-5" aria-label="Settings sections">{tabs.map((item) => <TabsTrigger key={item} value={item}>{item}</TabsTrigger>)}</TabsList>
         <div className="flex-1 overflow-auto px-5 py-4 text-sm">
           <TabsContent value="general"><div className="space-y-3"><div className="grid grid-cols-2 gap-3"><label className="space-y-1 text-xs">Theme<Select value={settings.theme} onValueChange={(value) => update('theme', value ?? 'dark')}><SelectTrigger aria-label="Theme"><span className="capitalize">{settings.theme}</span></SelectTrigger><SelectContent><SelectItem value="dark">Dark</SelectItem><SelectItem value="light">Light</SelectItem></SelectContent></Select></label><label className="space-y-1 text-xs">Close action<Select value={settings.closeAction} onValueChange={(value) => update('closeAction', value ?? 'keep_open')}><SelectTrigger aria-label="Close action"><span>{settings.closeAction}</span></SelectTrigger><SelectContent><SelectItem value="keep_open">Keep open</SelectItem><SelectItem value="minimize">Minimize</SelectItem></SelectContent></Select></label><label className="space-y-1 text-xs">Window mode<Select value={settings.windowMode} onValueChange={(value) => update('windowMode', value ?? 'Windowed')}><SelectTrigger aria-label="Window mode"><span>{settings.windowMode}</span></SelectTrigger><SelectContent><SelectItem value="Windowed">Windowed</SelectItem><SelectItem value="Fullscreen">Fullscreen</SelectItem><SelectItem value="Borderless">Borderless</SelectItem></SelectContent></Select></label><label className="space-y-1 text-xs">Default width<Input type="number" value={settings.defaultResolutionW} onChange={(e) => update('defaultResolutionW', Number(e.target.value))} /></label><label className="space-y-1 text-xs">Default height<Input type="number" value={settings.defaultResolutionH} onChange={(e) => update('defaultResolutionH', Number(e.target.value))} /></label><label className="space-y-1 text-xs">Min RAM (MB)<Input type="number" value={settings.defaultMinRamMB} onChange={(e) => update('defaultMinRamMB', Number(e.target.value))} /></label><label className="space-y-1 text-xs">Max RAM (MB)<Input type="number" value={settings.defaultMaxRamMB} onChange={(e) => update('defaultMaxRamMB', Number(e.target.value))} /></label></div></div></TabsContent>
           <TabsContent value="java"><div className="space-y-3"><label className="block space-y-1 text-xs">Default Java path<Input value={settings.defaultJavaPath} onChange={(e) => update('defaultJavaPath', e.target.value)} placeholder="Auto-detect" /></label><label className="block space-y-1 text-xs">Default JVM arguments<Input value={settings.defaultJvmArgs} onChange={(e) => update('defaultJvmArgs', e.target.value)} /></label><label className="block space-y-1 text-xs">Wrapper command<Input value={settings.wrapperCommand} onChange={(e) => update('wrapperCommand', e.target.value)} /></label><label className="block space-y-1 text-xs">GPU preference<Select value={settings.gpuPreference} onValueChange={(value) => update('gpuPreference', value ?? 'auto')}><SelectTrigger aria-label="GPU preference"><span className="capitalize">{settings.gpuPreference}</span></SelectTrigger><SelectContent><SelectItem value="auto">Auto</SelectItem><SelectItem value="discrete">Discrete</SelectItem><SelectItem value="integrated">Integrated</SelectItem></SelectContent></Select></label><Button variant="secondary" size="sm" disabled={!!busy} onClick={() => void scanJava()}><RefreshCw className="mr-1.5 size-3.5" />Scan Java</Button>{java.length > 0 && <ul className="space-y-1 text-xs text-muted-foreground">{java.map((item) => <li key={item.path} className="rounded border border-border px-2 py-1">Java {item.major}: {item.path}</li>)}</ul>}</div></TabsContent>
           <TabsContent value="data"><div className="space-y-3"><p className="text-xs text-muted-foreground">The new data root is applied after restarting the launcher. Existing data is not moved.</p><label className="block space-y-1 text-xs">Data root<Input value={dataRoot} onChange={(e) => setDataRoot(e.target.value)} /></label><div className="flex gap-2"><Button size="sm" disabled={!dataRoot || !!busy} onClick={() => void saveDataRoot()}>{busy === 'data-root' ? 'Saving...' : 'Use after restart'}</Button><Button variant="secondary" size="sm" disabled={!!busy} onClick={() => void openLogs()}>Open logs</Button></div></div></TabsContent>
           <TabsContent value="about"><div className="space-y-2 text-xs text-muted-foreground"><p className="font-medium text-foreground">Plume Launcher v1.0.0</p><p>Windows + Linux · Vanilla core</p><p>Manual updates are provided through the release page.</p></div></TabsContent>
           {error && <p role="alert" className="mt-4 text-xs text-destructive">{error}</p>}
         </div>
         </Tabs>
        <div className="flex justify-end border-t border-border px-5 py-3"><Button size="sm" disabled={!dirty || !!busy} onClick={() => void save()}>{busy === 'save' ? 'Saving...' : 'Save settings'}</Button></div>
         </Dialog.Popup>
       </Dialog.Portal>
     </Dialog.Root>
     <AlertDialog open={discardOpen} onOpenChange={setDiscardOpen}>
       <AlertDialogContent>
         <AlertDialogTitle>Discard unsaved settings?</AlertDialogTitle>
         <AlertDialogDescription>Your launcher settings changes will be lost.</AlertDialogDescription>
         <div className="mt-5 flex justify-end gap-2"><Button variant="ghost" onClick={() => setDiscardOpen(false)}>Keep editing</Button><Button variant="destructive" onClick={() => { setDiscardOpen(false); onClose(); }}>Discard</Button></div>
       </AlertDialogContent>
     </AlertDialog>
    </>
  );
}
