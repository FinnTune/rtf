import type { ReactNode } from 'react'
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
    </>
  )
}
