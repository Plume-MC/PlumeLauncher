import './index.css'
import { lazy, Suspense, useEffect, useState } from 'react'
import { AccountService, HomeService, SystemService } from '../bindings/plumelauncher/internal/services/index.js'
import type { Account } from '../bindings/plumelauncher/internal/services/models.js'
import type { Instance } from '../bindings/plumelauncher/internal/instances/models.js'
import plumeMark from '@/assets/plume-mark.webp'

const AppShell = lazy(() => import('@/layouts/AppShell').then((module) => ({ default: module.AppShell })))
const Home = lazy(() => import('@/screens/Home').then((module) => ({ default: module.Home })))
const SetupWizard = lazy(() => import('@/components/setup/SetupWizard').then((module) => ({ default: module.SetupWizard })))

function LoadingScreen() {
  return <div className="grid min-h-[100dvh] place-items-center text-sm text-muted-foreground">Loading...</div>
}

function App() {
  const [account, setAccount] = useState<Account | null>(null)
  const [accounts, setAccounts] = useState<Account[]>([])
  const [instances, setInstances] = useState<Instance[]>([])
  const [setup, setSetup] = useState(true)
  const [loading, setLoading] = useState(true)

  const refresh = async () => {
    const [accountsResult, instancesResult] = await Promise.all([
      AccountService.ListAccounts(),
      HomeService.ListInstances(),
    ])
    const nextAccounts = accountsResult ?? []
    setAccounts(nextAccounts)
    setAccount(nextAccounts.find((item) => item.selected) ?? nextAccounts[0] ?? null)
    setInstances(instancesResult ?? [])
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
    return <Suspense fallback={<LoadingScreen />}><SetupWizard onComplete={() => refresh().then(() => setSetup(false))} /></Suspense>
  }

  return (
    <Suspense fallback={<LoadingScreen />}>
      <AppShell account={account} accounts={accounts} onAccountsChanged={refresh}>
        <Suspense fallback={<LoadingScreen />}>
          <Home account={account} instances={instances} onRefresh={refresh} />
        </Suspense>
      </AppShell>
    </Suspense>
  )
}

export default App
