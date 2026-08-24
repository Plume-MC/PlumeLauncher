import { useEffect, useState } from 'react';
import { Dialog } from '@base-ui/react/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select';
import { HomeService } from '../../../bindings/plumelauncher/internal/services/index.js';

interface CreateInstanceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => Promise<void>;
}

export function CreateInstanceDialog({ open, onOpenChange, onCreated }: CreateInstanceDialogProps) {
  const [name, setName] = useState('');
  const [version, setVersion] = useState('1.21.4');
  const [versions, setVersions] = useState<string[]>([]);
  const [loader, setLoader] = useState('vanilla');
  const [loaderVersion, setLoaderVersion] = useState('');
  const [loaderVersions, setLoaderVersions] = useState<string[]>([]);
  const [error, setError] = useState('');

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError('');
    try {
      await HomeService.CreateInstance(name.trim(), version, loader, loaderVersion);
      setName('');
      onOpenChange(false);
      await onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to create instance');
    }
  };

  useEffect(() => {
    let cancelled = false;
    setError('');
    setVersions([]);
    setVersion('');
    void HomeService.SupportedVersions(loader).then((supported) => {
      if (cancelled) return;
       const nextVersions = supported ?? [];
       setVersions(nextVersions);
       setVersion((current) => nextVersions.includes(current) ? current : nextVersions[0] ?? '');
    }).catch((err) => {
      if (!cancelled) {
        setVersions([]);
        setVersion('');
        setError(err instanceof Error ? err.message : 'Unable to load supported versions');
      }
    });
    return () => { cancelled = true; };
  }, [loader]);

  useEffect(() => {
    let cancelled = false;
    setLoaderVersion('');
    setLoaderVersions([]);
    if (loader === 'vanilla' || !version) return;
    void HomeService.LoaderVersions(loader, version).then((supported) => {
      if (cancelled) return;
       const nextVersions = supported ?? [];
       setLoaderVersions(nextVersions);
       setLoaderVersion(nextVersions[0] ?? '');
    }).catch((err) => {
      if (!cancelled) setError(err instanceof Error ? err.message : 'Unable to load loader versions');
    });
    return () => { cancelled = true; };
  }, [loader, version]);

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/50" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-background p-5 shadow-xl">
          <Dialog.Title className="text-base font-semibold">New instance</Dialog.Title>
          <Dialog.Description className="mt-1 text-sm text-muted-foreground">Create an isolated Minecraft installation.</Dialog.Description>
          <form className="mt-5 space-y-4" onSubmit={submit}>
            <div className="space-y-2"><label htmlFor="instance-name" className="text-sm font-medium">Name</label><Input id="instance-name" value={name} maxLength={48} onChange={(event) => setName(event.target.value)} autoFocus required /></div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2"><label className="text-sm font-medium">Loader</label><Select value={loader} onValueChange={(value) => setLoader(value ?? 'vanilla')}><SelectTrigger aria-label="Loader"><span className="capitalize">{loader}</span></SelectTrigger><SelectContent><SelectItem value="vanilla">Vanilla</SelectItem><SelectItem value="fabric">Fabric</SelectItem><SelectItem value="quilt">Quilt</SelectItem></SelectContent></Select></div>
              <div className="space-y-2"><label className="text-sm font-medium">Minecraft version</label><Select value={version} onValueChange={(value) => setVersion(value ?? '')} disabled={!versions.length}><SelectTrigger aria-label="Minecraft version"><span>{version || 'Select version'}</span></SelectTrigger><SelectContent>{versions.map((supported) => <SelectItem key={supported} value={supported}>{supported}</SelectItem>)}</SelectContent></Select></div>
            </div>
            {loader !== 'vanilla' && <div className="space-y-2"><label className="text-sm font-medium">{loader === 'fabric' ? 'Fabric' : 'Quilt'} loader version</label><Select value={loaderVersion} onValueChange={(value) => setLoaderVersion(value ?? '')} disabled={!loaderVersions.length}><SelectTrigger aria-label="Loader version"><span>{loaderVersion || 'Select loader version'}</span></SelectTrigger><SelectContent>{loaderVersions.map((supported) => <SelectItem key={supported} value={supported}>{supported}</SelectItem>)}</SelectContent></Select></div>}
            {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
            <div className="flex justify-end gap-2"><Dialog.Close render={<Button type="button" variant="ghost" />}>Cancel</Dialog.Close><Button type="submit" disabled={!name.trim() || !version || (loader !== 'vanilla' && !loaderVersion) || !!error}>Create</Button></div>
          </form>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
