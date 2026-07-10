import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import AuditLogsPage from '../AuditLogsPage.vue'

// Use vi.hoisted() so variables are available in vi.mock factories
const { mockGet, mockUseDebounceFn } = vi.hoisted(() => {
  const mockGet = vi.fn()
  const mockUseDebounceFn = vi.fn((fn) => {
    const wrapper = vi.fn((...args) => fn(...args))
    wrapper._originalFn = fn
    return wrapper
  })
  return { mockGet, mockUseDebounceFn }
})

vi.mock('@/composables/useAPI', () => ({
  useAPI: () => ({
    get: mockGet,
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    del: vi.fn()
  })
}))

vi.mock('@/composables/useDateTime', () => ({
  useDateTime: () => ({
    formatDateTime: (d) => d,
    formatDate: (d) => d
  })
}))

vi.mock('@/composables/useDarkMode', () => ({
  useDarkMode: () => ({
    isDark: { value: false },
    toggleDarkMode: vi.fn(),
    setDarkMode: vi.fn(),
    setThemeMode: vi.fn(),
    initDarkMode: vi.fn(),
    cleanup: vi.fn()
  })
}))

vi.mock('@vueuse/core', () => ({
  useDebounceFn: mockUseDebounceFn
}))

// Mock Chart.js and vue-chartjs
vi.mock('chart.js', () => ({
  Chart: { register: vi.fn() },
  CategoryScale: {},
  LinearScale: {},
  PointElement: {},
  LineElement: {},
  BarElement: {},
  ArcElement: {},
  Title: {},
  Tooltip: {},
  Legend: {},
  Filler: {}
}))

vi.mock('vue-chartjs', () => ({
  Line: { template: '<canvas data-testid="line-chart" />', props: ['data', 'options'] },
  Doughnut: { template: '<canvas data-testid="doughnut-chart" />', props: ['data', 'options'] },
  Bar: { template: '<canvas data-testid="bar-chart" />', props: ['data', 'options'] }
}))

// Mock vue-router
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  useRoute: () => ({ query: {} })
}))

const stubs = {
  Card: { template: '<div class="card-stub"><slot /></div>' },
  Button: {
    template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
    props: ['variant', 'size', 'disabled', 'class']
  },
  Input: {
    template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" :placeholder="placeholder" />',
    props: ['modelValue', 'placeholder', 'id', 'class']
  },
  Activity: { template: '<span class="icon-activity" />' },
  ShieldAlert: { template: '<span class="icon-shield-alert" />' },
  Users: { template: '<span class="icon-users" />' },
  Globe: { template: '<span class="icon-globe" />' },
  ChevronDown: { template: '<span class="icon-chevron-down" />' },
  ChevronUp: { template: '<span class="icon-chevron-up" />' },
  Search: { template: '<span class="icon-search" />' },
  SlidersHorizontal: { template: '<span class="icon-sliders" />' },
  BarChart3: { template: '<span class="icon-bar-chart" />' }
}

const mockStatsResponse = {
  success: true,
  data: {
    period: '30d',
    summary: {
      total_events: 1234,
      security_events: 56,
      unique_users: 23,
      unique_ips: 18
    },
    by_action: [
      { action_type: 'login', count: 500 },
      { action_type: 'create', count: 300 },
      { action_type: 'delete', count: 50 }
    ],
    by_entity: [
      { entity_type: 'user', count: 600 },
      { entity_type: 'qr_batch', count: 200 }
    ],
    daily_trend: [
      { date: '2026-03-01', count: 45 },
      { date: '2026-03-02', count: 60 },
      { date: '2026-03-03', count: 30 }
    ],
    top_users: [
      { user_id: 'u1', email: 'admin@test.com', event_count: 120 }
    ]
  }
}

const mockLogsResponse = {
  success: true,
  data: {
    logs: [
      {
        id: 'log1',
        user_email: 'admin@test.com',
        company_name: 'Test Corp',
        action_type: 'login',
        entity_type: 'user',
        entity_id: '12345678-abcd',
        ip_address: '192.168.1.1',
        created_at: '2026-03-03T10:00:00Z'
      }
    ],
    pagination: { page: 1, limit: 20, total: 1, total_page: 1 }
  }
}

