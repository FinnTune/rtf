import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ControllableFakeWebSocket, chatWrapper as wrapper, checkLoginResponse } from '../testUtils/chatTestHarness'
import { useStatusMessage } from './StatusMessageContext'
import { useWebSocket } from './WebSocketContext'

// setup() drives the connection to 'open' using real timers (waitFor polls
// with real setTimeout under the hood, which never fires once fake timers
// are active) — each test switches to vi.useFakeTimers() only afterward, to
// deterministically control the reconnect backoff without waiting out real
// seconds.
async function setup(myUsername = 'alice') {
  ControllableFakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
  vi.stubGlobal('fetch', vi.fn().mockImplementation(() => Promise.resolve(checkLoginResponse(myUsername))))

  const { result, unmount } = renderHook(() => ({ ws: useWebSocket(), status: useStatusMessage() }), { wrapper })
  await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
  const socket = ControllableFakeWebSocket.instances[0]
  act(() => socket.simulateOpen())
  return { result, socket, unmount }
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe('WebSocketContext', () => {
  it('reaches open status and announces itself on connect', async () => {
    const { result, socket } = await setup()

    expect(result.current.ws.status).toBe('open')
    expect(socket.sent.map((frame) => (JSON.parse(frame) as { type: string }).type)).toEqual([
      'user-connect',
      'get-conversations',
    ])
  })

  it('an unexpected close enters reconnecting status and opens a fresh connection with a new OTP after backoff', async () => {
    const { result, socket } = await setup()
    const firstUrl = socket.url

    vi.useFakeTimers()
    act(() => socket.close())

    expect(result.current.ws.status).toBe('reconnecting')
    expect(result.current.status.text).toBe('Connection lost. Reconnecting...')

    // First reconnect attempt backs off 1000ms * attempt(1).
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })

    expect(ControllableFakeWebSocket.instances).toHaveLength(2)
    const newSocket = ControllableFakeWebSocket.instances[1]
    // A fresh OTP is minted for the reconnect rather than reusing the
    // original (single-use) one.
    expect(newSocket.url).not.toBe(firstUrl)

    act(() => newSocket.simulateOpen())
    expect(result.current.ws.status).toBe('open')
    expect(result.current.status.text).toBe('Reconnected.')
  })

  it('an intentional close (e.g. unmount/logout) does not trigger a reconnect', async () => {
    const { unmount } = await setup()

    vi.useFakeTimers()
    unmount()

    // Well beyond any backoff window, to prove no reconnect was ever scheduled.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10_000)
    })

    expect(ControllableFakeWebSocket.instances).toHaveLength(1)
  })

  it('gives up after the reconnect attempt budget and tells the user to refresh', async () => {
    const { result } = await setup()

    vi.useFakeTimers()

    // MAX_RECONNECT_ATTEMPTS is 3: each of these three unexpected closes
    // schedules and completes one more reconnect (backoff = 1000ms *
    // attempt number), leaving a new, never-opened socket behind each time.
    const backoffsMs = [1000, 2000, 3000]
    for (const backoff of backoffsMs) {
      const current = ControllableFakeWebSocket.instances[ControllableFakeWebSocket.instances.length - 1]
      act(() => current.close())
      await act(async () => {
        await vi.advanceTimersByTimeAsync(backoff)
      })
    }
    expect(ControllableFakeWebSocket.instances).toHaveLength(4)

    // A 4th unexpected close exhausts the budget — it gives up synchronously
    // instead of scheduling yet another attempt.
    const exhausted = ControllableFakeWebSocket.instances[3]
    act(() => exhausted.close())

    expect(result.current.status.text).toBe('Connection lost. Please refresh the page.')
    expect(ControllableFakeWebSocket.instances).toHaveLength(4)
  })

  it("tells the user their session ended if reconnecting's session check reports them logged out", async () => {
    let loggedIn = true
    ControllableFakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', ControllableFakeWebSocket)
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() =>
        Promise.resolve(
          loggedIn ? checkLoginResponse('alice') : new Response(JSON.stringify({ loggedIn: false }), { status: 200 }),
        ),
      ),
    )

    const { result } = renderHook(() => ({ ws: useWebSocket(), status: useStatusMessage() }), { wrapper })
    await waitFor(() => expect(ControllableFakeWebSocket.instances.length).toBe(1))
    act(() => ControllableFakeWebSocket.instances[0].simulateOpen())

    vi.useFakeTimers()
    loggedIn = false
    act(() => ControllableFakeWebSocket.instances[0].close())
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })

    expect(result.current.status.text).toBe('Your session has ended. Please log in again.')
    // No fresh connection is opened once the session check itself reports logged-out.
    expect(ControllableFakeWebSocket.instances).toHaveLength(1)
  })
})
