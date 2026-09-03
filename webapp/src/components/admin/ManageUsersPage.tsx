import { useCallback, useEffect, useState } from 'react'
import { listUsers } from '../../api/admin'
import { useAuth } from '../../contexts/AuthContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { UserSummary } from '../../types'
import { ManageUserRow } from './ManageUserRow'

export function ManageUsersPage() {
  const { user } = useAuth()

  // The backend is the real gate (RequireAdmin re-verifies on every write
  // regardless of what the client claims) — this just avoids showing a
  // page whose every action would visibly fail for a non-admin who
  // navigates here directly.
  if (user?.role !== 'admin') {
    return <p className="empty-state">Admin access required.</p>
  }

  return <ManageUsersList currentUsername={user.username} />
}

function ManageUsersList({ currentUsername }: { currentUsername: string }) {
  const [users, setUsers] = useState<UserSummary[]>([])
  const [loading, setLoading] = useState(true)
  const { showMessage } = useStatusMessage()

  const load = useCallback(() => {
    setLoading(true)
    listUsers()
      .then(setUsers)
      .catch((error: unknown) => {
        showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
      })
      .finally(() => setLoading(false))
  }, [showMessage])

  useEffect(() => {
    load()
  }, [load])

  return (
    <div id="manage-users">
      <h3>Manage Users</h3>
      {!loading && users.length === 0 && <p className="empty-state">No users yet.</p>}
      <ul className="manage-category-list">
        {users.map((targetUser) => (
          <ManageUserRow key={targetUser.id} targetUser={targetUser} isSelf={targetUser.username === currentUsername} onChanged={load} />
        ))}
      </ul>
    </div>
  )
}
