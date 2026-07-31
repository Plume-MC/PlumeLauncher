import { Settings, PanelRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Badge } from '@/components/ui/badge';

interface TopBarProps {
  onActivityToggle?: () => void;
  onSettingsClick?: () => void;
}

export function TopBar({ onActivityToggle, onSettingsClick }: TopBarProps) {
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
          onClick={onActivityToggle}
          aria-label="Toggle activity panel"
        >
          <PanelRight className="size-4" />
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
