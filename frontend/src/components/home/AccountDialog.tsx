import { useEffect, useRef, useState } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import { IconCheck, IconCloud, IconCopy, IconDeviceGamepad2, IconLoader2, IconLock, IconPlus, IconTrash, IconWorld, IconX } from '@tabler/icons-react';
import { Dialog } from '@base-ui/react/dialog';
import { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { AccountService, SystemService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { Account } from '../../../bindings/plumelauncher/internal/services/models.js';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useMotionPreference } from '@/components/motion/useMotionPreference';
import { cn } from '@/lib/utils';

interface AccountDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  accounts: Account[];
  onChanged: () => Promise<void>;
}

type Provider = 'offline' | 'ely.by' | 'microsoft';
type ElyMethod = 'browser' | 'password';

const providerLabel: Record<Provider, string> = {
  offline: 'Offline',
  'ely.by': 'Ely.by',
  microsoft: 'Microsoft',
};

function providerIcon(type: string, className?: string) {
  if (type === 'microsoft') return <IconCloud className={className} />;
  if (type === 'ely.by') return <IconWorld className={className} />;
  return <IconDeviceGamepad2 className={className} />;
}

function providerLine(account: Account) {
  if (account.type === 'microsoft') return 'Microsoft, sign-in window';
  if (account.type === 'ely.by') return account.oauth ? 'Ely.by, browser sign-in' : 'Ely.by, password sign-in';
  return 'Offline, no sign-in';
}

function initialOf(account: Account) {
  return (account.displayName || account.username || '?').trim().charAt(0).toUpperCase() || '?';
}

// Accounts whose skin lookup failed this session keep the initial tile;
// cleared on dialog close so reopening retries once.
const failedSkins = new Set<string>();

// AvatarTile shows the player's skin head (front face: 8px region at
// 8,8 of the 64x64 texture, scaled with pixelated rendering) or falls
// back to the initial letter while loading or when the lookup failed.
function AvatarTile({ account, skin, active = false, className }: { account: Account; skin?: string; active?: boolean; className?: string }) {
  if (skin) {
    return (
      <span
        aria-hidden="true"
        className={cn('grid shrink-0 place-items-center rounded-md bg-muted [background-position:14.28%_14.28%] [background-size:800%_800%] [image-rendering:pixelated]', className)}
        style={{ backgroundImage: `url("${skin}")` }}
      />
    );
  }
  return (
    <span aria-hidden="true" className={cn('grid shrink-0 place-items-center rounded-md bg-muted font-semibold', active ? 'bg-primary text-primary-foreground' : 'text-foreground', className)}>
      {initialOf(account)}
    </span>
  );
}

function formatSeconds(total: number) {
  const m = Math.floor(total / 60);
  const s = total % 60;
  return `${m}:${String(s).padStart(2, '0')}`;
}

