import { useState, useRef, useEffect } from 'react';
import { Events } from '@wailsio/runtime';
import { TopBar } from '@/components/home/TopBar';
import { ActivityPanel } from '@/components/panels/ActivityPanel';
import { SettingsSheet } from '@/components/home/SettingsSheet';
import { AccountDialog } from '@/components/home/AccountDialog';
import type { ActivityLogLine } from '@/components/panels/ActivityPanel';
import type { Account } from '../../bindings/plumelauncher/internal/services/models.js';
import type { DownloadProgressEvent, LaunchStateEvent, LogLineEvent } from '../../bindings/plumelauncher/internal/services/models.js';

interface AppShellProps {
  children: React.ReactNode;
  account?: Account | null;
  accounts?: Account[];
  onAccountsChanged?: () => Promise<void>;
}

export function AppShell({
  children,
  account = null,
  accounts = [],
  onAccountsChanged,
}: AppShellProps) {
  const [activityOpen, setActivityOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [accountsOpen, setAccountsOpen] = useState(false);
  const [downloadActive, setDownloadActive] = useState(false);
  const [launchActive, setLaunchActive] = useState(false);
  const [consoleLines, setConsoleLines] = useState<ActivityLogLine[]>([]);
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
        if (data.state === 'failed' || data.state === 'crashed') setActivityOpen(true);
      }),
      Events.On('log-line', (event) => {
        const data: LogLineEvent = event.data;
        setConsoleLines((lines) => [
          ...lines.slice(-499),
          { level: data.level, message: data.message },
        ]);
      }),
    ];
    return () => unsubs.forEach((unsubscribe) => unsubscribe());
  }, []);

  const activityCount = Number(downloadActive) + Number(launchActive);

  return (
    <div className="flex h-[100dvh] flex-col overflow-hidden bg-background text-foreground">
      <TopBar
        account={account}
        activityCount={activityCount}
        onActivityToggle={() => setActivityOpen((v) => !v)}
        onSettingsClick={() => setSettingsOpen(true)}
        onAccountsClick={() => setAccountsOpen(true)}
      />
      <main className="flex-1 overflow-auto">{children}</main>
      <ActivityPanel
        isOpen={activityOpen}
        onClose={() => {
          setActivityOpen(false);
          dismissedRef.current = true;
        }}
        consoleLines={consoleLines}
        download={download}
        onClearConsole={() => setConsoleLines([])}
      />
      <SettingsSheet isOpen={settingsOpen} onClose={() => setSettingsOpen(false)} />
      <AccountDialog
        open={accountsOpen}
        onOpenChange={setAccountsOpen}
        accounts={accounts}
        onChanged={onAccountsChanged ?? (async () => undefined)}
      />
    </div>
  );
}
