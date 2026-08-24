import { useState } from 'react';
import { useEffect } from 'react';
import { Events } from '@wailsio/runtime';
import { TopBar } from '@/components/home/TopBar';
import { ActivityPanel } from '@/components/panels/ActivityPanel';
import { SettingsSheet } from '@/components/home/SettingsSheet';
import type { DownloadProgressEvent, LaunchStateEvent } from '../../bindings/plumelauncher/internal/services/models.js';

interface AppShellProps {
  children: React.ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  const [activityOpen, setActivityOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [activityCount, setActivityCount] = useState(0);

  useEffect(() => {
    const unsubs = [
      Events.On('download-progress', (event) => {
        const data: DownloadProgressEvent = event.data;
        const active = data.status !== 'completed' && data.status !== 'failed' && data.status !== 'cancelled';
        setActivityCount(active ? 1 : 0);
        if (active || data.status === 'failed') setActivityOpen(true);
      }),
      Events.On('launch-state', (event) => {
        const data: LaunchStateEvent = event.data;
        const active = data.state !== 'stopped' && data.state !== 'failed';
        setActivityCount(active ? 1 : 0);
        if (active || data.state === 'failed') setActivityOpen(true);
      }),
    ];
    return () => unsubs.forEach((unsubscribe) => unsubscribe());
  }, []);

  return (
    <div className="flex h-[100dvh] flex-col overflow-hidden bg-background text-foreground">
      <TopBar
        activityCount={activityCount}
        onActivityToggle={() => { setActivityCount(0); setActivityOpen((v) => !v); }}
        onSettingsClick={() => setSettingsOpen(true)}
      />
      <main className="flex-1 overflow-auto">{children}</main>
      <ActivityPanel isOpen={activityOpen} onClose={() => setActivityOpen(false)} activityCount={activityCount} />
      <SettingsSheet isOpen={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </div>
  );
}
