import { vi } from 'vitest'

// Mock successful login response
export const mockLoginResponse = {
  success: true,
  message: 'Login successful',
  data: {
    user: {
      id: 'test-user-id',
      email: 'test@example.com',
      user_type: 'tenant_staff',
      role: 'admin',
      full_name: 'Test User',
      tenant_id: 'test-tenant-id',
      tenant_name: 'Test Company',
      must_change_password: false,
    },
    tokens: {
      access_token: 'mock-access-token',
      refresh_token: 'mock-refresh-token',
    },
    expires_in: 86400,
  },
}

// Mock failed login response
export const mockLoginErrorResponse = {
  success: false,
  message: 'Invalid email or password',
  data: null,
}

// Mock user profile response
export const mockUserResponse = {
  success: true,
  message: 'User profile',
  data: {
    id: 'test-user-id',
    email: 'test@example.com',
    user_type: 'tenant_staff',
    role: 'admin',
    full_name: 'Test User',
    tenant_id: 'test-tenant-id',
    tenant_name: 'Test Company',
    must_change_password: false,
  },
}

// Create a mock API instance
export function createMockApi() {
  return {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  }
}

// Mock useAPI composable
export function mockUseAPI() {
  const api = createMockApi()

  return {
    api,
    setupLoginSuccess: () => {
      api.post.mockResolvedValue({ data: mockLoginResponse })
    },
    setupLoginError: () => {
      api.post.mockRejectedValue({
        response: { data: mockLoginErrorResponse },
      })
    },
    setupGetUserSuccess: () => {
      api.get.mockResolvedValue({ data: mockUserResponse })
    },
  }
}
