import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App.tsx'
import { AuthProvider } from './contexts/AuthContext.tsx'
import { FeedViewProvider } from './contexts/FeedViewContext.tsx'
import { StatusMessageProvider } from './contexts/StatusMessageContext.tsx'
import './style.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <StatusMessageProvider>
        <AuthProvider>
          <FeedViewProvider>
            <App />
          </FeedViewProvider>
        </AuthProvider>
      </StatusMessageProvider>
    </BrowserRouter>
  </StrictMode>,
)
