import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { isTabInBackground, requestNotificationPermission, showBrowserNotification } from './notifications'

class FakeNotification {
  static permission: NotificationPermission = 'default'
  static requestPermission = vi.fn().mockResolvedValue('granted')
  onclick: (() => void) | null = null
  close = vi.fn()
  title: string
  options?: NotificationOptions
  constructor(title: string, options?: NotificationOptions) {
    this.title = title
    this.options = options
  }
}

describe('notifications', () => {
  beforeEach(() => {
    FakeNotification.requestPermission = vi.fn().mockResolvedValue('granted')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  describe('requestNotificationPermission', () => {
    it('requests permission when it has never been decided', () => {
      FakeNotification.permission = 'default'
      vi.stubGlobal('Notification', FakeNotification)
      requestNotificationPermission()
      expect(FakeNotification.requestPermission).toHaveBeenCalled()
    })

    it('does not re-request when already granted', () => {
      FakeNotification.permission = 'granted'
      vi.stubGlobal('Notification', FakeNotification)
      requestNotificationPermission()
      expect(FakeNotification.requestPermission).not.toHaveBeenCalled()
    })

    it('does not re-request when already denied', () => {
      FakeNotification.permission = 'denied'
      vi.stubGlobal('Notification', FakeNotification)
      requestNotificationPermission()
      expect(FakeNotification.requestPermission).not.toHaveBeenCalled()
    })

    it('is a safe no-op when the Notification API does not exist', () => {
      vi.stubGlobal('Notification', undefined)
      expect(() => requestNotificationPermission()).not.toThrow()
    })
  })

  describe('isTabInBackground', () => {
    beforeEach(() => {
      vi.spyOn(document, 'hidden', 'get').mockReturnValue(false)
      vi.spyOn(document, 'hasFocus').mockReturnValue(true)
    })

    it('is false when the tab is visible and focused', () => {
      expect(isTabInBackground()).toBe(false)
    })

    it('is true when the document is hidden', () => {
      vi.spyOn(document, 'hidden', 'get').mockReturnValue(true)
      expect(isTabInBackground()).toBe(true)
    })

    it('is true when the tab is visible but not focused', () => {
      vi.spyOn(document, 'hasFocus').mockReturnValue(false)
      expect(isTabInBackground()).toBe(true)
    })
  })

  describe('showBrowserNotification', () => {
    it('creates a notification with the given title/body when permission is granted', () => {
      FakeNotification.permission = 'granted'
      vi.stubGlobal('Notification', FakeNotification)
      const ctorSpy = vi.fn()
      class SpiedNotification extends FakeNotification {
        constructor(title: string, options?: NotificationOptions) {
          super(title, options)
          ctorSpy(title, options)
        }
      }
      vi.stubGlobal('Notification', SpiedNotification)

      showBrowserNotification('New message from bob', 'hi there')
      expect(ctorSpy).toHaveBeenCalledWith('New message from bob', expect.objectContaining({ body: 'hi there' }))
    })

    it('does nothing when permission has not been granted', () => {
      FakeNotification.permission = 'denied'
      const ctorSpy = vi.fn()
      class SpiedNotification extends FakeNotification {
        constructor(title: string, options?: NotificationOptions) {
          super(title, options)
          ctorSpy()
        }
      }
      vi.stubGlobal('Notification', SpiedNotification)

      showBrowserNotification('title', 'body')
      expect(ctorSpy).not.toHaveBeenCalled()
    })

    it('is a safe no-op when the Notification API does not exist', () => {
      vi.stubGlobal('Notification', undefined)
      expect(() => showBrowserNotification('title', 'body')).not.toThrow()
    })

    it('focuses the window and closes itself when clicked', () => {
      FakeNotification.permission = 'granted'
      const instances: FakeNotification[] = []
      class SpiedNotification extends FakeNotification {
        constructor(title: string, options?: NotificationOptions) {
          super(title, options)
          instances.push(this)
        }
      }
      vi.stubGlobal('Notification', SpiedNotification)
      const focusSpy = vi.spyOn(window, 'focus').mockImplementation(() => {})

      showBrowserNotification('title', 'body')
      const created = instances[0]
      created?.onclick?.()

      expect(focusSpy).toHaveBeenCalled()
      expect(created?.close).toHaveBeenCalled()
    })
  })
})
