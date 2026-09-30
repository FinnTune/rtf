import { useState } from 'react'
import { setUserBanned } from '../../api/admin'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { UserSummary } from '../../types'
import { LoadingButton } from '../common/LoadingButton'

interface ManageUserRowProps {
  targetUser: UserSummary
  isSelf: boolean
  onChanged: () => void
}

export function ManageUserRow({ targetUser, isSelf, onChanged }: ManageUserRowProps) {
  const [working, setWorking] = useState(false)
  const { showMessage } = useStatusMessage()

  // The backend is the real gate (SetUserBannedHandler re-checks this
  // regardless of what the client claims) — this just avoids showing a
  // Ban button whose click would visibly fail for a fellow admin's
  // account. Only the ban direction is blocked; an already-banned admin
  // (e.g. from before they were promoted) can still be unbanned.
  const banningBlocked = !targetUser.banned && (isSelf || targetUser.role === 'admin')
  const banningBlockedReason = isSelf
    ? "You can't ban your own account"
    : targetUser.role === 'admin'
      ? "You can't ban another admin account"
      : undefined

  async function handleToggleBanned() {
    const nextBanned = !targetUser.banned
    if (nextBanned && !window.confirm(`Ban ${targetUser.username}? They'll be signed out immediately and unable to log back in.`)) {
      return
    }
    setWorking(true)
    try {
      await setUserBanned(targetUser.id, nextBanned)
      showMessage(nextBanned ? `${targetUser.username} banned.` : `${targetUser.username} unbanned.`, 'success')
      onChanged()
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
    } finally {
      setWorking(false)
    }
  }

  return (
    <li className="manage-category-row">
      <span>
        {targetUser.username} ({targetUser.email}) — {targetUser.role}
        {targetUser.banned && ' — banned'}
      </span>
      <LoadingButton
        type="button"
        className={targetUser.banned ? 'btns' : 'btns btn-danger'}
        loading={working}
        loadingText="Saving..."
        disabled={banningBlocked}
        title={banningBlockedReason}
        onClick={() => void handleToggleBanned()}
      >
        {targetUser.banned ? 'Unban' : 'Ban'}
      </LoadingButton>
    </li>
  )
}
