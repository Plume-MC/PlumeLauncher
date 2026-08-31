import { useState } from 'react';
import { Check, ChevronLeft, Eye, EyeOff, Loader2, UserCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { AccountService } from '../../../bindings/plumelauncher/internal/services/index.js';

interface AccountStepProps {
  onNext: () => void;
  onBack: () => void;
}

export function AccountStep({ onNext, onBack }: AccountStepProps) {
  const [type, setType] = useState<'offline' | 'ely.by'>('offline');
  const [username, setUsername] = useState('Player');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const create = async () => {
    setSaving(true);
    setError('');
    try {
      if (type === 'offline') await AccountService.CreateOffline(username.trim());
      else await AccountService.LoginElyBy(username.trim(), password);
      onNext();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to create account');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="space-y-1 text-center">
        <h1 className="text-xl font-semibold tracking-tight">Add account</h1>
        <p className="text-sm text-muted-foreground">Pick offline play or Ely.by. Tokens stay in the OS keyring.</p>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <button
          type="button"
          onClick={() => setType('offline')}
          className={cn(
            'flex flex-col items-center gap-2 rounded-xl border p-4 text-center transition-colors',
            type === 'offline'
              ? 'border-primary/50 bg-primary/10'
              : 'border-border bg-card/30 hover:border-border hover:bg-muted/40'
          )}
        >
          <span className="flex size-11 items-center justify-center rounded-full border border-border bg-muted/40">
            <UserCircle className="size-5 text-muted-foreground" />
          </span>
          <span className="text-sm font-semibold">Offline</span>
          <span className="text-[10px] text-muted-foreground">Local profile</span>
        </button>
        <button
          type="button"
          onClick={() => setType('ely.by')}
          className={cn(
            'flex flex-col items-center gap-2 rounded-xl border p-4 text-center transition-colors',
            type === 'ely.by'
              ? 'border-primary/50 bg-primary/10'
              : 'border-border bg-card/30 hover:border-border hover:bg-muted/40'
          )}
        >
          <span className="flex size-11 items-center justify-center rounded-full border border-border bg-muted/40 text-xs font-bold tracking-tight text-foreground">
            Ely
          </span>
          <span className="text-sm font-semibold">Ely.by</span>
          <span className="text-[10px] text-muted-foreground">Online session</span>
        </button>
      </div>

      <div className="space-y-3">
        <div className="space-y-1.5">
          <label htmlFor="setup-username" className="text-xs font-medium text-muted-foreground">
            Username
          </label>
          <Input
            id="setup-username"
            value={username}
            maxLength={type === 'offline' ? 16 : undefined}
            onChange={(e) => setUsername(e.target.value)}
            autoFocus
            className="h-10"
          />
        </div>
        {type === 'ely.by' ? (
          <div className="space-y-1.5">
            <label htmlFor="setup-password" className="text-xs font-medium text-muted-foreground">
              Password
            </label>
            <div className="relative">
              <Input
                id="setup-password"
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
                className="h-10 pr-10"
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && username.trim() && password) void create();
                }}
              />
              <button
                type="button"
                className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground hover:text-foreground"
                onClick={() => setShowPassword((v) => !v)}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
              </button>
            </div>
          </div>
        ) : null}
      </div>

      {error ? (
        <p role="alert" className="text-xs text-destructive">
          {error}
        </p>
      ) : null}

      <div className="flex gap-2">
        <Button variant="outline" onClick={onBack} className="flex-1 gap-1 border-border" disabled={saving}>
          <ChevronLeft className="size-4" />
          Back
        </Button>
        <Button
          onClick={() => void create()}
          disabled={saving || !username.trim() || (type === 'ely.by' && !password)}
          className="flex-1 gap-1.5 bg-foreground font-semibold text-background hover:bg-foreground/90"
        >
          {saving ? <Loader2 className="size-4 animate-spin" /> : <Check className="size-4" />}
          {saving ? 'Working...' : type === 'ely.by' ? 'Sign in' : 'Create profile'}
        </Button>
      </div>
    </div>
  );
}
