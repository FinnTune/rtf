interface PaginationProps {
  offset: number
  pageSize: number
  total: number
  loading: boolean
  onNavigate: (offset: number) => void
}

export function Pagination({ offset, pageSize, total, loading, onNavigate }: PaginationProps) {
  const hasMore = offset + pageSize < total
  const rangeStart = offset + 1
  const rangeEnd = Math.min(offset + pageSize, total)

  return (
    <div id="posts-pagination">
      <button
        type="button"
        className="btns"
        disabled={loading || offset === 0}
        onClick={() => onNavigate(Math.max(0, offset - pageSize))}
      >
        Previous
      </button>
      <span className="pagination-range">
        {' '}
        {rangeStart}-{rangeEnd} of {total}{' '}
      </span>
      <button type="button" className="btns" disabled={loading || !hasMore} onClick={() => onNavigate(offset + pageSize)}>
        Next
      </button>
    </div>
  )
}
