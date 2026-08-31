import { useState } from 'react'
import { LoginForm } from '../auth/LoginForm'
import { RegisterForm } from '../auth/RegisterForm'
import { StatusBanner } from '../common/StatusBanner'

type View = 'intro' | 'login' | 'register'

export function LoggedOutShell() {
  const [view, setView] = useState<View>('intro')

  return (
    <>
      <header className="header">
        <h1 id="title">
          <a
            href="/"
            onClick={(event) => {
              event.preventDefault()
              setView('intro')
            }}
          >
            theDialectic
          </a>
        </h1>
        <button type="button" className="header-btns" onClick={() => setView('register')}>
          Register
        </button>
        <button type="button" className="header-btns" onClick={() => setView('login')}>
          Login
        </button>
      </header>

      <StatusBanner />

      {view === 'intro' && (
        <div className="intro" id="intro">
          <h2>Welcome to theDialectic</h2>
          <p>
            Please register and login to enter the converse universe.
            <br />
            <br />
          </p>
        </div>
      )}

      {view === 'login' && <LoginForm onSwitchToRegister={() => setView('register')} />}

      {view === 'register' && (
        <RegisterForm onSwitchToLogin={() => setView('login')} onRegistered={() => setView('login')} />
      )}
    </>
  )
}
