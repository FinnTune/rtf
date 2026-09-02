import { useState, type FormEvent } from 'react'
import { onActivationKey } from '../../a11y'
import { searchMessages } from '../../api/chat'
import { useChat } from '../../contexts/ChatContext'
import { useStatusMessage } from '../../contexts/StatusMessageContext'
import type { MessageSearchResult } from '../../types'

export function MessageSearchPanel() {
  const { openConversationById, getConversationTitle } = useChat()
  const { showMessage } = useStatusMessage()
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<MessageSearchResult[] | null>(null)
  const [searching, setSearching] = useState(false)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    const trimmed = query.trim()
    if (!trimmed) return
    setSearching(true)
    try {
      setResults(await searchMessages(trimmed))
    } catch (error) {
      showMessage('Err: ' + (error instanceof Error ? error.message : String(error)), 'error')
    } finally {
      setSearching(false)
    }
  }

  return (
    <div id="message-search">
      <h3>Search Messages</h3>
      <form onSubmit={(event) => void handleSubmit(event)}>
        <input
          type="text"
          placeholder="Search your messages..."
          aria-label="Search your messages"
          value={query}
          maxLength={100}
          onChange={(event) => setQuery(event.target.value)}
        />
        <button type="submit" className="btns" disabled={!query.trim() || searching}>
          {searching ? 'Searching...' : 'Search'}
        </button>
      </form>

      {results !== null && (
        <ul id="message-search-results">
          {results.length === 0 ? (
            <li className="empty-state">No messages found.</li>
          ) : (
            results.map((result) => (
              <li
                key={result.id}
                role="button"
                tabIndex={0}
                onClick={() => openConversationById(result.conversation_id)}
                onKeyDown={onActivationKey(() => openConversationById(result.conversation_id))}
              >
                <div className="message-search-meta">
                  <strong>{result.from}</strong> in {getConversationTitle(result.conversation_id)}
                </div>
                <div className="message-search-snippet">{result.message}</div>
              </li>
            ))
          )}
        </ul>
      )}
    </div>
  )
}
