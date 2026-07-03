import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import QRGenerationProgress from '../tenant/QRGenerationProgress.vue'

function build(props = {}) {
  return mount(QRGenerationProgress, {
    props: {
      status: 'processing',
      generatedCount: 500,
      totalQrCount: 1000,
      progressPercent: 50,
      etaSeconds: 30,
      errorMessage: '',
      ...props,
    },
  })
}

describe('QRGenerationProgress', () => {
  describe('Status rendering', () => {
    it('renders "Generating" label for processing status', () => {
      const wrapper = build({ status: 'processing' })
      expect(wrapper.text()).toContain('Generating')
    })

    it('renders "Queued" label for queued status', () => {
      const wrapper = build({ status: 'queued' })
      expect(wrapper.text()).toContain('Queued')
    })

    it('renders "Waiting for queue" label for pending_queue status', () => {
      const wrapper = build({ status: 'pending_queue' })
      expect(wrapper.text()).toContain('Waiting for queue')
    })

    it('renders "Completed" label for completed status', () => {
      const wrapper = build({ status: 'completed', progressPercent: 100 })
      expect(wrapper.text()).toContain('Completed')
    })

    it('renders "Failed" label for failed status', () => {
      const wrapper = build({ status: 'failed' })
      expect(wrapper.text()).toContain('Failed')
    })
  })

  describe('Progress bar', () => {
    it('displays generated/total counts with thousands separator', () => {
      const wrapper = build({ generatedCount: 1234, totalQrCount: 5000 })
      expect(wrapper.text()).toContain('1,234')
      expect(wrapper.text()).toContain('5,000')
    })

    it('caps progress at 100%', () => {
      const wrapper = build({ progressPercent: 150 })
      expect(wrapper.text()).toContain('(100%)')
    })

    it('floors progress at 0%', () => {
      const wrapper = build({ progressPercent: -10 })
      expect(wrapper.text()).toContain('(0%)')
    })

    it('rounds fractional progress to integer', () => {
      const wrapper = build({ progressPercent: 33.7 })
      expect(wrapper.text()).toContain('(34%)')
    })

    it('handles zero total without crashing', () => {
      const wrapper = build({ totalQrCount: 0, generatedCount: 0, progressPercent: 0 })
      expect(wrapper.exists()).toBe(true)
    })
  })

  describe('ETA display', () => {
    it('shows ETA in seconds when < 60', () => {
      const wrapper = build({ status: 'processing', etaSeconds: 45 })
      expect(wrapper.text()).toContain('45s remaining')
    })

    it('shows ETA in minutes when 60 <= seconds < 3600', () => {
      const wrapper = build({ status: 'processing', etaSeconds: 125 })
      expect(wrapper.text()).toContain('2m 5s remaining')
    })

    it('shows ETA in hours when >= 3600', () => {
      const wrapper = build({ status: 'processing', etaSeconds: 3720 })
      expect(wrapper.text()).toContain('1h 2m remaining')
    })

    it('does not show ETA for non-processing statuses', () => {
      const wrapper = build({ status: 'completed', etaSeconds: 60 })
      expect(wrapper.text()).not.toContain('remaining')
    })

    it('does not show ETA when etaSeconds is null', () => {
      const wrapper = build({ status: 'processing', etaSeconds: null })
      expect(wrapper.text()).not.toContain('remaining')
    })
  })

  describe('Error message', () => {
    it('shows error message for failed status', () => {
      const wrapper = build({
        status: 'failed',
        errorMessage: 'Out of memory',
      })
      expect(wrapper.text()).toContain('Out of memory')
    })

    it('does not show error message for non-failed status', () => {
      const wrapper = build({
        status: 'processing',
        errorMessage: 'Some error',
      })
      expect(wrapper.text()).not.toContain('Some error')
    })

    it('does not show empty error message even for failed status', () => {
      const wrapper = build({ status: 'failed', errorMessage: '' })
      // No error div should be rendered
      const errorDiv = wrapper.find('.bg-red-50')
      expect(errorDiv.exists()).toBe(false)
    })
  })

  describe('Visual styling', () => {
    it('applies spin animation to icon during processing', () => {
      const wrapper = build({ status: 'processing' })
      const spinningIcon = wrapper.find('.animate-spin')
      expect(spinningIcon.exists()).toBe(true)
    })

    it('does not apply spin animation for non-processing states', () => {
      const wrapper = build({ status: 'completed' })
      const spinningIcon = wrapper.find('.animate-spin')
      expect(spinningIcon.exists()).toBe(false)
    })

    it('progress bar uses green for completed', () => {
      const wrapper = build({ status: 'completed', progressPercent: 100 })
      const bar = wrapper.find('.bg-green-500')
      expect(bar.exists()).toBe(true)
    })

    it('progress bar uses red for failed', () => {
      const wrapper = build({ status: 'failed', progressPercent: 50 })
      const bar = wrapper.find('.bg-red-500')
      expect(bar.exists()).toBe(true)
    })

    it('progress bar uses cyan for processing', () => {
      const wrapper = build({ status: 'processing', progressPercent: 50 })
      const bar = wrapper.find('.bg-zinc-500')
      expect(bar.exists()).toBe(true)
    })
  })
})