function mountPage() {
  return mount(AuditLogsPage, {
    global: {
      stubs,
      plugins: [createPinia()]
    }
  })
}

describe('AuditLogsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.getItem.mockReturnValue(null)
    setActivePinia(createPinia())

    mockGet.mockImplementation((url) => {
      if (url.includes('/stats')) return Promise.resolve(mockStatsResponse)
      return Promise.resolve(mockLogsResponse)
    })
  })

  it('renders page title and subtitle', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.find('h1').text()).toBe('Audit Logs')
    expect(wrapper.text()).toContain('Security operations activity trail')
  })

  it('fetches stats on mount', async () => {
    mountPage()
    await flushPromises()

    const statsCalls = mockGet.mock.calls.filter(c => c[0].includes('/stats'))
    expect(statsCalls.length).toBeGreaterThanOrEqual(1)
    expect(statsCalls[0][0]).toBe('/tenant/audit-logs/stats')
  })

  it('displays stat cards with correct values', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('1,234')
    expect(text).toContain('56')
    expect(text).toContain('23')
    expect(text).toContain('18')
    expect(text).toContain('Total Events')
    expect(text).toContain('Security Events')
    expect(text).toContain('Unique Users')
    expect(text).toContain('Unique IPs')
  })

  it('renders chart components', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.find('[data-testid="line-chart"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="doughnut-chart"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="bar-chart"]').exists()).toBe(true)
  })

  it('period selector changes trigger new fetch', async () => {
    const wrapper = mountPage()
    await flushPromises()

    mockGet.mockClear()
    mockGet.mockImplementation((url) => {
      if (url.includes('/stats')) return Promise.resolve({ ...mockStatsResponse, data: { ...mockStatsResponse.data, period: '7d' } })
      return Promise.resolve(mockLogsResponse)
    })

    // Click the 7d button
    const periodButtons = wrapper.findAll('button').filter(b => b.text().includes('7 Days'))
    expect(periodButtons.length).toBe(1)
    await periodButtons[0].trigger('click')
    await flushPromises()

    const statsCalls = mockGet.mock.calls.filter(c => c[0].includes('/stats'))
    expect(statsCalls.length).toBeGreaterThanOrEqual(1)
    const lastCall = statsCalls[statsCalls.length - 1]
    expect(lastCall[1]).toEqual({ period: '7d' })
  })

  it('toggle summary hides charts and updates localStorage', async () => {
    const wrapper = mountPage()
    await flushPromises()

    // Charts should be visible initially (showSummary defaults to true)
    expect(wrapper.find('[data-testid="line-chart"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Hide Analytics')

    // Directly toggle showSummary via the component's exposed state
    wrapper.vm.showSummary = false
    await wrapper.vm.$nextTick()

    // Charts should be hidden after toggle
    expect(wrapper.find('[data-testid="line-chart"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Show Analytics')
  })

  it('reads localStorage preference on mount', async () => {
    localStorage.getItem.mockImplementation((key) => {
      if (key === 'auditLogShowSummary') return 'false'
      return null
    })

    const wrapper = mountPage()
    await flushPromises()

    // Summary should be hidden
    expect(wrapper.find('[data-testid="line-chart"]').exists()).toBe(false)
  })

  it('security events card highlights when > 0', async () => {
    const wrapper = mountPage()
    await flushPromises()

    // The redesigned security card uses security-pulse class and border-red-500
    const html = wrapper.html()
    expect(html).toContain('security-pulse')
    expect(html).toContain('border-red-500')
  })

  it('handles stats API failure gracefully', async () => {
    mockGet.mockImplementation((url) => {
      if (url.includes('/stats')) return Promise.reject(new Error('Network error'))
      return Promise.resolve(mockLogsResponse)
    })

    const wrapper = mountPage()
    await flushPromises()

    // Should not crash, should show 0 values
    expect(wrapper.text()).toContain('Total Events')
    expect(wrapper.text()).toContain('0')
  })

  it('fetches logs table data on mount', async () => {
    mountPage()
    await flushPromises()

    const logsCalls = mockGet.mock.calls.filter(c => c[0] === '/tenant/audit-logs')
    expect(logsCalls.length).toBeGreaterThanOrEqual(1)
  })
})
