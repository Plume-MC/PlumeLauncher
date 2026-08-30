import { AlertDialog as AlertDialogPrimitive } from '@base-ui/react/alert-dialog';
import { motion, AnimatePresence } from 'motion/react';
import type { ComponentProps } from 'react';

import { cn } from '@/lib/utils';

const AlertDialog = AlertDialogPrimitive.Root;

function AlertDialogContent({ className, ...props }: ComponentProps<typeof AlertDialogPrimitive.Popup>) {
  return (
    <AlertDialogPrimitive.Portal>
      <AlertDialogPrimitive.Backdrop className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm" />
      <AnimatePresence>
        <motion.div
          initial={{ opacity: 0, scale: 0.96, y: 8 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.96, y: 8 }}
          transition={{ duration: 0.2, ease: [0.16, 1, 0.3, 1] }}
          className="fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2"
        >
          <AlertDialogPrimitive.Popup
            className={cn(
              'rounded-xl border border-border bg-background/95 p-5 shadow-xl shadow-black/10 outline-none backdrop-blur-xl dark:bg-card/95 dark:shadow-black/25',
              className
            )}
            {...props}
          />
        </motion.div>
      </AnimatePresence>
    </AlertDialogPrimitive.Portal>
  );
}

function AlertDialogTitle({ className, ...props }: ComponentProps<typeof AlertDialogPrimitive.Title>) {
  return <AlertDialogPrimitive.Title className={cn('text-base font-semibold', className)} {...props} />;
}

function AlertDialogDescription({ className, ...props }: ComponentProps<typeof AlertDialogPrimitive.Description>) {
  return <AlertDialogPrimitive.Description className={cn('mt-2 text-sm text-muted-foreground', className)} {...props} />;
}

export { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogTitle };
