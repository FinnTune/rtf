import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from 'react'

export type MessageType = 'info' | 'success' | 'error'

interface StatusMessageValue {
  text: string
  type: MessageType
  showMessage: (text: string, type?: MessageType) => void
}

const StatusMessageContext = createContext<StatusMessageValue | null>(null)

export function StatusMessageProvider({ children }: { children: ReactNode }) {
  const [text, setText] = useState('')
  const [type, setType] = useState<MessageType>('info')
  const clearTimer = useRef<number | null>(null)

  const showMessage = useCallback((newText: string, newType: MessageType = 'info') => {
    if (clearTimer.current !== null) {
      window.clearTimeout(clearTimer.current)
      clearTimer.current = null
    }
    setText(newText)
    setType(newType)
    // Errors stay until the next message; success/info messages self-clear,
    // matching notify.js's original behavior.
    if (newType !== 'error') {
      clearTimer.current = window.setTimeout(() => {
        setText('')
      }, 5000)
    }
  }, [])

  return <StatusMessageContext.Provider value={{ text, type, showMessage }}>{children}</StatusMessageContext.Provider>
}

export function useStatusMessage(): StatusMessageValue {
  const ctx = useContext(StatusMessageContext)
  if (!ctx) {
    throw new Error('useStatusMessage must be used within a StatusMessageProvider')
  }
  return ctx
}
