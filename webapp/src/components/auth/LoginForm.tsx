import { useState, type FormEvent } from 'react'
import { useAuth } from '../../contexts/AuthContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import { LoadingButton } from '../common/LoadingButton'

export function LoginForm({ onSwitchToRegister }: { onSwitchToRegister: () => void }) {
  const { login } = useAuth()
  const { showMessage } = useStatusMessage()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setSubmitting(true)
    try {
      await login(username, password)
      // No `finally` resetting `submitting`: on success AuthContext's user
      // flips, App swaps to LoggedInShell, and this component unmounts.
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      setSubmitting(false)
    }
  }

  return (
    <form className="login-form" onSubmit={handleSubmit}>
      <label htmlFor="username">Login</label>
      <input
        type="text"
        id="username"
        placeholder="Enter your login"
        required
        value={username}
        onChange={(event) => setUsername(event.target.value)}
      />

      <label htmlFor="password">Password</label>
      <input
        type="password"
        id="password"
        placeholder="Enter your password"
        required
        value={password}
        onChange={(event) => setPassword(event.target.value)}
      />

      <LoadingButton type="submit" loading={submitting} loadingText="Logging in...">
        Login
      </LoadingButton>
      <button type="button" id="register-switch-button" onClick={onSwitchToRegister}>
        Go to registration
      </button>
    </form>
  )
}
