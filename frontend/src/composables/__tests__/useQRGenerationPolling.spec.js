import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'

const mockGet = vi.fn()
vi.mock('@/composables/useAPI', () => ({
  useAPI: () => ({ get: mockGet }),
}))

import { useQRGenerationPolling } from '../useQRGenerationPolling'

function buildStatus(overrides = {}) {
  return {
    success: true,
    data: {
      batch_id: 'batch-1',
      batch_name: 'Test Batch',
      status: 'processing',
      total_qr_count: 1000,
      generated_count: 250,
      progress_percent: 25,
      started_at: '2026-04-05T10:00:00Z',
      completed_at: null,
      eta_seconds: 60,
      error_message: '',
      ...overrides,
    },
  }
}

/**
 * Mount a tiny wrapper component so the composable runs in a real component
 * context (required for `onUnmounted` to work).
 */
function mountWithComposable() {
  let pollingApi
  const Wrapper = {
    template: '<div></div>',
    setup() {
      pollingApi = useQRGenerationPolling()
      return {}
    },
  }
  const wrapper = mount(Wrapper)
  return { wrapper, pollingApi: () => pollingApi }
}

describe('useQRGenerationPolling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('fetches status immediately on startPolling', async () => {
    mockGet.mockResolvedValueOnce(buildStatus())
    const { wrapper, pollingApi } = mountWithComposable()
    const api = pollingApi()

    api.startPolling('batch-1')
    await vi.runOnlyPendingTimersAsync()

    expect(mockGet).toHaveBeenCalledWith('/tenant/qr-batches/batch-1/generation-status')
    expect(api.status.value).toBeDefined()
    expect(api.status.value.status).toBe('processing')

    api.stopPolling()
    wrapper.unmount()
  })

  it('auto-stops polling when batch reaches terminal status', async () => {
    mockGet.mockResolvedValueOnce(buildStatus({ status: 'completed', progress_percent: 100 }))
    const { wrapper, pollingApi } = mountWithComposable()
    const api = pollingApi()

    api.startPolling('batch-1')
    await vi.runOnlyPendingTimersAsync()

    expect(api.status.value.status).toBe('completed')

    // Advance time; no further fetches should occur
    const callsBefore = mockGet.mock.calls.length
    await vi.advanceTimersByTimeAsync(5000)
    expect(mockGet.mock.calls.length).toBe(callsBefore)

    wrapper.unmount()
  })

  it('does not start polling if batchId is empty', () => {
    const { wrapper, pollingApi } = mountWithComposable()
    const api = pollingApi()

    api.startPolling('')
    api.startPolling(null)
    api.startPolling(undefined)

    expect(mockGet).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('stopPolling cancels the interval', async () => {
    mockGet.mockResolvedValue(buildStatus())
    const { wrapper, pollingApi } = mountWithComposable()
    const api = pollingApi()

    api.startPolling('batch-1')
    await vi.runOnlyPendingTimersAsync()
    const callsAfterStart = mockGet.mock.calls.length

    api.stopPolling()
    await vi.advanceTimersByTimeAsync(10000)

    // No additional calls after stop
    expect(mockGet.mock.calls.length).toBe(callsAfterStart)

    wrapper.unmount()
  })

  it('handles fetch errors gracefully', async () => {
    mockGet.mockRejectedValue(new Error('Network down'))
    const { wrapper, pollingApi } = mountWithComposable()
    const api = pollingApi()

    api.startPolling('batch-1')
    // Flush pending microtasks (the rejected promise) before checking state
    await vi.runOnlyPendingTimersAsync()
    await Promise.resolve()
    await Promise.resolve()

    expect(api.error.value).toBeTruthy()
    expect(api.status.value).toBeNull()

    api.stopPolling()
    wrapper.unmount()
  })

  it('restarts polling with a different batchId', async () => {
    mockGet.mockResolvedValue(buildStatus())
    const { wrapper, pollingApi } = mountWithComposable()
    const api = pollingApi()

    api.startPolling('batch-1')
    await vi.runOnlyPendingTimersAsync()

    api.startPolling('batch-2')
    await vi.runOnlyPendingTimersAsync()

    // Last call should be for batch-2
    const lastCall = mockGet.mock.calls[mockGet.mock.calls.length - 1]
    expect(lastCall[0]).toContain('batch-2')

    api.stopPolling()
    wrapper.unmount()
  })

  it('cleans up on component unmount', async () => {
    mockGet.mockResolvedValue(buildStatus())
    const { wrapper, pollingApi } = mountWithComposable()
    const api = pollingApi()

    api.startPolling('batch-1')
    await vi.runOnlyPendingTimersAsync()
    const callsBeforeUnmount = mockGet.mock.calls.length

    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(10000)

    // No polling after unmount
    expect(mockGet.mock.calls.length).toBe(callsBeforeUnmount)
  })
})
