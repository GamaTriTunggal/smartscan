import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import WarrantySettingsPreview from '../WarrantySettingsPreview.vue'

// Mock the constants module
vi.mock('@/constants/previewOptions', () => ({
  ASPECT_RATIOS: [
    { value: '9/16', label: '9:16' },
    { value: '9/19.5', label: '9:19.5' }
  ],
  DEFAULT_ASPECT_RATIO: '9/19.5'
}))

const defaultFieldsConfig = {
  enabled: true,
  fields: {
    store_name: 'hidden',
    country: 'hidden',
    province: 'hidden',
    city: 'hidden',
    address: 'hidden'
  }
}

function createWrapper(props = {}) {
  return mount(WarrantySettingsPreview, {
    props: {
      fieldsConfig: defaultFieldsConfig,
      ...props
    }
  })
}

describe('WarrantySettingsPreview', () => {
  // === 1. Rendering ===
  describe('Rendering', () => {
    it('should render the preview container', () => {
      const wrapper = createWrapper()
      expect(wrapper.find('.bg-gray-100').exists()).toBe(true)
    })

    it('should render "Warranty Preview" header text', () => {
      const wrapper = createWrapper()
      expect(wrapper.text()).toContain('Warranty Preview')
    })

    it('should render aspect ratio selector', () => {
      const wrapper = createWrapper()
      const select = wrapper.find('select')
      expect(select.exists()).toBe(true)
      const options = select.findAll('option')
      expect(options.length).toBe(2)
    })

    it('should render phone frame with border', () => {
      const wrapper = createWrapper()
      expect(wrapper.find('.rounded-\\[32px\\]').exists()).toBe(true)
    })

    it('should render three preview mode buttons', () => {
      const wrapper = createWrapper()
      const buttons = wrapper.findAll('button')
      const modeButtons = buttons.filter(b =>
        ['Form', 'Success', 'Error'].includes(b.text().trim())
      )
      expect(modeButtons.length).toBe(3)
    })
  })

  // === 2. Disabled state ===
  describe('Disabled state', () => {
    it('should show "Warranty disabled" when fieldsConfig.enabled is false', () => {
      const wrapper = createWrapper({
        fieldsConfig: { enabled: false, fields: {} }
      })
      expect(wrapper.text()).toContain('Warranty disabled')
    })

    it('should not show form fields when disabled', () => {
      const wrapper = createWrapper({
        fieldsConfig: { enabled: false, fields: {} }
      })
      expect(wrapper.text()).not.toContain('Full Name')
      expect(wrapper.text()).not.toContain('Email')
    })

    it('should still show header when disabled', () => {
      const wrapper = createWrapper({
        fieldsConfig: { enabled: false, fields: {} },
        productName: 'Test Product'
      })
      expect(wrapper.text()).toContain('Warranty Activation')
      expect(wrapper.text()).toContain('Test Product')
    })
  })

  // === 3. Preview modes ===
  describe('Preview modes', () => {
    it('should show form view by default', () => {
      const wrapper = createWrapper()
      expect(wrapper.text()).toContain('Full Name')
      expect(wrapper.text()).toContain('Activate Warranty')
    })

    it('should switch to success view when Success button is clicked', async () => {
      const wrapper = createWrapper()
      const successBtn = wrapper.findAll('button').find(b => b.text().trim() === 'Success')
      await successBtn.trigger('click')

      expect(wrapper.text()).toContain('Warranty Activated!')
      expect(wrapper.text()).toContain('Your product warranty has been successfully activated.')
      expect(wrapper.text()).not.toContain('Full Name')
    })

    it('should switch to error view when Error button is clicked', async () => {
      const wrapper = createWrapper()
      const errorBtn = wrapper.findAll('button').find(b => b.text().trim() === 'Error')
      await errorBtn.trigger('click')

      expect(wrapper.text()).toContain('Already Activated')
      expect(wrapper.text()).toContain('This product warranty has already been activated.')
      expect(wrapper.text()).not.toContain('Full Name')
    })

    it('should switch back to form view from success', async () => {
      const wrapper = createWrapper()
      // Go to success
      const successBtn = wrapper.findAll('button').find(b => b.text().trim() === 'Success')
      await successBtn.trigger('click')
      expect(wrapper.text()).toContain('Warranty Activated!')

      // Back to form
      const formBtn = wrapper.findAll('button').find(b => b.text().trim() === 'Form')
      await formBtn.trigger('click')
      expect(wrapper.text()).toContain('Full Name')
      expect(wrapper.text()).not.toContain('Warranty Activated!')
    })

  })

  // === 4. Fixed fields ===
  describe('Fixed fields', () => {
    it('should always show Full Name with asterisk', () => {
      const wrapper = createWrapper()
      const labels = wrapper.findAll('label')
      const nameLabel = labels.find(l => l.text().includes('Full Name'))
      expect(nameLabel).toBeDefined()
      expect(nameLabel.text()).toContain('*')
    })

    it('should always show Email with asterisk', () => {
      const wrapper = createWrapper()
      const labels = wrapper.findAll('label')
      const emailLabel = labels.find(l => l.text().includes('Email'))
      expect(emailLabel).toBeDefined()
      expect(emailLabel.text()).toContain('*')
    })

    it('should always show Phone with asterisk', () => {
      const wrapper = createWrapper()
      const labels = wrapper.findAll('label')
      const phoneLabel = labels.find(l => l.text().includes('Phone'))
      expect(phoneLabel).toBeDefined()
      expect(phoneLabel.text()).toContain('*')
    })

    it('should always show Purchase Date with asterisk', () => {
      const wrapper = createWrapper()
      const labels = wrapper.findAll('label')
      const dateLabel = labels.find(l => l.text().includes('Purchase Date'))
      expect(dateLabel).toBeDefined()
      expect(dateLabel.text()).toContain('*')
    })

    it('should render all fixed fields as disabled inputs', () => {
      const wrapper = createWrapper()
      const inputs = wrapper.findAll('input[disabled]')
      // At least 4 fixed inputs: name, email, phone, date
      expect(inputs.length).toBeGreaterThanOrEqual(4)
    })
  })

  // === 5. Configurable fields ===
  describe('Configurable fields', () => {
    it('should show store_name when set to optional', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'optional', country: 'hidden', province: 'hidden', city: 'hidden', address: 'hidden' }
        }
      })
      expect(wrapper.text()).toContain('Store Name')
    })

    it('should show store_name when set to required', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'required', country: 'hidden', province: 'hidden', city: 'hidden', address: 'hidden' }
        }
      })
      expect(wrapper.text()).toContain('Store Name')
    })

    it('should hide store_name when set to hidden', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'hidden', country: 'hidden', province: 'hidden', city: 'hidden', address: 'hidden' }
        }
      })
      expect(wrapper.text()).not.toContain('Store Name')
    })

    it('should show country when visible', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'hidden', country: 'optional', province: 'hidden', city: 'hidden', address: 'hidden' }
        }
      })
      expect(wrapper.text()).toContain('Country')
    })

    it('should show province when visible', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'hidden', country: 'hidden', province: 'required', city: 'hidden', address: 'hidden' }
        }
      })
      expect(wrapper.text()).toContain('Province')
    })

    it('should show city when visible', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'hidden', country: 'hidden', province: 'hidden', city: 'optional', address: 'hidden' }
        }
      })
      expect(wrapper.text()).toContain('City')
    })

    it('should show Full Address when address is visible', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'hidden', country: 'hidden', province: 'hidden', city: 'hidden', address: 'required' }
        }
      })
      expect(wrapper.text()).toContain('Full Address')
    })

    it('should reactively show/hide fields when props change', async () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'hidden', country: 'hidden', province: 'hidden', city: 'hidden', address: 'hidden' }
        }
      })
      expect(wrapper.text()).not.toContain('Store Name')

      await wrapper.setProps({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'optional', country: 'hidden', province: 'hidden', city: 'hidden', address: 'hidden' }
        }
      })
      expect(wrapper.text()).toContain('Store Name')
    })
  })

  // === 6. Required indicators ===
  describe('Required indicators', () => {
    it('should show asterisk for required store_name', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'required', country: 'hidden', province: 'hidden', city: 'hidden', address: 'hidden' }
        }
      })
      const labels = wrapper.findAll('label')
      const storeLabel = labels.find(l => l.text().includes('Store Name'))
      expect(storeLabel.text()).toContain('*')
    })

    it('should not show asterisk for optional store_name', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'optional', country: 'hidden', province: 'hidden', city: 'hidden', address: 'hidden' }
        }
      })
      const labels = wrapper.findAll('label')
      const storeLabel = labels.find(l => l.text().includes('Store Name'))
      // The label should contain "Store Name" but NOT have a red asterisk span
      const asteriskSpan = storeLabel.find('.text-red-500')
      expect(asteriskSpan.exists()).toBe(false)
    })

    it('should show asterisk for required address fields', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'hidden', country: 'required', province: 'required', city: 'required', address: 'required' }
        }
      })
      const labels = wrapper.findAll('label')

      const countryLabel = labels.find(l => l.text().includes('Country'))
      expect(countryLabel.find('.text-red-500').exists()).toBe(true)

      const provinceLabel = labels.find(l => l.text().includes('Province'))
      expect(provinceLabel.find('.text-red-500').exists()).toBe(true)

      const cityLabel = labels.find(l => l.text().includes('City'))
      expect(cityLabel.find('.text-red-500').exists()).toBe(true)

      const addressLabel = labels.find(l => l.text().includes('Full Address'))
      expect(addressLabel.find('.text-red-500').exists()).toBe(true)
    })

    it('should not show asterisk for optional address fields', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'hidden', country: 'optional', province: 'optional', city: 'optional', address: 'optional' }
        }
      })
      const labels = wrapper.findAll('label')

      const countryLabel = labels.find(l => l.text().includes('Country'))
      expect(countryLabel.find('.text-red-500').exists()).toBe(false)

      const cityLabel = labels.find(l => l.text().includes('City'))
      expect(cityLabel.find('.text-red-500').exists()).toBe(false)
    })
  })

  // === 7. Address section ===
  describe('Address section', () => {
    it('should show "Your Address" divider when at least one address field is visible', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'hidden', country: 'optional', province: 'hidden', city: 'hidden', address: 'hidden' }
        }
      })
      expect(wrapper.text()).toContain('Your Address')
      expect(wrapper.text()).toContain('Required for warranty service')
    })

    it('should hide "Your Address" divider when all address fields are hidden', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'hidden', country: 'hidden', province: 'hidden', city: 'hidden', address: 'hidden' }
        }
      })
      expect(wrapper.text()).not.toContain('Your Address')
    })

    it('should show divider when only address field is visible', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'hidden', country: 'hidden', province: 'hidden', city: 'hidden', address: 'optional' }
        }
      })
      expect(wrapper.text()).toContain('Your Address')
    })

    it('should show divider when multiple address fields are visible', () => {
      const wrapper = createWrapper({
        fieldsConfig: {
          enabled: true,
          fields: { store_name: 'hidden', country: 'required', province: 'optional', city: 'required', address: 'optional' }
        }
      })
      expect(wrapper.text()).toContain('Your Address')
      expect(wrapper.text()).toContain('Country')
      expect(wrapper.text()).toContain('Province')
      expect(wrapper.text()).toContain('City')
      expect(wrapper.text()).toContain('Full Address')
    })
  })

  // === 8. Custom fields ===
  describe('Custom fields', () => {
    it('should show "Additional Information" header when custom fields exist', () => {
      const wrapper = createWrapper({
        customFields: [
          { id: '1', label: 'Serial Number', type: 'text', required: false }
        ]
      })
      expect(wrapper.text()).toContain('Additional Information')
    })

    it('should not show "Additional Information" when no custom fields', () => {
      const wrapper = createWrapper({
        customFields: []
      })
      expect(wrapper.text()).not.toContain('Additional Information')
    })

    it('should render text input for text type field', () => {
      const wrapper = createWrapper({
        customFields: [
          { id: '1', label: 'Serial Number', type: 'text', required: false }
        ]
      })
      expect(wrapper.text()).toContain('Serial Number')
      const inputs = wrapper.findAll('input[type="text"]')
      // Fixed fields (name, phone) + custom text field
      expect(inputs.length).toBeGreaterThanOrEqual(3)
    })

    it('should render textarea for textarea type field', () => {
      const wrapper = createWrapper({
        customFields: [
          { id: '1', label: 'Notes', type: 'textarea', required: false }
        ]
      })
      expect(wrapper.text()).toContain('Notes')
      const textareas = wrapper.findAll('textarea')
      expect(textareas.length).toBeGreaterThanOrEqual(1)
    })

    it('should render select for select type field with options', () => {
      const wrapper = createWrapper({
        customFields: [
          { id: '1', label: 'Color', type: 'select', required: false, options: ['Red', 'Blue', 'Green'] }
        ]
      })
      expect(wrapper.text()).toContain('Color')
      // Find select that has "Select..." option + custom options
      const selects = wrapper.findAll('select')
      const colorSelect = selects.find(s => s.text().includes('Red'))
      expect(colorSelect).toBeDefined()
      expect(colorSelect.text()).toContain('Blue')
      expect(colorSelect.text()).toContain('Green')
    })

    it('should render number input for number type', () => {
      const wrapper = createWrapper({
        customFields: [
          { id: '1', label: 'Quantity', type: 'number', required: false }
        ]
      })
      expect(wrapper.text()).toContain('Quantity')
      const numberInput = wrapper.find('input[type="number"]')
      expect(numberInput.exists()).toBe(true)
    })

    it('should render date input for date type', () => {
      const wrapper = createWrapper({
        customFields: [
          { id: '1', label: 'Install Date', type: 'date', required: false }
        ]
      })
      expect(wrapper.text()).toContain('Install Date')
      // There are 2 date inputs: Purchase Date (fixed) + Install Date (custom)
      const dateInputs = wrapper.findAll('input[type="date"]')
      expect(dateInputs.length).toBe(2)
    })

    it('should show asterisk for required custom fields', () => {
      const wrapper = createWrapper({
        customFields: [
          { id: '1', label: 'Serial Number', type: 'text', required: true }
        ]
      })
      const labels = wrapper.findAll('label')
      const serialLabel = labels.find(l => l.text().includes('Serial Number'))
      expect(serialLabel.find('.text-red-500').exists()).toBe(true)
    })

    it('should not show asterisk for non-required custom fields', () => {
      const wrapper = createWrapper({
        customFields: [
          { id: '1', label: 'Serial Number', type: 'text', required: false }
        ]
      })
      const labels = wrapper.findAll('label')
      const serialLabel = labels.find(l => l.text().includes('Serial Number'))
      expect(serialLabel.find('.text-red-500').exists()).toBe(false)
    })

    it('should show "Untitled Field" when label is empty', () => {
      const wrapper = createWrapper({
        customFields: [
          { id: '1', label: '', type: 'text', required: false }
        ]
      })
      expect(wrapper.text()).toContain('Untitled Field')
    })
  })

  // === 9. Template config styling ===
  describe('Template config styling', () => {
    it('should use default colors when no templateConfig', () => {
      const wrapper = createWrapper()
      const header = wrapper.find('.px-4.py-6.text-center')
      expect(header.attributes('style')).toContain('background-color: #18181b')
    })

    it('should apply custom header background color', () => {
      const wrapper = createWrapper({
        templateConfig: {
          styling: { header_bg_color: '#ff0000' }
        }
      })
      const header = wrapper.find('.px-4.py-6.text-center')
      expect(header.attributes('style')).toContain('background-color: #ff0000')
    })

    it('should apply custom submit button text', () => {
      const wrapper = createWrapper({
        templateConfig: {
          submit_button: { text: 'Register Warranty' }
        }
      })
      expect(wrapper.text()).toContain('Register Warranty')
      expect(wrapper.text()).not.toContain('Activate Warranty')
    })

    it('should apply custom submit button colors', () => {
      const wrapper = createWrapper({
        templateConfig: {
          submit_button: { bg_color: '#22c55e', text_color: '#000000' }
        }
      })
      // Find the submit button (last button in form area that is not a mode toggle button)
      const allButtons = wrapper.findAll('button')
      const submitBtn = allButtons.find(b => b.text().trim() === 'Activate Warranty')
      expect(submitBtn.attributes('style')).toContain('background-color: #22c55e')
      expect(submitBtn.attributes('style')).toContain('color: #000000')
    })

    it('should apply custom success messages', async () => {
      const wrapper = createWrapper({
        templateConfig: {
          messages: {
            success_title: 'Registered!',
            success_message: 'Your warranty is now active.'
          }
        }
      })
      const successBtn = wrapper.findAll('button').find(b => b.text().trim() === 'Success')
      await successBtn.trigger('click')

      expect(wrapper.text()).toContain('Registered!')
      expect(wrapper.text()).toContain('Your warranty is now active.')
    })

    it('should apply custom error messages', async () => {
      const wrapper = createWrapper({
        templateConfig: {
          messages: {
            already_activated_title: 'Duplicate!',
            already_activated_message: 'This product was already registered.'
          }
        }
      })
      const errorBtn = wrapper.findAll('button').find(b => b.text().trim() === 'Error')
      await errorBtn.trigger('click')

      expect(wrapper.text()).toContain('Duplicate!')
      expect(wrapper.text()).toContain('This product was already registered.')
    })

    it('should apply custom text color to labels', () => {
      const wrapper = createWrapper({
        templateConfig: {
          styling: { text_color: '#dc2626' }
        }
      })
      const labels = wrapper.findAll('label')
      const nameLabel = labels.find(l => l.text().includes('Full Name'))
      expect(nameLabel.attributes('style')).toContain('color: #dc2626')
    })
  })

  // === 10. Product info ===
  describe('Product info', () => {
    it('should display product name in header', () => {
      const wrapper = createWrapper({
        productName: 'Premium Widget'
      })
      expect(wrapper.text()).toContain('Premium Widget')
    })

    it('should show "Product Name" fallback when productName is empty', () => {
      const wrapper = createWrapper({
        productName: ''
      })
      expect(wrapper.text()).toContain('Product Name')
    })

    it('should display warranty months', () => {
      const wrapper = createWrapper({
        warrantyMonths: 24
      })
      expect(wrapper.text()).toContain('24 months warranty')
    })

    it('should use default 12 months when warrantyMonths not provided', () => {
      const wrapper = createWrapper()
      expect(wrapper.text()).toContain('12 months warranty')
    })

    it('should hide warranty months text when warrantyMonths is 0', () => {
      const wrapper = createWrapper({
        warrantyMonths: 0
      })
      expect(wrapper.text()).not.toContain('months warranty')
    })

    it('should always show "Warranty Activation" title', () => {
      const wrapper = createWrapper()
      expect(wrapper.text()).toContain('Warranty Activation')
    })
  })

  // === Edge cases ===
  describe('Edge cases', () => {
    it('should handle fieldsConfig with missing fields object', () => {
      const wrapper = createWrapper({
        fieldsConfig: { enabled: true }
      })
      // Should render form without crashing, no configurable fields shown
      expect(wrapper.text()).toContain('Full Name')
      expect(wrapper.text()).not.toContain('Store Name')
      expect(wrapper.text()).not.toContain('Your Address')
    })

    it('should handle null templateConfig gracefully', () => {
      const wrapper = createWrapper({
        templateConfig: null
      })
      // Should use all defaults
      expect(wrapper.text()).toContain('Activate Warranty')
      expect(wrapper.text()).toContain('Warranty Activation')
    })

    it('should handle empty customFields array', () => {
      const wrapper = createWrapper({
        customFields: []
      })
      expect(wrapper.text()).not.toContain('Additional Information')
    })

    it('should handle custom field with undefined options for select type', () => {
      const wrapper = createWrapper({
        customFields: [
          { id: '1', label: 'Choice', type: 'select', required: false }
          // no options property
        ]
      })
      expect(wrapper.text()).toContain('Choice')
      expect(wrapper.text()).toContain('Select...')
    })

    it('should handle unknown custom field type', () => {
      const wrapper = createWrapper({
        customFields: [
          { id: '1', label: 'Custom', type: 'unknown_type', required: false }
        ]
      })
      // Should render as text input (fallback)
      expect(wrapper.text()).toContain('Custom')
    })
  })
})
