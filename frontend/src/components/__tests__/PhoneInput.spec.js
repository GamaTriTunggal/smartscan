import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import PhoneInput from '../PhoneInput.vue'

describe('PhoneInput', () => {
  function createWrapper(props = {}) {
    return mount(PhoneInput, {
      props: {
        modelValue: '',
        ...props
      }
    })
  }

  describe('Initial Rendering', () => {
    it('should render country code selector and input', () => {
      const wrapper = createWrapper()

      expect(wrapper.find('button').exists()).toBe(true)
      expect(wrapper.find('input[type="tel"]').exists()).toBe(true)
    })

    it('should default to Indonesia (+62)', async () => {
      const wrapper = createWrapper()
      await flushPromises()

      expect(wrapper.text()).toContain('+62')
    })

    it('should show dropdown arrow', () => {
      const wrapper = createWrapper()

      expect(wrapper.find('svg').exists()).toBe(true)
    })

    it('should respect defaultCountry prop', async () => {
      const wrapper = createWrapper({ defaultCountry: 'SG' })
      await flushPromises()

      expect(wrapper.text()).toContain('+65')
    })

    it('should respect disabled prop', () => {
      const wrapper = createWrapper({ disabled: true })

      expect(wrapper.find('button').attributes('disabled')).toBeDefined()
      expect(wrapper.find('input').attributes('disabled')).toBeDefined()
    })

    it('should show custom placeholder', () => {
      const wrapper = createWrapper({ placeholder: 'Enter phone' })

      expect(wrapper.find('input').attributes('placeholder')).toBe('Enter phone')
    })
  })

  describe('Country Code Selection', () => {
    it('should toggle dropdown on button click', async () => {
      const wrapper = createWrapper()

      // Initially dropdown should be hidden
      expect(wrapper.findAll('.absolute.z-50').length).toBe(0)

      // Click to open
      await wrapper.find('button').trigger('click')
      expect(wrapper.find('.absolute.z-50').exists()).toBe(true)

      // Click to close
      await wrapper.find('button').trigger('click')
      // Dropdown should be hidden (closed)
    })

    it('should show country list in dropdown', async () => {
      const wrapper = createWrapper()

      await wrapper.find('button').trigger('click')

      expect(wrapper.text()).toContain('Indonesia')
      expect(wrapper.text()).toContain('Malaysia')
      expect(wrapper.text()).toContain('Singapore')
      expect(wrapper.text()).toContain('United States')
    })

    it('should select country from dropdown', async () => {
      const wrapper = createWrapper()
      await flushPromises()

      await wrapper.find('button').trigger('click')

      // Find and click Malaysia option
      const malaysiaButton = wrapper.findAll('.z-50 button').find(b => b.text().includes('Malaysia'))
      await malaysiaButton.trigger('click')
      await flushPromises()

      // Should show Malaysia dial code
      expect(wrapper.find('button').text()).toContain('+60')
    })

    it('should close dropdown after selection', async () => {
      const wrapper = createWrapper()

      await wrapper.find('button').trigger('click')

      const malaysiaButton = wrapper.findAll('.z-50 button').find(b => b.text().includes('Malaysia'))
      await malaysiaButton.trigger('click')
      await flushPromises()

      // Dropdown should be hidden
      expect(wrapper.findAll('.absolute.z-50').length).toBe(0)
    })
  })

  describe('Number Input', () => {
    it('should accept numeric input', async () => {
      const wrapper = createWrapper()
      await flushPromises()

      await wrapper.find('input').setValue('812345678')

      expect(wrapper.find('input').element.value).toBe('812345678')
    })

    it('should strip leading zeros', async () => {
      const wrapper = createWrapper()
      await flushPromises()

      await wrapper.find('input').setValue('0812345678')
      await wrapper.find('input').trigger('input')
      await flushPromises()

      // Leading zeros should be removed
      expect(wrapper.find('input').element.value).toBe('812345678')
    })

    it('should strip non-numeric characters', async () => {
      const wrapper = createWrapper()
      await flushPromises()

      const input = wrapper.find('input')
      await input.setValue('812-345-678')
      await input.trigger('input')
      await flushPromises()

      expect(input.element.value).toBe('812345678')
    })
  })

  describe('E.164 Formatting', () => {
    it('should emit E.164 format on input', async () => {
      const wrapper = createWrapper()
      await flushPromises()

      await wrapper.find('input').setValue('812345678')
      await wrapper.find('input').trigger('input')
      await flushPromises()

      // Should emit +62812345678
      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      const emissions = wrapper.emitted('update:modelValue')
      const lastEmission = emissions[emissions.length - 1][0]
      expect(lastEmission).toBe('+62812345678')
    })

    it('should show formatted number preview', async () => {
      const wrapper = createWrapper()
      await flushPromises()

      await wrapper.find('input').setValue('812345678')
      await wrapper.find('input').trigger('input')
      await flushPromises()

      expect(wrapper.text()).toContain('Format: +62812345678')
    })

    it('should change format when country changes', async () => {
      const wrapper = createWrapper()
      await flushPromises()

      // Enter number first
      await wrapper.find('input').setValue('123456789')
      await wrapper.find('input').trigger('input')
      await flushPromises()

      // Change to US
      await wrapper.find('button').trigger('click')
      const usButton = wrapper.findAll('.z-50 button').find(b => b.text().includes('United States'))
      await usButton.trigger('click')
      await flushPromises()

      // Should now be +1
      expect(wrapper.text()).toContain('Format: +1123456789')
    })
  })

  describe('Parsing E.164 Input', () => {
    it('should parse E.164 number from modelValue', async () => {
      const wrapper = createWrapper({ modelValue: '+62812345678' })
      await flushPromises()

      expect(wrapper.find('button').text()).toContain('+62')
      expect(wrapper.find('input').element.value).toBe('812345678')
    })

    it('should parse different country codes', async () => {
      const wrapper = createWrapper({ modelValue: '+1234567890' })
      await flushPromises()

      expect(wrapper.find('button').text()).toContain('+1')
      expect(wrapper.find('input').element.value).toBe('234567890')
    })

    it('should handle Singapore code correctly', async () => {
      const wrapper = createWrapper({ modelValue: '+6591234567' })
      await flushPromises()

      expect(wrapper.find('button').text()).toContain('+65')
      expect(wrapper.find('input').element.value).toBe('91234567')
    })

    it('should handle empty modelValue', async () => {
      const wrapper = createWrapper({ modelValue: '' })
      await flushPromises()

      expect(wrapper.find('input').element.value).toBe('')
    })
  })

  describe('Required Validation', () => {
    it('should add required attribute when required prop is true', () => {
      const wrapper = createWrapper({ required: true })

      expect(wrapper.find('input').attributes('required')).toBeDefined()
    })

    it('should not have required attribute when required is false', () => {
      const wrapper = createWrapper({ required: false })

      expect(wrapper.find('input').attributes('required')).toBeUndefined()
    })
  })

  describe('Click Outside', () => {
    it('should have overlay when dropdown is open', async () => {
      const wrapper = createWrapper()

      await wrapper.find('button').trigger('click')

      // Should have fixed overlay for click outside
      expect(wrapper.find('.fixed.inset-0').exists()).toBe(true)
    })
  })
})
