import { Select as SelectPrimitive } from '@base-ui/react/select';
import { Check, ChevronDown } from 'lucide-react';
import type { ComponentProps } from 'react';

import { cn } from '@/lib/utils';

const Select = SelectPrimitive.Root;

function SelectTrigger({ className, children, ...props }: ComponentProps<typeof SelectPrimitive.Trigger>) {
  return <SelectPrimitive.Trigger className={cn('flex h-8 w-full items-center justify-between gap-2 rounded-md border border-border bg-background px-2 text-xs outline-none focus-visible:ring-2 focus-visible:ring-ring/50', className)} {...props}>{children}<ChevronDown className="size-3.5 opacity-60" /></SelectPrimitive.Trigger>;
}

function SelectContent({ className, children, ...props }: ComponentProps<typeof SelectPrimitive.Popup>) {
  return <SelectPrimitive.Portal><SelectPrimitive.Positioner className="z-50" sideOffset={4}><SelectPrimitive.Popup className={cn('max-h-72 min-w-32 overflow-auto rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-lg outline-none', className)} {...props}><SelectPrimitive.List><SelectPrimitive.Group>{children}</SelectPrimitive.Group></SelectPrimitive.List></SelectPrimitive.Popup></SelectPrimitive.Positioner></SelectPrimitive.Portal>;
}

function SelectItem({ className, children, ...props }: ComponentProps<typeof SelectPrimitive.Item>) {
  return <SelectPrimitive.Item className={cn('relative flex cursor-default select-none items-center rounded-sm py-1.5 pl-7 pr-2 text-xs outline-none data-highlighted:bg-accent data-highlighted:text-accent-foreground', className)} {...props}><span className="absolute left-2 flex size-3.5 items-center justify-center"><SelectPrimitive.ItemIndicator><Check className="size-3.5" /></SelectPrimitive.ItemIndicator></span><SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText></SelectPrimitive.Item>;
}

function SelectLabel({ className, ...props }: ComponentProps<typeof SelectPrimitive.GroupLabel>) {
  return <SelectPrimitive.GroupLabel className={cn('px-2 py-1.5 text-xs font-medium text-muted-foreground', className)} {...props} />;
}

export { Select, SelectContent, SelectItem, SelectLabel, SelectTrigger };
