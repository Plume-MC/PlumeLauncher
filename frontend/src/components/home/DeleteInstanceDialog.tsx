import { Button } from '@/components/ui/button';
import { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { HomeService } from '../../../bindings/plumelauncher/internal/services/index.js';

interface DeleteInstanceDialogProps {
  instanceId: string | null;
  instanceName?: string;
  onOpenChange: (open: boolean) => void;
  onDeleted: () => Promise<void>;
}

export function DeleteInstanceDialog({ instanceId, instanceName, onOpenChange, onDeleted }: DeleteInstanceDialogProps) {
  const open = instanceId !== null;
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const confirm = async () => {
    if (!instanceId) return;
    setBusy(true);
    setError('');
    try {
      await HomeService.DeleteInstance(instanceId);
      onOpenChange(false);
      await onDeleted();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to delete instance');
    } finally {
      setBusy(false);
    }
  };

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogTitle>Delete instance?</AlertDialogTitle>
        <AlertDialogDescription>This permanently removes <strong className="text-foreground">{instanceName}</strong> and its isolated game files.</AlertDialogDescription>
        {error && <p role="alert" className="text-xs text-destructive">{error}</p>}
        <div className="mt-5 flex justify-end gap-2"><Button type="button" variant="ghost" disabled={busy} onClick={() => onOpenChange(false)}>Cancel</Button><Button type="button" variant="destructive" disabled={busy} onClick={() => void confirm()}>{busy ? 'Deleting...' : 'Delete'}</Button></div>
      </AlertDialogContent>
    </AlertDialog>
  );
}
import { useState } from 'react';
