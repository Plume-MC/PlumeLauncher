import './index.css'
import { useEffect, useState } from 'react'
import { AppShell } from '@/layouts/AppShell'
import { Home } from '@/screens/Home'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { AccountService, HomeService, SystemService } from '../bindings/plumelauncher/internal/services/index.js'
import type { Account } from '../bindings/plumelauncher/internal/services/models.js'
import type { Instance } from '../bindings/plumelauncher/internal/instances/models.js'

function App() {
  const [account, setAccount] = useState<Account | null>(null)
  const [accounts, setAccounts] = useState<Account[]>([])
  const [instances, setInstances] = useState<Instance[]>([])
  const [setup, setSetup] = useState(true)
  const [loading, setLoading] = useState(true)

  const refresh = async () => {
    const accounts = await AccountService.ListAccounts()
    setAccounts(accounts)
    setAccount(accounts.find((item) => item.selected) ?? accounts[0] ?? null)
    setInstances((await HomeService.ListInstances()) ?? [])
    setSetup(accounts.length === 0)
  }

  useEffect(() => {
    Promise.all([SystemService.GetSettings(), refresh()]).then(([settings]) => {
      document.documentElement.classList.toggle('dark', settings.theme !== 'light')
    }).catch(() => setSetup(true)).finally(() => setLoading(false))
  }, [])

  if (loading) {
    return <div className="grid min-h-[100dvh] place-items-center text-sm text-muted-foreground">Loading Plume Launcher...</div>
  }

  if (setup) {
    return <SetupGate onComplete={() => refresh().then(() => setSetup(false))} />
  }

  return (
    <AppShell>
      <Home account={account} accounts={accounts} instances={instances} onRefresh={refresh} />
    </AppShell>
  )
}

function SetupGate({ onComplete }: { onComplete: () => void }) {
  const [username, setUsername] = useState('Player')
  const [type, setType] = useState<'offline' | 'ely.by'>('offline')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const createAccount = async () => {
    setSaving(true)
    setError('')
    try {
      if (type === 'offline') await AccountService.CreateOffline(username.trim())
      else await AccountService.LoginElyBy(username.trim(), password)
      await onComplete()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to create account')
    } finally {
      setSaving(false)
    }
  }

  return (
    <main className="grid min-h-[100dvh] place-items-center bg-background px-5 text-foreground">
      <section className="w-full max-w-md space-y-6 rounded-lg border border-border bg-card p-6 shadow-sm" aria-labelledby="setup-title">
        <div>
          <p className="text-xs font-medium uppercase tracking-wider text-primary">Plume Launcher</p>
          <h1 id="setup-title" className="mt-2 text-2xl font-semibold tracking-tight">Set up your launcher</h1>
          <p className="mt-2 text-sm text-muted-foreground">Choose an offline profile or sign in with Ely.by.</p>
        </div>
        <div className="flex gap-2"><Button type="button" size="sm" variant={type === 'offline' ? 'default' : 'ghost'} onClick={() => setType('offline')}>Offline</Button><Button type="button" size="sm" variant={type === 'ely.by' ? 'default' : 'ghost'} onClick={() => setType('ely.by')}>Ely.by</Button></div>
        <div className="space-y-2">
          <label htmlFor="offline-username" className="text-sm font-medium">Username</label>
          <Input id="offline-username" value={username} maxLength={type === 'offline' ? 16 : undefined} onChange={(event) => setUsername(event.target.value)} autoFocus />
        </div>
        {type === 'ely.by' && <div className="space-y-2"><label htmlFor="ely-password" className="text-sm font-medium">Password</label><Input id="ely-password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="current-password" /></div>}
        {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
        <Button className="w-full" onClick={createAccount} disabled={saving || !username.trim() || (type === 'ely.by' && !password)}>
          {saving ? 'Creating account...' : 'Continue'}
        </Button>
      </section>
    </main>
  )
}

export default App
