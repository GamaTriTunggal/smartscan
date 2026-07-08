<script setup>
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAPI } from '@/composables/useAPI'
import { useDateTime } from '@/composables/useDateTime'
import { useEscapeKey } from '@/composables/useEscapeKey'
import { useQRGenerationStore, isTerminalStatus, isInProgressStatus } from '@/stores/qrGeneration'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import CreateBatchModal from '@/components/products/CreateBatchModal.vue'
import BatchPdfDownload from '@/components/products/BatchPdfDownload.vue'
import QRGenerationProgress from '@/components/tenant/QRGenerationProgress.vue'
import { ArrowLeft, QrCode, Download, Plus, Package, Eye, Search, Calendar, Shield, Megaphone, Trash2, RotateCcw, MapPin, RefreshCw } from 'lucide-vue-next'
import { useTour, isTourActive, getTourNonce } from '@/composables/useTour.js'
import { getPagination } from '@/lib/pagination'

const route = useRoute()
const router = useRouter()
const { get, post, del, put, getAuthHeaders } = useAPI()
const { formatDate } = useDateTime()
const qrGenerationStore = useQRGenerationStore()

// Retry state
const retryingBatchId = ref(null)

const productId = computed(() => route.params.productId)

const loading = ref(true)
const product = ref(null)
const batches = ref([])
const searchQuery = ref('')
const showFilter = ref('active')

// Delete/Restore state
const deletingBatch = ref(null)
const showDeleteConfirm = ref(false)
const deleteError = ref('')
const isDeleting = ref(false)
const restoringBatchId = ref(null)

// Pagination
const page = ref(1)
const limit = ref(20)
const total = ref(0)
const totalPages = computed(() => Math.ceil(total.value / limit.value))

// Filtered batches based on search
const filteredBatches = computed(() => {
  if (!searchQuery.value) return batches.value
  const q = searchQuery.value.toLowerCase()
  return batches.value.filter(b =>
    b.batch_name.toLowerCase().includes(q) ||
    b.batch_code?.toLowerCase().includes(q)
  )
})

// Create Batch Modal
const showBatchModal = ref(false)

// Close delete confirmation on Escape key (CreateBatchModal handles its own Escape)
useEscapeKey(() => {
  showDeleteConfirm.value = false
})

const fetchProduct = async () => {
  try {
    const response = await get(`/tenant/products/${productId.value}`)
    if (response.success && response.data) {
      product.value = response.data
    }
  } catch (error) {
    console.error('Failed to fetch product:', error)
  }
}

const fetchBatches = async () => {
  try {
    loading.value = true
    const response = await get('/tenant/qr-batches', {
      product_id: productId.value,
      page: page.value,
      limit: limit.value,
      show: showFilter.value,
    })
    if (response.success && response.data) {
      batches.value = response.data.batches || []
      total.value = getPagination(response.data).total
    }
  } catch (error) {
    console.error('Failed to fetch batches:', error)
  } finally {
    loading.value = false
  }
}

const goBack = () => {
  router.push('/tenant/products/dynamic')
}

const openBatchModal = () => {
  showBatchModal.value = true
}

// Batch created by CreateBatchModal (it owns the POST + trackNewBatch). Refresh list.
const onBatchCreated = async () => {
  showBatchModal.value = false
  page.value = 1
  await fetchBatches()
}

// Merge live generation status from store into batches list for reactive progress bars
const batchesWithLiveStatus = computed(() => {
  return filteredBatches.value.map(batch => {
    const liveStatus = qrGenerationStore.getBatchStatus(batch.id)
    if (liveStatus) {
      return {
        ...batch,
        status: liveStatus.status,
        _liveProgress: liveStatus, // full status object for progress bar
      }
    }
    return batch
  })
})

// When a tracked batch completes (removed from store's active list), re-fetch
// the batch list so the UI immediately shows updated status + action buttons.
// This watcher is reactive-only (no polling) — zero cost when idle.
watch(
  () => qrGenerationStore.activeBatchList.length,
  (newLen, oldLen) => {
    if (oldLen > 0 && newLen < oldLen) {
      fetchBatches()
    }
  }
)

// Retry a failed batch
const retryBatch = async (batch) => {
  if (retryingBatchId.value) return
  retryingBatchId.value = batch.id
  try {
    const response = await post(`/tenant/qr-batches/${batch.id}/retry-generation`)
    if (response.success) {
      if (response.data) {
        qrGenerationStore.trackNewBatch(response.data)
      }
      await fetchBatches()
    }
  } catch (error) {
    console.error('Failed to retry batch:', error)
  } finally {
    retryingBatchId.value = null
  }
}

