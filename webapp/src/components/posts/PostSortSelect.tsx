import type { PostSort } from '../../api/posts'

const SORT_OPTIONS: { value: PostSort; label: string }[] = [
  { value: 'newest', label: 'Newest' },
  { value: 'most_liked', label: 'Most liked' },
  { value: 'most_commented', label: 'Most commented' },
]

export function PostSortSelect({ value, onChange }: { value: PostSort; onChange: (sort: PostSort) => void }) {
  return (
    <select
      className="post-sort-select"
      aria-label="Sort posts by"
      value={value}
      onChange={(event) => onChange(event.target.value as PostSort)}
    >
      {SORT_OPTIONS.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  )
}
