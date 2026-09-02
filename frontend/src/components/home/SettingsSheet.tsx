import { useEffect, useState } from 'react';
import { IconX, IconSettings, IconCoffee, IconFolderOpen, IconInfoCircle, IconDeviceDesktop, IconDeviceLaptop } from '@tabler/icons-react';
import { Dialog } from '@base-ui/react/dialog';
import { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select';
import { JavaRuntimeManager } from '@/components/home/JavaRuntimeManager';
import { SystemService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { LauncherDefaults } from '../../../bindings/plumelauncher/internal/instances/models.js';
import { toast } from '@/components/ui/toast';
import { cn } from '@/lib/utils';

type SettingsTab = 'general' | 'java' | 'data' | 'about';

interface SettingsSheetProps {
  isOpen: boolean;
  onClose: () => void;
}

const NAV_ITEMS = [
  { id: 'general' as const, label: 'General', icon: IconSettings },
  { id: 'java' as const, label: 'Java', icon: IconCoffee },
  { id: 'data' as const, label: 'Data', icon: IconFolderOpen },
  { id: 'about' as const, label: 'About', icon: IconInfoCircle },
] as const;

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
      toast.add({ type: 'success', title: 'Settings saved' });
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

  const openFolder = async (kind: 'app' | 'game') => {
    setBusy(kind);
    setError('');
    try {
      if (kind === 'app') await SystemService.OpenAppRoot();
      else await SystemService.OpenGameRoot();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to open folder');
    } finally {
      setBusy('');
    }
  };

  return (
    <>
      <Dialog.Root open={isOpen} onOpenChange={(open) => { if (!open) close(); }}>
        <Dialog.Portal>
          <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm transition-[opacity] duration-200 data-ending-style:opacity-0 data-starting-style:opacity-0" />
          <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 flex max-h-[min(90vh,720px)] w-[min(720px,calc(100vw-1.5rem))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-xl bg-background shadow-xl outline-none transition-[opacity,transform] duration-200 ease-[cubic-bezier(0.16,1,0.3,1)] data-ending-style:scale-95 data-ending-style:opacity-0 data-starting-style:scale-95 data-starting-style:opacity-0">
            <div className="flex items-start justify-between border-b border-border px-5 py-4">
              <div>
                <Dialog.Title className="text-sm font-semibold">Settings</Dialog.Title>
                {dirty && <p className="text-xs text-amber-500">Unsaved changes</p>}
              </div>
              <Button variant="ghost" size="icon-xs" onClick={close} aria-label="Close settings"><IconX className="size-4" /></Button>
            </div>

            <div className="flex min-h-0 flex-1">
              <nav className="flex w-[140px] shrink-0 flex-col gap-1 border-r border-border p-2" aria-label="Settings sections">
                {NAV_ITEMS.map((item) => {
                  const Icon = item.icon;
                  const active = tab === item.id;
                  return (
                    <button
                      key={item.id}
                      type="button"
                      onClick={() => setTab(item.id)}
                      className={cn(
                        'flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-xs font-medium transition-colors',
                        active
                          ? 'bg-primary/10 text-primary'
                          : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'
                      )}
                      aria-current={active ? 'page' : undefined}
                    >
                      <Icon className="size-4 shrink-0" />
                      {item.label}
                    </button>
                  );
                })}
              </nav>

              <div className="min-h-0 flex-1 overflow-auto p-5 text-sm">
                {tab === 'general' && (
                  <div className="space-y-4 animate-in fade-in">
                    <div className="rounded-lg bg-card/40 p-4">
                      <h3 className="mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">Appearance</h3>
                      <div className="grid grid-cols-2 gap-3">
                        <label className="flex flex-col gap-1 text-xs">Theme<Select value={settings.theme} onValueChange={(value) => update('theme', value ?? 'dark')}><SelectTrigger aria-label="Theme"><span className="capitalize">{settings.theme}</span></SelectTrigger><SelectContent><SelectItem value="dark">Dark</SelectItem><SelectItem value="light">Light</SelectItem></SelectContent></Select></label>
                        <label className="flex flex-col gap-1 text-xs">Close action<Select value={settings.closeAction} onValueChange={(value) => update('closeAction', value ?? 'keep_open')}><SelectTrigger aria-label="Close action"><span>{settings.closeAction === 'keep_open' ? 'Keep open' : settings.closeAction === 'minimize' ? 'Minimize' : settings.closeAction}</span></SelectTrigger><SelectContent><SelectItem value="keep_open">Keep open</SelectItem><SelectItem value="minimize">Minimize</SelectItem></SelectContent></Select></label>
                      </div>
                    </div>

                    <div className="rounded-lg bg-card/40 p-4">
                      <h3 className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">Global instances</h3>
                      <p className="mb-3 text-[11px] text-muted-foreground">Applied when you create a new instance.</p>
                      <div className="grid grid-cols-2 gap-3">
                        <label className="flex flex-col gap-1 text-xs">Window mode<Select value={settings.windowMode} onValueChange={(value) => update('windowMode', value ?? 'Windowed')}><SelectTrigger aria-label="Window mode"><span>{settings.windowMode}</span></SelectTrigger><SelectContent><SelectItem value="Windowed">Windowed</SelectItem><SelectItem value="Fullscreen">Fullscreen</SelectItem><SelectItem value="Borderless">Borderless</SelectItem></SelectContent></Select></label>
                        <label className="flex flex-col gap-1 text-xs">GPU preference<Select value={settings.gpuPreference} onValueChange={(value) => update('gpuPreference', value ?? 'auto')}><SelectTrigger aria-label="GPU preference"><span className="capitalize">{settings.gpuPreference}</span></SelectTrigger><SelectContent><SelectItem value="auto">Auto</SelectItem><SelectItem value="discrete">Discrete</SelectItem><SelectItem value="integrated">Integrated</SelectItem></SelectContent></Select></label>
                        <label className="flex flex-col gap-1 text-xs">Default width<Input type="number" value={settings.defaultResolutionW} onChange={(event) => update('defaultResolutionW', Number(event.target.value))} /></label>
                        <label className="flex flex-col gap-1 text-xs">Default height<Input type="number" value={settings.defaultResolutionH} onChange={(event) => update('defaultResolutionH', Number(event.target.value))} /></label>
                      </div>
                    </div>

                    <div className="rounded-lg bg-card/40 p-4">
                      <h3 className="mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">Memory</h3>
                      <div className="grid grid-cols-2 gap-3">
                        <label className="flex flex-col gap-1 text-xs">Min RAM (MB)<Input type="number" value={settings.defaultMinRamMB} onChange={(event) => update('defaultMinRamMB', Number(event.target.value))} /></label>
                        <label className="flex flex-col gap-1 text-xs">Max RAM (MB)<Input type="number" value={settings.defaultMaxRamMB} onChange={(event) => update('defaultMaxRamMB', Number(event.target.value))} /></label>
                      </div>
                    </div>
                  </div>
                )}

                {tab === 'java' && (
                  <div className="space-y-4 animate-in fade-in">
                    <JavaRuntimeManager defaultPath={settings.defaultJavaPath} onDefaultPathChange={(path) => update('defaultJavaPath', path)} onCustomPathAdded={(path) => update('customJavaPaths', (settings.customJavaPaths ?? []).includes(path) ? settings.customJavaPaths : [...(settings.customJavaPaths ?? []), path])} />
                    <div className="rounded-lg bg-card/40 p-4">
                      <h3 className="mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">JVM</h3>
                      <div className="space-y-3">
                        <label className="flex flex-col gap-1 text-xs">Default JVM arguments<Input value={settings.defaultJvmArgs} onChange={(event) => update('defaultJvmArgs', event.target.value)} placeholder="e.g. -XX:+UseG1GC" /></label>
                        <label className="flex flex-col gap-1 text-xs">Wrapper command<Input value={settings.wrapperCommand} onChange={(event) => update('wrapperCommand', event.target.value)} placeholder="e.g. gamemoderun" /></label>
                      </div>
                    </div>
                  </div>
                )}

                {tab === 'data' && (
                  <div className="space-y-4 animate-in fade-in">
                    <div className="rounded-lg bg-card/40 p-4">
                      <h3 className="mb-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">Game data root</h3>
                      <p className="mb-3 text-[11px] text-muted-foreground">Instances, assets, versions, cache. Applies after restart. Existing game data is not moved.</p>
                      <label className="flex flex-col gap-1 text-xs">Path<Input value={dataRoot} onChange={(event) => setDataRoot(event.target.value)} /></label>
                    </div>

                    <div className="rounded-lg bg-card/40 p-4">
                      <h3 className="mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">Folders</h3>
                      {appRoot ? <div className="flex items-center justify-between gap-3 text-[11px] text-muted-foreground"><span>App data (fixed): <span className="font-mono text-foreground/80">{appRoot}</span></span><Button size="sm" variant="ghost" disabled={!!busy} onClick={() => void openFolder('app')}>{busy === 'app' ? 'Opening...' : 'Open folder'}</Button></div> : null}
                    </div>

                    <div className="rounded-lg bg-card/40 p-4">
                      <p className="mb-3 text-[11px] text-muted-foreground">Change applies after restart. Game data stays in place.</p>
                      <div className="flex gap-2">
                        <Button size="sm" disabled={!dataRoot || !!busy} onClick={() => void saveDataRoot()}>{busy === 'data-root' ? 'Saving...' : 'Use after restart'}</Button>
                        <Button variant="secondary" size="sm" disabled={!!busy} onClick={() => void openFolder('game')}>{busy === 'game' ? 'Opening...' : 'Open game folder'}</Button>
                        <Button variant="secondary" size="sm" disabled={!!busy} onClick={() => void SystemService.OpenLogFolder()}>Open logs</Button>
                      </div>
                      {dataNotice ? <p className="mt-2 text-xs text-amber-500" role="status">{dataNotice}</p> : null}
                    </div>
                  </div>
                )}

                {tab === 'about' && (
                  <div className="space-y-4 animate-in fade-in">
                    <div className="flex items-center gap-3">
                      <div className="flex size-10 items-center justify-center rounded-lg bg-primary text-xs font-bold text-primary-foreground">P</div>
                      <div>
                        <p className="text-sm font-semibold text-foreground">Plume Launcher</p>
                        <p className="font-mono text-xs text-muted-foreground">v1.0.0</p>
                      </div>
                    </div>

                    <p className="text-sm text-foreground/80">Install, repair, launch. Isolated instances.</p>

                    <div className="grid grid-cols-2 gap-2">
                      <div className="rounded-lg bg-card/40 p-3">
                        <p className="text-[10px] font-medium uppercase tracking-wide text-muted-foreground">Platform</p>
                        <div className="mt-1 flex items-center gap-1.5 text-xs font-medium text-foreground">
                          {navigator.platform.includes('Win') ? <IconDeviceDesktop className="size-3.5" /> : <IconDeviceLaptop className="size-3.5" />}
                          {navigator.platform.includes('Win') ? 'Windows' : navigator.platform.includes('Linux') ? 'Linux' : navigator.platform}
                          <span className="text-muted-foreground">amd64</span>
                        </div>
                      </div>

                      <a
                        href="https://github.com/Plume-MC/PlumeLauncher"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex flex-col justify-center rounded-lg bg-card/40 p-3 transition-colors hover:bg-muted/40"
                      >
                        <p className="text-[10px] font-medium uppercase tracking-wide text-muted-foreground">Source</p>
                        <p className="mt-1 text-xs font-medium text-foreground">GitHub</p>
                      </a>
                    </div>
                  </div>
                )}

                {error && <p role="alert" className="mt-4 text-xs text-destructive">{error}</p>}
              </div>
            </div>

            <div className="flex justify-end border-t border-border px-5 py-3">
              <Button
                size="sm"
                disabled={!dirty || !!busy}
                onClick={() => void save()}
                className="bg-foreground font-semibold text-background hover:bg-foreground/90"
              >
                {busy === 'save' ? 'Saving...' : 'Save settings'}
              </Button>
            </div>
          </Dialog.Popup>
        </Dialog.Portal>
      </Dialog.Root>

      <AlertDialog open={discardOpen} onOpenChange={setDiscardOpen}>
        <AlertDialogContent>
          <AlertDialogTitle>Discard unsaved settings?</AlertDialogTitle>
          <AlertDialogDescription>Your launcher settings changes will be lost.</AlertDialogDescription>
          <div className="mt-5 flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setDiscardOpen(false)}>Keep editing</Button>
            <Button variant="destructive" onClick={() => { setDiscardOpen(false); onClose(); }}>Discard</Button>
          </div>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
