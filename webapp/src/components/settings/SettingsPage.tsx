import { useEffect, useState, type FormEvent } from 'react'
import { getProfile, updatePassword, updateProfile, type Profile } from '../../api/profile'
import { useAuth } from '../../contexts/AuthContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import { LoadingButton } from '../common/LoadingButton'

export function SettingsPage() {
  const [profile, setProfile] = useState<Profile | null>(null)
  const [loading, setLoading] = useState(true)
  const { showMessage } = useStatusMessage()

  useEffect(() => {
    let cancelled = false
    getProfile()
      .then((result) => {
        if (!cancelled) setProfile(result)
      })
      .catch((error: unknown) => {
        showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [showMessage])

  if (loading) {
    return <p>Loading…</p>
  }
  if (!profile) {
    return <p className="empty-state">Unable to load your profile.</p>
  }

  return (
    <div id="settings-page">
      <h3>Profile</h3>
      <ProfileForm initialProfile={profile} />
      <h3>Password</h3>
      <PasswordForm />
    </div>
  )
}

function ProfileForm({ initialProfile }: { initialProfile: Profile }) {
  const { refresh } = useAuth()
  const { showMessage } = useStatusMessage()
  const [fname, setFname] = useState(initialProfile.fname)
  const [lname, setLname] = useState(initialProfile.lname)
  const [email, setEmail] = useState(initialProfile.email)
  const [saving, setSaving] = useState(false)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setSaving(true)
    try {
      await updateProfile({ fname, lname, email })
      // AuthContext's own `user.email` was set at login and won't
      // otherwise reflect this change until the next full login (see
      // UpdateProfileHandler's comment on why it updates the server-side
      // in-memory client too) — refresh() re-runs checkLogin, which does.
      await refresh()
      showMessage('Profile updated.', 'success')
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <form className="settings-form" onSubmit={handleSubmit}>
      <label htmlFor="settings-fname">First Name: </label>
      <input
        type="text"
        id="settings-fname"
        maxLength={50}
        required
        value={fname}
        onChange={(event) => setFname(event.target.value)}
      />

      <label htmlFor="settings-lname">Last Name: </label>
      <input
        type="text"
        id="settings-lname"
        maxLength={50}
        required
        value={lname}
        onChange={(event) => setLname(event.target.value)}
      />

      <label htmlFor="settings-email">Email: </label>
      <input
        type="email"
        id="settings-email"
        maxLength={254}
        required
        value={email}
        onChange={(event) => setEmail(event.target.value)}
      />

      <LoadingButton type="submit" className="btns btn-primary" loading={saving} loadingText="Saving...">
        Save Profile
      </LoadingButton>
    </form>
  )
}

function PasswordForm() {
  const { showMessage } = useStatusMessage()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [saving, setSaving] = useState(false)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    if (newPassword !== confirmPassword) {
      showMessage('New passwords do not match', 'error')
      return
    }
    setSaving(true)
    try {
      await updatePassword(currentPassword, newPassword)
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      showMessage('Password updated.', 'success')
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <form className="settings-form" onSubmit={handleSubmit}>
      <label htmlFor="settings-current-password">Current Password: </label>
      <input
        type="password"
        id="settings-current-password"
        minLength={8}
        maxLength={72}
        required
        value={currentPassword}
        onChange={(event) => setCurrentPassword(event.target.value)}
      />

      <label htmlFor="settings-new-password">New Password: </label>
      <input
        type="password"
        id="settings-new-password"
        minLength={8}
        maxLength={72}
        required
        value={newPassword}
        onChange={(event) => setNewPassword(event.target.value)}
      />

      <label htmlFor="settings-confirm-password">Confirm New Password: </label>
      <input
        type="password"
        id="settings-confirm-password"
        minLength={8}
        maxLength={72}
        required
        value={confirmPassword}
        onChange={(event) => setConfirmPassword(event.target.value)}
      />

      <LoadingButton type="submit" className="btns btn-primary" loading={saving} loadingText="Saving...">
        Change Password
      </LoadingButton>
    </form>
  )
}
