import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ConfirmDialog from '../ui/ConfirmDialog.vue'
import { AlertTriangle, Info } from 'lucide-vue-next'

describe('ConfirmDialog', () => {
  function createWrapper(props = {}) {
    return mount(ConfirmDialog, {
      props: {
        open: true,
        ...props,
      },
      global: {
        stubs: {
          Teleport: { template: '<div><slot /></div>' },
        },
      },
    })
  }

  it('does not render when open is false', () => {
    const wrapper = createWrapper({ open: false })

    // v-if="open" should prevent the dialog from rendering
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })

  it('renders title and message when open', () => {
    const wrapper = createWrapper({
      title: 'Delete Item',
      message: 'This action cannot be undone.',
    })

    expect(wrapper.find('h2').text()).toBe('Delete Item')
    expect(wrapper.find('p').text()).toBe('This action cannot be undone.')
  })

  it('renders with default props', () => {
    const wrapper = createWrapper()

    expect(wrapper.find('h2').text()).toBe('Confirm Action')
    expect(wrapper.find('p').text()).toBe('Are you sure you want to proceed?')
    expect(wrapper.text()).toContain('Confirm')
    expect(wrapper.text()).toContain('Cancel')
  })

  it('renders custom button text', () => {
    const wrapper = createWrapper({
      confirmText: 'Yes, Delete',
      cancelText: 'No, Keep It',
    })

    expect(wrapper.text()).toContain('No, Keep It')
    expect(wrapper.text()).toContain('Yes, Delete')
  })

  it('emits confirm event on confirm button click', async () => {
    const wrapper = createWrapper()

    // The confirm button is the last button in the dialog (after cancel)
    const buttons = wrapper.findAll('button')
    const confirmButton = buttons[buttons.length - 1]
    await confirmButton.trigger('click')

    expect(wrapper.emitted('confirm')).toBeTruthy()
    expect(wrapper.emitted('confirm')).toHaveLength(1)
  })

  it('emits cancel event on cancel button click', async () => {
    const wrapper = createWrapper()

    // The cancel button is the first button in the actions area
    const buttons = wrapper.findAll('button')
    const cancelButton = buttons[0]
    await cancelButton.trigger('click')

    expect(wrapper.emitted('cancel')).toBeTruthy()
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('emits cancel event on backdrop click', async () => {
    const wrapper = createWrapper()

    // The backdrop is the div with bg-black/50 class
    const backdrop = wrapper.find('.bg-black\\/50')
    await backdrop.trigger('click')

    expect(wrapper.emitted('cancel')).toBeTruthy()
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('renders destructive variant with AlertTriangle icon', () => {
    const wrapper = createWrapper({ variant: 'destructive' })

    // Use the imported component reference to find the icon
    expect(wrapper.findComponent(AlertTriangle).exists()).toBe(true)
    expect(wrapper.findComponent(Info).exists()).toBe(false)
  })

  it('renders default variant with Info icon', () => {
    const wrapper = createWrapper({ variant: 'default' })

    // Use the imported component reference to find the icon
    expect(wrapper.findComponent(Info).exists()).toBe(true)
    expect(wrapper.findComponent(AlertTriangle).exists()).toBe(false)
  })

  it('shows loading state when loading is true', () => {
    const wrapper = createWrapper({ loading: true })

    expect(wrapper.text()).toContain('Processing...')
  })

  it('disables buttons when loading', () => {
    const wrapper = createWrapper({ loading: true })

    const buttons = wrapper.findAll('button')
    buttons.forEach(button => {
      expect(button.attributes('disabled')).toBeDefined()
    })
  })

  it('has proper ARIA attributes (role=dialog, aria-modal)', () => {
    const wrapper = createWrapper({ title: 'Test Dialog' })

    const dialog = wrapper.find('[role="dialog"]')
    expect(dialog.exists()).toBe(true)
    expect(dialog.attributes('aria-modal')).toBe('true')
    expect(dialog.attributes('aria-label')).toBe('Test Dialog')
  })
})
