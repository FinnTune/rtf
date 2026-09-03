import { Component, type ErrorInfo, type ReactNode } from 'react'

interface ErrorBoundaryProps {
  children: ReactNode
}

interface ErrorBoundaryState {
  hasError: boolean
}

// React error boundaries have no hook equivalent — this has to be a class
// component. Sits once at the top of the tree (see main.tsx) so a render
// error anywhere (a malformed post, a bad chat payload, ...) shows a
// recoverable fallback instead of unmounting the whole app to a blank page.
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false }

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Unhandled error in component tree:', error, info.componentStack)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="empty-state">
          <p>Something went wrong.</p>
          <button type="button" className="btns btn-primary" onClick={() => window.location.reload()}>
            Reload page
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