const viewBatchDetail = (batchId) => {
  router.push(`/tenant/qr-batches/${batchId}`)
}

const downloadBatch = async (batchId) => {
  try {
    const endpoint = `/tenant/qr-batches/${batchId}/export/csv`
    const response = await fetch(`${import.meta.env.VITE_API_URL}${endpoint}`, {
      method: 'GET',
      headers: getAuthHeaders(),
      credentials: 'include',
    })
    if (!response.ok) throw new Error('Export failed')
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `qr_batch_${batchId}.csv`
    a.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Failed to export:', error)
    alert('Failed to export CSV. Please try again.')
  }
}

const changePage = (newPage) => {
  if (newPage >= 1 && newPage <= totalPages.value) {
    page.value = newPage
    fetchBatches()
  }
}

const onShowFilterChange = () => {
  page.value = 1
  fetchBatches()
}

const confirmDelete = (batch) => {
  deletingBatch.value = batch
  deleteError.value = ''
  showDeleteConfirm.value = true
}

const deleteBatch = async () => {
  if (!deletingBatch.value || isDeleting.value) return
  try {
    isDeleting.value = true
    deleteError.value = ''
    const response = await del(`/tenant/qr-batches/${deletingBatch.value.id}`)
    if (response.success) {
      showDeleteConfirm.value = false
      deletingBatch.value = null
      fetchBatches()
    } else {
      deleteError.value = response.message || 'Failed to delete batch'
    }
  } catch (error) {
    deleteError.value = error.response?.data?.message || 'Failed to delete batch'
  } finally {
    isDeleting.value = false
  }
}

const restoreBatch = async (batch) => {
  if (restoringBatchId.value) return
  restoringBatchId.value = batch.id
  try {
    const response = await put(`/tenant/qr-batches/${batch.id}/restore`)
    if (response.success) {
      fetchBatches()
    }
  } catch (error) {
    console.error('Failed to restore batch:', error)
  } finally {
    restoringBatchId.value = null
  }
}

const tour = useTour()

function handleTourSetValue(e) {
  if (!isTourActive()) return
  if (e.detail._nonce !== getTourNonce()) return
  const { field, value } = e.detail
  switch (field) {
    case 'batch_search_query':
      searchQuery.value = value
      break
  }
}

onMounted(async () => {
  await Promise.all([fetchProduct(), fetchBatches()])
  tour.resumeIfActive()
  window.addEventListener('tour-set-value', handleTourSetValue)
})

