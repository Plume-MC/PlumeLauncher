import { Select as SelectPrimitive } from '@base-ui/react/select';
import { Check, ChevronDown } from 'lucide-react';
import type { ComponentProps } from 'react';

import { cn } from '@/lib/utils';

const Select = SelectPrimitive.Root;

function SelectTrigger({ className, children, ...props }: ComponentProps<typeof SelectPrimitive.Trigger>) {
  return (
    <SelectPrimitive.Trigger
      className={cn(
        'group flex h-8 w-full items-center justify-between gap-2 rounded-lg border border-input bg-background px-2.5 text-xs shadow-[inset_0_1px_2px_rgba(0,0,0,0.04)] outline-none transition-[box-shadow,border-color,transform] hover:border-primary/40 focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/30 active:scale-[0.99] disabled:cursor-not-allowed disabled:opacity-50 dark:shadow-[inset_0_1px_2px_rgba(0,0,0,0.18)]',
        className
      )}
      {...props}
    >
      {children}
      <ChevronDown className="size-3.5 shrink-0 opacity-60 transition-transform duration-200 ease-out group-data-[popup-open]:rotate-180" />
    </SelectPrimitive.Trigger>
  );
}

function SelectContent({ className, children, ...props }: ComponentProps<typeof SelectPrimitive.Popup>) {
  return (
    <SelectPrimitive.Portal>
      <SelectPrimitive.Positioner className="z-50 outline-none" sideOffset={4} alignItemWithTrigger={false}>
        <SelectPrimitive.Popup
          className={cn(
            'box-border max-h-48 w-[var(--anchor-width)] min-w-[var(--anchor-width)] origin-top overflow-y-auto overscroll-contain rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-lg shadow-black/8 outline-none backdrop-blur-xl transition-[opacity,transform] duration-150 ease-out data-ending-style:scale-95 data-ending-style:opacity-0 data-starting-style:scale-95 data-starting-style:opacity-0 dark:shadow-black/20',
            className
          )}
          {...props}
        >
          <SelectPrimitive.List>
            <SelectPrimitive.Group>{children}</SelectPrimitive.Group>
          </SelectPrimitive.List>
        </SelectPrimitive.Popup>
      </SelectPrimitive.Positioner>
    </SelectPrimitive.Portal>
  );
}

function SelectItem({ className, children, ...props }: ComponentProps<typeof SelectPrimitive.Item>) {
  return (
    <SelectPrimitive.Item
      className={cn(
        'relative flex w-full cursor-default select-none items-center rounded-md py-1.5 pl-7 pr-2 text-xs outline-none transition-colors data-highlighted:bg-accent data-highlighted:text-accent-foreground',
        className
      )}
      {...props}
    >
      <span className="absolute left-2 flex size-3.5 items-center justify-center">
        <SelectPrimitive.ItemIndicator>
          <Check className="size-3.5" />
        </SelectPrimitive.ItemIndicator>
      </span>
      <SelectPrimitive.ItemText className="truncate">{children}</SelectPrimitive.ItemText>
    </SelectPrimitive.Item>
  );
}

function SelectLabel({ className, ...props }: ComponentProps<typeof SelectPrimitive.GroupLabel>) {
  return <SelectPrimitive.GroupLabel className={cn('px-2 py-1.5 text-xs font-medium text-muted-foreground', className)} {...props} />;
}

export { Select, SelectContent, SelectItem, SelectLabel, SelectTrigger };
