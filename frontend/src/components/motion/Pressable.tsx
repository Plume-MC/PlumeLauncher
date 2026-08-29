import { motion } from 'motion/react';
import { useMotionPreference } from './useMotionPreference';
import { cn } from '@/lib/utils';

interface PressableProps {
  children: React.ReactNode;
  className?: string;
  hoverScale?: number;
  pressScale?: number;
  as?: 'div' | 'span' | 'button';
  onClick?: () => void;
}

export function Pressable({
  children,
  className,
  hoverScale = 1.02,
  pressScale = 0.98,
  as = 'div',
  onClick,
}: PressableProps) {
  const { reduced } = useMotionPreference();

  const Component = motion[as] as typeof motion.div;

  return (
    <Component
      className={cn(className)}
      whileHover={reduced ? undefined : { scale: hoverScale }}
      whileTap={reduced ? undefined : { scale: pressScale }}
      transition={{ duration: 0.15, ease: [0.16, 1, 0.3, 1] }}
      onClick={onClick}
    >
      {children}
    </Component>
  );
}