export function AccountDialog({ open, onOpenChange, accounts, onChanged }: AccountDialogProps) {
  const [type, setType] = useState<Provider>('offline');
  const [elyMethod, setElyMethod] = useState<ElyMethod>('browser');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [device, setDevice] = useState<{ deviceCode: string; userCode: string; verificationUri: string; interval: number; browserOpened: boolean; expiresAt: number; totalSeconds: number } | null>(null);
  const [remaining, setRemaining] = useState(0);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [adding, setAdding] = useState(false);
  const [pendingRemove, setPendingRemove] = useState<Account | null>(null);
  const { reduced } = useMotionPreference();
  const pendingLogin = useRef<{ cancel: () => void } | null>(null);
  const deviceTimer = useRef<ReturnType<typeof setInterval> | null>(null);
  // Cancel support for the device-code flow: flowGen invalidates stale flows,
  // deviceReject settles the outer run() promise, finishCancel aborts the
  // in-flight Finish poll, mirroring the Microsoft pendingLogin pattern.
  const flowGen = useRef(0);
  const deviceReject = useRef<(() => void) | null>(null);
  const finishCancel = useRef<(() => void) | null>(null);
  const bandRef = useRef<HTMLFormElement | null>(null);
  const usernameRef = useRef<HTMLInputElement | null>(null);

  const active = accounts.find((account) => account.selected) ?? null;

  const [skins, setSkins] = useState<Record<string, string>>({});

  // Resolve skin heads while the dialog is open. The backend caches results,
  // failures fall back to the initial tile and retry on the next open.
  useEffect(() => {
    if (!open) {
      failedSkins.clear();
      return;
    }
    let cancelled = false;
    for (const account of accounts) {
      if (failedSkins.has(account.uuid)) continue;
      void AccountService.GetAccountSkin(account.uuid)
        .then((dataURL) => {
          if (cancelled || !dataURL) return;
          setSkins((prev) => (prev[account.uuid] === dataURL ? prev : { ...prev, [account.uuid]: dataURL }));
        })
        .catch(() => {
          failedSkins.add(account.uuid);
        });
    }
    return () => {
      cancelled = true;
    };
  }, [open, accounts]);

  // Code expiry countdown: bounded by the expires_in Ely.by returned.
  useEffect(() => {
    if (!device) {
      setRemaining(0);
      return;
    }
    const tick = () => setRemaining(Math.max(0, Math.ceil((device.expiresAt - Date.now()) / 1000)));
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [device]);

  const stopDevicePoll = () => {
    if (deviceTimer.current) {
      clearInterval(deviceTimer.current);
      deviceTimer.current = null;
    }
  };

  const cancelLogin = () => {
    pendingLogin.current?.cancel();
    pendingLogin.current = null;
  };

  const cancelDeviceFlow = () => {
    // Invalidate the current flow first so an in-flight Finish cannot settle
    // state after cancel, then abort network work and settle run().
    flowGen.current += 1;
    stopDevicePoll();
    finishCancel.current?.();
    finishCancel.current = null;
    const reject = deviceReject.current;
    deviceReject.current = null;
    if (device) {
      void AccountService.CancelElyByOAuth(device.deviceCode).catch(() => undefined);
      setDevice(null);
    }
    setBusy(false);
    setError('');
    reject?.();
  };

  // The add band is hidden by default; the grid card opens it.
  const openAdd = () => {
    setAdding(true);
    requestAnimationFrame(() => {
      bandRef.current?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
      usernameRef.current?.focus();
    });
  };

  const closeAdd = () => {
    if (device) cancelDeviceFlow();
    setAdding(false);
    setError('');
  };

  // Closing the dialog also tears down any running device-code flow so no
  // polling keeps running in the background.
  const handleOpenChange = (next: boolean) => {
    if (!next) {
      if (device) cancelDeviceFlow();
      setAdding(false);
    }
    onOpenChange(next);
  };

  const usePasswordInstead = () => {
    cancelDeviceFlow();
    setElyMethod('password');
    setError('');
    requestAnimationFrame(() => usernameRef.current?.focus());
  };

  const openDevicePage = () => {
    if (!device) return;
    void SystemService.OpenBrowserURL(device.verificationUri).catch(() => {
      setError('Unable to open the Ely.by page. Use the URL shown below instead.');
    });
  };

  const copyCode = async () => {
    if (!device) return;
    try {
      await navigator.clipboard.writeText(device.userCode);
      setCopied(true);
      setTimeout(() => setCopied(false), 1600);
    } catch {
      setError('Unable to copy the code. Select it and copy manually.');
    }
  };

  const run = async (action: () => Promise<unknown>) => {
    setBusy(true);
    setError('');
    try {
      await action();
      await onChanged();
      setPassword('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to update account');
    } finally {
      setBusy(false);
    }
  };

  const confirmRemove = () => {
    if (!pendingRemove) return;
    const uuid = pendingRemove.uuid;
    setPendingRemove(null);
    void run(() => AccountService.DeleteAccount(uuid));
  };

  const addAccount = (event: React.FormEvent) => {
    event.preventDefault();
    if (type === 'microsoft') {
      const call = AccountService.LoginMicrosoft();
      pendingLogin.current = call;
      void run(async () => {
        try {
          await call;
          setUsername('');
          setAdding(false);
        } catch (err) {
          if (err instanceof Error && /cancel/i.test(err.message)) return;
          throw err;
        } finally {
          pendingLogin.current = null;
        }
      });
      return;
    }
    void run(async () => {
      if (type === 'offline') {
        await AccountService.CreateOffline(username.trim());
        setUsername('');
        setAdding(false);
        return;
      }
      if (elyMethod === 'password') {
        await AccountService.LoginElyBy(username.trim(), password);
        setUsername('');
        setAdding(false);
        return;
      }
      // Browser (device-code) flow: fetch the user code, then poll until
      // the player confirms on the Ely.by website.
      const gen = ++flowGen.current;
      const start = await AccountService.StartElyByOAuth();
      if (flowGen.current !== gen) return;
      if (!start) throw new Error('Unable to start Ely.by sign-in');
      let browserOpened = true;
      try {
        await SystemService.OpenBrowserURL(start.verificationUri);
      } catch {
        browserOpened = false;
      }
      if (flowGen.current !== gen) return;
      const totalSeconds = Math.max(1, start.expiresIn || 600);
      setDevice({
        deviceCode: start.deviceCode,
        userCode: start.userCode,
        verificationUri: start.verificationUri,
        interval: Math.max(5, start.interval || 5),
        browserOpened,
        expiresAt: Date.now() + totalSeconds * 1000,
        totalSeconds,
      });
      try {
        await new Promise<void>((resolve, reject) => {
          deviceReject.current = () => reject(new Error('Cancelled'));
          const poll = async () => {
            if (flowGen.current !== gen) return;
            try {
              const call = AccountService.FinishElyByOAuth(start.deviceCode);
              finishCancel.current = () => call.cancel();
              const account = await call;
              if (flowGen.current !== gen) return;
              if (account) {
                stopDevicePoll();
                deviceReject.current = null;
                finishCancel.current = null;
                resolve();
              }
            } catch (err) {
              if (flowGen.current !== gen) return;
              const message = err instanceof Error ? err.message : '';
              // Still waiting: keep polling. Anything else fails fast.
              if (/waiting|approval/i.test(message)) return;
              stopDevicePoll();
              deviceReject.current = null;
              finishCancel.current = null;
              reject(err);
            }
          };
          void poll();
          deviceTimer.current = setInterval(() => void poll(), Math.max(5, start.interval || 5) * 1000);
        });
        setDevice(null);
        setUsername('');
        setAdding(false);
      } catch (err) {
        setDevice(null);
        // Player cancelled on purpose: return quietly like the Microsoft flow.
        if (err instanceof Error && /cancel/i.test(err.message)) return;
        throw err;
      }
    });
  };

  const submitLabel = busy
    ? 'Working…'
    : type === 'microsoft'
      ? 'Continue with Microsoft'
      : type === 'ely.by'
        ? elyMethod === 'browser'
          ? 'Continue with Ely.by'
          : 'Sign in'
        : 'Add offline profile';

  const submitDisabled =
    busy ||
    (type === 'offline' && !username.trim()) ||
    (type === 'ely.by' && elyMethod === 'password' && (!username.trim() || !password));

  return (
    <Dialog.Root open={open} onOpenChange={handleOpenChange}>
      <AnimatePresence>
        {open ? (
          <Dialog.Portal>
            <Dialog.Backdrop
              render={
                <motion.div
                  className="fixed inset-0 z-40 bg-black/55 backdrop-blur-sm"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  transition={{ duration: reduced ? 0.01 : 0.18 }}
                />
              }
            />
            <Dialog.Popup
              render={
                <motion.div
                  className="fixed left-1/2 top-1/2 z-50 max-h-[calc(100vh-2rem)] w-[min(860px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-border bg-background shadow-xl outline-none"
                  initial={{ opacity: 0, scale: 0.96, y: 8 }}
                  animate={{ opacity: 1, scale: 1, y: 0 }}
                  exit={{ opacity: 0, scale: 0.96, y: 8 }}
                  transition={{ duration: reduced ? 0.01 : 0.22, ease: [0.16, 1, 0.3, 1] }}
                />
              }
            >
          <header className="flex items-center justify-between border-b border-border px-5 py-3 sm:px-6">
            <div className="flex items-baseline gap-2">
              <Dialog.Title className="text-base font-semibold tracking-tight">Accounts</Dialog.Title>
              <span className="rounded-md border border-border px-1.5 py-0.5 text-[10px] text-muted-foreground">{accounts.length}</span>
            </div>
            <Dialog.Description className="sr-only">Select, add, or remove a launcher account.</Dialog.Description>
            <Button type="button" variant="ghost" size="icon-sm" onClick={() => onOpenChange(false)} aria-label="Close accounts"><IconX className="size-4" /></Button>
          </header>

          <div className="space-y-4 px-5 py-4 sm:px-6">
            {active ? (
              <motion.section
                aria-label="Active identity"
                initial={{ opacity: 0, y: -6 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: reduced ? 0.01 : 0.26, ease: [0.16, 1, 0.3, 1] }}
                className="flex items-center gap-3.5 rounded-md border border-border bg-muted/60 py-3 pr-4 pl-3.5 sm:pl-4"
              >
                <AvatarTile account={active} skin={skins[active.uuid]} active className="size-11 text-base sm:size-12 sm:text-lg" />
                <div className="min-w-0 flex-1">
                  <p className="flex flex-wrap items-center gap-x-2.5 gap-y-1 text-[15px] font-semibold">
                    <span className="truncate">{active.displayName || active.username}</span>
                    <span className="inline-flex items-center gap-1 text-xs font-medium text-primary">
                      <IconCheck className="size-3.5" />
                      Active
                    </span>
                  </p>
                  <p className="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground">
                    {providerIcon(active.type, 'size-3.5 shrink-0')}
                    <span className="truncate">{providerLine(active)}</span>
                  </p>
                </div>
              </motion.section>
            ) : null}

            <section aria-label="Switch identity">
              <p className="text-xs font-semibold text-muted-foreground">Switch identity</p>

              {accounts.length ? (
                <div className="mt-2 grid gap-2.5 sm:grid-cols-2">
                  {accounts.map((account, index) => (
                    <motion.div
                      key={account.uuid}
                      aria-current={account.selected ? 'true' : undefined}
                      initial={{ opacity: 0, y: 6 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{
                        duration: reduced ? 0.01 : 0.22,
                        delay: reduced ? 0 : Math.min(index, 5) * 0.035,
                        ease: [0.16, 1, 0.3, 1],
                      }}
                      className={cn(
                        'flex min-w-0 items-center gap-3 rounded-md border border-border bg-card p-2.5 transition-colors',
                        account.selected && 'border-primary bg-primary/5',
                      )}
                    >
                      <AvatarTile account={account} skin={skins[account.uuid]} active={account.selected} className="size-9 text-sm" />
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">{account.displayName || account.username}</p>
                        <p className="mt-0.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                          {providerIcon(account.type, 'size-3.5 shrink-0')}
                          <span className="truncate">{account.type === 'offline' ? 'Offline, no sign-in' : providerLabel[account.type as Provider] ?? account.type}</span>
                        </p>
                      </div>
                      <div className="flex shrink-0 items-center gap-1.5">
                        {account.selected ? (
                          <span className="inline-flex items-center gap-1 text-xs font-medium text-primary">
                            <IconCheck className="size-3.5" />
                            Active
                          </span>
                        ) : (
                          <Button type="button" size="sm" variant="secondary" disabled={busy} onClick={() => void run(() => AccountService.SelectAccount(account.uuid))}>
                            Use
                          </Button>
                        )}
                        <Button
                          type="button"
                          size="icon-sm"
                          variant="ghost"
                          className="text-muted-foreground hover:text-destructive"
                          disabled={busy || accounts.length <= 1}
                          title={accounts.length <= 1 ? 'Cannot remove the last account' : undefined}
                          aria-label={`Remove ${account.displayName || account.username}`}
                          onClick={() => setPendingRemove(account)}
                        >
                          <IconTrash className="size-4" />
                        </Button>
                      </div>
                    </motion.div>
                  ))}

                  <motion.button
                    type="button"
                    onClick={openAdd}
                    initial={{ opacity: 0, y: 6 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{
                      duration: reduced ? 0.01 : 0.22,
                      delay: reduced ? 0 : Math.min(accounts.length, 5) * 0.035,
                      ease: [0.16, 1, 0.3, 1],
                    }}
                    className="flex min-h-14 items-center justify-center gap-2 rounded-md border border-dashed border-border px-3 text-sm font-medium text-muted-foreground transition-colors hover:border-muted-foreground hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring"
                  >
                    <IconPlus className="size-4" />
                    Add account
                  </motion.button>
                </div>
              ) : (
                <div className="mt-2 grid min-h-32 place-items-center rounded-md border border-dashed border-border px-4 text-center text-sm text-muted-foreground">
                  No identities yet. Add one below to start playing.
                </div>
              )}
            </section>

            <AnimatePresence initial={false}>
              {adding ? (
                <motion.div
                  key="add-band"
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: 'auto' }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={{ duration: reduced ? 0.01 : 0.2, ease: [0.16, 1, 0.3, 1] }}
                  className="overflow-hidden"
                >
            <form ref={bandRef} onSubmit={addAccount} aria-label="Add account" className="space-y-3 rounded-md border border-border bg-muted/60 p-3.5 sm:p-4">
              <div className="flex flex-wrap items-center justify-end gap-2">
                <p className="mr-auto text-xs font-semibold text-muted-foreground">Add account</p>
                <Button type="button" size="icon-sm" variant="ghost" aria-label="Hide add account" onClick={closeAdd}>
                  <IconX className="size-3.5" />
                </Button>
              </div>

              <div className="flex flex-wrap items-center gap-2">
                <div className="inline-flex rounded-md border border-border bg-background p-1" role="group" aria-label="Account type">
                  {(Object.keys(providerLabel) as Provider[]).map((value) => (
                    <Button
                      key={value}
                      type="button"
                      size="sm"
                      variant={type === value && !device ? 'secondary' : 'ghost'}
                      aria-pressed={type === value && !device}
                      onClick={() => { setType(value); setError(''); }}
                    >
                      {providerLabel[value]}
                    </Button>
                  ))}
                </div>
                {type === 'ely.by' && !device ? (
                  <div className="inline-flex rounded-md border border-border bg-background p-1" role="group" aria-label="Ely.by sign-in method">
                    <Button type="button" size="sm" variant={elyMethod === 'browser' ? 'secondary' : 'ghost'} aria-pressed={elyMethod === 'browser'} onClick={() => { setElyMethod('browser'); setError(''); }}>Browser</Button>
                    <Button type="button" size="sm" variant={elyMethod === 'password' ? 'secondary' : 'ghost'} aria-pressed={elyMethod === 'password'} onClick={() => { setElyMethod('password'); setError(''); }}>Password</Button>
                  </div>
                ) : null}
              </div>

              <AnimatePresence mode="wait" initial={false}>
              {device ? (
                <motion.div
                  key="device"
                  className="space-y-3"
                  aria-live="polite"
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -4 }}
                  transition={{ duration: reduced ? 0.01 : 0.16, ease: [0.16, 1, 0.3, 1] }}
                >
                  <div className="flex items-start justify-between gap-3 rounded-md border border-border bg-background p-3.5">
                    <div className="min-w-0">
                      <p className="text-xs text-muted-foreground">Enter this code on Ely.by</p>
                      <p className="mt-1 select-all text-3xl font-bold tracking-[0.16em] uppercase">{device.userCode}</p>
                      <p className="mt-1 select-all break-all text-xs text-muted-foreground">{device.verificationUri}</p>
                    </div>
                    <Button type="button" size="icon-sm" variant="ghost" aria-label={copied ? 'Code copied' : 'Copy code'} onClick={() => void copyCode()}>
                      {copied ? <IconCheck className="size-4 text-primary" /> : <IconCopy className="size-4" />}
                    </Button>
                  </div>

                  <div className="space-y-1.5">
                    <div
                      role="progressbar"
                      aria-label="Code time remaining"
                      aria-valuemin={0}
                      aria-valuemax={device.totalSeconds}
                      aria-valuenow={remaining}
                      className="h-1.5 overflow-hidden rounded-md border border-border bg-background"
                    >
                      <div
                        className="h-full bg-primary transition-[width] duration-1000 ease-linear"
                        style={{ width: `${Math.min(100, Math.max(0, (remaining / device.totalSeconds) * 100))}%` }}
                      />
                    </div>
                    <p className="text-[11px] text-muted-foreground">Code expires in {formatSeconds(remaining)}</p>
                  </div>

                  <p className="flex items-center gap-2 text-xs text-muted-foreground">
                    <IconLoader2 className="size-3.5 shrink-0 animate-spin text-ring" />
                    {device.browserOpened ? 'Browser opened. Waiting for approval on Ely.by.' : 'Open the page manually. Waiting for approval on Ely.by.'}
                  </p>

                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <Button type="button" size="sm" variant="ghost" onClick={cancelDeviceFlow}>
                      Cancel
                    </Button>
                    <div className="flex flex-wrap gap-2">
                      <Button type="button" size="sm" variant="secondary" onClick={openDevicePage}>Open page again</Button>
                      <Button type="button" size="sm" variant="secondary" onClick={usePasswordInstead}>
                        Use password instead
                      </Button>
                    </div>
                  </div>
                </motion.div>
              ) : type === 'microsoft' ? (
                <motion.div
                  key="microsoft"
                  className="space-y-3"
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -4 }}
                  transition={{ duration: reduced ? 0.01 : 0.16, ease: [0.16, 1, 0.3, 1] }}
                >
                  <p className="flex items-start gap-1.5 text-xs text-muted-foreground">
                    <IconLock className="mt-px size-3.5 shrink-0" />
                    {busy ? 'Waiting for Microsoft sign-in. Close the sign-in window to cancel.' : 'Opens a Microsoft sign-in window. Tokens stay safely on this device.'}
                  </p>
                  <div className="flex items-center justify-end gap-2">
                    <div className="flex gap-2">
                      {busy ? <Button type="button" size="sm" variant="ghost" onClick={cancelLogin}>Cancel</Button> : null}
                      <Button type="submit" size="sm" disabled={submitDisabled}>
                        {busy ? <IconLoader2 className="mr-1.5 size-3.5 animate-spin" /> : null}
                        {submitLabel}
                      </Button>
                    </div>
                  </div>
                </motion.div>
              ) : (
                <motion.div
                  key={`${type}-${elyMethod}`}
                  className="space-y-3"
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -4 }}
                  transition={{ duration: reduced ? 0.01 : 0.16, ease: [0.16, 1, 0.3, 1] }}
                >
                  {type === 'ely.by' && elyMethod === 'browser' ? (
                    <p className="flex items-start gap-1.5 text-xs text-muted-foreground">
                      <IconWorld className="mt-px size-3.5 shrink-0" />
                      Opens the Ely.by page in your browser. Your password never enters the launcher, and the code expires after 10 minutes.
                    </p>
                  ) : (
                    <div className="space-y-2.5">
                      <label className="block space-y-1.5 text-xs font-medium">
                        {type === 'ely.by' ? 'Email or username' : 'Username'}
                        <Input
                          ref={usernameRef}
                          value={username}
                          maxLength={type === 'offline' ? 16 : undefined}
                          onChange={(event) => setUsername(event.target.value)}
                          autoFocus={!active}
                          required
                        />
                      </label>
                      {type === 'ely.by' ? (
                        <label className="block space-y-1.5 text-xs font-medium">
                          Password
                          <Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="current-password" required />
                        </label>
                      ) : null}
                      <p className="flex items-start gap-1.5 text-xs text-muted-foreground">
                        <IconLock className="mt-px size-3.5 shrink-0" />
                        {type === 'ely.by' ? 'Session is stored safely on this device. Password sign-in skips 2FA, use Browser for that.' : 'Username only. The profile never leaves this device.'}
                      </p>
                    </div>
                  )}

                  <div className="flex items-center justify-end gap-2">
                    <Button type="submit" size="sm" disabled={submitDisabled}>
                      {busy ? <IconLoader2 className="mr-1.5 size-3.5 animate-spin" /> : null}
                      {submitLabel}
                    </Button>
                  </div>
                </motion.div>
              )}
              </AnimatePresence>

              {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
            </form>
                </motion.div>
              ) : null}
            </AnimatePresence>
          </div>
        </Dialog.Popup>
          </Dialog.Portal>
        ) : null}
      </AnimatePresence>
      <AlertDialog open={pendingRemove !== null} onOpenChange={(open) => { if (!open) setPendingRemove(null); }}>
        <AlertDialogContent>
          <AlertDialogTitle>Remove account?</AlertDialogTitle>
          <AlertDialogDescription>This removes <strong className="text-foreground">{pendingRemove?.displayName || pendingRemove?.username}</strong> from this device{pendingRemove?.type === 'microsoft' || pendingRemove?.type === 'ely.by' ? ' and signs it out' : ''}.</AlertDialogDescription>
          <div className="mt-5 flex justify-end gap-2"><Button type="button" variant="ghost" disabled={busy} onClick={() => setPendingRemove(null)}>Cancel</Button><Button type="button" variant="destructive" disabled={busy} onClick={confirmRemove}>{busy ? 'Removing...' : 'Remove'}</Button></div>
        </AlertDialogContent>
      </AlertDialog>
    </Dialog.Root>
  );
}
