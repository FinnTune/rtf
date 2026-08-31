import { useStatusMessage } from '../../contexts/StatusMessageContext'

// The #msg equivalent — a single global status banner. style.css collapses
// it to nothing via `#msg:empty` when there's no text, so an idle banner
// takes up no visible space.
export function StatusBanner() {
  const { text, type } = useStatusMessage()
  return (
    <div id="msg" className={`msg-${type}`}>
      {text}
    </div>
  )
}
