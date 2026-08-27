import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { AccountService } from '../../../bindings/plumelauncher/internal/services/index.js';

interface AccountStepProps {
  onNext: () => void;
  onBack: () => void;
}

export function AccountStep({ onNext, onBack }: AccountStepProps) {
  const [type, setType] = useState<'offline' | 'ely.by'>('offline');
  const [username, setUsername] = useState('Player');
  const [password, setPassword] = useState('');
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
      <div className="text-center space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Add Account</h1>
        <p className="text-sm text-muted-foreground">Choose how you want to play.</p>
      </div>

      <div className="flex gap-2">
        <Button size="sm" variant={type === 'offline' ? 'default' : 'ghost'} onClick={() => setType('offline')}>
          Offline
        </Button>
        <Button size="sm" variant={type === 'ely.by' ? 'default' : 'ghost'} onClick={() => setType('ely.by')}>
          Ely.by
        </Button>
      </div>

      <div className="space-y-3">
        <div className="space-y-1.5">
          <label htmlFor="setup-username" className="text-xs font-medium">Username</label>
          <Input
            id="setup-username"
            value={username}
            maxLength={type === 'offline' ? 16 : undefined}
            onChange={(e) => setUsername(e.target.value)}
            autoFocus
          />
        </div>
        {type === 'ely.by' && (
          <div className="space-y-1.5">
            <label htmlFor="setup-password" className="text-xs font-medium">Password</label>
            <Input
              id="setup-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
            />
          </div>
        )}
      </div>

      {error && <p role="alert" className="text-xs text-destructive">{error}</p>}

      <div className="flex gap-2">
        <Button variant="outline" onClick={onBack} className="flex-1">Back</Button>
        <Button
          onClick={create}
          disabled={saving || !username.trim() || (type === 'ely.by' && !password)}
          className="flex-1"
        >
          {saving ? 'Creating...' : 'Continue'}
        </Button>
      </div>
    </div>
  );
}
