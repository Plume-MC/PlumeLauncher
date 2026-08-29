import { motion, AnimatePresence } from 'motion/react';
import { useMotionPreference } from './useMotionPreference';
import { cn } from '@/lib/utils';

interface PageTransitionProps {
  children: React.ReactNode;
  className?: string;
  stepKey: string | number;
  direction?: 1 | -1;
}

export function PageTransition({
  children,
  className,
  stepKey,
  direction = 1,
}: PageTransitionProps) {
  const { reduced } = useMotionPreference();
  const x = direction * 24;

  return (
    <AnimatePresence mode="wait" initial={false}>
      <motion.div
        key={stepKey}
        className={cn(className)}
        initial={{ opacity: 0, x: reduced ? 0 : x }}
        animate={{ opacity: 1, x: 0 }}
        exit={{ opacity: 0, x: reduced ? 0 : -x }}
        transition={{
          duration: reduced ? 0.01 : 0.28,
          ease: [0.16, 1, 0.3, 1],
        }}
      >
        {children}
      </motion.div>
    </AnimatePresence>
  );
}
