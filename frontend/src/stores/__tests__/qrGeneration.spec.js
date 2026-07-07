import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import {
  useQRGenerationStore,
  isTerminalStatus,
  isInProgressStatus,
  QRGenerationStatus,
} from '../qrGeneration'

// Mock useAPI composable
const mockGet = vi.fn()

vi.mock('@/composables/useAPI', () => ({
  useAPI: () => ({
    get: mockGet,
  }),
}))

// Mock useToast composable
const mockToastSuccess = vi.fn()
const mockToastError = vi.fn()

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    success: mockToastSuccess,
    error: mockToastError,
  }),
}))

function mockGenResponse(generations) {
  return {
    success: true,
    data: {
      active_generations: generations,
      truncated: false,
    },
  }
}

function buildGeneration(overrides = {}) {
  return {
    batch_id: 'batch-1',
    batch_name: 'Test Batch',
    status: 'processing',
    total_qr_count: 1000,
    generated_count: 500,
    progress_percent: 50,
    started_at: '2026-04-05T10:00:00Z',
    completed_at: null,
    eta_seconds: 30,
    error_message: '',
    ...overrides,
  }
}

describe('QRGeneration Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  afterEach(() => {
    // Make sure any timers started during tests are cleaned up
    const store = useQRGenerationStore()
    store.stopPolling()
    vi.useRealTimers()
  })

  describe('Helper exports', () => {
    it('isTerminalStatus returns true for completed and failed', () => {
      expect(isTerminalStatus('completed')).toBe(true)
      expect(isTerminalStatus('failed')).toBe(true)
    })

    it('isTerminalStatus returns false for non-terminal statuses', () => {
      expect(isTerminalStatus('pending_queue')).toBe(false)
      expect(isTerminalStatus('queued')).toBe(false)
      expect(isTerminalStatus('processing')).toBe(false)
    })

    it('isInProgressStatus returns true for non-terminal statuses', () => {
      expect(isInProgressStatus('pending_queue')).toBe(true)
      expect(isInProgressStatus('queued')).toBe(true)
      expect(isInProgressStatus('processing')).toBe(true)
    })

    it('isInProgressStatus returns false for terminal statuses', () => {
      expect(isInProgressStatus('completed')).toBe(false)
      expect(isInProgressStatus('failed')).toBe(false)
    })

    it('QRGenerationStatus exports all status constants', () => {
      expect(QRGenerationStatus.PENDING_QUEUE).toBe('pending_queue')
      expect(QRGenerationStatus.QUEUED).toBe('queued')
      expect(QRGenerationStatus.PROCESSING).toBe('processing')
      expect(QRGenerationStatus.COMPLETED).toBe('completed')
      expect(QRGenerationStatus.FAILED).toBe('failed')
    })
  })

  describe('Initial state', () => {
    it('has empty activeBatches and is not polling', () => {
      const store = useQRGenerationStore()
      expect(store.activeBatches).toEqual({})
      expect(store.isPolling).toBe(false)
      expect(store.hasActiveBatches).toBe(false)
      expect(store.activeBatchList).toEqual([])
    })
  })

  describe('fetchActive', () => {
    it('populates activeBatches when backend returns generations', async () => {
      const store = useQRGenerationStore()
      const gen = buildGeneration()
      mockGet.mockResolvedValueOnce(mockGenResponse([gen]))

      await store.fetchActive()

      expect(mockGet).toHaveBeenCalledWith('/tenant/qr-batches/active-generations')
      expect(store.activeBatches[gen.batch_id]).toEqual(gen)
      expect(store.hasActiveBatches).toBe(true)
    })

    it('returns empty array when no active generations', async () => {
      const store = useQRGenerationStore()
      mockGet.mockResolvedValueOnce(mockGenResponse([]))

      const result = await store.fetchActive()

      expect(result).toEqual([])
      expect(store.hasActiveBatches).toBe(false)
    })

    it('silently handles fetch errors', async () => {
      const store = useQRGenerationStore()
      mockGet.mockRejectedValueOnce(new Error('Network error'))

      const result = await store.fetchActive()

      expect(result).toEqual([])
      expect(store.hasActiveBatches).toBe(false)
    })
  })

  describe('Reconciliation and toasts', () => {
    it('fires success toast when batch transitions to completed', async () => {
      const store = useQRGenerationStore()
      const processing = buildGeneration({ status: 'processing' })

      // First poll: in-progress
      mockGet.mockResolvedValueOnce(mockGenResponse([processing]))
      await store.fetchActive()
      expect(mockToastSuccess).not.toHaveBeenCalled()

      // Second poll: completed (still in list)
      const completed = buildGeneration({ status: 'completed', progress_percent: 100 })
      mockGet.mockResolvedValueOnce(mockGenResponse([completed]))
      await store.fetchActive()

      expect(mockToastSuccess).toHaveBeenCalledTimes(1)
      expect(mockToastSuccess.mock.calls[0][0]).toContain('Test Batch')
    })

    it('fires error toast when batch transitions to failed', async () => {
      const store = useQRGenerationStore()
      const processing = buildGeneration({ status: 'processing' })
      mockGet.mockResolvedValueOnce(mockGenResponse([processing]))
      await store.fetchActive()

      const failed = buildGeneration({ status: 'failed', error_message: 'Out of memory' })
      mockGet.mockResolvedValueOnce(mockGenResponse([failed]))
      await store.fetchActive()

      expect(mockToastError).toHaveBeenCalledTimes(1)
      expect(mockToastError.mock.calls[0][0]).toContain('Test Batch')
      expect(mockToastError.mock.calls[0][0]).toContain('Retry')
    })

    it('notifies completion when a batch disappears and its status confirms terminal', async () => {
      const store = useQRGenerationStore()
      const processing = buildGeneration({ status: 'processing' })
      mockGet.mockResolvedValueOnce(mockGenResponse([processing]))
      await store.fetchActive()

      // Batch disappears from the active list; reconcile confirms the real terminal
      // state via the per-batch generation-status endpoint before notifying.
      mockGet.mockResolvedValueOnce(mockGenResponse([]))
      mockGet.mockResolvedValueOnce({ success: true, data: { status: 'completed' } })
      await store.fetchActive()

      expect(mockToastSuccess).toHaveBeenCalledTimes(1)
      expect(store.activeBatches[processing.batch_id]).toBeUndefined()
    })

    it('does NOT fire a false completion toast when the status check fails or is non-terminal', async () => {
      const store = useQRGenerationStore()
      const processing = buildGeneration({ status: 'processing' })
      mockGet.mockResolvedValueOnce(mockGenResponse([processing]))
      await store.fetchActive()

      // Batch temporarily absent from the (possibly truncated) active list, and the
      // status GET fails transiently — the store must NOT assume completion.
      mockGet.mockResolvedValueOnce(mockGenResponse([]))
      mockGet.mockRejectedValueOnce(new Error('network'))
      await store.fetchActive()

      expect(mockToastSuccess).not.toHaveBeenCalled()
      expect(mockToastError).not.toHaveBeenCalled()
      // Batch retained for a later re-check rather than dropped with a false toast.
      expect(store.activeBatches[processing.batch_id]).toBeDefined()
    })

    it('does not fire duplicate toasts for the same batch', async () => {
      const store = useQRGenerationStore()
      const completed = buildGeneration({ status: 'completed' })

      // First poll: completed (fires toast)
      mockGet.mockResolvedValueOnce(mockGenResponse([completed]))
      await store.fetchActive()
      expect(mockToastSuccess).toHaveBeenCalledTimes(1)

      // Second poll: batch removed from active list (already terminal)
      mockGet.mockResolvedValueOnce(mockGenResponse([]))
      await store.fetchActive()

      // Should NOT fire again
      expect(mockToastSuccess).toHaveBeenCalledTimes(1)
    })

    it('removes batches no longer in active list and stops polling when empty', async () => {
      const store = useQRGenerationStore()

      mockGet.mockResolvedValueOnce(mockGenResponse([buildGeneration()]))
      await store.fetchActive()
      store.startPolling()
      expect(store.isPolling).toBe(true)

      // Second poll: no active batches; reconcile confirms the batch is terminal.
      mockGet.mockResolvedValueOnce(mockGenResponse([]))
      mockGet.mockResolvedValueOnce({ success: true, data: { status: 'completed' } })
      await store.fetchActive()

      expect(store.isPolling).toBe(false)
      expect(store.hasActiveBatches).toBe(false)
    })
  })

  describe('trackNewBatch', () => {
    it('tracks a new batch with non-terminal status and starts polling', () => {
      const store = useQRGenerationStore()
      mockGet.mockResolvedValue(mockGenResponse([buildGeneration()]))

      store.trackNewBatch({
        id: 'new-batch-1',
        batch_name: 'New Batch',
        status: 'queued',
        qr_count: 5000,
      })

      expect(store.activeBatches['new-batch-1']).toBeDefined()
      expect(store.activeBatches['new-batch-1'].status).toBe('queued')
      expect(store.activeBatches['new-batch-1'].total_qr_count).toBe(5000)
      expect(store.isPolling).toBe(true)
    })

    it('does not track a batch that is already in terminal state', () => {
      const store = useQRGenerationStore()
      store.trackNewBatch({
        id: 'done-batch',
        batch_name: 'Done',
        status: 'completed',
        qr_count: 10,
      })

      expect(store.activeBatches['done-batch']).toBeUndefined()
      expect(store.isPolling).toBe(false)
    })

    it('handles missing batch gracefully', () => {
      const store = useQRGenerationStore()
      store.trackNewBatch(null)
      store.trackNewBatch({})
      expect(store.hasActiveBatches).toBe(false)
    })
  })

  describe('Polling lifecycle', () => {
    it('startPolling is idempotent (multiple calls do not create duplicate timers)', async () => {
      const store = useQRGenerationStore()
      mockGet.mockResolvedValue(mockGenResponse([buildGeneration()]))

      store.startPolling()
      store.startPolling()
      store.startPolling()

      expect(store.isPolling).toBe(true)
    })

    it('stopPolling clears timer and resets flag', async () => {
      const store = useQRGenerationStore()
      mockGet.mockResolvedValue(mockGenResponse([buildGeneration()]))
      store.startPolling()
      expect(store.isPolling).toBe(true)

      store.stopPolling()
      expect(store.isPolling).toBe(false)
    })

    it('checkAndStartPolling only starts polling if backend returns active batches', async () => {
      const store = useQRGenerationStore()

      // No active batches
      mockGet.mockResolvedValueOnce(mockGenResponse([]))
      await store.checkAndStartPolling()
      expect(store.isPolling).toBe(false)

      // Active batches
      mockGet.mockResolvedValueOnce(mockGenResponse([buildGeneration()]))
      await store.checkAndStartPolling()
      expect(store.isPolling).toBe(true)
    })
  })

  describe('reset', () => {
    it('clears all state and stops polling', async () => {
      const store = useQRGenerationStore()
      mockGet.mockResolvedValue(mockGenResponse([buildGeneration()]))
      store.startPolling()
      store.activeBatches['batch-1'] = buildGeneration()

      store.reset()

      expect(store.activeBatches).toEqual({})
      expect(store.isPolling).toBe(false)
    })
  })

  describe('getBatchStatus getter', () => {
    it('returns status for tracked batch', () => {
      const store = useQRGenerationStore()
      const gen = buildGeneration({ batch_id: 'my-batch' })
      store.activeBatches['my-batch'] = gen

      expect(store.getBatchStatus('my-batch')).toEqual(gen)
    })

    it('returns null for untracked batch', () => {
      const store = useQRGenerationStore()
      expect(store.getBatchStatus('nonexistent')).toBeNull()
    })
  })
})
