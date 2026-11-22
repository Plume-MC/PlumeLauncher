import { Switch as SwitchPrimitive } from '@base-ui/react/switch';
import type { ComponentProps } from 'react';

import { cn } from '@/lib/utils';

function Switch({ className, ...props }: ComponentProps<typeof SwitchPrimitive.Root>) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        'relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border border-input bg-muted p-0.5 outline-none transition-[background-color,border-color] duration-150 ease-out group',
        'data-checked:border-primary data-checked:bg-primary',
        'focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/30',
        'data-disabled:cursor-not-allowed data-disabled:opacity-50',
        className
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        className={cn(
          'pointer-events-none block size-3.5 rounded-full bg-background shadow-sm transition-transform duration-150 ease-out',
          'translate-x-0 group-data-checked:translate-x-4'
        )}
      />
    </SwitchPrimitive.Root>
  );
}

export { Switch };
