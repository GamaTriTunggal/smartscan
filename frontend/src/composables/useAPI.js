import axios from 'axios'
import { useAuthStore } from '@/stores/auth'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'

// Module-level state for logout coordination
// Prevents race conditions when multiple requests fail with 401 simultaneously
let isLoggingOut = false
let refreshPromise = null

/**
 * Check if logout is in progress
 * Used by components to avoid showing broken state during redirect
 * @returns {boolean}
 */
export function isLogoutInProgress() {
  return isLoggingOut
}

/**
 * Set logout state (for manual logout from components)
 * Intentionally exported as public API for components that need to coordinate logout state
 * (e.g., LogoutButton, session timeout handlers)
 * @param {boolean} value
 */
export function setLoggingOut(value) {
  isLoggingOut = value
  if (value) {
    refreshPromise = null // Clear any pending refresh
  }
}

/**
 * Clear local storage auth data synchronously
 * Called during forced logout to ensure clean state before redirect
 */
function clearAuthStorage() {
  // Read user ID BEFORE clearing 'user' — needed for tour state cleanup below
  let tourUserId = 'anonymous'
  try {
    const u = JSON.parse(localStorage.getItem('user'))
    if (u?.id) tourUserId = u.id
  } catch { /* ignore */ }

  localStorage.removeItem('authenticated')
  localStorage.removeItem('token_expires_at')
  localStorage.removeItem('user')
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')

  // Clear active tour state so in-progress tours don't resume after re-login.
  // Completed tours history is intentionally kept. This is an inline removal
  // (no useTour import) because the page is about to fully reload via
  // window.location.replace — module-level state doesn't need cleanup.
  localStorage.removeItem(`smartscan_active_tour_${tourUserId}`)
}

const api = axios.create({
  baseURL: API_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true, // Include cookies in cross-origin requests (for HttpOnly token cookies)
})

// Request interceptor
api.interceptors.request.use(
  (config) => {
    // Early exit if logging out - prevent new requests during logout
    if (isLoggingOut) {
      return Promise.reject(new Error('LOGOUT_IN_PROGRESS'))
    }

    // Let axios set Content-Type automatically for FormData (multipart/form-data with boundary)
    if (config.data instanceof FormData) {
      delete config.headers['Content-Type']
    }

    // Auth uses HttpOnly cookies - no Authorization header needed
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    // Early exit if logging out - prevent cascade of errors
    if (isLoggingOut) {
      return Promise.reject(new Error('LOGOUT_IN_PROGRESS'))
    }

    const authStore = useAuthStore()
    const originalRequest = error.config

    // Check for inactivity timeout - redirect first, then cleanup
    // This prevents Vue from re-rendering with null state before redirect completes
    if (error.response?.status === 401 && error.response?.data?.code === 'INACTIVITY_TIMEOUT') {
      // Set flag FIRST to prevent other requests from processing
      isLoggingOut = true

      // Clear local storage synchronously - no API call needed
      // Server will clear cookies, we just need to clear client state
      clearAuthStorage()

      // Redirect using replace (prevents back button returning to broken state)
      if (typeof window !== 'undefined') {
        window.location.replace('/login?reason=inactivity')
      }

      return Promise.reject(error)
    }

    // If 401 and this is not already a retry, try to refresh
    if (error.response?.status === 401 && !originalRequest._retry && authStore.isAuthenticated) {
      originalRequest._retry = true

      try {
        // Deduplicate refresh requests - if one is in progress, wait for it
        // This prevents multiple concurrent 401s from triggering multiple refresh attempts
        if (!refreshPromise) {
          refreshPromise = axios.post(`${API_URL}/auth/refresh`, {}, {
            withCredentials: true,
          }).finally(() => {
            refreshPromise = null
          })
        }

        const response = await refreshPromise

        if (response.data.success) {
          // Tokens are now in HttpOnly cookies, just update expiry info
          if (response.data.data?.expires_in) {
            authStore.setTokenExpiry(response.data.data.expires_in)
          }

          // Retry original request (cookies will be sent automatically)
          return api(originalRequest)
        } else {
          // Refresh returned falsy success - force logout to prevent infinite loop
          isLoggingOut = true
          clearAuthStorage()

          if (typeof window !== 'undefined') {
            window.location.replace('/login?reason=session_expired')
          }

          return Promise.reject(new Error('SESSION_EXPIRED'))
        }
      } catch {
        // Refresh failed - redirect to login
        isLoggingOut = true
        clearAuthStorage()

        if (typeof window !== 'undefined') {
          window.location.replace('/login?reason=session_expired')
        }

        return Promise.reject(new Error('SESSION_EXPIRED'))
      }
    }

    return Promise.reject(error)
  }
)

export function useAPI() {
  const get = async (url, params) => {
    const response = await api.get(url, { params })
    return response.data
  }

  const post = async (url, data) => {
    const response = await api.post(url, data)
    return response.data
  }

  const put = async (url, data) => {
    const response = await api.put(url, data)
    return response.data
  }

  const patch = async (url, data) => {
    const response = await api.patch(url, data)
    return response.data
  }

  const del = async (url, data) => {
    const config = data ? { data } : undefined
    const response = await api.delete(url, config)
    return response.data
  }

  // Upload file with FormData (for multipart/form-data uploads)
  const upload = async (url, formData) => {
    const response = await api.post(url, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
    return response.data
  }

  // Get auth headers for direct fetch calls (e.g., file downloads)
  // Auth uses HttpOnly cookies, so only Content-Type is needed
  const getAuthHeaders = () => {
    return {
      'Content-Type': 'application/json',
    }
  }

  return {
    api,
    get,
    post,
    put,
    patch,
    del,
    upload,
    getAuthHeaders,
  }
}
