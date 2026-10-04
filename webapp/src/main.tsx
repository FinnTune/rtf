import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App.tsx'
import { ErrorBoundary } from './components/common/ErrorBoundary.tsx'
import { AuthProvider } from './contexts/AuthContext.tsx'
import { ChatProvider } from './contexts/ChatContext.tsx'
import { FeedViewProvider } from './contexts/FeedViewContext.tsx'
import { NotificationsProvider } from './contexts/NotificationsContext.tsx'
import { StatusMessageProvider } from './contexts/StatusMessageContext.tsx'
import { WebSocketProvider } from './contexts/WebSocketContext.tsx'
import './style.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ErrorBoundary>
      <BrowserRouter>
        <StatusMessageProvider>
          <AuthProvider>
            <WebSocketProvider>
              <ChatProvider>
                <NotificationsProvider>
                  <FeedViewProvider>
                    <App />
                  </FeedViewProvider>
                </NotificationsProvider>
              </ChatProvider>
            </WebSocketProvider>
          </AuthProvider>
        </StatusMessageProvider>
      </BrowserRouter>
    </ErrorBoundary>
  </StrictMode>,
)
