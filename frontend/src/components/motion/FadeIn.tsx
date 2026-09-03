import { motion, type Variants } from 'motion/react';
import { useMotionPreference } from './useMotionPreference';
import { cn } from '@/lib/utils';

interface FadeInProps {
  children: React.ReactNode;
  className?: string;
  delay?: number;
  duration?: number;
  direction?: 'up' | 'down' | 'left' | 'right' | 'none';
  distance?: number;
}

const directionOffset: Record<string, { x?: number; y?: number }> = {
  up: { y: 16 },
  down: { y: -16 },
  left: { x: 16 },
  right: { x: -16 },
  none: {},
};

export function FadeIn({
  children,
  className,
  delay = 0,
  duration = 0.35,
  direction = 'up',
  distance,
}: FadeInProps) {
  const { reduced } = useMotionPreference();
  const offset = directionOffset[direction];
  const customDistance = distance ?? (direction === 'none' ? 0 : 16);

  const variants: Variants = {
    hidden: {
      opacity: 0,
      x: offset.x !== undefined ? offset.x * (customDistance / 16) : 0,
      y: offset.y !== undefined ? offset.y * (customDistance / 16) : 0,
    },
    visible: {
      opacity: 1,
      x: 0,
      y: 0,
      transition: {
        duration: reduced ? 0.01 : duration,
        delay: reduced ? 0 : delay,
        ease: [0.16, 1, 0.3, 1],
      },
    },
  };

  return (
    <motion.div
      className={cn(className)}
      initial="hidden"
      animate="visible"
      variants={variants}
    >
      {children}
    </motion.div>
  );
}
