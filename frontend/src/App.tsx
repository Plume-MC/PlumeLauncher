import './index.css'
import { useEffect } from 'react'
import { AppShell } from '@/layouts/AppShell'
import { Home } from '@/screens/Home'

function App() {
  useEffect(() => {
    document.documentElement.classList.add('dark')
  }, [])

  return (
    <AppShell>
      <Home />
    </AppShell>
  )
}

export default App
