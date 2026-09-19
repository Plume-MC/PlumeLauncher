import { IconSettings, IconLayoutSidebarRight, IconUser } from '@tabler/icons-react';
import { motion } from 'motion/react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { cn } from '@/lib/utils';
import plumeMark from '@/assets/plume-mark.webp';
import type { Account } from '../../../bindings/plumelauncher/internal/services/models.js';

interface TopBarProps {
  onActivityToggle?: () => void;
  onSettingsClick?: () => void;
  onAccountsClick?: () => void;
  activityCount?: number;
  account?: Account | null;
}

export function TopBar({
  onActivityToggle,
  onSettingsClick,
  onAccountsClick,
  activityCount = 0,
  account = null,
}: TopBarProps) {
  const name = account?.displayName || account?.username || 'No account';
  const typeLabel = account?.type === 'ely.by' ? 'Ely.by' : account?.type === 'microsoft' ? 'Microsoft' : account ? 'Offline' : 'Sign in';

  return (
    <header
      className="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-border bg-background/95 px-4 backdrop-blur-sm sm:px-5"
      data-slot="top-bar"
    >
      <div className="flex min-w-0 items-center gap-3">
        <img src={plumeMark} alt="" className="size-8 shrink-0 rounded-lg" />
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="truncate text-sm font-semibold tracking-tight">Plume</span>
          </div>
          <p className="hidden text-[10px] text-muted-foreground sm:block">Launcher</p>
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-1.5 sm:gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onAccountsClick}
          className="h-9 gap-2 border-border/80 px-2.5 hover:bg-muted/60"
          aria-label="Open accounts"
        >
          <span
            className={cn(
              'flex size-6 items-center justify-center rounded-full bg-muted',
              account && 'text-primary'
            )}
          >
            <IconUser className="size-3.5" />
          </span>
          <span className="hidden min-w-0 flex-col items-start leading-none sm:flex">
            <span className="text-[10px] font-medium text-muted-foreground">Account</span>
            <span className="max-w-[110px] truncate text-xs font-semibold">{name}</span>
          </span>
          <span className="hidden text-[10px] text-muted-foreground md:inline">({typeLabel})</span>
        </Button>

        <Separator orientation="vertical" className="mx-0.5 hidden h-5 sm:block" />

        <Button
          variant="ghost"
          size="icon-sm"
          className="relative"
          onClick={onActivityToggle}
          aria-label="Toggle activity panel"
          aria-pressed={activityCount > 0}
        >
          <IconLayoutSidebarRight className="size-4" />
          {activityCount > 0 && (
            <motion.span
              initial={{ scale: 0.6, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              className="absolute right-1 top-1 grid size-3 place-items-center rounded-full bg-info font-mono text-[8px] text-info-foreground"
              aria-label={`${activityCount} active operation`}
            >
              {activityCount}
            </motion.span>
          )}
        </Button>
        <Button variant="ghost" size="icon-sm" onClick={onSettingsClick} aria-label="Open settings">
          <IconSettings className="size-4" />
        </Button>
      </div>
    </header>
  );
}
