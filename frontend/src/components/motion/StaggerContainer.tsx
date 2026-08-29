import { motion, type Variants } from 'motion/react';
import { useMotionPreference } from './useMotionPreference';
import { cn } from '@/lib/utils';

interface StaggerContainerProps {
  children: React.ReactNode;
  className?: string;
  staggerDelay?: number;
  childDuration?: number;
  childDirection?: 'up' | 'down' | 'left' | 'right' | 'none';
}

export function StaggerContainer({
  children,
  className,
  staggerDelay = 0.04,
  childDuration = 0.3,
  childDirection = 'up',
}: StaggerContainerProps) {
  const { reduced } = useMotionPreference();

  const offsets: Record<string, { x?: number; y?: number }> = {
    up: { y: 14 },
    down: { y: -14 },
    left: { x: 14 },
    right: { x: -14 },
    none: {},
  };

  const offset = offsets[childDirection];

  const container: Variants = {
    hidden: { opacity: 1 },
    visible: {
      opacity: 1,
      transition: {
        staggerChildren: reduced ? 0 : staggerDelay,
        delayChildren: reduced ? 0 : 0.02,
      },
    },
  };

  const child: Variants = {
    hidden: { opacity: 0, ...offset },
    visible: {
      opacity: 1,
      x: 0,
      y: 0,
      transition: {
        duration: reduced ? 0.01 : childDuration,
        ease: [0.16, 1, 0.3, 1],
      },
    },
  };

  return (
    <motion.div
      className={cn(className)}
      variants={container}
      initial="hidden"
      animate="visible"
    >
      {Array.isArray(children)
        ? children.map((c, i) =>
            c ? (
              <motion.div key={i} variants={child}>
                {c}
              </motion.div>
            ) : null
          )
        : children}
    </motion.div>
  );
}
