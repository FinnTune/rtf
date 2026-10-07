import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { useDismissibleOpen } from './useDismissibleOpen'

function dispatchEscape() {
  document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
}

describe('useDismissibleOpen', () => {
  it('starts closed', () => {
    const { result } = renderHook(() => useDismissibleOpen())
    expect(result.current.isOpen).toBe(false)
  })

  it('Escape closes it and returns focus to the toggle button', () => {
    const { result } = renderHook(() => useDismissibleOpen())

    // The toggle button has to actually be the ref target and mounted in
    // the document for .focus() to do anything observable.
    const button = document.createElement('button')
    document.body.appendChild(button)
    result.current.toggleRef.current = button

    act(() => result.current.setIsOpen(true))
    expect(result.current.isOpen).toBe(true)

    act(() => dispatchEscape())
    expect(result.current.isOpen).toBe(false)
    expect(document.activeElement).toBe(button)

    document.body.removeChild(button)
  })

  it('Escape is a no-op while already closed (no stray listener firing setIsOpen unnecessarily)', () => {
    const { result } = renderHook(() => useDismissibleOpen())
    act(() => dispatchEscape())
    expect(result.current.isOpen).toBe(false)
  })

  it('ignores every other key', () => {
    const { result } = renderHook(() => useDismissibleOpen())
    act(() => result.current.setIsOpen(true))

    act(() => document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' })))
    expect(result.current.isOpen).toBe(true)
  })
})
