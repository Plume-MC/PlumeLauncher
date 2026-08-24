import { Tabs as TabsPrimitive } from '@base-ui/react/tabs';
import type { ComponentProps } from 'react';

import { cn } from '@/lib/utils';

const Tabs = TabsPrimitive.Root;

function TabsList({ className, ...props }: ComponentProps<typeof TabsPrimitive.List>) {
  return <TabsPrimitive.List className={cn('flex border-b border-border', className)} {...props} />;
}

function TabsTrigger({ className, ...props }: ComponentProps<typeof TabsPrimitive.Tab>) {
  return <TabsPrimitive.Tab className={cn('border-b-2 border-transparent px-3 py-2.5 text-xs font-medium capitalize text-muted-foreground outline-none transition-colors hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50 data-active:border-primary data-active:text-foreground', className)} {...props} />;
}

function TabsContent({ className, ...props }: ComponentProps<typeof TabsPrimitive.Panel>) {
  return <TabsPrimitive.Panel className={cn('outline-none', className)} {...props} />;
}

export { Tabs, TabsContent, TabsList, TabsTrigger };
