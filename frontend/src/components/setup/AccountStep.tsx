import { useState } from 'react';
import { IconCheck, IconChevronLeft, IconEye, IconEyeOff, IconLoader2, IconUserCircle } from '@tabler/icons-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { AccountService } from '../../../bindings/plumelauncher/internal/services/index.js';

interface AccountStepProps {
  onNext: () => void;
  onBack: () => void;
}

export function AccountStep({ onNext, onBack }: AccountStepProps) {
  const [type, setType] = useState<'offline' | 'ely.by' | 'microsoft'>('offline');
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
      else if (type === 'ely.by') await AccountService.LoginElyBy(username.trim(), password);
      else await AccountService.LoginMicrosoft();
      onNext();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to create account');
    } finally {
      setSaving(false);
    }
  };

  const cards = [
    { id: 'offline' as const, title: 'Offline', sub: 'Local profile', icon: <IconUserCircle className="size-5 text-muted-foreground" /> },
    { id: 'ely.by' as const, title: 'Ely.by', sub: 'Online session', icon: <span className="text-xs font-bold tracking-tight text-foreground">Ely</span> },
    { id: 'microsoft' as const, title: 'Microsoft', sub: 'Online session', icon: <span className="text-xs font-bold tracking-tight text-foreground">MS</span> },
  ];

  return (
    <div className="space-y-6">
      <div className="space-y-1 text-center">
        <h1 className="text-xl font-semibold tracking-tight">Add account</h1>
        <p className="text-sm text-muted-foreground">Pick offline play, Ely.by, or Microsoft. Tokens stay in the OS keyring.</p>
      </div>

      <div className="grid grid-cols-3 gap-2">
        {cards.map((card) => (
          <button
            key={card.id}
            type="button"
            onClick={() => setType(card.id)}
            className={cn(
              'flex flex-col items-center gap-2 rounded-xl border p-4 text-center transition-colors',
              type === card.id
                ? 'border-primary/50 bg-primary/10'
                : 'border-border bg-card/30 hover:border-border hover:bg-muted/40'
            )}
          >
            <span className="flex size-11 items-center justify-center rounded-full border border-border bg-muted/40">
              {card.icon}
            </span>
            <span className="text-sm font-semibold">{card.title}</span>
            <span className="text-[10px] text-muted-foreground">{card.sub}</span>
          </button>
        ))}
      </div>

      {type === 'microsoft' ? (
        <p className="text-center text-xs text-muted-foreground">Opens a Microsoft sign-in window. Tokens stay in the OS keyring.</p>
      ) : (
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
                {showPassword ? <IconEyeOff className="size-4" /> : <IconEye className="size-4" />}
              </button>
            </div>
          </div>
        ) : null}
      </div>
      )}

      {error ? (
        <p role="alert" className="text-xs text-destructive">
          {error}
        </p>
      ) : null}

      <div className="flex gap-2">
        <Button variant="outline" onClick={onBack} className="flex-1 gap-1 border-border" disabled={saving}>
          <IconChevronLeft className="size-4" />
          Back
        </Button>
        <Button
          onClick={() => void create()}
          disabled={saving || (type !== 'microsoft' && (!username.trim() || (type === 'ely.by' && !password)))}
          className="flex-1 gap-1.5 bg-foreground font-semibold text-background hover:bg-foreground/90"
        >
          {saving ? <IconLoader2 className="size-4 animate-spin" /> : <IconCheck className="size-4" />}
          {saving ? 'Working...' : type === 'microsoft' ? 'Sign in with Microsoft' : type === 'ely.by' ? 'Sign in' : 'Create profile'}
        </Button>
      </div>
    </div>
  );
}
