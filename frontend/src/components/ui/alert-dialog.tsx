import { AlertDialog as AlertDialogPrimitive } from '@base-ui/react/alert-dialog';
import type { ComponentProps } from 'react';

import { cn } from '@/lib/utils';

const AlertDialog = AlertDialogPrimitive.Root;

function AlertDialogContent({ className, ...props }: ComponentProps<typeof AlertDialogPrimitive.Popup>) {
  return <AlertDialogPrimitive.Portal><AlertDialogPrimitive.Backdrop className="fixed inset-0 z-40 bg-black/50" /><AlertDialogPrimitive.Popup className={cn('fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-background p-5 shadow-xl outline-none', className)} {...props} /></AlertDialogPrimitive.Portal>;
}

function AlertDialogTitle({ className, ...props }: ComponentProps<typeof AlertDialogPrimitive.Title>) {
  return <AlertDialogPrimitive.Title className={cn('text-base font-semibold', className)} {...props} />;
}

function AlertDialogDescription({ className, ...props }: ComponentProps<typeof AlertDialogPrimitive.Description>) {
  return <AlertDialogPrimitive.Description className={cn('mt-2 text-sm text-muted-foreground', className)} {...props} />;
}

export { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogTitle };
