import { useRef, useState } from 'react';
import { IconKey, IconLoader2, IconPlus, IconTrash, IconUser, IconX } from '@tabler/icons-react';
import { Dialog } from '@base-ui/react/dialog';
import { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { AccountService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { Account } from '../../../bindings/plumelauncher/internal/services/models.js';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';

interface AccountDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  accounts: Account[];
  onChanged: () => Promise<void>;
}

export function AccountDialog({ open, onOpenChange, accounts, onChanged }: AccountDialogProps) {
  const [type, setType] = useState<'offline' | 'ely.by' | 'microsoft'>('offline');
  const [elyMethod, setElyMethod] = useState<'browser' | 'password'>('browser');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [device, setDevice] = useState<{ deviceCode: string; userCode: string; verificationUri: string; interval: number } | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [pendingRemove, setPendingRemove] = useState<Account | null>(null);
  const pendingLogin = useRef<{ cancel: () => void } | null>(null);
  const deviceTimer = useRef<ReturnType<typeof setInterval> | null>(null);

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
    stopDevicePoll();
    if (device) {
      void AccountService.CancelElyByOAuth(device.deviceCode).catch(() => undefined);
      setDevice(null);
    }
    setBusy(false);
    setError('');
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
        return;
      }
      if (elyMethod === 'password') {
        await AccountService.LoginElyBy(username.trim(), password);
        setUsername('');
        return;
      }
      // Browser (device-code) flow: fetch the user code, then poll until
      // the player confirms on the Ely.by website.
      const start = await AccountService.StartElyByOAuth();
      if (!start) throw new Error('Unable to start Ely.by sign-in');
      setDevice({ deviceCode: start.deviceCode, userCode: start.userCode, verificationUri: start.verificationUri, interval: Math.max(5, start.interval || 5) });
      try {
        await new Promise<void>((resolve, reject) => {
          const poll = async () => {
            try {
              const account = await AccountService.FinishElyByOAuth(start.deviceCode);
              if (account) {
                stopDevicePoll();
                resolve();
              }
            } catch (err) {
              const message = err instanceof Error ? err.message : '';
              // Still waiting: keep polling. Anything else fails fast.
              if (/waiting|approval/i.test(message)) return;
              stopDevicePoll();
              reject(err);
            }
          };
          void poll();
          deviceTimer.current = setInterval(() => void poll(), Math.max(5, start.interval || 5) * 1000);
        });
        setDevice(null);
        setUsername('');
      } catch (err) {
        setDevice(null);
        throw err;
      }
    });
  };

  const typeLabel = (t: string) => (t === 'ely.by' ? 'Ely.by' : t === 'microsoft' ? 'Microsoft' : 'Offline');

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/55 backdrop-blur-sm transition-[opacity] duration-200 data-ending-style:opacity-0 data-starting-style:opacity-0" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 w-[min(860px,calc(100vw-2rem))] max-h-[calc(100vh-2rem)] -translate-x-1/2 -translate-y-1/2 overflow-hidden rounded-xl border border-border bg-background shadow-xl outline-none transition-[opacity,transform] duration-200 ease-[cubic-bezier(0.16,1,0.3,1)] data-ending-style:scale-95 data-ending-style:opacity-0 data-starting-style:scale-95 data-starting-style:opacity-0">
          <header className="flex items-center justify-between border-b border-border px-5 py-4 sm:px-6">
            <div className="flex items-baseline gap-2">
              <Dialog.Title className="text-base font-semibold tracking-tight">Accounts</Dialog.Title>
              <span className="font-mono text-[10px] text-muted-foreground">{accounts.length}</span>
            </div>
            <Dialog.Description className="sr-only">Select, add, or remove a launcher account.</Dialog.Description>
            <Button type="button" variant="ghost" size="icon-sm" onClick={() => onOpenChange(false)} aria-label="Close accounts"><IconX className="size-4" /></Button>
          </header>

          <div className="grid items-start md:grid-cols-[minmax(260px,0.85fr)_minmax(360px,1.15fr)]">
            <section aria-label="Saved accounts" className="border-b border-border p-3 md:border-b-0 md:border-r">
              <div className="max-h-[390px] overflow-y-auto rounded-lg border border-border">
                {accounts.length ? (
                  <div className="divide-y divide-border">
                    {accounts.map((account) => (
                      <div key={account.uuid} className={cn('flex min-h-16 items-center gap-3 px-3 py-2.5 transition-colors', account.selected && 'bg-primary/8')}>
                        <span className={cn('grid size-8 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground', account.selected && 'bg-primary/15 text-primary')}>
                          <IconUser className="size-3.5" />
                        </span>
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-sm font-medium">{account.displayName || account.username}</p>
                          <div className="mt-1 flex items-center gap-1.5">
                            <span className="rounded-md bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">{typeLabel(account.type)}</span>
                            {account.selected ? <span className="text-[10px] font-medium text-primary">Active</span> : null}
                          </div>
                        </div>
                        <div className="flex shrink-0 items-center gap-1">
                          {!account.selected ? <Button type="button" size="sm" variant="secondary" disabled={busy} onClick={() => void run(() => AccountService.SelectAccount(account.uuid))}>Select</Button> : null}
                          <Button type="button" size="icon-xs" variant="ghost" className="text-muted-foreground hover:text-destructive" disabled={busy || accounts.length <= 1} title={accounts.length <= 1 ? 'Cannot remove the last account' : undefined} aria-label={`Remove ${account.displayName || account.username}`} onClick={() => setPendingRemove(account)}>
                            <IconTrash className="size-3.5" />
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="grid min-h-40 place-items-center text-center text-sm text-muted-foreground">No saved accounts.</div>
                )}
              </div>
            </section>

            <form className="w-full self-start" onSubmit={addAccount}>
              <div className="px-5 py-5 sm:px-6">
                <h2 className="text-sm font-semibold">Add account</h2>
                <div className="mt-4 inline-flex rounded-lg border border-border p-1">
                  <Button type="button" size="sm" variant={type === 'offline' ? 'secondary' : 'ghost'} onClick={() => setType('offline')}>Offline</Button>
                  <Button type="button" size="sm" variant={type === 'ely.by' ? 'secondary' : 'ghost'} onClick={() => setType('ely.by')}>Ely.by</Button>
                  <Button type="button" size="sm" variant={type === 'microsoft' ? 'secondary' : 'ghost'} onClick={() => setType('microsoft')}>Microsoft</Button>
                </div>

                {type === 'microsoft' ? (
                  <div className="mt-5 space-y-4">
                    <p className="flex items-center gap-1.5 text-xs text-muted-foreground"><IconKey className="size-3.5" />{busy ? 'Waiting for Microsoft sign-in… Close the sign-in window to cancel.' : 'Opens a Microsoft sign-in window. Tokens stay safely on this device.'}</p>
                    {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
                  </div>
                ) : (
                <div className="mt-5 space-y-4">
                  {!(type === 'ely.by' && elyMethod === 'browser') ? (
                  <label className="block space-y-1.5 text-xs font-medium">
                    {type === 'ely.by' ? 'Email or username' : 'Username'}
                    <Input value={username} maxLength={type === 'offline' ? 16 : undefined} onChange={(event) => setUsername(event.target.value)} autoFocus required />
                  </label>
                  ) : null}
                  {type === 'ely.by' ? (
                    <>
                      <div className="inline-flex rounded-lg border border-border p-1" role="group" aria-label="Ely.by sign-in method">
                        <Button type="button" size="sm" variant={elyMethod === 'browser' ? 'secondary' : 'ghost'} onClick={() => { setElyMethod('browser'); setError(''); }}>Browser</Button>
                        <Button type="button" size="sm" variant={elyMethod === 'password' ? 'secondary' : 'ghost'} onClick={() => { setElyMethod('password'); setError(''); }}>Password</Button>
                      </div>
                      {elyMethod === 'browser' ? (
                        device ? (
                          <div className="space-y-2 rounded-lg border border-border p-3">
                            <p className="text-xs text-muted-foreground">Open this page and enter the code:</p>
                            <p className="break-all font-mono text-xs text-primary">{device.verificationUri}</p>
                            <p className="font-mono text-2xl font-bold tracking-widest">{device.userCode}</p>
                            <p className="flex items-center gap-1.5 text-xs text-muted-foreground"><IconLoader2 className="size-3.5 animate-spin" />{busy ? 'Waiting for approval on the Ely.by website…' : ''}</p>
                            <Button type="button" size="sm" variant="ghost" onClick={cancelDeviceFlow}>Cancel</Button>
                          </div>
                        ) : (
                          <p className="flex items-center gap-1.5 text-xs text-muted-foreground"><IconKey className="size-3.5" />Opens a code on the Ely.by website. Your password never enters the launcher.</p>
                        )
                      ) : (
                        <>
                          <label className="block space-y-1.5 text-xs font-medium">
                            Password
                            <Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="current-password" required />
                          </label>
                          <p className="flex items-center gap-1.5 text-xs text-muted-foreground"><IconKey className="size-3.5" />Session is stored safely on this device.</p>
                        </>
                      )}
                    </>
                  ) : null}
                  {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
                </div>
                )}
              </div>
              <footer className="flex justify-end gap-2 border-t border-border px-5 py-3 sm:px-6">
                {busy && type === 'microsoft' ? <Button type="button" size="sm" variant="ghost" onClick={cancelLogin}>Cancel</Button> : null}
                {busy && type === 'ely.by' && elyMethod === 'browser' ? <Button type="button" size="sm" variant="ghost" onClick={cancelDeviceFlow}>Cancel</Button> : null}
                <Button type="submit" size="sm" disabled={busy || (type === 'offline' && !username.trim()) || (type === 'ely.by' && elyMethod === 'password' && (!username.trim() || !password)) || (type === 'ely.by' && elyMethod === 'browser' && !!device)}>
                  {busy ? <IconLoader2 className="mr-1.5 size-3.5 animate-spin" /> : <IconPlus className="mr-1.5 size-3.5" />}
                  {type === 'microsoft' ? 'Sign in with Microsoft' : type === 'ely.by' && elyMethod === 'browser' ? 'Get Ely.by code' : type === 'ely.by' ? 'Sign in' : 'Add profile'}
                </Button>
              </footer>
            </form>
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
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
