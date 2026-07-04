import { defineStore } from 'pinia'
import { useAPI } from '@/composables/useAPI'
import { useToast } from '@/composables/useToast'

// Terminal statuses — once reached, no further polling needed
const TERMINAL_STATUSES = new Set(['completed', 'failed'])
// In-progress statuses — need polling
const IN_PROGRESS_STATUSES = new Set(['pending_queue', 'queued', 'processing'])

// Polling interval in ms (2 seconds per product spec)
const POLL_INTERVAL_MS = 2000

// Module-level timer handle. Kept OUTSIDE Pinia state because:
//   1. setInterval handles are not serialisable (Pinia devtools / persistence plugins would break)
//   2. The reactive proxy would wrap the numeric ID unnecessarily
// There's one store instance per app, so a single module-level ref is safe.
let _pollingTimer = null

export const useQRGenerationStore = defineStore('qrGeneration', {
  state: () => ({
    // Map of batch_id -> generation status object
    // Example: { 'uuid1': { batch_id, batch_name, status, progress_percent, ... } }
    activeBatches: {},
    isPolling: false,
    // Plain object used as a set: batch_id -> true
    // Using a Set would break Pinia reactivity because Vue's proxy does not track Set mutations
    notifiedBatchIds: {},
  }),

  getters: {
    activeBatchList(state) {
      return Object.values(state.activeBatches)
    },
    hasActiveBatches(state) {
      return Object.keys(state.activeBatches).length > 0
    },
    getBatchStatus: (state) => (batchId) => {
      return state.activeBatches[batchId] || null
    },
  },

  actions: {
    /**
     * Fetch active generations from the backend
     * @returns {Promise<Array>} array of generation status objects
     */
    async fetchActive() {
      const { get } = useAPI()
      try {
        const response = await get('/tenant/qr-batches/active-generations')
        if (response.success && response.data) {
          const generations = response.data.active_generations || []
          this._reconcile(generations)
          return generations
        }
      } catch (error) {
        // Silent fail — polling will retry
        console.error('[qrGeneration] Failed to fetch active generations:', error)
      }
      return []
    },

    /**
     * Reconcile fetched state with local state
     * - New batches added
     * - Existing batches updated
     * - Batches that transitioned to terminal state trigger toasts
     * - Batches that disappeared from the fetch are assumed completed successfully
     *   (backend filters terminal statuses, so removal implies completion)
     * @private
     */
    async _reconcile(generations) {
      const toast = useToast()
      const { get } = useAPI()
      const fetchedIds = new Set(generations.map(g => g.batch_id))

      // Update / add batches
      for (const gen of generations) {
        this.activeBatches[gen.batch_id] = gen

        // If transitioning to terminal status (or already terminal but not notified), trigger toast
        if (TERMINAL_STATUSES.has(gen.status) && !this.notifiedBatchIds[gen.batch_id]) {
          this._notifyCompletion(gen, toast)
          this.notifiedBatchIds[gen.batch_id] = true
        }
      }

      // Remove batches that are no longer in the active list.
      // If a batch was in-progress last tick and is gone now, the backend has filtered it out
      // because it reached a terminal state. Notify the user if we haven't already.
      for (const id of Object.keys(this.activeBatches)) {
        if (!fetchedIds.has(id)) {
          if (!this.notifiedBatchIds[id]) {
            // The batch left the in-progress list — confirm its actual terminal
            // state before notifying (it may have FAILED, not completed).
            const lastKnown = this.activeBatches[id]
            let finalStatus = 'completed'
            try {
              const statusResp = await get(`/tenant/qr-batches/${id}/generation-status`)
              if (statusResp?.success && statusResp.data?.status) {
                finalStatus = statusResp.data.status
              }
            } catch (e) { /* keep optimistic default */ }
            this._notifyCompletion({ ...lastKnown, status: finalStatus }, toast)
            this.notifiedBatchIds[id] = true
          }
          delete this.activeBatches[id]
        }
      }

      // Stop polling if no more active batches
      if (Object.keys(this.activeBatches).length === 0) {
        this.stopPolling()
      }
    },

    /**
     * Notify user about a completion or failure
     * @private
     */
    _notifyCompletion(gen, toast) {
      const count = (gen.total_qr_count || 0).toLocaleString()
      if (gen.status === 'completed') {
        toast.success(
          `Batch "${gen.batch_name}" ready — ${count} QR codes generated`,
          { duration: 8000 }
        )
      } else if (gen.status === 'failed') {
        toast.error(
          `Batch "${gen.batch_name}" generation failed. Click Retry to resume.`,
          { duration: 10000 }
        )
      }
    },

    /**
     * Start polling for active generations.
     * Idempotent: does nothing if already polling.
     */
    startPolling() {
      if (this.isPolling) return
      this.isPolling = true

      // Fetch immediately then at interval
      this.fetchActive()
      _pollingTimer = setInterval(() => {
        this.fetchActive()
      }, POLL_INTERVAL_MS)
    },

    /**
     * Stop polling
     */
    stopPolling() {
      if (_pollingTimer) {
        clearInterval(_pollingTimer)
        _pollingTimer = null
      }
      this.isPolling = false
    },

    /**
     * Check backend for active generations and start polling if any exist.
     * Called on app mount (e.g., after login or page refresh).
     */
    async checkAndStartPolling() {
      const generations = await this.fetchActive()
      if (generations && generations.length > 0) {
        this.startPolling()
      }
    },

    /**
     * Add a batch to tracking and start polling.
     * Called right after user submits a new batch successfully.
     * @param {object} batch - The batch object returned from the create endpoint
     */
    trackNewBatch(batch) {
      if (!batch || !batch.id) return

      // Only track batches that are in non-terminal state
      if (TERMINAL_STATUSES.has(batch.status)) return

      this.activeBatches[batch.id] = {
        batch_id: batch.id,
        batch_name: batch.batch_name,
        status: batch.status,
        total_qr_count: batch.qr_count,
        generated_count: 0,
        progress_percent: 0,
        started_at: null,
        completed_at: null,
        eta_seconds: null,
        error_message: '',
      }
      this.startPolling()
    },

    /**
     * Reset store state (e.g., on logout or tenant switch).
     * Clears all tracking and stops polling.
     */
    reset() {
      this.stopPolling()
      this.activeBatches = {}
      this.notifiedBatchIds = {}
    },
  },
})

// Export status helpers for component use
export const QRGenerationStatus = {
  PENDING_QUEUE: 'pending_queue',
  QUEUED: 'queued',
  PROCESSING: 'processing',
  COMPLETED: 'completed',
  FAILED: 'failed',
}

export function isTerminalStatus(status) {
  return TERMINAL_STATUSES.has(status)
}

export function isInProgressStatus(status) {
  return IN_PROGRESS_STATUSES.has(status)
}
