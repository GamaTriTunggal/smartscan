import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../auth'

// Mock useAPI composable
vi.mock('@/composables/useAPI', () => ({
  useAPI: () => ({
    get: vi.fn(),
    post: vi.fn(),
  }),
}))

describe('Auth Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  describe('Initial State', () => {
    it('should have null user initially', () => {
      const store = useAuthStore()
      expect(store.user).toBeNull()
    })

    it('should not be authenticated initially', () => {
      const store = useAuthStore()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('setUser', () => {
    it('should set user data', () => {
      const store = useAuthStore()
      const userData = {
        id: 'test-id',
        email: 'test@example.com',
        user_type: 'tenant_staff',
        role: 'admin',
      }

      store.setUser(userData)

      expect(store.user).toEqual(userData)
    })

    it('should persist user to localStorage', () => {
      const store = useAuthStore()
      const userData = { id: 'test-id', email: 'test@example.com' }

      store.setUser(userData)

      expect(localStorage.setItem).toHaveBeenCalledWith('user', JSON.stringify(userData))
    })
  })

  describe('setAuthenticated', () => {
    it('should set authenticated to true', () => {
      const store = useAuthStore()

      store.setAuthenticated(true)

      expect(store.isAuthenticated).toBe(true)
    })

    it('should set authenticated to false', () => {
      const store = useAuthStore()
      store.setAuthenticated(true)

      store.setAuthenticated(false)

      expect(store.isAuthenticated).toBe(false)
    })

    it('should persist to localStorage', () => {
      const store = useAuthStore()

      store.setAuthenticated(true)

      expect(localStorage.setItem).toHaveBeenCalledWith('authenticated', 'true')
    })
  })

  describe('Role Computed Properties', () => {
    it('should identify tenant staff', () => {
      const store = useAuthStore()
      store.setUser({ user_type: 'tenant_staff', role: 'admin' })

      expect(store.isTenant).toBe(true)
    })

    it('should not identify unknown user types as tenant staff', () => {
      const store = useAuthStore()
      store.setUser({ user_type: 'unknown_type', role: 'admin' })

      expect(store.isTenant).toBe(false)
    })

    it('should identify tenant admin', () => {
      const store = useAuthStore()
      store.setUser({ user_type: 'tenant_staff', role: 'admin' })

      expect(store.isAdmin).toBe(true)
    })

    it('should identify QC staff', () => {
      const store = useAuthStore()
      store.setUser({ user_type: 'tenant_staff', role: 'qc_staff' })

      expect(store.isQCStaff).toBe(true)
    })

    it('should identify warehouse staff', () => {
      const store = useAuthStore()
      store.setUser({ user_type: 'tenant_staff', role: 'warehouse_staff' })

      expect(store.isWarehouseStaff).toBe(true)
    })
  })

  // Regression guard for bug 260506-1157: unknown user types used to fall through
  // to /tenant/dashboard, which the tenantOnly route guard rejected → infinite
  // redirect loop, page unresponsive.
  // dashboardPath centralizes the user_type → path mapping in ONE place.
  describe('dashboardPath computed', () => {
    it('routes tenant_staff to /tenant/dashboard', () => {
      const store = useAuthStore()
      store.setUser({ user_type: 'tenant_staff', role: 'admin' })
      expect(store.dashboardPath).toBe('/tenant/dashboard')
    })

    it('falls back to /login when user is null', () => {
      const store = useAuthStore()
      // explicit null — defensive fallback for unknown user_type
      store.user = null
      expect(store.dashboardPath).toBe('/login')
    })

    it('falls back to /login for unknown user_type', () => {
      const store = useAuthStore()
      store.setUser({ user_type: 'unknown_role', role: 'admin' })
      expect(store.dashboardPath).toBe('/login')
    })
  })

  describe('Access Control Computed Properties', () => {
    it('should allow admin to access QC', () => {
      const store = useAuthStore()
      store.setUser({ user_type: 'tenant_staff', role: 'admin' })

      expect(store.canAccessQC).toBe(true)
    })

    it('should allow QC staff to access QC', () => {
      const store = useAuthStore()
      store.setUser({ user_type: 'tenant_staff', role: 'qc_staff' })

      expect(store.canAccessQC).toBe(true)
    })

    it('should not allow warehouse staff to access QC', () => {
      const store = useAuthStore()
      store.setUser({ user_type: 'tenant_staff', role: 'warehouse_staff' })

      expect(store.canAccessQC).toBe(false)
    })

    it('should allow admin to access warehouse', () => {
      const store = useAuthStore()
      store.setUser({ user_type: 'tenant_staff', role: 'admin' })

      expect(store.canAccessWarehouse).toBe(true)
    })

    it('should allow warehouse staff to access warehouse', () => {
      const store = useAuthStore()
      store.setUser({ user_type: 'tenant_staff', role: 'warehouse_staff' })

      expect(store.canAccessWarehouse).toBe(true)
    })

    it('should allow only admin to access settings', () => {
      const store = useAuthStore()

      store.setUser({ user_type: 'tenant_staff', role: 'admin' })
      expect(store.canAccessSettings).toBe(true)

      store.setUser({ user_type: 'tenant_staff', role: 'qc_staff' })
      expect(store.canAccessSettings).toBe(false)
    })
  })

  describe('mustChangePassword', () => {
    it('should return true when user must change password', () => {
      const store = useAuthStore()
      store.setUser({ must_change_password: true })

      expect(store.mustChangePassword).toBe(true)
    })

    it('should return false when user does not need to change password', () => {
      const store = useAuthStore()
      store.setUser({ must_change_password: false })

      expect(store.mustChangePassword).toBe(false)
    })

    it('should return false when must_change_password is not set', () => {
      const store = useAuthStore()
      store.setUser({ email: 'test@example.com' })

      expect(store.mustChangePassword).toBe(false)
    })
  })

  describe('logout', () => {
    it('should clear user data', async () => {
      const store = useAuthStore()
      store.setUser({ id: 'test-id' })
      store.setAuthenticated(true)

      await store.logout()

      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })

    it('should clear localStorage', async () => {
      const store = useAuthStore()
      store.setUser({ id: 'test-id' })
      store.setAuthenticated(true)

      await store.logout()

      expect(localStorage.removeItem).toHaveBeenCalledWith('authenticated')
      expect(localStorage.removeItem).toHaveBeenCalledWith('user')
      expect(localStorage.removeItem).toHaveBeenCalledWith('token_expires_at')
    })
  })

  describe('setTokenExpiry', () => {
    it('should set token expiry time', () => {
      const store = useAuthStore()
      const now = Date.now()
      vi.spyOn(Date, 'now').mockReturnValue(now)

      store.setTokenExpiry(3600) // 1 hour

      expect(localStorage.setItem).toHaveBeenCalledWith(
        'token_expires_at',
        (now + 3600000).toString()
      )
    })
  })

  describe('initFromStorage', () => {
    it('should restore authenticated state from localStorage', () => {
      const futureExpiry = (Date.now() + 3600000).toString() // 1 hour from now
      localStorage.getItem.mockImplementation((key) => {
        if (key === 'authenticated') return 'true'
        if (key === 'user') return JSON.stringify({ id: 'stored-id', email: 'stored@example.com' })
        if (key === 'token_expires_at') return futureExpiry
        return null
      })

      // Create a new store - initFromStorage is called in the store definition
      const store = useAuthStore()

      // Manually call initFromStorage since pinia might not trigger it in tests
      store.initFromStorage()

      expect(store.isAuthenticated).toBe(true)
    })

    it('should restore user from localStorage', () => {
      const storedUser = { id: 'stored-id', email: 'stored@example.com' }
      const futureExpiry = (Date.now() + 3600000).toString() // 1 hour from now
      localStorage.getItem.mockImplementation((key) => {
        if (key === 'authenticated') return 'true'
        if (key === 'user') return JSON.stringify(storedUser)
        if (key === 'token_expires_at') return futureExpiry
        return null
      })

      const store = useAuthStore()
      store.initFromStorage()

      expect(store.user).toEqual(storedUser)
    })
  })
})
