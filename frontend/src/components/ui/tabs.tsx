import { Tabs as TabsPrimitive } from '@base-ui/react/tabs';
import { motion } from 'motion/react';
import type { ComponentProps } from 'react';

import { cn } from '@/lib/utils';

const Tabs = TabsPrimitive.Root;

function TabsList({ className, ...props }: ComponentProps<typeof TabsPrimitive.List>) {
  return <TabsPrimitive.List className={cn('relative flex border-b border-border', className)} {...props} />;
}

function TabsTrigger({ className, ...props }: ComponentProps<typeof TabsPrimitive.Tab>) {
  return (
    <TabsPrimitive.Tab
      className={cn(
        'relative z-10 border-b-2 border-transparent px-3 py-2.5 text-xs font-medium capitalize text-muted-foreground outline-none transition-colors hover:text-foreground focus-visible:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 data-active:text-foreground',
        className
      )}
      {...props}
    />
  );
}

function TabsIndicator() {
  return (
    <motion.div
      layoutId="tabs-indicator"
      className="absolute bottom-0 h-[2px] bg-primary"
      transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
    />
  );
}

function TabsContent({ className, ...props }: ComponentProps<typeof TabsPrimitive.Panel>) {
  return <TabsPrimitive.Panel className={cn('outline-none', className)} {...props} />;
}

export { Tabs, TabsContent, TabsIndicator, TabsList, TabsTrigger };
