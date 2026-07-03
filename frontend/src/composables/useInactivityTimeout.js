import { ref, onMounted, onUnmounted } from 'vue'

// Inactivity timeout in milliseconds (30 minutes)
const INACTIVITY_TIMEOUT_MS = 30 * 60 * 1000

// Activity events to track
const ACTIVITY_EVENTS = ['mousemove', 'keydown', 'click', 'scroll', 'touchstart']

/**
 * Composable for tracking user inactivity and auto-logout
 * @param {Function} onTimeout - Callback function to call when timeout occurs (typically logout)
 * @param {Object} options - Optional configuration
 * @param {number} options.timeoutMs - Custom timeout in milliseconds (default: 30 minutes)
 */
export function useInactivityTimeout(onTimeout, options = {}) {
  const timeoutMs = options.timeoutMs || INACTIVITY_TIMEOUT_MS
  const isActive = ref(true)
  let timeoutId = null
  let isInitialized = false

  /**
   * Reset the inactivity timer
   */
  function resetTimer() {
    if (timeoutId) {
      clearTimeout(timeoutId)
    }

    timeoutId = setTimeout(() => {
      isActive.value = false
      if (onTimeout && typeof onTimeout === 'function') {
        onTimeout()
      }
    }, timeoutMs)
  }

  /**
   * Handle user activity event
   */
  function handleActivity() {
    if (!isActive.value) return
    resetTimer()
  }

  /**
   * Start tracking user activity
   */
  function startTracking() {
    if (isInitialized) return

    ACTIVITY_EVENTS.forEach(event => {
      document.addEventListener(event, handleActivity, { passive: true })
    })

    // Start the initial timer
    resetTimer()
    isInitialized = true
  }

  /**
   * Stop tracking user activity
   */
  function stopTracking() {
    ACTIVITY_EVENTS.forEach(event => {
      document.removeEventListener(event, handleActivity)
    })

    if (timeoutId) {
      clearTimeout(timeoutId)
      timeoutId = null
    }

    isInitialized = false
  }

  // Auto-start on mount, cleanup on unmount
  onMounted(() => {
    startTracking()
  })

  onUnmounted(() => {
    stopTracking()
  })

  return {
    isActive,
    startTracking,
    stopTracking,
    resetTimer
  }
}
