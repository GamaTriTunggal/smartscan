import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import ProductBatchHistoryPage from '../ProductBatchHistoryPage.vue'

// Mock vue-router
const mockPush = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({
    params: { productId: 'product-uuid-1' },
  }),
  useRouter: () => ({
    push: mockPush,
  }),
}))

// Mock useAPI
const mockGet = vi.fn()
const mockPost = vi.fn()
const mockDel = vi.fn()
const mockPut = vi.fn()
vi.mock('@/composables/useAPI', () => ({
  useAPI: () => ({
    get: mockGet,
    post: mockPost,
    del: mockDel,
    put: mockPut,
    getAuthHeaders: vi.fn(() => ({})),
  }),
}))

// Mock useDateTime
vi.mock('@/composables/useDateTime', () => ({
  useDateTime: () => ({
    formatDate: vi.fn((date) => date ? new Date(date).toLocaleDateString() : ''),
    toUTCString: vi.fn((date) => date || null),
  }),
}))

// Mock useEscapeKey
vi.mock('@/composables/useEscapeKey', () => ({
  useEscapeKey: vi.fn(),
}))

// Mock useBatchGeofence
vi.mock('@/composables/useBatchGeofence', () => ({
  useBatchGeofence: () => ({
    geofenceData: ref({ latitude: null, longitude: null, radius_km: 25, label: '' }),
    zoneTemplates: ref([]),
    selectedZoneTemplateId: ref(null),
    getDefaultGeofenceFields: () => ({
      geofence_enabled: false,
      geofence_latitude: null,
      geofence_longitude: null,
      geofence_radius_km: 25,
      geofence_label: '',
    }),
    onGeofenceUpdate: vi.fn(),
    loadZoneTemplate: vi.fn(),
    fetchZoneTemplates: vi.fn(),
    resetGeofence: vi.fn(),
    buildGeofencePayload: vi.fn(() => ({})),
  }),
}))

// Stub components
const stubs = {
  Card: { template: '<div class="card"><slot /></div>' },
  Button: {
    template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
    props: ['disabled', 'variant', 'size'],
  },
  Input: {
    template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
    props: ['modelValue', 'disabled', 'type', 'placeholder'],
  },
  GeofenceMapPicker: { template: '<div class="geofence-picker-stub" />' },
  CreateBatchModal: {
    template: '<div v-if="open" class="create-batch-modal" />',
    props: ['open', 'productId', 'productName', 'warrantyEnabled'],
    emits: ['close', 'created'],
  },
  BatchPdfDownload: {
    template: '<button class="pdf-download-stub" />',
    props: ['batchId', 'qrCount', 'batchCode', 'size'],
  },
  ArrowLeft: { template: '<svg />' },
  QrCode: { template: '<svg />' },
  Download: { template: '<svg />' },
  Printer: { template: '<svg />' },
  Plus: { template: '<svg />' },
  Package: { template: '<svg />' },
  Eye: { template: '<svg />' },
  Search: { template: '<svg />' },
  Calendar: { template: '<svg />' },
  Shield: { template: '<svg />' },
  Megaphone: { template: '<svg />' },
  Trash2: { template: '<svg />' },
  RotateCcw: { template: '<svg />' },
  MapPin: { template: '<svg class="map-pin-icon" />' },
}

// Test data
const mockProduct = {
  id: 'product-uuid-1',
  product_name: 'Test Product',
  product_code: 'TP-001',
  warranty_enabled: false,
}

const batchWithGeofence = {
  id: 'batch-geo-1',
  batch_name: 'Geofenced Batch',
  batch_code: 'GEO-001',
  qr_count: 100,
  production_date: '2026-01-01T00:00:00Z',
  expiry_date: '2027-01-01T00:00:00Z',
  geofence_enabled: true,
  geofence_latitude: -6.2088,
  geofence_longitude: 106.8456,
  geofence_radius_km: 50,
  geofence_label: 'Jakarta Zone',
  deleted_at: null,
  scan_count: 5,
}

