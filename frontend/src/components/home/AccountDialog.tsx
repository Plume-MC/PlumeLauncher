import { useState } from 'react';
import { Dialog } from '@base-ui/react/dialog';
import { AccountService } from '../../../bindings/plumelauncher/internal/services/index.js';
import type { Account } from '../../../bindings/plumelauncher/internal/services/models.js';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

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
    setBusy(true); setError('');
    try { await action(); await onChanged(); setPassword(''); }
    catch (err) { setError(err instanceof Error ? err.message : 'Unable to update account'); }
    finally { setBusy(false); }
  };

  const addAccount = (event: React.FormEvent) => {
    event.preventDefault();
    void run(async () => {
      if (type === 'offline') await AccountService.CreateOffline(username.trim());
      else await AccountService.LoginElyBy(username.trim(), password);
      setUsername('');
    });
  };

  return <Dialog.Root open={open} onOpenChange={onOpenChange}><Dialog.Portal>
    <Dialog.Backdrop className="fixed inset-0 z-40 bg-black/50" />
    <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-background p-5 shadow-xl">
      <Dialog.Title className="text-base font-semibold">Accounts</Dialog.Title>
      <Dialog.Description className="mt-1 text-sm text-muted-foreground">Ely.by sessions are kept in your OS keyring.</Dialog.Description>
      <div className="mt-4 space-y-2">{accounts.map((account) => <div key={account.uuid} className={`flex items-center justify-between rounded-md border p-3 text-sm ${account.selected ? 'border-primary/60 bg-primary/10' : 'border-border'}`}>
        <div className="min-w-0"><p className="truncate font-medium">{account.displayName || account.username}</p><p className="mt-0.5 text-xs text-muted-foreground">{account.type === 'ely.by' ? 'Ely.by session' : 'Offline profile'}{account.selected && ' · Active'}</p></div>
        <div className="ml-3 flex shrink-0 gap-1">{!account.selected && <Button size="sm" variant="secondary" disabled={busy} onClick={() => void run(() => AccountService.SelectAccount(account.uuid))}>Use</Button>}{account.type === 'ely.by' && <Button size="sm" variant="ghost" disabled={busy} onClick={() => void run(() => AccountService.LogoutElyBy(account.uuid))}>Log out</Button>}</div>
      </div>)}</div>
      <form className="mt-5 space-y-3 border-t border-border pt-4" onSubmit={addAccount}>
        <p className="text-sm font-medium">Add account</p>
        <div className="flex gap-2"><Button type="button" size="sm" variant={type === 'offline' ? 'default' : 'ghost'} onClick={() => setType('offline')}>Offline</Button><Button type="button" size="sm" variant={type === 'ely.by' ? 'default' : 'ghost'} onClick={() => setType('ely.by')}>Ely.by</Button></div>
        <div><label htmlFor="account-username" className="text-sm font-medium">Username</label><Input id="account-username" value={username} maxLength={type === 'offline' ? 16 : undefined} onChange={(event) => setUsername(event.target.value)} required /></div>
        {type === 'ely.by' && <div><label htmlFor="account-password" className="text-sm font-medium">Password</label><Input id="account-password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="current-password" required /></div>}
        {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
        <div className="flex justify-end gap-2"><Dialog.Close render={<Button type="button" variant="ghost" />}>Close</Dialog.Close><Button type="submit" disabled={busy || !username.trim() || (type === 'ely.by' && !password)}>{busy ? 'Working...' : type === 'ely.by' ? 'Sign in' : 'Add offline profile'}</Button></div>
      </form>
    </Dialog.Popup>
  </Dialog.Portal></Dialog.Root>;
}
