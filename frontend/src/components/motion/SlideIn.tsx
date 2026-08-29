import { motion, AnimatePresence } from 'motion/react';
import { useMotionPreference } from './useMotionPreference';
import { cn } from '@/lib/utils';

interface SlideInProps {
  children: React.ReactNode;
  className?: string;
  isOpen: boolean;
  direction?: 'top' | 'bottom' | 'left' | 'right';
  duration?: number;
}

const directionOffset: Record<string, { x?: string; y?: string }> = {
  top: { y: '-100%' },
  bottom: { y: '100%' },
  left: { x: '-100%' },
  right: { x: '100%' },
};

export function SlideIn({
  children,
  className,
  isOpen,
  direction = 'right',
  duration = 0.3,
}: SlideInProps) {
  const { reduced } = useMotionPreference();
  const offset = directionOffset[direction];

  return (
    <AnimatePresence>
      {isOpen && (
        <motion.div
          className={cn(className)}
          initial={{ opacity: 0, ...offset }}
          animate={{ opacity: 1, x: 0, y: 0 }}
          exit={{ opacity: 0, ...offset }}
          transition={{
            duration: reduced ? 0.01 : duration,
            ease: [0.16, 1, 0.3, 1],
          }}
        >
          {children}
        </motion.div>
      )}
    </AnimatePresence>
  );
}
