import type { ButtonHTMLAttributes, ReactNode } from 'react'

interface LoadingButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  loading: boolean
  loadingText: string
  children: ReactNode
}

// A controlled-prop equivalent of notify.js's setButtonLoading(): disables
// the button and swaps its label while `loading` is true.
export function LoadingButton({ loading, loadingText, children, disabled, ...rest }: LoadingButtonProps) {
  return (
    <button {...rest} disabled={loading || disabled}>
      {loading ? loadingText : children}
    </button>
  )
}
