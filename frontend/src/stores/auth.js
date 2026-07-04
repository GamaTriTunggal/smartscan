import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useAPI } from '@/composables/useAPI'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  // Track authentication state (tokens are in HttpOnly cookies, not accessible from JS)
  const authenticated = ref(false)
  // Token expiry tracking for proactive refresh
  const tokenExpiresAt = ref(null)

  const isAuthenticated = computed(() => authenticated.value)
  const isTenant = computed(() => user.value?.user_type === 'tenant_staff')
  const isAdmin = computed(() => isTenant.value && user.value?.role === 'admin')
  const isQCStaff = computed(() => isTenant.value && user.value?.role === 'qc_staff')
  const isWarehouseStaff = computed(() => isTenant.value && user.value?.role === 'warehouse_staff')
  const mustChangePassword = computed(() => user.value?.must_change_password === true)

  // Centralized post-auth redirect target. Single source of truth used by LoginPage
  // and ChangePasswordPage.
  const dashboardPath = computed(() => {
    if (isTenant.value) return '/tenant/dashboard'
    return '/login' // Unknown user_type — defensive fallback (shouldn't happen)
  })

  // Combined access helpers (staff OR admin can access)
  const canAccessQC = computed(() => isAdmin.value || isQCStaff.value)
  const canAccessWarehouse = computed(() => isAdmin.value || isWarehouseStaff.value)
  const canAccessSettings = computed(() => isAdmin.value)
  const canManageStaff = computed(() => isAdmin.value)
  const canManageProducts = computed(() => isAdmin.value)
  const canManageLocations = computed(() => isAdmin.value)

  // For backward compatibility - these now just track auth state
  const accessToken = computed(() => authenticated.value ? 'cookie' : null)
  const refreshToken = computed(() => authenticated.value ? 'cookie' : null)

  function setAuthenticated(value) {
    authenticated.value = value
    localStorage.setItem('authenticated', value ? 'true' : 'false')
  }

  function setTokenExpiry(expiresInSeconds) {
    tokenExpiresAt.value = Date.now() + (expiresInSeconds * 1000)
    localStorage.setItem('token_expires_at', tokenExpiresAt.value.toString())
  }

  // Check if token is expired (with 30 second buffer for edge cases)
  const isTokenExpired = computed(() => {
    if (!tokenExpiresAt.value) return true
    return Date.now() > (tokenExpiresAt.value - 30000)
  })

  function setUser(userData) {
    user.value = userData
    localStorage.setItem('user', JSON.stringify(userData))
  }

  // Backward compatibility - setTokens now just marks as authenticated
  function setTokens(access, refresh) {
    setAuthenticated(true)
  }

  async function login(email, password) {
    const { post } = useAPI()
    try {
      const response = await post('/auth/login', { email, password })
      if (response.success && response.data) {
        setAuthenticated(true)
        setUser(response.data.user)
        if (response.data.expires_in) {
          setTokenExpiry(response.data.expires_in)
        }
        return true
      }
      return false
    } catch {
      return false
    }
  }


  async function fetchUser() {
    if (!authenticated.value) return false

    const { get } = useAPI()
    try {
      const response = await get('/me')
      if (response.success && response.data) {
        setUser(response.data)
        return true
      }
      return false
    } catch {
      // If we get a 401, the session is invalid
      logout()
      return false
    }
  }

  async function logout() {
    // Capture user ID before clearing state — needed by clearActiveTourState
    // later, since localStorage['user'] will already be removed by then.
    const logoutUserId = user.value?.id || null

    // Clear local state FIRST to prevent race conditions
    // This ensures the UI immediately reflects logged-out state
    user.value = null
    authenticated.value = false
    tokenExpiresAt.value = null
    localStorage.removeItem('authenticated')
    localStorage.removeItem('token_expires_at')
    localStorage.removeItem('user')
    // Remove legacy token storage
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')


    // Reset QR generation polling state (stop timers, clear tracking)
    // Dynamic import avoids circular dependency with Pinia store registration
    try {
      const { useQRGenerationStore } = await import('@/stores/qrGeneration')
      useQRGenerationStore().reset()
    } catch (e) {
      // Non-fatal: if the store isn't loaded, there's nothing to reset
      console.warn('[auth.logout] Failed to reset QR generation store:', e)
    }

    // Clear active tour state (in-progress tours) so they don't resume after
    // re-login. Completed tours history is kept. Dynamic import avoids circular
    // dependency (useTour → useAuthStore → useTour). We pass logoutUserId
    // explicitly because localStorage['user'] was already removed above.
    try {
      const { clearActiveTourState } = await import('@/composables/useTour')
      clearActiveTourState(logoutUserId)
    } catch (e) {
      // Non-fatal fallback: remove directly if dynamic import fails
      if (logoutUserId) {
        localStorage.removeItem(`smartscan_active_tour_${logoutUserId}`)
      }
      console.warn('[auth.logout] Failed to clear tour state:', e)
    }

    // Call logout API to clear cookies on server (fire-and-forget)
    try {
      const { post } = useAPI()
      await post('/auth/logout', {})
    } catch {
      // Ignore errors during logout - cookies will expire naturally
    }
  }

  function initFromStorage() {
    // Check if we were previously authenticated
    const wasAuthenticated = localStorage.getItem('authenticated') === 'true'
    if (wasAuthenticated) {
      // Load token expiry FIRST to check if session is still valid
      const storedExpiry = localStorage.getItem('token_expires_at')
      if (storedExpiry) {
        tokenExpiresAt.value = parseInt(storedExpiry, 10)
      }

      // Check if token is expired BEFORE restoring auth state
      if (isTokenExpired.value) {
        // Token expired - clear stale state instead of restoring
        localStorage.removeItem('authenticated')
        localStorage.removeItem('user')
        localStorage.removeItem('token_expires_at')
        authenticated.value = false
        user.value = null
        tokenExpiresAt.value = null
        return  // Don't restore stale auth
      }

      // Token still valid - restore auth state
      authenticated.value = true

      // Load cached user data
      const storedUser = localStorage.getItem('user')
      if (storedUser) {
        try {
          user.value = JSON.parse(storedUser)
        } catch {
          localStorage.removeItem('user')
        }
      }
    }

    // Migrate from old localStorage token storage
    const oldAccessToken = localStorage.getItem('access_token')
    if (oldAccessToken && !wasAuthenticated) {
      // Old tokens exist but new auth flag not set - mark as authenticated
      authenticated.value = true
      localStorage.setItem('authenticated', 'true')
      // Clean up old storage
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
    }
  }

  // Initialize from storage
  initFromStorage()

  async function changePassword(currentPassword, newPassword) {
    const { post } = useAPI()
    try {
      const response = await post('/auth/change-password', {
        current_password: currentPassword,
        new_password: newPassword,
      })
      if (response.success) {
        // Update user to clear must_change_password flag
        if (user.value) {
          user.value.must_change_password = false
          localStorage.setItem('user', JSON.stringify(user.value))
        }
        return { success: true }
      }
      return { success: false, error: response.message || 'Failed to change password' }
    } catch (e) {
      return { success: false, error: 'Failed to change password' }
    }
  }

  return {
    user,
    accessToken,
    refreshToken,
    isAuthenticated,
    isTokenExpired,
    isTenant,
    isAdmin,
    isQCStaff,
    isWarehouseStaff,
    mustChangePassword,
    dashboardPath,
    canAccessQC,
    canAccessWarehouse,
    canAccessSettings,
    canManageStaff,
    canManageProducts,
    canManageLocations,
    setTokens,
    setUser,
    setAuthenticated,
    setTokenExpiry,
    login,
    fetchUser,
    logout,
    changePassword,
    initFromStorage,
  }
})
