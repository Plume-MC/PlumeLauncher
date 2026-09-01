import { useState } from 'react';
import { FolderKey, Loader2, Plus, Trash2, UserRound, X } from 'lucide-react';
import { Dialog } from '@base-ui/react/dialog';
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
  const [type, setType] = useState<'offline' | 'ely.by'>('offline');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

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

  const addAccount = (event: React.FormEvent) => {
    event.preventDefault();
    void run(async () => {
      if (type === 'offline') await AccountService.CreateOffline(username.trim());
      else await AccountService.LoginElyBy(username.trim(), password);
      setUsername('');
    });
  };

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
            <Button type="button" variant="ghost" size="icon-sm" onClick={() => onOpenChange(false)} aria-label="Close accounts"><X className="size-4" /></Button>
          </header>

          <div className="grid items-start md:grid-cols-[minmax(260px,0.85fr)_minmax(360px,1.15fr)]">
            <section aria-label="Saved accounts" className="border-b border-border p-3 md:border-b-0 md:border-r">
              <div className="max-h-[390px] overflow-y-auto rounded-lg border border-border">
                {accounts.length ? (
                  <div className="divide-y divide-border">
                    {accounts.map((account) => (
                      <div key={account.uuid} className={cn('flex min-h-16 items-center gap-3 px-3 py-2.5 transition-colors', account.selected && 'bg-primary/8')}>
                        <span className={cn('grid size-8 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground', account.selected && 'bg-primary/15 text-primary')}>
                          <UserRound className="size-3.5" />
                        </span>
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-sm font-medium">{account.displayName || account.username}</p>
                          <div className="mt-1 flex items-center gap-1.5">
                            <span className="rounded-md bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">{account.type === 'ely.by' ? 'Ely.by' : 'Offline'}</span>
                            {account.selected ? <span className="text-[10px] font-medium text-primary">Active</span> : null}
                          </div>
                        </div>
                        <div className="flex shrink-0 items-center gap-1">
                          {!account.selected ? <Button type="button" size="sm" variant="secondary" disabled={busy} onClick={() => void run(() => AccountService.SelectAccount(account.uuid))}>Use</Button> : null}
                          <Button type="button" size="icon-xs" variant="ghost" className="text-muted-foreground hover:text-destructive" disabled={busy} aria-label={`Remove ${account.displayName || account.username}`} onClick={() => void run(() => AccountService.DeleteAccount(account.uuid))}>
                            <Trash2 className="size-3.5" />
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="grid min-h-40 place-items-center text-center text-sm text-muted-foreground">No saved profiles.</div>
                )}
              </div>
            </section>

            <form className="w-full self-start" onSubmit={addAccount}>
              <div className="px-5 py-5 sm:px-6">
                <h2 className="text-sm font-semibold">Add account</h2>
                <div className="mt-4 inline-flex rounded-lg border border-border p-1">
                  <Button type="button" size="sm" variant={type === 'offline' ? 'secondary' : 'ghost'} onClick={() => setType('offline')}>Offline</Button>
                  <Button type="button" size="sm" variant={type === 'ely.by' ? 'secondary' : 'ghost'} onClick={() => setType('ely.by')}>Ely.by</Button>
                </div>

                <div className="mt-5 space-y-4">
                  <label className="block space-y-1.5 text-xs font-medium">
                    {type === 'ely.by' ? 'Email or username' : 'Username'}
                    <Input value={username} maxLength={type === 'offline' ? 16 : undefined} onChange={(event) => setUsername(event.target.value)} autoFocus required />
                  </label>
                  {type === 'ely.by' ? (
                    <>
                      <label className="block space-y-1.5 text-xs font-medium">
                        Password
                        <Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="current-password" required />
                      </label>
                      <p className="flex items-center gap-1.5 text-xs text-muted-foreground"><FolderKey className="size-3.5" />Session is stored in your OS keyring.</p>
                    </>
                  ) : null}
                  {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
                </div>
              </div>
              <footer className="flex justify-end border-t border-border px-5 py-3 sm:px-6">
                <Button type="submit" size="sm" disabled={busy || !username.trim() || (type === 'ely.by' && !password)}>
                  {busy ? <Loader2 className="mr-1.5 size-3.5 animate-spin" /> : <Plus className="mr-1.5 size-3.5" />}
                  {type === 'ely.by' ? 'Sign in' : 'Add profile'}
                </Button>
              </footer>
            </form>
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
