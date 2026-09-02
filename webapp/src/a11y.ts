import type { KeyboardEvent } from 'react'

// For elements given role="button" on something that isn't a real <button>
// (native buttons get Enter/Space activation for free; ARIA roles don't).
export function onActivationKey(handler: () => void) {
  return (event: KeyboardEvent) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      handler()
    }
  }
}
