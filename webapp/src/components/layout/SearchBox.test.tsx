import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { SearchBox } from './SearchBox'

describe('SearchBox', () => {
  it('does not submit when the query trims to empty', async () => {
    const onSubmit = vi.fn()
    render(<SearchBox showClear={false} onSubmit={onSubmit} onClear={vi.fn()} />)

    await userEvent.type(screen.getByLabelText('Search posts'), '   ')
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))

    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('submits the trimmed query', async () => {
    const onSubmit = vi.fn()
    render(<SearchBox showClear={false} onSubmit={onSubmit} onClear={vi.fn()} />)

    await userEvent.type(screen.getByLabelText('Search posts'), '  hello world  ')
    await userEvent.click(screen.getByRole('button', { name: 'Search' }))

    expect(onSubmit).toHaveBeenCalledWith('hello world')
  })

  it('submits on Enter, not just the Search button', async () => {
    const onSubmit = vi.fn()
    render(<SearchBox showClear={false} onSubmit={onSubmit} onClear={vi.fn()} />)

    await userEvent.type(screen.getByLabelText('Search posts'), 'hello{Enter}')

    expect(onSubmit).toHaveBeenCalledWith('hello')
  })

  it('only renders Clear when showClear is true, and wires it to onClear', async () => {
    const onClear = vi.fn()
    const { rerender } = render(<SearchBox showClear={false} onSubmit={vi.fn()} onClear={onClear} />)
    expect(screen.queryByRole('button', { name: 'Clear' })).not.toBeInTheDocument()

    rerender(<SearchBox showClear={true} onSubmit={vi.fn()} onClear={onClear} />)
    await userEvent.click(screen.getByRole('button', { name: 'Clear' }))
    expect(onClear).toHaveBeenCalledTimes(1)
  })
})
