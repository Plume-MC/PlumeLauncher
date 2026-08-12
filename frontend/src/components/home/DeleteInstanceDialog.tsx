import { Dialog } from '@base-ui/react/dialog';
import { Button } from '@/components/ui/button';
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
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/50" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-background p-5 shadow-xl">
          <Dialog.Title className="text-base font-semibold">Delete instance?</Dialog.Title>
          <Dialog.Description className="mt-2 text-sm text-muted-foreground">This permanently removes <strong className="text-foreground">{instanceName}</strong> and its isolated game files.</Dialog.Description>
          <div className="mt-5 flex justify-end gap-2"><Dialog.Close render={<Button type="button" variant="ghost" />}>Cancel</Dialog.Close><Button type="button" variant="destructive" onClick={() => void confirm()}>Delete</Button></div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
