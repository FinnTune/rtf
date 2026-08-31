import type { ReactNode } from 'react'
import { ChatWindowsLayer } from '../chat/ChatWindowsLayer'
import { StatusBanner } from '../common/StatusBanner'
import { SidebarLeft } from './SidebarLeft'
import { SidebarRight } from './SidebarRight'
import { Topbar } from './Topbar'

export function LoggedInShell({ children }: { children: ReactNode }) {
  return (
    <>
      <Topbar />
      <StatusBanner />
      <div className="app-shell">
        <SidebarLeft />
        <main className="content-pane" id="main-content">
          {children}
        </main>
        <SidebarRight />
      </div>
      {/* Siblings of .app-shell, same region the vanilla app appended
          floating chat windows to (#main) — relies on .chat-window's
          float:right for layout. */}
      <ChatWindowsLayer />
    </>
  )
}