onUnmounted(() => {
  window.removeEventListener('tour-set-value', handleTourSetValue)
})
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex items-center gap-4 mb-6">
      <button
        @click="goBack"
        class="p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
      >
        <ArrowLeft class="w-5 h-5" />
      </button>
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Batch History</h1>
        <p v-if="product" class="text-sm text-gray-500 dark:text-gray-400">
          {{ product.product_name }}
          <span v-if="product.product_code" class="ml-2">({{ product.product_code }})</span>
        </p>
      </div>
    </div>

    <!-- Product Info Card -->
    <Card v-if="product" class="p-4 mb-6">
      <div class="flex items-center gap-4">
        <div class="p-3 bg-zinc-100 dark:bg-zinc-900/30 rounded-lg">
          <Package class="w-6 h-6 text-zinc-600 dark:text-zinc-400" />
        </div>
        <div class="flex-1">
          <h2 class="font-semibold text-gray-900 dark:text-white">{{ product.product_name }}</h2>
          <p v-if="product.product_code" class="text-sm text-gray-500 dark:text-gray-400">
            Code: {{ product.product_code }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <span class="px-2 py-1 text-xs rounded-full bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400">
            Dynamic QR
          </span>
          <span v-if="product.warranty_enabled" class="px-2 py-1 text-xs rounded-full bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400">
            Warranty
          </span>
        </div>
      </div>
    </Card>

    <!-- Action Bar -->
    <div class="flex flex-col sm:flex-row justify-between gap-4 mb-6">
      <div class="flex items-center gap-3">
        <div class="relative max-w-xs w-full">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input
            v-model="searchQuery"
            placeholder="Search batches..."
            class="pl-9"
            data-tour="batch-search-input"
          />
        </div>
        <select
          v-model="showFilter"
          @change="onShowFilterChange"
          class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm"
        >
          <option value="active">Active</option>
          <option value="deleted">Deleted</option>
          <option value="all">All</option>
        </select>
      </div>
      <Button @click="openBatchModal">
        <Plus class="w-4 h-4 mr-2" />
        Create New Batch
      </Button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <div v-else>
      <!-- Empty State -->
      <Card v-if="batches.length === 0" class="p-6">
        <div class="text-center py-8">
          <QrCode class="w-16 h-16 text-gray-300 dark:text-gray-600 mx-auto mb-4" />
          <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No batches created yet</h3>
          <p class="text-gray-500 dark:text-gray-400 mb-4">
            Create your first batch of QR codes for this product.
          </p>
          <Button @click="openBatchModal">
            <Plus class="w-4 h-4 mr-2" />
            Create First Batch
          </Button>
        </div>
      </Card>

      <!-- Batches List -->
      <div v-else class="space-y-3">
        <Card
          v-for="(batch, bIdx) in batchesWithLiveStatus"
          :key="batch.id"
          class="p-3 transition-shadow"
          :class="batch.deleted_at ? 'opacity-60 bg-gray-50 dark:bg-gray-800/50 border-dashed' : 'hover:shadow-md'"
        >
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div class="flex-1 min-w-0">
              <!-- Row 1: Batch name + code count -->
              <div class="flex items-center gap-3">
                <QrCode class="w-5 h-5 flex-shrink-0" :class="batch.deleted_at ? 'text-gray-400' : 'text-zinc-600'" />
                <span class="font-semibold truncate" :class="batch.deleted_at ? 'text-gray-500 dark:text-gray-400' : 'text-gray-900 dark:text-white'">{{ batch.batch_name }}</span>
                <span class="px-2 py-0.5 text-xs rounded-full flex-shrink-0" :class="batch.deleted_at ? 'bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400' : 'bg-zinc-100 text-zinc-700 dark:bg-zinc-900/30 dark:text-zinc-400'">
                  {{ batch.qr_count.toLocaleString() }} codes
                </span>
                <span v-if="batch.deleted_at" class="px-2 py-0.5 text-xs rounded-full bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400 flex-shrink-0">
                  Deleted
                </span>
                <!-- Generation status badges -->
                <span v-else-if="batch.status === 'failed'" class="px-2 py-0.5 text-xs font-medium rounded-full bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400 flex-shrink-0">
                  Failed
                </span>
                <span v-else-if="batch.status === 'processing'" class="px-2 py-0.5 text-xs font-medium rounded-full bg-zinc-100 text-zinc-700 dark:bg-zinc-900/30 dark:text-zinc-400 flex-shrink-0">
                  Generating
                </span>
                <span v-else-if="batch.status === 'queued' || batch.status === 'pending_queue'" class="px-2 py-0.5 text-xs font-medium rounded-full bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400 flex-shrink-0">
                  {{ batch.status === 'pending_queue' ? 'Waiting for queue' : 'Queued' }}
                </span>
              </div>
              <!-- Row 2: Code, dates, and badges - all in one line -->
              <div class="ml-8 mt-1 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-500 dark:text-gray-400">
                <span>Code: {{ batch.batch_code }}</span>
                <span v-if="batch.production_date" class="flex items-center gap-1">
                  <Calendar class="w-3 h-3" />
                  Prod: {{ formatDate(batch.production_date) }}
                </span>
                <span v-if="batch.expiry_date" class="flex items-center gap-1">
                  <Calendar class="w-3 h-3" />
                  Exp: {{ formatDate(batch.expiry_date) }}
                </span>
                <span
                  v-if="batch.geofence_enabled"
                  class="flex items-center gap-1 px-2 py-0.5 text-xs rounded-full bg-zinc-50 text-zinc-700 dark:bg-zinc-900/30 dark:text-zinc-400 cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-900/50 transition-colors"
                  @click.stop="viewBatchDetail(batch.id)"
                  :title="`Geofence: ${batch.geofence_label || 'Distribution Zone'} (${batch.geofence_radius_km} km)`"
                >
                  <MapPin class="w-3 h-3" />
                  {{ batch.geofence_label || 'Distribution Zone' }}
                  <span v-if="batch.geofence_radius_km" class="text-zinc-500 dark:text-zinc-500">&middot; {{ batch.geofence_radius_km }} km</span>
                </span>
              </div>
              <!-- Progress bar for in-progress generations -->
              <div v-if="batch._liveProgress && isInProgressStatus(batch.status)" class="ml-8 mt-3">
                <QRGenerationProgress
                  :status="batch._liveProgress.status"
                  :generated-count="batch._liveProgress.generated_count"
                  :total-qr-count="batch._liveProgress.total_qr_count"
                  :progress-percent="batch._liveProgress.progress_percent"
                  :eta-seconds="batch._liveProgress.eta_seconds"
                  :error-message="batch._liveProgress.error_message"
                />
              </div>
            </div>
            <div class="flex items-center gap-2 sm:flex-shrink-0">
              <template v-if="batch.deleted_at">
                <Button
                  variant="outline"
                  size="sm"
                  :disabled="restoringBatchId === batch.id"
                  @click="restoreBatch(batch)"
                  class="text-green-600 border-green-200 hover:bg-green-50 dark:text-green-400 dark:border-green-800 dark:hover:bg-green-900/30"
                >
                  <RotateCcw class="w-4 h-4 mr-1" :class="{ 'animate-spin': restoringBatchId === batch.id }" />
                  {{ restoringBatchId === batch.id ? 'Restoring...' : 'Restore' }}
                </Button>
              </template>
              <template v-else-if="batch.status === 'failed'">
                <Button
                  variant="outline"
                  size="sm"
                  :disabled="retryingBatchId === batch.id"
                  @click="retryBatch(batch)"
                  class="text-zinc-600 border-zinc-200 hover:bg-zinc-50 dark:text-zinc-400 dark:border-zinc-800 dark:hover:bg-zinc-900/30"
                >
                  <RefreshCw class="w-4 h-4 mr-1" :class="{ 'animate-spin': retryingBatchId === batch.id }" />
                  {{ retryingBatchId === batch.id ? 'Retrying...' : 'Retry' }}
                </Button>
                <Button variant="outline" size="sm" @click="viewBatchDetail(batch.id)">
                  <Eye class="w-4 h-4 mr-1" />
                  Details
                </Button>
              </template>
              <template v-else-if="isInProgressStatus(batch.status)">
                <span class="text-xs text-gray-500 dark:text-gray-400 italic px-2">
                  Actions available after generation completes
                </span>
                <Button variant="outline" size="sm" @click="viewBatchDetail(batch.id)">
                  <Eye class="w-4 h-4 mr-1" />
                  View
                </Button>
              </template>
              <template v-else>
                <Button variant="outline" size="sm" @click="viewBatchDetail(batch.id)" :data-tour="bIdx === 0 ? 'batch-insights-btn' : undefined">
                  <Eye class="w-4 h-4 mr-1" />
                  Insights
                </Button>
                <Button variant="outline" size="sm" @click="downloadBatch(batch.id)">
                  <Download class="w-4 h-4 mr-1" />
                  CSV
                </Button>
                <BatchPdfDownload
                  :batch-id="batch.id"
                  :qr-count="batch.qr_count"
                  :batch-code="batch.batch_code"
                  size="sm"
                />
                <Button
                  v-if="!batch.scan_count"
                  variant="outline"
                  size="sm"
                  @click="confirmDelete(batch)"
                  class="text-red-600 border-red-200 hover:bg-red-50 dark:text-red-400 dark:border-red-800 dark:hover:bg-red-900/30"
                >
                  <Trash2 class="w-4 h-4" />
                </Button>
              </template>
            </div>
          </div>
        </Card>

        <!-- Pagination -->
        <div v-if="totalPages > 1" class="flex justify-center items-center gap-2 mt-6">
          <Button
            variant="outline"
            size="sm"
            :disabled="page === 1"
            @click="changePage(page - 1)"
          >
            Previous
          </Button>
          <span class="text-sm text-gray-600 dark:text-gray-400 px-4">
            Page {{ page }} of {{ totalPages }}
          </span>
          <Button
            variant="outline"
            size="sm"
            :disabled="page === totalPages"
            @click="changePage(page + 1)"
          >
            Next
          </Button>
        </div>
      </div>
    </div>

    <!-- Create Batch Modal -->
    <CreateBatchModal
      :open="showBatchModal"
      :product-id="product?.id"
      :product-name="product?.product_name"
      :warranty-enabled="!!product?.warranty_enabled"
      @close="showBatchModal = false"
      @created="onBatchCreated"
    />

    <!-- Delete Confirmation Dialog -->
    <div v-if="showDeleteConfirm" class="fixed inset-0 bg-black/50 flex items-center justify-center z-[60]">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md mx-4 shadow-xl">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          Delete Batch?
        </h3>
        <p class="text-sm text-gray-600 dark:text-gray-400 mb-2">
          Are you sure you want to delete "<strong>{{ deletingBatch?.batch_name }}</strong>"?
        </p>
        <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
          The batch will be soft deleted and can be restored later from the "Deleted" filter. QR codes from this batch will no longer be scannable.
        </p>
        <div v-if="deleteError" class="p-3 mb-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <p class="text-sm text-red-700 dark:text-red-300">{{ deleteError }}</p>
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="showDeleteConfirm = false; deleteError = ''">
            Cancel
          </Button>
          <Button variant="destructive" :disabled="isDeleting || deleteError" @click="deleteBatch">
            {{ isDeleting ? 'Deleting...' : 'Delete' }}
          </Button>
        </div>
      </div>
    </div>

  </div>
</template>
