import { useState } from 'react';
import { TopBar } from '@/components/home/TopBar';
import { ActivityPanel } from '@/components/panels/ActivityPanel';
import { SettingsSheet } from '@/components/home/SettingsSheet';

interface AppShellProps {
  children: React.ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  const [activityOpen, setActivityOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);

  return (
    <div className="flex h-[100dvh] flex-col overflow-hidden bg-background text-foreground">
      <TopBar
        onActivityToggle={() => setActivityOpen((v) => !v)}
        onSettingsClick={() => setSettingsOpen(true)}
      />
      <main className="flex-1 overflow-auto">{children}</main>
      <ActivityPanel isOpen={activityOpen} onClose={() => setActivityOpen(false)} />
      <SettingsSheet isOpen={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </div>
  );
}
