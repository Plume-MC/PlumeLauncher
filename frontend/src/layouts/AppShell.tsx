import { lazy, Suspense, useState, useRef, useEffect } from 'react';
import { Events } from '@wailsio/runtime';
import type { ActivityLogLine } from '@/components/panels/ActivityPanel';
import type { Account } from '../../bindings/plumelauncher/internal/services/models.js';
import type { DownloadProgressEvent, LaunchStateEvent, LogLineEvent, JavaDownloadProgressEvent } from '../../bindings/plumelauncher/internal/services/models.js';

const TopBar = lazy(() => import('@/components/home/TopBar').then((module) => ({ default: module.TopBar })));
const ActivityPanel = lazy(() => import('@/components/panels/ActivityPanel').then((module) => ({ default: module.ActivityPanel })));
const SettingsSheet = lazy(() => import('@/components/home/SettingsSheet').then((module) => ({ default: module.SettingsSheet })));
const AccountDialog = lazy(() => import('@/components/home/AccountDialog').then((module) => ({ default: module.AccountDialog })));

function LoadingOverlay() {
  return <div className="sr-only" role="status">Loading...</div>;
}

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
  // Mirrors for the event handler below, which cannot see fresh state.
  // Only one overlay is visible at a time; activity auto-open yields to
  // settings/accounts and surfaces through the TopBar badge instead.
  const settingsOpenRef = useRef(false);
  const accountsOpenRef = useRef(false);
  const setSettings = (open: boolean) => {
    settingsOpenRef.current = open;
    setSettingsOpen(open);
  };
  const setAccounts = (open: boolean) => {
    accountsOpenRef.current = open;
    setAccountsOpen(open);
  };
  const [downloadActive, setDownloadActive] = useState(false);
  const [launchActive, setLaunchActive] = useState(false);
  const [consoleLines, setConsoleLines] = useState<ActivityLogLine[]>([]);
  const [download, setDownload] = useState<DownloadProgressEvent | null>(null);
  const [launch, setLaunch] = useState<LaunchStateEvent | null>(null);
  const [javaDownloads, setJavaDownloads] = useState<JavaDownloadProgressEvent[]>([]);
  const javaDownloadsRef = useRef(javaDownloads);
  javaDownloadsRef.current = javaDownloads;
  const dismissedRef = useRef(false);

  useEffect(() => {
    let frame = 0;
    let nextDownload: DownloadProgressEvent | null = null;
    let nextLaunch: LaunchStateEvent | null = null;
    let nextLines: ActivityLogLine[] = [];
    const flush = () => {
      frame = 0;
      if (nextDownload) {
        const data = nextDownload;
        nextDownload = null;
        setDownload(data);
        const active = !['completed', 'failed', 'cancelled'].includes(data.status);
        setDownloadActive(active);
        if (!active) dismissedRef.current = false;
        if ((active || data.status === 'failed') && !dismissedRef.current && !settingsOpenRef.current && !accountsOpenRef.current) setActivityOpen(true);
      }
      if (nextLaunch) {
        const data = nextLaunch;
        nextLaunch = null;
        setLaunch(data);
        const active = !['stopped', 'failed', 'crashed'].includes(data.state);
        setLaunchActive(active);
        if (active && !dismissedRef.current && !settingsOpenRef.current && !accountsOpenRef.current) setActivityOpen(true);
        if ((data.state === 'failed' || data.state === 'crashed') && !settingsOpenRef.current && !accountsOpenRef.current) setActivityOpen(true);
      }
      if (nextLines.length) {
        const lines = nextLines;
        nextLines = [];
        setConsoleLines((current) => [...current, ...lines].slice(-500));
      }
    };
    const schedule = () => {
      if (!frame) frame = requestAnimationFrame(flush);
    };
    const flushNow = () => {
      if (frame) cancelAnimationFrame(frame);
      flush();
    };
    const unsubs = [
      Events.On('download-progress', (event) => {
        const data: DownloadProgressEvent = event.data;
        nextDownload = data;
        if (['completed', 'failed', 'cancelled'].includes(data.status)) flushNow();
        else schedule();
      }),
      Events.On('launch-state', (event) => {
        const data: LaunchStateEvent = event.data;
        nextLaunch = data;
        if (['stopped', 'failed', 'crashed'].includes(data.state)) flushNow();
        else schedule();
      }),
      Events.On('log-line', (event) => {
        const data: LogLineEvent = event.data;
        nextLines.push({ level: data.level, message: data.message });
        if (data.level === 'error') flushNow();
        else schedule();
      }),
      Events.On('java-download-progress', (event) => {
        const data: JavaDownloadProgressEvent = event.data;
        setJavaDownloads((current) => {
          const existing = current.findIndex((dl) => dl.major === data.major);
          if (existing >= 0) {
            const next = [...current];
            next[existing] = data;
            return next;
          }
          return [...current, data];
        });
        if (['downloading', 'extracting'].includes(data.status) && !dismissedRef.current && !settingsOpenRef.current && !accountsOpenRef.current) {
          setActivityOpen(true);
        }
      }),
    ];
    return () => {
      if (frame) cancelAnimationFrame(frame);
      unsubs.forEach((unsubscribe) => unsubscribe());
    };
  }, []);

  const activityCount = Number(downloadActive) + Number(launchActive);

  return (
    <div className="flex h-[100dvh] flex-col overflow-hidden bg-background text-foreground">
      <Suspense fallback={<LoadingOverlay />}>
        <TopBar
          account={account}
          activityCount={activityCount}
          onActivityToggle={() => {
            setSettings(false);
            setAccounts(false);
            setActivityOpen((v) => !v);
          }}
          onSettingsClick={() => {
            setAccounts(false);
            setSettings(true);
          }}
          onAccountsClick={() => {
            setSettings(false);
            setAccounts(true);
          }}
        />
      </Suspense>
      <main className="flex-1 overflow-auto">{children}</main>
      <Suspense fallback={<LoadingOverlay />}>
        {activityOpen && <ActivityPanel
          isOpen={activityOpen}
          onClose={() => {
            setActivityOpen(false);
            dismissedRef.current = true;
          }}
          consoleLines={consoleLines}
          download={download}
          launch={launch}
          javaDownloads={javaDownloads}
          onClearConsole={() => setConsoleLines([])}
        />}
        {settingsOpen && <SettingsSheet isOpen={settingsOpen} onClose={() => setSettings(false)} />}
        {accountsOpen && <AccountDialog
          open={accountsOpen}
          onOpenChange={setAccounts}
          accounts={accounts}
          onChanged={onAccountsChanged ?? (async () => undefined)}
        />}
      </Suspense>
    </div>
  );
}