const batchWithoutGeofence = {
  id: 'batch-no-geo',
  batch_name: 'Normal Batch',
  batch_code: 'NRM-001',
  qr_count: 50,
  production_date: null,
  expiry_date: null,
  geofence_enabled: false,
  geofence_latitude: null,
  geofence_longitude: null,
  geofence_radius_km: null,
  geofence_label: '',
  deleted_at: null,
  scan_count: 0,
}

const batchGeofenceNoLabel = {
  id: 'batch-geo-nolabel',
  batch_name: 'Unlabeled Geo Batch',
  batch_code: 'ULB-001',
  qr_count: 25,
  production_date: null,
  expiry_date: null,
  geofence_enabled: true,
  geofence_latitude: -7.25,
  geofence_longitude: 112.75,
  geofence_radius_km: 30,
  geofence_label: '',
  deleted_at: null,
  scan_count: 0,
}

const deletedBatchWithGeofence = {
  id: 'batch-deleted-geo',
  batch_name: 'Deleted Geo Batch',
  batch_code: 'DG-001',
  qr_count: 75,
  production_date: null,
  expiry_date: null,
  geofence_enabled: true,
  geofence_latitude: -6.9,
  geofence_longitude: 107.6,
  geofence_radius_km: 15,
  geofence_label: 'Deleted Zone',
  deleted_at: '2026-02-15T00:00:00Z',
  scan_count: 0,
}

function setupMockAPI(batches = [batchWithGeofence, batchWithoutGeofence]) {
  mockGet.mockImplementation((url) => {
    if (url.includes('/tenant/products/')) {
      return Promise.resolve({ success: true, data: mockProduct })
    }
    if (url.includes('/tenant/qr-batches')) {
      return Promise.resolve({
        success: true,
        data: {
          batches,
          pagination: { total: batches.length, page: 1, limit: 20, total_page: 1 },
        },
      })
    }
    return Promise.resolve({ success: true, data: null })
  })
}

describe('ProductBatchHistoryPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockGet.mockReset()
    mockPost.mockReset()
    mockDel.mockReset()
    mockPut.mockReset()
    mockPush.mockReset()
  })

  function createWrapper() {
    return mount(ProductBatchHistoryPage, {
      global: { stubs },
    })
  }

  describe('Geofence Badge', () => {
    it('should show geofence badge when geofence_enabled is true', async () => {
      setupMockAPI([batchWithGeofence])
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('Jakarta Zone')
      expect(wrapper.text()).toContain('50 km')
    })

    it('should not show geofence badge when geofence_enabled is false', async () => {
      setupMockAPI([batchWithoutGeofence])
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).not.toContain('Distribution Zone')
      expect(wrapper.find('.map-pin-icon').exists()).toBe(false)
    })

    it('should show fallback "Distribution Zone" when geofence_label is empty', async () => {
      setupMockAPI([batchGeofenceNoLabel])
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('Distribution Zone')
      expect(wrapper.text()).toContain('30 km')
    })

    it('should navigate to batch detail when geofence badge is clicked', async () => {
      setupMockAPI([batchWithGeofence])
      const wrapper = createWrapper()
      await flushPromises()

      // Find the geofence badge span by matching text content
      const allSpans = wrapper.findAll('span')
      const geoBadge = allSpans.find(s => {
        const text = s.text()
        return text.includes('Jakarta Zone') && text.includes('50 km')
      })
      expect(geoBadge).toBeTruthy()
      await geoBadge.trigger('click')

      expect(mockPush).toHaveBeenCalledWith('/tenant/qr-batches/batch-geo-1')
    })

    it('should show geofence badge on deleted batches', async () => {
      setupMockAPI([deletedBatchWithGeofence])
      mockGet.mockImplementation((url) => {
        if (url.includes('/tenant/products/')) {
          return Promise.resolve({ success: true, data: mockProduct })
        }
        if (url.includes('/tenant/qr-batches')) {
          return Promise.resolve({
            success: true,
            data: {
              batches: [deletedBatchWithGeofence],
              pagination: { total: 1, page: 1, limit: 20, total_page: 1 },
            },
          })
        }
        return Promise.resolve({ success: true, data: null })
      })
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('Deleted Zone')
      expect(wrapper.text()).toContain('15 km')
      expect(wrapper.text()).toContain('Deleted') // deleted badge also shows
    })
  })
})
