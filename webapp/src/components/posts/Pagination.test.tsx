import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Pagination } from './Pagination'

describe('Pagination', () => {
  it('disables Previous on the first page and enables Next when there is more', () => {
    render(<Pagination offset={0} pageSize={10} total={25} loading={false} onNavigate={vi.fn()} />)
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Next' })).toBeEnabled()
    expect(screen.getByText('1-10 of 25')).toBeInTheDocument()
  })

  it('disables Next on the last (partial) page and enables Previous', () => {
    render(<Pagination offset={20} pageSize={10} total={25} loading={false} onNavigate={vi.fn()} />)
    expect(screen.getByRole('button', { name: 'Previous' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
    expect(screen.getByText('21-25 of 25')).toBeInTheDocument()
  })

  it('disables both buttons while loading, even mid-list', () => {
    render(<Pagination offset={10} pageSize={10} total={25} loading={true} onNavigate={vi.fn()} />)
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
  })

  it('Next/Previous call onNavigate with the correct offset', async () => {
    const onNavigate = vi.fn()
    render(<Pagination offset={10} pageSize={10} total={25} loading={false} onNavigate={onNavigate} />)
    await userEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(onNavigate).toHaveBeenCalledWith(20)
    await userEvent.click(screen.getByRole('button', { name: 'Previous' }))
    expect(onNavigate).toHaveBeenCalledWith(0)
  })
})
