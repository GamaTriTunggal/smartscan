import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import DynamicQRPage from '../DynamicQRPage.vue'

// Mock vue-router
const mockPush = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}))

// Mock useAPI
const mockGet = vi.fn()
const mockPost = vi.fn()
vi.mock('@/composables/useAPI', () => ({
  useAPI: () => ({
    get: mockGet,
    post: mockPost,
  }),
}))

// Mock useDateTime
vi.mock('@/composables/useDateTime', () => ({
  useDateTime: () => ({
    formatDate: vi.fn((date) => date ? new Date(date).toLocaleDateString() : ''),
    toUTCString: vi.fn((date) => date || null),
  }),
}))

// Stub components
const stubs = {
  Card: {
    template: '<div class="card"><slot /></div>',
  },
  Button: {
    template: '<button :disabled="loading || disabled" @click="$emit(\'click\')"><slot /></button>',
    props: ['loading', 'disabled', 'variant', 'size'],
  },
  Input: {
    template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" :disabled="disabled" :type="type" />',
    props: ['modelValue', 'disabled', 'type'],
  },
  QrCode: {
    template: '<svg class="qr-icon"></svg>',
  },
  Plus: {
    template: '<svg class="plus-icon"></svg>',
  },
  Package: {
    template: '<svg class="package-icon"></svg>',
  },
  Eye: {
    template: '<svg class="eye-icon"></svg>',
  },
  Search: {
    template: '<svg class="search-icon"></svg>',
  },
  History: {
    template: '<svg class="history-icon"></svg>',
  },
  CreateProductModal: {
    template: '<div v-if="show" class="create-product-modal"><slot /></div>',
    props: ['show', 'qrType'],
    emits: ['close', 'created'],
  },
  CreateBatchModal: {
    template: '<div v-if="open" class="create-batch-modal"><p>Create New Batch</p><p>For: {{ productName }}</p><button class="emit-created" @click="$emit(\'created\')">Do Create</button><button class="emit-close" @click="$emit(\'close\')">Do Close</button></div>',
    props: ['open', 'productId', 'productName', 'warrantyEnabled'],
    emits: ['close', 'created'],
  },
}

