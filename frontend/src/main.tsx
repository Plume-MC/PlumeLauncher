import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { Toaster } from '@/components/ui/toast'

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <Toaster>
      <App />
    </Toaster>
  </React.StrictMode>,
)
