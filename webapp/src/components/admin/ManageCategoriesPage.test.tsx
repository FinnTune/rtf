import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../../contexts/AuthContext'
import { StatusMessageProvider } from '../../contexts/StatusMessageContext'
import { StatusBanner } from '../common/StatusBanner'
import { ManageCategoriesPage } from './ManageCategoriesPage'

function requestUrl(input: string | URL | Request): string {
  return typeof input === 'string' ? input : input.toString()
}

function checkLoginResponse(role: string) {
  return new Response(
    JSON.stringify({ loggedIn: true, id: 1, username: 'admin', email: 'a@example.com', joined: '2026-01-01', otp: 'x', role }),
    { status: 200 },
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ManageCategoriesPage', () => {
  it('shows "Admin access required" for a non-admin user, without ever fetching categories', async () => {
    const fetchMock = vi.fn().mockResolvedValue(checkLoginResponse('user'))
    vi.stubGlobal('fetch', fetchMock)
    render(
      <StatusMessageProvider>
        <AuthProvider>
          <ManageCategoriesPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )
    expect(await screen.findByText('Admin access required.')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([input]) => requestUrl(input as string | URL | Request).startsWith('/getCategories'))).toBe(
      false,
    )
  })

  it('lets an admin view, create, rename, and delete categories', async () => {
    // A mutable in-memory list backing the mock, so create/rename/delete
    // (and the getCategories() cache each of them busts) are all reflected
    // in the next fetch, exactly like the real backend.
    let categories = [{ id: 1, name: 'Sports' }]
    let nextId = 2

    const fetchMock = vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
      const url = requestUrl(input)
      if (url.startsWith('/checkLogin')) {
        return checkLoginResponse('admin')
      }
      if (url.startsWith('/getCategories')) {
        return new Response(JSON.stringify(categories), { status: 200 })
      }
      if (url.startsWith('/createCategory')) {
        const { name } = JSON.parse(init!.body as string)
        const created = { id: nextId++, name }
        categories = [...categories, created]
        return new Response(JSON.stringify(created), { status: 201 })
      }
      if (url.startsWith('/editCategory')) {
        const { id, name } = JSON.parse(init!.body as string)
        categories = categories.map((c) => (c.id === id ? { id, name } : c))
        return new Response(JSON.stringify({ id, name }), { status: 200 })
      }
      if (url.startsWith('/deleteCategory')) {
        const { id } = JSON.parse(init!.body as string)
        categories = categories.filter((c) => c.id !== id)
        return new Response('Category deleted', { status: 200 })
      }
      throw new Error('Unexpected fetch: ' + url)
    })
    vi.stubGlobal('fetch', fetchMock)

    render(
      <StatusMessageProvider>
        <AuthProvider>
          <StatusBanner />
          <ManageCategoriesPage />
        </AuthProvider>
      </StatusMessageProvider>,
    )

    expect(await screen.findByText('Sports')).toBeInTheDocument()

    // Create
    await userEvent.type(screen.getByPlaceholderText('New category name'), 'Music')
    await userEvent.click(screen.getByRole('button', { name: 'Add Category' }))
    expect(await screen.findByText('Music')).toBeInTheDocument()

    // Rename the newly-created one
    const musicRow = screen.getByText('Music').closest('li')!
    await userEvent.click(within(musicRow).getByRole('button', { name: 'Rename' }))
    const renameInput = within(musicRow).getByDisplayValue('Music')
    await userEvent.clear(renameInput)
    await userEvent.type(renameInput, 'Jazz')
    await userEvent.click(within(musicRow).getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Jazz')).toBeInTheDocument()
    expect(screen.queryByText('Music')).not.toBeInTheDocument()

    // Delete it
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const jazzRow = screen.getByText('Jazz').closest('li')!
    await userEvent.click(within(jazzRow).getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(screen.queryByText('Jazz')).not.toBeInTheDocument())

    // Sports should still be there throughout
    expect(screen.getByText('Sports')).toBeInTheDocument()
  })
})
