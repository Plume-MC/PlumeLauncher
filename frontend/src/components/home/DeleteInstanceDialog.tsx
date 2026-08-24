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
  const confirm = async () => {
    if (!instanceId) return;
    await HomeService.DeleteInstance(instanceId);
    onOpenChange(false);
    await onDeleted();
  };

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogTitle>Delete instance?</AlertDialogTitle>
        <AlertDialogDescription>This permanently removes <strong className="text-foreground">{instanceName}</strong> and its isolated game files.</AlertDialogDescription>
        <div className="mt-5 flex justify-end gap-2"><Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>Cancel</Button><Button type="button" variant="destructive" onClick={() => void confirm()}>Delete</Button></div>
      </AlertDialogContent>
    </AlertDialog>
  );
}
