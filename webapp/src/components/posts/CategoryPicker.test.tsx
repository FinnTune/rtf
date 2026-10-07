import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { PostCategoryRef } from '../../api/posts'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { CategoryPicker } from './CategoryPicker'

vi.mock('../../api/categories', () => ({
  getCategories: vi.fn(),
  subscribeToCategoryChanges: vi.fn(() => vi.fn()),
}))
import { getCategories, subscribeToCategoryChanges } from '../../api/categories'

const categories = [
  { id: 1, name: 'Sports' },
  { id: 2, name: 'Tech' },
]

beforeEach(() => {
  vi.mocked(getCategories).mockReset().mockResolvedValue(categories)
  vi.mocked(subscribeToCategoryChanges).mockReset().mockReturnValue(vi.fn())
})

function renderPicker(selected: PostCategoryRef[] = []) {
  const onChange = vi.fn()
  render(
    <StatusMessageProvider>
      <CategoryPicker selected={selected} onChange={onChange} />
    </StatusMessageProvider>,
  )
  return { onChange }
}

describe('CategoryPicker', () => {
  it('keeps the list collapsed until the toggle is clicked', async () => {
    renderPicker()
    expect(screen.queryByText('Sports')).not.toBeInTheDocument()
    await userEvent.click(screen.getByText('Select Categories>>'))
    expect(await screen.findByText('Sports')).toBeInTheDocument()
  })

  it('checks exactly the categories already in `selected`', async () => {
    renderPicker([{ id: 2, name: 'Tech' }])
    await userEvent.click(screen.getByText('Select Categories>>'))
    expect(await screen.findByLabelText('Tech')).toBeChecked()
    expect(screen.getByLabelText('Sports')).not.toBeChecked()
  })

  it('adds an unselected category on check', async () => {
    const { onChange } = renderPicker([{ id: 2, name: 'Tech' }])
    await userEvent.click(screen.getByText('Select Categories>>'))
    await userEvent.click(await screen.findByLabelText('Sports'))
    expect(onChange).toHaveBeenCalledWith([
      { id: 2, name: 'Tech' },
      { id: 1, name: 'Sports' },
    ])
  })

  it('removes an already-selected category on uncheck', async () => {
    const { onChange } = renderPicker([
      { id: 1, name: 'Sports' },
      { id: 2, name: 'Tech' },
    ])
    await userEvent.click(screen.getByText('Select Categories>>'))
    await userEvent.click(await screen.findByLabelText('Tech'))
    expect(onChange).toHaveBeenCalledWith([{ id: 1, name: 'Sports' }])
  })

  it('Escape closes the dropdown and returns focus to the toggle button', async () => {
    renderPicker()
    const toggle = screen.getByText('Select Categories>>')
    await userEvent.click(toggle)
    await screen.findByText('Sports')

    await userEvent.keyboard('{Escape}')
    expect(screen.queryByText('Sports')).not.toBeInTheDocument()
    expect(toggle).toHaveFocus()
  })

  it('reloads the list when notified of an external category change', async () => {
    renderPicker()
    await userEvent.click(screen.getByText('Select Categories>>'))
    await screen.findByText('Sports')
    expect(getCategories).toHaveBeenCalledTimes(1)

    vi.mocked(getCategories).mockResolvedValue([{ id: 3, name: 'Food' }])
    const listener = vi.mocked(subscribeToCategoryChanges).mock.calls[0][0]
    await act(async () => listener())

    expect(await screen.findByText('Food')).toBeInTheDocument()
    expect(screen.queryByText('Sports')).not.toBeInTheDocument()
  })
})
