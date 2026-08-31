import { afterEach, describe, expect, it, vi } from 'vitest'
import { requestJson, requestVoid } from './client'

function mockFetch(response: Response) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response))
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('requestJson', () => {
  it('parses the JSON body and X-Total-Count header on success', async () => {
    mockFetch(
      new Response(JSON.stringify([{ Title: 'Hello' }]), {
        status: 200,
        headers: { 'X-Total-Count': '7' },
      }),
    )
    const { data, total } = await requestJson<{ Title: string }[]>('/getAllPosts')
    expect(data).toEqual([{ Title: 'Hello' }])
    expect(total).toBe(7)
  })

  it('reports total as null when X-Total-Count is absent (e.g. search)', async () => {
    mockFetch(new Response(JSON.stringify([]), { status: 200 }))
    const { total } = await requestJson('/searchPosts?q=x')
    expect(total).toBeNull()
  })

  it('throws the server-provided error message on a non-ok response', async () => {
    mockFetch(new Response('username must be 3-30 characters', { status: 400 }))
    await expect(requestJson('/register')).rejects.toThrow('username must be 3-30 characters')
  })

  it('falls back to a generic message when the error body is empty', async () => {
    mockFetch(new Response('', { status: 401 }))
    await expect(requestJson('/login')).rejects.toThrow('Request failed (401)')
  })
})

describe('requestVoid', () => {
  it('resolves without parsing the body on success', async () => {
    mockFetch(new Response('Registration successful.', { status: 200 }))
    await expect(requestVoid('/register')).resolves.toBeUndefined()
  })

  it('throws on a non-ok response, same as requestJson', async () => {
    mockFetch(new Response('Username or email already exists', { status: 409 }))
    await expect(requestVoid('/register')).rejects.toThrow('Username or email already exists')
  })
})
