import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App.tsx'
import { AuthProvider } from './contexts/AuthContext.tsx'
import { ChatProvider } from './contexts/ChatContext.tsx'
import { FeedViewProvider } from './contexts/FeedViewContext.tsx'
import { StatusMessageProvider } from './contexts/StatusMessageContext.tsx'
import { WebSocketProvider } from './contexts/WebSocketContext.tsx'
import './style.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <StatusMessageProvider>
        <AuthProvider>
          <WebSocketProvider>
            <ChatProvider>
              <FeedViewProvider>
                <App />
              </FeedViewProvider>
            </ChatProvider>
          </WebSocketProvider>
        </AuthProvider>
      </StatusMessageProvider>
    </BrowserRouter>
  </StrictMode>,
)
