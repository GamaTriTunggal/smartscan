import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ProductTemplatePreview from '../ProductTemplatePreview.vue'

// Mock the constants module
vi.mock('@/constants/previewOptions', () => ({
  ASPECT_RATIOS: [
    { value: '9/16', label: '9:16' },
    { value: '9/19.5', label: '9:19.5' }
  ],
  DEFAULT_ASPECT_RATIO: '9/19.5'
}))

// Stub lucide icons
const iconStub = { template: '<span class="icon-stub" />' }
const stubs = {
  Image: iconStub,
  Video: iconStub,
  Share2: iconStub,
  Award: iconStub,
  ExternalLink: iconStub,
  FileText: iconStub,
  Shield: iconStub,
  Megaphone: iconStub
}

function createWrapper(props = {}) {
  return mount(ProductTemplatePreview, {
    props: {
      config: {},
      ...props
    },
    global: { stubs }
  })
}

describe('ProductTemplatePreview', () => {
  describe('Loading state', () => {
    it('should render loading spinner when loading is true', () => {
      const wrapper = createWrapper({ loading: true })

      expect(wrapper.find('.animate-spin').exists()).toBe(true)
      expect(wrapper.text()).toContain('Loading template...')
    })

    it('should not render loading spinner when loading is false', () => {
      const wrapper = createWrapper({ loading: false })

      expect(wrapper.find('.animate-spin').exists()).toBe(false)
    })
  })

  describe('Header', () => {
    it('should render browser bar with Live Preview text', () => {
      const wrapper = createWrapper()

      expect(wrapper.text()).toContain('Live Preview')
    })

    it('should render aspect ratio selector', () => {
      const wrapper = createWrapper()

      const select = wrapper.find('select')
      expect(select.exists()).toBe(true)
    })

    it('should display default badge text', () => {
      const wrapper = createWrapper()

      expect(wrapper.text()).toContain('Authentic Product')
    })

    it('should display custom badge text from config', () => {
      const wrapper = createWrapper({
        config: {
          header: { badge_text: 'Premium Quality' }
        }
      })

      expect(wrapper.text()).toContain('Premium Quality')
    })
  })

  describe('Product info', () => {
    it('should render product name and brand', () => {
      const wrapper = createWrapper({
        productName: 'Test Product',
        brandName: 'Test Brand'
      })

      expect(wrapper.text()).toContain('Test Product')
      expect(wrapper.text()).toContain('Test Brand')
    })

    it('should show default product name when not provided', () => {
      const wrapper = createWrapper()

      expect(wrapper.text()).toContain('Sample Product')
    })

    it('should show product code when displayConfig enables it', () => {
      const wrapper = createWrapper({
        productCode: 'PROD-001',
        displayConfig: { product_code: true }
      })

      expect(wrapper.text()).toContain('Product Code')
      expect(wrapper.text()).toContain('PROD-001')
    })
  })

  describe('Verification count', () => {
    it('should show placeholder scan count by default', () => {
      const wrapper = createWrapper()

      expect(wrapper.text()).toContain('3 times')
    })

    it('should hide scan count when displayConfig disables it', () => {
      const wrapper = createWrapper({
        displayConfig: { show_verification_count: false }
      })

      expect(wrapper.text()).not.toContain('3 times')
    })
  })

  describe('Batch fields', () => {
    it('should show batch fields when enabled', () => {
      const wrapper = createWrapper({
        displayConfig: { batch_code: true, production_date: true, expiry_date: true }
      })

      expect(wrapper.text()).toContain('Batch')
      expect(wrapper.text()).toContain('Production Date')
      expect(wrapper.text()).toContain('Expiry Date')
    })
  })

  describe('Certifications section', () => {
    it('should render certifications when provided', () => {
      const wrapper = createWrapper({
        certifications: [
          { id: '1', name: 'ISO 9001' },
          { id: '2', name: 'BPOM' }
        ]
      })

      expect(wrapper.text()).toContain('ISO 9001')
      expect(wrapper.text()).toContain('BPOM')
    })

    it('should hide certifications when empty', () => {
      const wrapper = createWrapper({
        certifications: []
      })

      expect(wrapper.text()).not.toContain('Certifications')
    })

    it('should use custom certifications title from config', () => {
      const wrapper = createWrapper({
        certifications: [{ id: '1', name: 'SNI' }],
        config: {
          certifications_section: { header_text: 'Quality Standards' }
        }
      })

      expect(wrapper.text()).toContain('Quality Standards')
    })
  })

  describe('Social accounts section', () => {
    it('should render social accounts when provided', () => {
      const wrapper = createWrapper({
        socialAccounts: [
          { platform_code: 'INSTAGRAM', platform_name: 'Instagram', url: 'https://instagram.com/test' }
        ]
      })

      // Social section should be visible (either sticky or inline)
      const socialLinks = wrapper.findAll('a[title]')
      expect(socialLinks.length).toBeGreaterThan(0)
    })

    it('should hide social accounts when empty', () => {
      const wrapper = createWrapper({
        socialAccounts: []
      })

      // No social links
      const socialLinks = wrapper.findAll('a[title]')
      expect(socialLinks.length).toBe(0)
    })
  })

  describe('Gallery section', () => {
    it('should render gallery images when provided', () => {
      const wrapper = createWrapper({
        images: [
          { id: '1', image_url: '/uploads/img1.jpg', is_main: true },
          { id: '2', image_url: '/uploads/img2.jpg' },
          { id: '3', image_url: '/uploads/img3.jpg' }
        ]
      })

      expect(wrapper.text()).toContain('Gallery')
    })

    it('should not render gallery when only main image exists', () => {
      const wrapper = createWrapper({
        images: [
          { id: '1', image_url: '/uploads/img1.jpg', is_main: true }
        ]
      })

      // Gallery section shouldn't show when there are no non-main images
      expect(wrapper.text()).not.toContain('Gallery')
    })
  })

  describe('Warranty button', () => {
    it('should show warranty button when enabled', () => {
      const wrapper = createWrapper({
        warrantyEnabled: true
      })

      expect(wrapper.text()).toContain('Activate Warranty')
    })

    it('should show custom warranty button text from config', () => {
      const wrapper = createWrapper({
        warrantyEnabled: true,
        config: {
          warranty_button: { text: 'Register Now' }
        }
      })

      expect(wrapper.text()).toContain('Register Now')
    })

    it('should hide warranty button when disabled', () => {
      const wrapper = createWrapper({
        warrantyEnabled: false
      })

      expect(wrapper.text()).not.toContain('Activate Warranty')
    })
  })

  describe('Description section', () => {
    it('should render description when provided', () => {
      const wrapper = createWrapper({
        description: 'This is a high quality product.'
      })

      expect(wrapper.text()).toContain('Description')
      expect(wrapper.text()).toContain('This is a high quality product.')
    })

    it('should hide description when empty', () => {
      const wrapper = createWrapper({
        description: ''
      })

      expect(wrapper.text()).not.toContain('Description')
    })
  })

  describe('Website button', () => {
    it('should show website button when URL is provided', () => {
      const wrapper = createWrapper({
        websiteUrl: 'https://example.com',
        websiteCaption: 'Visit Our Store'
      })

      expect(wrapper.text()).toContain('Visit Our Store')
    })

    it('should show default text when no caption', () => {
      const wrapper = createWrapper({
        websiteUrl: 'https://example.com'
      })

      expect(wrapper.text()).toContain('Visit Website')
    })

    it('should hide website button when no URL', () => {
      const wrapper = createWrapper({
        websiteUrl: ''
      })

      expect(wrapper.text()).not.toContain('Visit Website')
    })
  })

  // === FEATURE TESTS: Field order reactivity ===
  // Note: product_name is always rendered first (separate header card),
  // then info fields (product_code, verification, batch, dates) follow in a consolidated card.
  // The consolidated card respects field_order for non-header fields.
  describe('Field order (feature test)', () => {
    function getFieldOrder(wrapper) {
      const fields = wrapper.findAll('[data-field]')
      return fields.map(f => f.attributes('data-field'))
    }

    // Get only the info fields (excluding product_name header card)
    function getInfoFieldOrder(wrapper) {
      return getFieldOrder(wrapper).filter(f => f !== 'product_name')
    }

    it('should render info fields in custom order from displayConfig.field_order', () => {
      const wrapper = createWrapper({
        productName: 'My Product',
        productCode: 'CODE-123',
        displayConfig: {
          product_code: true,
          show_verification_count: true,
          field_order: ['show_verification_count', 'product_code', 'product_name']
        }
      })

      // product_name is always first (header card)
      const allFields = getFieldOrder(wrapper)
      expect(allFields[0]).toBe('product_name')

      // Info fields follow the custom order
      const infoFields = getInfoFieldOrder(wrapper)
      expect(infoFields).toEqual(['show_verification_count', 'product_code'])
    })

    it('should render fields in default order when no field_order specified', () => {
      const wrapper = createWrapper({
        productName: 'My Product',
        productCode: 'CODE-123',
        displayConfig: {
          product_code: true,
          show_verification_count: true
        }
      })

      const order = getFieldOrder(wrapper)
      // product_name header is always first
      expect(order[0]).toBe('product_name')
      // Default info field order: product_code before show_verification_count
      const infoFields = getInfoFieldOrder(wrapper)
      expect(infoFields.indexOf('product_code')).toBeLessThan(infoFields.indexOf('show_verification_count'))
    })

    it('should update info field order reactively when displayConfig changes', async () => {
      const wrapper = createWrapper({
        productName: 'My Product',
        productCode: 'CODE-123',
        displayConfig: {
          product_code: true,
          show_verification_count: true,
          field_order: ['product_name', 'product_code', 'show_verification_count']
        }
      })

      // Initial order: product_code before verification
      let infoFields = getInfoFieldOrder(wrapper)
      expect(infoFields).toEqual(['product_code', 'show_verification_count'])

      // Change field order: verification first
      await wrapper.setProps({
        displayConfig: {
          product_code: true,
          show_verification_count: true,
          field_order: ['show_verification_count', 'product_name', 'product_code']
        }
      })

      infoFields = getInfoFieldOrder(wrapper)
      expect(infoFields).toEqual(['show_verification_count', 'product_code'])
    })
  })
})
