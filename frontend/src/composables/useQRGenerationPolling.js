import { ref, onUnmounted } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { isTerminalStatus } from '@/stores/qrGeneration'

const POLL_INTERVAL_MS = 2000

/**
 * Composable for polling a single batch's generation status.
 * Used on the batch detail page to show progress bar + ETA.
 *
 * Auto-stops polling when the batch reaches a terminal state (completed/failed).
 * Auto-cleans up on component unmount.
 *
 * Usage:
 *   const { status, loading, error, startPolling, stopPolling } = useQRGenerationPolling()
 *   startPolling(batchId)
 */
export function useQRGenerationPolling() {
  const { get } = useAPI()

  const status = ref(null)
  const loading = ref(false)
  const error = ref(null)
  let pollingTimer = null
  let currentBatchId = null

  async function fetchStatus(batchId) {
    try {
      const response = await get(`/tenant/qr-batches/${batchId}/generation-status`)
      if (response.success && response.data) {
        status.value = response.data
        error.value = null

        // Auto-stop when terminal
        if (isTerminalStatus(response.data.status)) {
          stopPolling()
        }
      }
    } catch (err) {
      // Silent retry — polling will try again
      console.error('[useQRGenerationPolling] Failed to fetch status:', err)
      error.value = err
    }
  }

  function startPolling(batchId) {
    if (!batchId) return

    // Stop any existing polling
    stopPolling()

    currentBatchId = batchId
    loading.value = true

    // Fetch immediately, then at interval
    fetchStatus(batchId).finally(() => {
      loading.value = false
    })

    pollingTimer = setInterval(() => {
      if (currentBatchId) {
        fetchStatus(currentBatchId)
      }
    }, POLL_INTERVAL_MS)
  }

  function stopPolling() {
    if (pollingTimer) {
      clearInterval(pollingTimer)
      pollingTimer = null
    }
    currentBatchId = null
  }

  // Auto-cleanup on component unmount
  onUnmounted(() => {
    stopPolling()
  })

  return {
    status,
    loading,
    error,
    startPolling,
    stopPolling,
  }
}
