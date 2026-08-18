import { useEffect, useState } from 'react';
import { Dialog } from '@base-ui/react/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
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
  const [error, setError] = useState('');

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError('');
    try {
      await HomeService.CreateInstance(name.trim(), version, loader);
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
    void HomeService.SupportedVersions(loader).then((supported) => {
      if (cancelled) return;
      setVersions(supported);
      setVersion((current) => supported.includes(current) ? current : supported[0] ?? '');
    }).catch((err) => {
      if (!cancelled) setError(err instanceof Error ? err.message : 'Unable to load supported versions');
    });
    return () => { cancelled = true; };
  }, [loader]);

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
              <div className="space-y-2"><label htmlFor="instance-loader" className="text-sm font-medium">Loader</label><select id="instance-loader" value={loader} onChange={(event) => setLoader(event.target.value)} className="h-9 w-full rounded-md border border-border bg-background px-2 text-sm"><option value="vanilla">Vanilla</option><option value="fabric">Fabric</option><option value="quilt">Quilt</option></select></div>
              <div className="space-y-2"><label htmlFor="instance-version" className="text-sm font-medium">Version</label><select id="instance-version" value={version} onChange={(event) => setVersion(event.target.value)} className="h-9 w-full rounded-md border border-border bg-background px-2 text-sm" disabled={!versions.length}>{versions.map((supported) => <option key={supported} value={supported}>{supported}</option>)}</select></div>
            </div>
            {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
            <div className="flex justify-end gap-2"><Dialog.Close render={<Button type="button" variant="ghost" />}>Cancel</Dialog.Close><Button type="submit" disabled={!name.trim()}>Create</Button></div>
          </form>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
