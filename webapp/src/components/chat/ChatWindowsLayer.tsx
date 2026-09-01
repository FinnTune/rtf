import { useChat } from '../../contexts/ChatContext'
import { ChatWindow } from './ChatWindow'

// Rendered as a sibling inside #main, same region the vanilla app appended
// chat windows to — relies on the unchanged `.chat-window { float: right }`
// CSS for multi-window layout, no JS-computed positioning needed.
export function ChatWindowsLayer() {
  const { openWindows } = useChat()

  return (
    <>
      {Object.values(openWindows).map((state) => (
        <ChatWindow key={state.conversationId} state={state} />
      ))}
    </>
  )
}
