import { Settings, PanelRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Badge } from '@/components/ui/badge';

interface TopBarProps {
  onActivityToggle?: () => void;
  onSettingsClick?: () => void;
  activityCount?: number;
}

export function TopBar({ onActivityToggle, onSettingsClick, activityCount = 0 }: TopBarProps) {
  return (
    <header
      className="flex h-14 items-center justify-between border-b border-border bg-background px-4"
      data-slot="top-bar"
    >
      <div className="flex items-center gap-3">
        <span className="text-sm font-semibold tracking-tight">Plume Launcher</span>
        <Badge variant="outline" className="text-[10px]">v1.0.0</Badge>
      </div>

      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="icon-sm"
          className="relative"
          onClick={onActivityToggle}
          aria-label="Toggle activity panel"
          aria-pressed={activityCount > 0}
        >
          <PanelRight className="size-4" />
          {activityCount > 0 && <span className="absolute right-1 top-1 grid size-3 place-items-center rounded-full bg-info text-[8px] text-info-foreground" aria-label={`${activityCount} active operation`}>{activityCount}</span>}
        </Button>
        <Separator orientation="vertical" className="h-5 bg-border" />
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={onSettingsClick}
          aria-label="Open settings"
        >
          <Settings className="size-4" />
        </Button>
      </div>
    </header>
  );
}
