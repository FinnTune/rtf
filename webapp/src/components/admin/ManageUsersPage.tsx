import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { listUsers } from '../../api/admin'
import { useAuth } from '../../contexts/AuthContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { UserSummary } from '../../types'
import { Pagination } from '../posts/Pagination'
import { ManageUserRow } from './ManageUserRow'

const USERS_PAGE_SIZE = 20

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
  const [offset, setOffset] = useState(0)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [query, setQuery] = useState('')
  const { showMessage } = useStatusMessage()

  const load = useCallback(
    (targetOffset: number, targetQuery: string) => {
      setLoading(true)
      listUsers(targetOffset, USERS_PAGE_SIZE, targetQuery || undefined)
        .then((result) => {
          setUsers(result.users)
          // Falls back to the page's own length if the backend ever omits
          // X-Total-Count (it doesn't today, but matches how every other
          // paginated list in this app treats a null total defensively).
          setTotal(result.total ?? result.users.length)
          setOffset(targetOffset)
        })
        .catch((error: unknown) => {
          showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
        })
        .finally(() => setLoading(false))
    },
    [showMessage],
  )

  useEffect(() => {
    load(0, '')
    // eslint-disable-next-line react-hooks/exhaustive-deps -- load only on mount, like every other paginated list's initial fetch
  }, [])

  // A new search always starts back at offset 0 — the previous page's
  // offset is almost certainly meaningless against the new, filtered
  // result set.
  function handleSearchSubmit(event: FormEvent) {
    event.preventDefault()
    load(0, query)
  }

  return (
    <div id="manage-users">
      <h3>Manage Users</h3>
      <form className="manage-users-search" onSubmit={handleSearchSubmit}>
        <input
          type="text"
          aria-label="Search users by username or email"
          placeholder="Search by username or email…"
          maxLength={100}
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
        <button type="submit" className="btns">
          Search
        </button>
      </form>
      {!loading && users.length === 0 && <p className="empty-state">No users found.</p>}
      <ul className="manage-category-list">
        {users.map((targetUser) => (
          <ManageUserRow
            key={targetUser.id}
            targetUser={targetUser}
            isSelf={targetUser.username === currentUsername}
            onChanged={() => load(offset, query)}
          />
        ))}
      </ul>
      {total > USERS_PAGE_SIZE && (
        <Pagination offset={offset} pageSize={USERS_PAGE_SIZE} total={total} loading={loading} onNavigate={(nextOffset) => load(nextOffset, query)} />
      )}
    </div>
  )
}
