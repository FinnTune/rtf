import { useState, type FormEvent } from 'react'

interface SearchBoxProps {
  showClear: boolean
  onSubmit: (query: string) => void
  onClear: () => void
}

// Its own component so it can be `key`-remounted from Topbar (see there) —
// that's what resets the typed text when the feed view is cleared
// elsewhere, without syncing state via an effect.
export function SearchBox({ showClear, onSubmit, onClear }: SearchBoxProps) {
  const [query, setQuery] = useState('')

  function handleSubmit(event: FormEvent) {
    event.preventDefault()
    const trimmed = query.trim()
    if (!trimmed) {
      return
    }
    onSubmit(trimmed)
  }

  return (
    <form id="search-form" className="topbar-search" onSubmit={handleSubmit}>
      <input
        type="text"
        id="search-input"
        name="q"
        placeholder="Search posts..."
        aria-label="Search posts"
        maxLength={100}
        value={query}
        onChange={(event) => setQuery(event.target.value)}
      />
      <button type="submit" className="btns" id="search-submit-button">
        Search
      </button>
      {showClear && (
        <button type="button" className="btns" id="clear-search-button" onClick={onClear}>
          Clear
        </button>
      )}
    </form>
  )
}
