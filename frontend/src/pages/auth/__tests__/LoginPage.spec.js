import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import LoginPage from '../LoginPage.vue'

// Mock vue-router
const mockPush = vi.fn()
const mockReplace = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockPush,
    replace: mockReplace,
  }),
  useRoute: () => ({
    query: {},
  }),
}))

// Mock auth store
const mockLogin = vi.fn()
const mockLogout = vi.fn()
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    login: mockLogin,
    logout: mockLogout,
    isTenant: true,
    // Phase 6.1.5 follow-up (bug 260506-1157): LoginPage now reads dashboardPath
    // from the auth store instead of hard-coding the user-type → path mapping.
    dashboardPath: '/tenant/dashboard',
  }),
}))

// Mock useDarkMode
vi.mock('@/composables/useDarkMode', () => ({
  useDarkMode: vi.fn(),
}))

// Stub components
const stubs = {
  Button: {
    template: '<button :disabled="loading" type="submit" @click="$emit(\'click\')"><slot /></button>',
    props: ['loading', 'type', 'class'],
  },
  Input: {
    template: '<input :value="modelValue" :type="type || \'text\'" @input="$emit(\'update:modelValue\', $event.target.value)" :disabled="disabled" :placeholder="placeholder" />',
    props: ['modelValue', 'disabled', 'type', 'placeholder', 'id', 'class'],
  },
  Label: {
    template: '<label><slot /></label>',
    props: ['for', 'class'],
  },
  Alert: {
    template: '<div class="alert" role="alert"><slot /></div>',
    props: ['variant', 'class'],
  },
  ThemeSwitcher: {
    template: '<div class="theme-switcher"></div>',
  },
  // Stub Lucide icons
  CheckCircle: { template: '<svg class="check-circle"></svg>' },
  ShieldCheck: { template: '<svg class="shield-check"></svg>' },
  Zap: { template: '<svg class="zap"></svg>' },
}

describe('LoginPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockLogin.mockReset()
    mockPush.mockReset()
  })

  function createWrapper() {
    return mount(LoginPage, {
      global: {
        stubs,
      },
    })
  }

  describe('Rendering', () => {
    it('should render the login form', () => {
      const wrapper = createWrapper()

      // Check for key elements
      expect(wrapper.find('h2').text()).toBe('Welcome back')
      expect(wrapper.findAll('input').length).toBe(2)
      expect(wrapper.find('button[type="submit"]').exists()).toBe(true)
    })

    it('should have email and password inputs', () => {
      const wrapper = createWrapper()
      const inputs = wrapper.findAll('input')

      expect(inputs[0].attributes('type')).toBe('email')
      expect(inputs[1].attributes('type')).toBe('password')
    })

    it('should show sign in button', () => {
      const wrapper = createWrapper()

      expect(wrapper.text()).toContain('Sign in')
    })

    it('should show terms and privacy links', () => {
      const wrapper = createWrapper()

      expect(wrapper.text()).toContain('Terms of Service')
      expect(wrapper.text()).toContain('Privacy Policy')
    })
  })

  describe('Validation', () => {
    it('should show error when submitting empty form', async () => {
      const wrapper = createWrapper()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.text()).toContain('Please enter email and password')
      expect(mockLogin).not.toHaveBeenCalled()
    })

    it('should show error when email is empty', async () => {
      const wrapper = createWrapper()
      const inputs = wrapper.findAll('input')

      await inputs[1].setValue('password123')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.text()).toContain('Please enter email and password')
      expect(mockLogin).not.toHaveBeenCalled()
    })

    it('should show error when password is empty', async () => {
      const wrapper = createWrapper()
      const inputs = wrapper.findAll('input')

      await inputs[0].setValue('test@example.com')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.text()).toContain('Please enter email and password')
      expect(mockLogin).not.toHaveBeenCalled()
    })
  })

  describe('Form Submission', () => {
    it('should call login with email and password', async () => {
      mockLogin.mockResolvedValue(true)
      const wrapper = createWrapper()
      const inputs = wrapper.findAll('input')

      await inputs[0].setValue('test@example.com')
      await inputs[1].setValue('password123')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(mockLogin).toHaveBeenCalledWith('test@example.com', 'password123')
    })

    it('should show error on failed login', async () => {
      mockLogin.mockResolvedValue(false)
      const wrapper = createWrapper()
      const inputs = wrapper.findAll('input')

      await inputs[0].setValue('test@example.com')
      await inputs[1].setValue('wrongpassword')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.text()).toContain('Invalid email or password')
    })

    it('should show error on login exception', async () => {
      mockLogin.mockRejectedValue(new Error('Network error'))
      const wrapper = createWrapper()
      const inputs = wrapper.findAll('input')

      await inputs[0].setValue('test@example.com')
      await inputs[1].setValue('password123')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.text()).toContain('Login failed. Please try again.')
    })
  })

  describe('Navigation', () => {
    it('should redirect tenant to tenant dashboard on success', async () => {
      mockLogin.mockResolvedValue(true)
      const wrapper = createWrapper()
      const inputs = wrapper.findAll('input')

      await inputs[0].setValue('admin@example.com')
      await inputs[1].setValue('password')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(mockPush).toHaveBeenCalledWith('/tenant/dashboard')
    })
  })
})
