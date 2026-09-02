// Thin wrapper around the browser Notification API — used to alert the user
// to a new chat message when their tab isn't the one they're looking at.
// Every function is a safe no-op in an environment without Notification
// support (older browsers, some embedded webviews, and jsdom in tests).

export function requestNotificationPermission(): void {
  if (typeof Notification === 'undefined') return
  if (Notification.permission === 'default') {
    void Notification.requestPermission()
  }
}

// True when the user isn't actually looking at this tab right now — the
// signal for "a native OS notification is worth interrupting them with",
// as opposed to a message arriving while they're already watching the
// conversation update live in the UI.
export function isTabInBackground(): boolean {
  return document.hidden || !document.hasFocus()
}

export function showBrowserNotification(title: string, body: string): void {
  if (typeof Notification === 'undefined' || Notification.permission !== 'granted') return
  const notification = new Notification(title, { body, icon: '/img/favglobe.png' })
  notification.onclick = () => {
    window.focus()
    notification.close()
  }
}
