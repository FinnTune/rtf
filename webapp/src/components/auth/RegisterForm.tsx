import { useState, type FormEvent } from 'react'
import { useAuth } from '../../contexts/AuthContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import { LoadingButton } from '../common/LoadingButton'

interface RegisterFormProps {
  onSwitchToLogin: () => void
  onRegistered: () => void
}

const initialFields = {
  fname: '',
  lname: '',
  uname: '',
  email: '',
  age: '',
  gender: 'male',
  password: '',
}

export function RegisterForm({ onSwitchToLogin, onRegistered }: RegisterFormProps) {
  const { register } = useAuth()
  const { showMessage } = useStatusMessage()
  const [fields, setFields] = useState(initialFields)
  const [confirmPassword, setConfirmPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)

  function setField<K extends keyof typeof fields>(key: K, value: (typeof fields)[K]) {
    setFields((prev) => ({ ...prev, [key]: value }))
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    if (fields.password !== confirmPassword) {
      showMessage('Passwords do not match', 'error')
      return
    }
    setSubmitting(true)
    try {
      await register(fields)
      showMessage('You are now registered. Please login.', 'success')
      onRegistered()
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form className="register-form" onSubmit={handleSubmit}>
      <label htmlFor="regfname">First Name: </label>
      <input
        type="text"
        id="regfname"
        placeholder="First Name"
        maxLength={50}
        required
        value={fields.fname}
        onChange={(event) => setField('fname', event.target.value)}
      />

      <label htmlFor="reglname">Last Name: </label>
      <input
        type="text"
        id="reglname"
        placeholder="Last Name"
        maxLength={50}
        required
        value={fields.lname}
        onChange={(event) => setField('lname', event.target.value)}
      />

      <label htmlFor="reguname">Username: </label>
      <input
        type="text"
        id="reguname"
        placeholder="Username"
        minLength={3}
        maxLength={30}
        required
        value={fields.uname}
        onChange={(event) => setField('uname', event.target.value)}
      />

      <label htmlFor="regemail">Email: </label>
      <input
        type="email"
        id="regemail"
        placeholder="Email"
        maxLength={254}
        required
        value={fields.email}
        onChange={(event) => setField('email', event.target.value)}
      />

      <label htmlFor="regage">Age: </label>
      <input
        type="number"
        id="regage"
        placeholder="Age"
        min={13}
        max={120}
        required
        value={fields.age}
        onChange={(event) => setField('age', event.target.value)}
      />

      <label htmlFor="reggender">Gender: </label>
      <select id="reggender" value={fields.gender} onChange={(event) => setField('gender', event.target.value)}>
        <option value="male">Male</option>
        <option value="female">Female</option>
        <option value="other">Other</option>
      </select>

      <label htmlFor="regpassword">Password: </label>
      <input
        type="password"
        id="regpassword"
        placeholder="Password"
        minLength={8}
        maxLength={72}
        required
        value={fields.password}
        onChange={(event) => setField('password', event.target.value)}
      />

      <label htmlFor="regconfpassword">Confirm Password: </label>
      <input
        type="password"
        id="regconfpassword"
        placeholder="Confirm Password"
        minLength={8}
        maxLength={72}
        required
        value={confirmPassword}
        onChange={(event) => setConfirmPassword(event.target.value)}
      />

      <LoadingButton type="submit" loading={submitting} loadingText="Registering...">
        Register
      </LoadingButton>
      <button type="button" id="login-switch-button" onClick={onSwitchToLogin}>
        Go to login
      </button>
    </form>
  )
}
