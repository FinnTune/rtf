import type { ReactNode } from 'react'
import { AuthProvider } from '../contexts/AuthContext'
import { ChatProvider } from '../contexts/ChatContext'
import { StatusMessageProvider } from '../contexts/StatusMessageContext'
import { WebSocketProvider } from '../contexts/WebSocketContext'

// A controllable stand-in for the browser WebSocket, shared by every test
// that needs to drive ChatContext/WebSocketContext — tests simulate the
// server side (open, incoming frames) and inspect what the client sent.
export class ControllableFakeWebSocket {
  static instances: ControllableFakeWebSocket[] = []
  static readonly OPEN = 1
  readyState = 0
  onopen: (() => void) | null = null
  onclose: (() => void) | null = null
  onmessage: ((event: MessageEvent<string>) => void) | null = null
  sent: string[] = []
  url: string
  constructor(url: string) {
    this.url = url
    ControllableFakeWebSocket.instances.push(this)
  }
  send(data: string) {
    this.sent.push(data)
  }
  close() {
    this.onclose?.()
  }
  simulateOpen() {
    this.readyState = 1
    this.onopen?.()
  }
  simulateMessage(type: string, payload: unknown) {
    this.onmessage?.({ data: JSON.stringify({ type, payload }) } as MessageEvent<string>)
  }
}

export function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

export function checkLoginResponse(username: string) {
  return new Response(
    JSON.stringify({
      loggedIn: true,
      id: 1,
      username,
      email: 'a@example.com',
      joined: '2026-01-01',
      otp: 'otp-' + Math.random(),
    }),
    { status: 200 },
  )
}

// The full provider stack ChatContext needs to function (it reads auth
// state and talks through WebSocketContext).
export function chatWrapper({ children }: { children: ReactNode }) {
  return (
    <StatusMessageProvider>
      <AuthProvider>
        <WebSocketProvider>
          <ChatProvider>{children}</ChatProvider>
        </WebSocketProvider>
      </AuthProvider>
    </StatusMessageProvider>
  )
}
