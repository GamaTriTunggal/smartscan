import { ref } from 'vue'
import { useToast } from '@/composables/useToast'

// Global notifications state (shared across all components)
const notifications = ref([])
let nextId = 1

export function useNotifications() {
  const toast = useToast()

  /**
   * Show a notification
   * @param {Object} options
   * @param {string} options.type - 'success' | 'error' | 'warning' | 'info'
   * @param {string} options.title - Notification title
   * @param {string} options.message - Notification message
   * @param {number} options.duration - Auto-dismiss duration in ms (0 = no auto-dismiss)
   * @returns {number} Notification ID
   */
  const notify = ({ type = 'info', title = '', message = '', duration = 5000 }) => {
    const id = nextId++
    const notification = { id, type, title, message, createdAt: Date.now() }
    notifications.value.push(notification)

    // Bridge to useToast so notifications are actually rendered
    const displayMessage = title && message ? `${title}: ${message}` : message || title
    const toastFn = toast[type] || toast.info
    toastFn(displayMessage, { duration })

    // Auto-remove after duration
    if (duration > 0) {
      setTimeout(() => {
        removeNotification(id)
      }, duration)
    }

    return id
  }

  /**
   * Remove a notification by ID
   * @param {number} id - Notification ID
   */
  const removeNotification = (id) => {
    const index = notifications.value.findIndex(n => n.id === id)
    if (index !== -1) {
      notifications.value.splice(index, 1)
    }
  }

  /**
   * Clear all notifications
   */
  const clearAll = () => {
    notifications.value = []
  }

  // Convenience methods
  const success = (title, message = '') => notify({ type: 'success', title, message })
  const error = (title, message = '') => notify({ type: 'error', title, message })
  const warning = (title, message = '') => notify({ type: 'warning', title, message })
  const info = (title, message = '') => notify({ type: 'info', title, message })

  // Aliases for backward compatibility (used by useBundle.js and other composables)
  const showSuccess = (title, message = '') => success(title, message)
  const showError = (title, message = '') => error(title, message)
  const showWarning = (title, message = '') => warning(title, message)
  const showInfo = (title, message = '') => info(title, message)

  return {
    notifications,
    notify,
    removeNotification,
    clearAll,
    // Primary methods
    success,
    error,
    warning,
    info,
    // Aliases
    showSuccess,
    showError,
    showWarning,
    showInfo
  }
}
