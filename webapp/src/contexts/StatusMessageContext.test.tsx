import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { StatusMessageProvider, useStatusMessage } from './StatusMessageContext'

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('StatusMessageContext', () => {
  it('auto-clears a success message after 5s', () => {
    const { result } = renderHook(() => useStatusMessage(), { wrapper: StatusMessageProvider })

    act(() => result.current.showMessage('Saved.', 'success'))
    expect(result.current.text).toBe('Saved.')

    act(() => vi.advanceTimersByTime(5000))
    expect(result.current.text).toBe('')
  })

  it('does not auto-clear an error message', () => {
    const { result } = renderHook(() => useStatusMessage(), { wrapper: StatusMessageProvider })

    act(() => result.current.showMessage('Something broke.', 'error'))
    act(() => vi.advanceTimersByTime(10_000))
    expect(result.current.text).toBe('Something broke.')
  })

  it('a new message cancels the previous auto-clear timer', () => {
    const { result } = renderHook(() => useStatusMessage(), { wrapper: StatusMessageProvider })

    act(() => result.current.showMessage('First.', 'success'))
    act(() => vi.advanceTimersByTime(4000))
    act(() => result.current.showMessage('Second.', 'success'))
    // If the first timer weren't cancelled, this tick (4000+2000=6000ms
    // since "First." was shown, but only 2000ms since "Second.") would
    // incorrectly clear "Second." early.
    act(() => vi.advanceTimersByTime(2000))
    expect(result.current.text).toBe('Second.')
  })
})
