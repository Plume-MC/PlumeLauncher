import { useEffect, useState } from 'react';
import { IconLoader2, IconX } from '@tabler/icons-react';
import { Dialog } from '@base-ui/react/dialog';
import { AnimatePresence, motion } from 'motion/react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select';
import { cn } from '@/lib/utils';
import { HomeService } from '../../../bindings/plumelauncher/internal/services/index.js';
import vanillaLogo from '@/assets/loaders/vanilla.png';
import fabricLogo from '@/assets/loaders/fabric.png';
import quiltLogo from '@/assets/loaders/quilt.png';

interface CreateInstanceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => Promise<void>;
}

const LOADERS = [
  { id: 'vanilla', label: 'Vanilla', src: vanillaLogo },
  { id: 'fabric', label: 'Fabric', src: fabricLogo },
  { id: 'quilt', label: 'Quilt', src: quiltLogo },
] as const;

export function CreateInstanceDialog({ open, onOpenChange, onCreated }: CreateInstanceDialogProps) {
  const [name, setName] = useState('');
  const [version, setVersion] = useState('');
  const [versions, setVersions] = useState<string[]>([]);
  const [loader, setLoader] = useState('vanilla');
  const [loaderVersion, setLoaderVersion] = useState('');
  const [loaderVersions, setLoaderVersions] = useState<string[]>([]);
  const [error, setError] = useState('');
  const [loadingVersions, setLoadingVersions] = useState(false);
  const [loadingLoaders, setLoadingLoaders] = useState(false);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    if (!open) return;
    setName('');
    setLoader('vanilla');
    setError('');
    setLoaderVersion('');
    setLoaderVersions([]);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setError('');
    setVersions([]);
    setVersion('');
    setLoadingVersions(true);
    void HomeService.SupportedVersions(loader)
      .then((supported) => {
        if (cancelled) return;
        const nextVersions = supported ?? [];
        setVersions(nextVersions);
        setVersion(nextVersions[0] ?? '');
      })
      .catch((err) => {
        if (!cancelled) {
          setVersions([]);
          setVersion('');
          setError(err instanceof Error ? err.message : 'Unable to load supported versions');
        }
      })
      .finally(() => {
        if (!cancelled) setLoadingVersions(false);
      });
    return () => {
      cancelled = true;
    };
  }, [loader, open]);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setLoaderVersion('');
    setLoaderVersions([]);
    if (loader === 'vanilla' || !version) {
      setLoadingLoaders(false);
      return;
    }
    setLoadingLoaders(true);
    setError('');
    void HomeService.LoaderVersions(loader, version)
      .then((supported) => {
        if (cancelled) return;
        const nextVersions = supported ?? [];
        setLoaderVersions(nextVersions);
        setLoaderVersion(nextVersions[0] ?? '');
        if (nextVersions.length === 0) {
          setError(`No ${loader} loader versions for ${version}`);
        }
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Unable to load loader versions');
      })
      .finally(() => {
        if (!cancelled) setLoadingLoaders(false);
      });
    return () => {
      cancelled = true;
    };
  }, [loader, version, open]);

  const canSubmit =
    !!name.trim() &&
    !!version &&
    !creating &&
    !loadingVersions &&
    (loader === 'vanilla' || (!!loaderVersion && !loadingLoaders));

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!canSubmit) return;
    setCreating(true);
    setError('');
    try {
      await HomeService.CreateInstance(name.trim(), version, loader, loaderVersion);
      setName('');
      await onCreated();
      onOpenChange(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to create instance');
    } finally {
      setCreating(false);
    }
  };

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <AnimatePresence>
        {open ? (
          <Dialog.Portal>
            <Dialog.Backdrop
              render={
                <motion.div
                  className="fixed inset-0 z-40 bg-black/55 backdrop-blur-sm"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  transition={{ duration: 0.18 }}
                />
              }
            />
            <Dialog.Popup
              render={
                <motion.div
                  className="fixed left-1/2 top-1/2 z-50 flex max-h-[min(90vh,720px)] w-[min(560px,calc(100vw-1.5rem))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-xl border border-border bg-background shadow-xl outline-none"
                  initial={{ opacity: 0, scale: 0.96, y: 8 }}
                  animate={{ opacity: 1, scale: 1, y: 0 }}
                  exit={{ opacity: 0, scale: 0.96, y: 8 }}
                  transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
                />
              }
            >
              <div className="flex items-start justify-between border-b border-border px-5 py-4">
                <div>
                  <Dialog.Title className="text-base font-semibold tracking-tight">New instance</Dialog.Title>
                  <Dialog.Description className="mt-0.5 text-sm text-muted-foreground">
                    Isolated install under your game data root.
                  </Dialog.Description>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Close"
                  onClick={() => onOpenChange(false)}
                >
                  <IconX className="size-4" />
                </Button>
              </div>

              <form className="flex min-h-0 flex-1 flex-col" onSubmit={submit}>
                <div className="flex-1 space-y-5 overflow-auto px-5 py-4">
                  <div className="space-y-2">
                    <label htmlFor="instance-name" className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                      Name
                    </label>
                    <Input
                      id="instance-name"
                      value={name}
                      maxLength={48}
                      onChange={(event) => setName(event.target.value)}
                      placeholder="e.g. Survival 1.21"
                      autoFocus
                      required
                      className="h-10"
                    />
                  </div>

                  <div className="space-y-2">
                    <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Loader</span>
                    <div className="grid grid-cols-3 gap-2">
                      {LOADERS.map((item) => {
                        const active = loader === item.id;
                        return (
                          <motion.button
                            key={item.id}
                            type="button"
                            onClick={() => setLoader(item.id)}
                            whileTap={{ scale: 0.98 }}
                            className={cn(
                              'relative flex h-11 items-center gap-2 rounded-lg border px-2.5 text-left transition-colors',
                              active
                                ? 'border-primary/50 bg-primary/10'
                                : 'border-border bg-card/40 hover:bg-muted/40'
                            )}
                          >
                            {active && (
                              <motion.span
                                layoutId="loader-tile-active"
                                className="absolute inset-0 rounded-lg border border-primary/40"
                                transition={{ type: 'spring', stiffness: 420, damping: 32 }}
                              />
                            )}
                            <img src={item.src} alt="" className="relative size-7 shrink-0 rounded-md object-cover ring-1 ring-border/50" />
                            <span className="relative truncate text-xs font-semibold">{item.label}</span>
                          </motion.button>
                        );
                      })}
                    </div>
                  </div>

                  <div className="space-y-2">
                    <label className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                      Minecraft version
                    </label>
                    <Select
                      value={version}
                      onValueChange={(value) => {
                        setError('');
                        setVersion(value ?? '');
                      }}
                      disabled={loadingVersions || !versions.length}
                    >
                      <SelectTrigger aria-label="Minecraft version" className="h-9 w-full">
                        <span className="min-w-0 flex-1 truncate text-left">
                          {loadingVersions ? 'Loading versions...' : version || 'Select version'}
                        </span>
                      </SelectTrigger>
                      <SelectContent>
                        {versions.map((supported) => (
                          <SelectItem key={supported} value={supported}>
                            {supported}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <AnimatePresence initial={false}>
                    {loader !== 'vanilla' && (
                      <motion.div
                        key="loader-version"
                        initial={{ opacity: 0, height: 0 }}
                        animate={{ opacity: 1, height: 'auto' }}
                        exit={{ opacity: 0, height: 0 }}
                        transition={{ duration: 0.2, ease: [0.16, 1, 0.3, 1] }}
                        className="space-y-2 overflow-hidden"
                      >
                        <label className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                          {loader === 'fabric' ? 'Fabric' : 'Quilt'} loader version
                        </label>
                        <Select
                          value={loaderVersion}
                          onValueChange={(value) => {
                            setError('');
                            setLoaderVersion(value ?? '');
                          }}
                          disabled={loadingLoaders || !loaderVersions.length}
                        >
                          <SelectTrigger aria-label="Loader version" className="h-9 w-full">
                            <span className="min-w-0 flex-1 truncate text-left">
                              {loadingLoaders ? 'Loading loaders...' : loaderVersion || 'Select loader version'}
                            </span>
                          </SelectTrigger>
                          <SelectContent>
                            {loaderVersions.map((supported) => (
                              <SelectItem key={supported} value={supported}>
                                {supported}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </motion.div>
                    )}
                  </AnimatePresence>

                  {error ? (
                    <p role="alert" className="text-sm text-destructive">
                      {error}
                    </p>
                  ) : null}
                </div>

                <div className="flex justify-end gap-2 border-t border-border px-5 py-3">
                  <Button type="button" variant="ghost" onClick={() => onOpenChange(false)} disabled={creating}>
                    Cancel
                  </Button>
                  <Button
                    type="submit"
                    disabled={!canSubmit}
                    className="min-w-24 gap-1.5 bg-foreground font-semibold text-background hover:bg-foreground/90"
                  >
                    {creating ? <IconLoader2 className="size-4 animate-spin" /> : null}
                    {creating ? 'Creating...' : 'Create'}
                  </Button>
                </div>
              </form>
            </Dialog.Popup>
          </Dialog.Portal>
        ) : null}
      </AnimatePresence>
    </Dialog.Root>
  );
}
