import { useEffect, useRef, useState } from 'react'

// Shared by every custom dropdown-style toggle (CategoryPicker,
// NotificationsBell): Escape closes it and returns focus to the toggle
// button that opened it, matching native <select>/<details> keyboard
// behavior. Without this, a keyboard user who opens one of these has no
// quick way to dismiss it short of tabbing all the way through its content
// (or shift-tabbing back past it) to reach the toggle again.
export function useDismissibleOpen() {
  const [isOpen, setIsOpen] = useState(false)
  const toggleRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!isOpen) return
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key !== 'Escape') return
      setIsOpen(false)
      toggleRef.current?.focus()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen])

  return { isOpen, setIsOpen, toggleRef }
}
