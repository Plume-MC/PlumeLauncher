import { useEffect, useState } from 'react';
import { X, RefreshCw } from 'lucide-react';
import { Dialog } from '@base-ui/react/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
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
    if (dirty && !window.confirm('Discard unsaved launcher settings?')) return;
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
    <Dialog.Root open={isOpen} onOpenChange={(open) => { if (!open) close(); }}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/40" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 flex h-[min(590px,90vh)] w-[min(620px,92vw)] -translate-x-1/2 -translate-y-1/2 flex-col rounded-lg border border-border bg-background shadow-xl">
        <div className="flex items-center justify-between border-b border-border px-5 py-4"><div><Dialog.Title className="text-sm font-semibold">Settings</Dialog.Title>{dirty && <p className="text-xs text-amber-500">Unsaved changes</p>}</div><Button variant="ghost" size="icon-xs" onClick={close} aria-label="Close settings"><X className="size-4" /></Button></div>
        <div className="flex border-b border-border px-5" role="tablist" aria-label="Settings sections">{tabs.map((item) => <button key={item} type="button" role="tab" aria-selected={tab === item} tabIndex={tab === item ? 0 : -1} className={`border-b-2 px-3 py-2.5 text-xs font-medium capitalize transition-colors ${tab === item ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'}`} onClick={() => setTab(item)}>{item}</button>)}</div>
        <div className="flex-1 overflow-auto px-5 py-4 text-sm">
          {tab === 'general' && <div className="space-y-3"><div className="grid grid-cols-2 gap-3"><label className="space-y-1 text-xs">Theme<select value={settings.theme} onChange={(e) => update('theme', e.target.value)} className="h-8 w-full rounded-md border border-border bg-background px-2"><option value="dark">Dark</option><option value="light">Light</option></select></label><label className="space-y-1 text-xs">Close action<select value={settings.closeAction} onChange={(e) => update('closeAction', e.target.value)} className="h-8 w-full rounded-md border border-border bg-background px-2"><option value="keep_open">Keep open</option><option value="minimize">Minimize</option></select></label><label className="space-y-1 text-xs">Window mode<select value={settings.windowMode} onChange={(e) => update('windowMode', e.target.value)} className="h-8 w-full rounded-md border border-border bg-background px-2"><option>Windowed</option><option>Fullscreen</option><option>Borderless</option></select></label><label className="space-y-1 text-xs">Default width<Input type="number" value={settings.defaultResolutionW} onChange={(e) => update('defaultResolutionW', Number(e.target.value))} /></label><label className="space-y-1 text-xs">Default height<Input type="number" value={settings.defaultResolutionH} onChange={(e) => update('defaultResolutionH', Number(e.target.value))} /></label><label className="space-y-1 text-xs">Min RAM (MB)<Input type="number" value={settings.defaultMinRamMB} onChange={(e) => update('defaultMinRamMB', Number(e.target.value))} /></label><label className="space-y-1 text-xs">Max RAM (MB)<Input type="number" value={settings.defaultMaxRamMB} onChange={(e) => update('defaultMaxRamMB', Number(e.target.value))} /></label></div></div>}
          {tab === 'java' && <div className="space-y-3"><label className="block space-y-1 text-xs">Default Java path<Input value={settings.defaultJavaPath} onChange={(e) => update('defaultJavaPath', e.target.value)} placeholder="Auto-detect" /></label><label className="block space-y-1 text-xs">Default JVM arguments<Input value={settings.defaultJvmArgs} onChange={(e) => update('defaultJvmArgs', e.target.value)} /></label><label className="block space-y-1 text-xs">Wrapper command<Input value={settings.wrapperCommand} onChange={(e) => update('wrapperCommand', e.target.value)} /></label><label className="block space-y-1 text-xs">GPU preference<select value={settings.gpuPreference} onChange={(e) => update('gpuPreference', e.target.value)} className="h-8 w-full rounded-md border border-border bg-background px-2"><option value="auto">Auto</option><option value="discrete">Discrete</option><option value="integrated">Integrated</option></select></label><Button variant="secondary" size="sm" disabled={!!busy} onClick={() => void scanJava()}><RefreshCw className="mr-1.5 size-3.5" />Scan Java</Button>{java.length > 0 && <ul className="space-y-1 text-xs text-muted-foreground">{java.map((item) => <li key={item.path} className="rounded border border-border px-2 py-1">Java {item.major}: {item.path}</li>)}</ul>}</div>}
          {tab === 'data' && <div className="space-y-3"><p className="text-xs text-muted-foreground">The new data root is applied after restarting the launcher. Existing data is not moved.</p><label className="block space-y-1 text-xs">Data root<Input value={dataRoot} onChange={(e) => setDataRoot(e.target.value)} /></label><div className="flex gap-2"><Button size="sm" disabled={!dataRoot || !!busy} onClick={() => void saveDataRoot()}>{busy === 'data-root' ? 'Saving...' : 'Use after restart'}</Button><Button variant="secondary" size="sm" disabled={!!busy} onClick={() => void openLogs()}>Open logs</Button></div></div>}
          {tab === 'about' && <div className="space-y-2 text-xs text-muted-foreground"><p className="font-medium text-foreground">Plume Launcher v1.0.0</p><p>Windows + Linux · Vanilla core</p><p>Manual updates are provided through the release page.</p></div>}
          {error && <p role="alert" className="mt-4 text-xs text-destructive">{error}</p>}
        </div>
        <div className="flex justify-end border-t border-border px-5 py-3"><Button size="sm" disabled={!dirty || !!busy} onClick={() => void save()}>{busy === 'save' ? 'Saving...' : 'Save settings'}</Button></div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