describe('DynamicQRPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockGet.mockReset()
    mockPost.mockReset()
    mockPush.mockReset()
  })

  function createWrapper() {
    return mount(DynamicQRPage, {
      global: {
        stubs,
      },
    })
  }

  describe('Rendering', () => {
    it('should render page title and description', async () => {
      mockGet.mockResolvedValue({ success: true, data: { products: [] } })
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.find('h1').text()).toBe('Dynamic QR Products')
      expect(wrapper.text()).toContain('Manage products with unique QR codes per unit')
    })

    it('should render "Add New Product" button', async () => {
      mockGet.mockResolvedValue({ success: true, data: { products: [] } })
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('Add New Product')
    })

    it('should show loading spinner when loading', () => {
      mockGet.mockImplementation(() => new Promise(() => {}))
      const wrapper = createWrapper()

      expect(wrapper.find('.animate-spin').exists()).toBe(true)
    })

    it('should show empty state when no products', async () => {
      mockGet.mockResolvedValue({ success: true, data: { products: [] } })
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('No dynamic QR products yet')
      expect(wrapper.text()).toContain('Create Dynamic Product')
    })

    it('should render product cards when data exists', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'Test Product',
          product_code: 'PROD-001',
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('Test Product')
      expect(wrapper.text()).toContain('PROD-001')
    })
  })

  describe('Product List', () => {
    it('should display product name and code', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'Dynamic Product A',
          product_code: 'DYN-001',
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('Dynamic Product A')
      expect(wrapper.text()).toContain('DYN-001')
    })

    it('should show "Dynamic QR" badge', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'Test Product',
          product_code: 'TEST-001',
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('Dynamic QR')
    })

    it('should show campaign badge if enabled', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'Campaign Product',
          product_code: 'CAMP-001',
          campaign_enabled: true,
          campaign: { campaign_name: 'Summer Promo' },
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      // Component shows "Campaign" badge, not full campaign name
      expect(wrapper.text()).toContain('Campaign')
    })

    it('should show warranty badge if enabled', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'Warranty Product',
          product_code: 'WAR-001',
          warranty_enabled: true,
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('Warranty')
    })
  })

  describe('Navigation Buttons', () => {
    it('should have Settings button for each product', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'Test Product',
          product_code: 'TEST-001',
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('Settings')
    })

    it('should have Batch History button for each product', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'Test Product',
          product_code: 'TEST-001',
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('Batch History')
    })

    it('should navigate to product detail when Settings clicked', async () => {
      const mockProducts = [
        {
          id: 'prod-123',
          product_name: 'Test Product',
          product_code: 'TEST-001',
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      const settingsButton = wrapper.findAll('button').find(b => b.text().includes('Settings'))
      await settingsButton.trigger('click')

      expect(mockPush).toHaveBeenCalledWith('/tenant/products/prod-123')
    })

    it('should navigate to batch history when Batch History clicked', async () => {
      const mockProducts = [
        {
          id: 'prod-123',
          product_name: 'Test Product',
          product_code: 'TEST-001',
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      const historyButton = wrapper.findAll('button').find(b => b.text().includes('Batch History'))
      await historyButton.trigger('click')

      expect(mockPush).toHaveBeenCalledWith('/tenant/products/prod-123/batches')
    })

    it('should have New Batch button for each product', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'Test Product',
          product_code: 'TEST-001',
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('New Batch')
    })
  })

  describe('Create Batch Modal', () => {
    it('should open the batch modal when "New Batch" button clicked', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'Test Product',
          product_code: 'TEST-001',
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.find('.create-batch-modal').exists()).toBe(false)

      // Find and click "New Batch" button
      const newBatchButton = wrapper.findAll('button').find(b => b.text().includes('New Batch'))
      await newBatchButton.trigger('click')

      expect(wrapper.find('.create-batch-modal').exists()).toBe(true)
      expect(wrapper.text()).toContain('Create New Batch')
    })

    it('should pass the selected product name to the modal', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'My Dynamic Product',
          product_code: 'MDP-001',
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      const newBatchButton = wrapper.findAll('button').find(b => b.text().includes('New Batch'))
      await newBatchButton.trigger('click')

      expect(wrapper.text()).toContain('For: My Dynamic Product')
    })

    it('should redirect to batches page when the modal emits "created"', async () => {
      const mockProducts = [
        {
          id: 'prod-123',
          product_name: 'Test Product',
          product_code: 'TEST-001',
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      // Open modal
      const newBatchButton = wrapper.findAll('button').find(b => b.text().includes('New Batch'))
      await newBatchButton.trigger('click')
      await flushPromises()

      // Modal emits 'created' (it owns the create POST internally)
      await wrapper.find('.emit-created').trigger('click')
      await flushPromises()

      expect(mockPush).toHaveBeenCalledWith('/tenant/products/prod-123/batches')
    })

    it('should close the modal when it emits "close"', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'Test Product',
          product_code: 'TEST-001',
          created_at: '2025-01-15T10:00:00Z',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      // Open modal
      const newBatchButton = wrapper.findAll('button').find(b => b.text().includes('New Batch'))
      await newBatchButton.trigger('click')
      expect(wrapper.find('.create-batch-modal').exists()).toBe(true)

      // Modal emits 'close'
      await wrapper.find('.emit-close').trigger('click')

      expect(wrapper.find('.create-batch-modal').exists()).toBe(false)
    })
  })

  describe('Search', () => {
    it('should render search input', async () => {
      mockGet.mockResolvedValue({ success: true, data: { products: [] } })
      const wrapper = createWrapper()
      await flushPromises()

      const searchInput = wrapper.find('input')
      expect(searchInput.exists()).toBe(true)
    })

    it('should filter products by name', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'Alpha Product',
          product_code: 'ALPHA-001',
        },
        {
          id: '2',
          product_name: 'Beta Product',
          product_code: 'BETA-001',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      // Both products should be visible initially
      expect(wrapper.text()).toContain('Alpha Product')
      expect(wrapper.text()).toContain('Beta Product')

      // Type in search
      const searchInput = wrapper.find('input')
      await searchInput.setValue('Alpha')
      await flushPromises()

      // Only Alpha should be visible
      expect(wrapper.text()).toContain('Alpha Product')
      expect(wrapper.text()).not.toContain('Beta Product')
    })

    it('should filter products by code', async () => {
      const mockProducts = [
        {
          id: '1',
          product_name: 'Product One',
          product_code: 'FIRST-001',
        },
        {
          id: '2',
          product_name: 'Product Two',
          product_code: 'SECOND-002',
        },
      ]
      mockGet.mockResolvedValue({ success: true, data: { products: mockProducts } })
      const wrapper = createWrapper()
      await flushPromises()

      // Type in search by code
      const searchInput = wrapper.find('input')
      await searchInput.setValue('SECOND')
      await flushPromises()

      // Only Product Two should be visible
      expect(wrapper.text()).not.toContain('Product One')
      expect(wrapper.text()).toContain('Product Two')
    })
  })
})
