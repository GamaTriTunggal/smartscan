import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import CreateBatchModal from '../CreateBatchModal.vue'

// Mock useAPI
const mockGet = vi.fn()
const mockPost = vi.fn()
vi.mock('@/composables/useAPI', () => ({
  useAPI: () => ({ get: mockGet, post: mockPost, getAuthHeaders: () => ({}) }),
}))

// Mock useDateTime
vi.mock('@/composables/useDateTime', () => ({
  useDateTime: () => ({
    toUTCString: (d) => d || null,
    formatDate: (d) => d,
  }),
}))

// Mock useEscapeKey
vi.mock('@/composables/useEscapeKey', () => ({ useEscapeKey: vi.fn() }))

// Mock qr generation store (avoid polling side effects)
const mockTrackNewBatch = vi.fn()
vi.mock('@/stores/qrGeneration', () => ({
  useQRGenerationStore: () => ({ trackNewBatch: mockTrackNewBatch }),
}))

// Mock useBatchGeofence
vi.mock('@/composables/useBatchGeofence', () => ({
  useBatchGeofence: () => ({
    geofenceData: ref({ latitude: null, longitude: null, radius_km: 25, label: '' }),
    zoneTemplates: ref([]),
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
    buildGeofencePayload: () => ({}),
  }),
}))

const stubs = {
  Button: {
    template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
    props: ['disabled', 'variant', 'size'],
  },
  Input: {
    template: '<input :value="modelValue" :type="type" @input="$emit(\'update:modelValue\', $event.target.value)" />',
    props: ['modelValue', 'type', 'placeholder'],
  },
  GeofenceMapPicker: { template: '<div class="geofence-picker-stub" />' },
}

function createWrapper(props = {}) {
  return mount(CreateBatchModal, {
    props: { open: true, productId: 'prod-1', productName: 'Test Product', ...props },
    global: { stubs },
  })
}

describe('CreateBatchModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders nothing when closed', () => {
    const wrapper = createWrapper({ open: false })
    expect(wrapper.text()).not.toContain('Create New Batch')
  })

  it('renders the form and product name when open', () => {
    const wrapper = createWrapper()
    expect(wrapper.text()).toContain('Create New Batch')
    expect(wrapper.text()).toContain('For: Test Product')
    expect(wrapper.text()).toContain('Batch Name')
    expect(wrapper.text()).toContain('Number of QR Codes')
  })

  it('creates a batch, tracks it, and emits created + close', async () => {
    mockPost.mockResolvedValue({ success: true, data: { id: 'batch-1' } })
    const wrapper = createWrapper()

    // batch name is the first input
    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('Spring Batch')

    const createBtn = wrapper.findAll('button').find(b => b.text().includes('Create Batch'))
    await createBtn.trigger('click')
    await flushPromises()

    expect(mockPost).toHaveBeenCalledWith('/tenant/qr-batches', expect.objectContaining({
      product_id: 'prod-1',
      batch_name: 'Spring Batch',
    }))
    expect(mockTrackNewBatch).toHaveBeenCalledWith({ id: 'batch-1' })
    expect(wrapper.emitted('created')).toBeTruthy()
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('shows an error message when creation fails', async () => {
    mockPost.mockResolvedValue({ success: false, message: 'Batch limit reached' })
    const wrapper = createWrapper()

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('Overflow Batch')

    const createBtn = wrapper.findAll('button').find(b => b.text().includes('Create Batch'))
    await createBtn.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Batch limit reached')
    expect(wrapper.emitted('created')).toBeFalsy()
  })
})
