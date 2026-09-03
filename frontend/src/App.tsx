import './index.css'
import { useEffect, useState } from 'react'
import { AppShell } from '@/layouts/AppShell'
import { Home } from '@/screens/Home'
import { SetupWizard } from '@/components/setup/SetupWizard'
import { AccountService, HomeService, SystemService } from '../bindings/plumelauncher/internal/services/index.js'
import type { Account } from '../bindings/plumelauncher/internal/services/models.js'
import type { Instance } from '../bindings/plumelauncher/internal/instances/models.js'
import plumeMark from '@/assets/plume-mark.webp'

function App() {
  const [account, setAccount] = useState<Account | null>(null)
  const [accounts, setAccounts] = useState<Account[]>([])
  const [instances, setInstances] = useState<Instance[]>([])
  const [setup, setSetup] = useState(true)
  const [loading, setLoading] = useState(true)

  const refresh = async () => {
    const nextAccounts = (await AccountService.ListAccounts()) ?? []
    setAccounts(nextAccounts)
    setAccount(nextAccounts.find((item) => item.selected) ?? nextAccounts[0] ?? null)
    setInstances((await HomeService.ListInstances()) ?? [])
    setSetup((current) => current && nextAccounts.length === 0)
  }

  useEffect(() => {
    Promise.all([SystemService.GetSettings(), refresh()]).then(([settings]) => {
      document.documentElement.classList.toggle('dark', settings.theme !== 'light')
    }).catch(() => setSetup(true)).finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="grid min-h-[100dvh] place-items-center gap-3 text-sm text-muted-foreground">
        <img src={plumeMark} alt="Plume Launcher" className="size-12 rounded-xl" />
        <span>Loading Plume Launcher...</span>
      </div>
    )
  }

  if (setup) {
    return <SetupWizard onComplete={() => refresh().then(() => setSetup(false))} />
  }

  return (
    <AppShell account={account} accounts={accounts} onAccountsChanged={refresh}>
      <Home account={account} instances={instances} onRefresh={refresh} />
    </AppShell>
  )
}

export default App
