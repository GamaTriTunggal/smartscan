import { ref } from 'vue'

// Global toast state - shared across all components
const toasts = ref([])
let toastId = 0

/**
 * Toast notification composable
 * Provides a simple API for showing toast notifications anywhere in the app
 *
 * Usage:
 *   const toast = useToast()
 *   toast.success('Operation completed!')
 *   toast.error('Something went wrong')
 */
export function useToast() {
  /**
   * Add a toast notification
   * @param {string} message - The message to display
   * @param {string} [type='info'] - Toast type: 'success' | 'error' | 'info' | 'warning'
   * @param {object} [options] - Optional settings
   * @param {number} [options.duration] - Auto-dismiss duration in ms (default: 5000, 0 = no auto-dismiss)
   */
  function addToast(message, type = 'info', options = {}) {
    const id = ++toastId
    const duration = options.duration !== undefined ? options.duration : 5000

    const toast = {
      id,
      message,
      type,
      createdAt: Date.now()
    }

    toasts.value.push(toast)

    // Auto-dismiss after duration (if not 0)
    if (duration > 0) {
      setTimeout(() => {
        dismiss(id)
      }, duration)
    }

    return id
  }

  /**
   * Dismiss a specific toast by ID
   * @param {number} id - Toast ID to dismiss
   */
  function dismiss(id) {
    const index = toasts.value.findIndex(t => t.id === id)
    if (index > -1) {
      toasts.value.splice(index, 1)
    }
  }

  /**
   * Dismiss all toasts
   */
  function dismissAll() {
    toasts.value = []
  }

  /**
   * Show success toast (green)
   * @param {string} message
   * @param {object} options
   */
  function success(message, options = {}) {
    return addToast(message, 'success', options)
  }

  /**
   * Show error toast (red)
   * @param {string} message
   * @param {object} options
   */
  function error(message, options = {}) {
    return addToast(message, 'error', options)
  }

  /**
   * Show info toast (blue)
   * @param {string} message
   * @param {object} options
   */
  function info(message, options = {}) {
    return addToast(message, 'info', options)
  }

  /**
   * Show warning toast (yellow)
   * @param {string} message
   * @param {object} options
   */
  function warning(message, options = {}) {
    return addToast(message, 'warning', options)
  }

  return {
    toasts,
    success,
    error,
    info,
    warning,
    dismiss,
    dismissAll
  }
}
