import { useState, useRef } from 'react';
import { useEffect } from 'react';
import { Events } from '@wailsio/runtime';
import { TopBar } from '@/components/home/TopBar';
import { ActivityPanel } from '@/components/panels/ActivityPanel';
import { SettingsSheet } from '@/components/home/SettingsSheet';
import type { DownloadProgressEvent, LaunchStateEvent, LogLineEvent } from '../../bindings/plumelauncher/internal/services/models.js';

interface AppShellProps {
  children: React.ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  const [activityOpen, setActivityOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [downloadActive, setDownloadActive] = useState(false);
  const [launchActive, setLaunchActive] = useState(false);
  const [consoleLines, setConsoleLines] = useState<string[]>([]);
  const [download, setDownload] = useState<DownloadProgressEvent | null>(null);
  const dismissedRef = useRef(false);

  useEffect(() => {
    const unsubs = [
      Events.On('download-progress', (event) => {
        const data: DownloadProgressEvent = event.data;
        setDownload(data);
        const active = !['completed', 'failed', 'cancelled'].includes(data.status);
        setDownloadActive(active);
        if (!active) dismissedRef.current = false;
        if ((active || data.status === 'failed') && !dismissedRef.current) setActivityOpen(true);
      }),
      Events.On('launch-state', (event) => {
        const data: LaunchStateEvent = event.data;
        const active = !['stopped', 'failed', 'crashed'].includes(data.state);
        setLaunchActive(active);
        if (active || data.state === 'failed' || data.state === 'crashed') setActivityOpen(true);
      }),
	  Events.On('log-line', (event) => {
		const data: LogLineEvent = event.data;
		setConsoleLines((lines) => [...lines.slice(-499), `${data.level === 'error' ? '[ERR] ' : ''}${data.message}`]);
	  }),
    ];
    return () => unsubs.forEach((unsubscribe) => unsubscribe());
  }, []);

  const activityCount = Number(downloadActive) + Number(launchActive);

  return (
    <div className="flex h-[100dvh] flex-col overflow-hidden bg-background text-foreground">
      <TopBar
        activityCount={activityCount}
        onActivityToggle={() => setActivityOpen((v) => !v)}
        onSettingsClick={() => setSettingsOpen(true)}
      />
      <main className="flex-1 overflow-auto">{children}</main>
      <ActivityPanel isOpen={activityOpen} onClose={() => { setActivityOpen(false); dismissedRef.current = true; }} activityCount={activityCount} consoleLines={consoleLines} download={download} />
      <SettingsSheet isOpen={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </div>
  );
}
