<script setup>
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAPI } from '@/composables/useAPI'
import { useDateTime } from '@/composables/useDateTime'
import { useAuthStore } from '@/stores/auth'
import { useQRGenerationPolling } from '@/composables/useQRGenerationPolling'
import { isTerminalStatus, isInProgressStatus } from '@/stores/qrGeneration'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import BatchPdfDownload from '@/components/products/BatchPdfDownload.vue'
import ScanHeatmap from '@/components/ScanHeatmap.vue'
import GeofenceMapPicker from '@/components/GeofenceMapPicker.vue'
import QRGenerationProgress from '@/components/tenant/QRGenerationProgress.vue'
import { Line } from 'vue-chartjs'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, Filler, Title, Tooltip, Legend } from 'chart.js'
import { ArrowLeft, Eye, BarChart3, MapPin as MapPinIcon, Activity, Navigation, RefreshCw } from 'lucide-vue-next'
import { useTour, isTourActive, getTourNonce } from '@/composables/useTour.js'
import { getPagination } from '@/lib/pagination'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Filler, Title, Tooltip, Legend)

const route = useRoute()
const router = useRouter()
const { get, put, post, getAuthHeaders } = useAPI()
const { formatDate, formatDateTime } = useDateTime()
const authStore = useAuthStore()

// Async generation polling
const generationPolling = useQRGenerationPolling()
const retrying = ref(false)

// Effective batch status (live status from polling overrides stale batch.status)
const effectiveStatus = computed(() => {
  if (generationPolling.status.value) {
    return generationPolling.status.value.status
  }
  return batch.value?.status || 'completed'
})

const isGenerating = computed(() => isInProgressStatus(effectiveStatus.value))
const isFailed = computed(() => effectiveStatus.value === 'failed')
const isCompleted = computed(() => effectiveStatus.value === 'completed')

// Export state
const exporting = ref({ csv: false, excel: false })

const batchId = computed(() => route.params.id)

const loading = ref(true)
const loadingCodes = ref(false)
const batch = ref(null)
const actualQRCount = ref(0)
const codes = ref([])
const pagination = ref({ page: 1, limit: 50, total: 0, total_page: 0 })
const statusFilter = ref('')
const counterfeitFilter = ref('')

// Template editing
const editingTemplates = ref(false)
const savingTemplates = ref(false)
const validationTemplates = ref([])
const warrantyTemplates = ref([])
const selectedValidationTemplateId = ref('')
const selectedWarrantyTemplateId = ref('')

// Heatmap state
const heatmapLoading = ref(false)
const heatmapData = ref(null)
const heatmapError = ref(false)

// Analytics state
const analyticsLoading = ref(false)
const analyticsData = ref(null)
const analyticsError = ref(false)

// Geofence analytics state
const geofenceAnalytics = ref(null)

const heatmapGeofenceViolations = computed(() => {
  return heatmapData.value?.geofence_violations || []
})

const heatmapGeofenceViolationCount = computed(() => {
  return heatmapGeofenceViolations.value.length
})

const geofenceZone = computed(() => {
  if (batch.value.geofence_latitude == null) return []
  return [{
    batch_id: batchId.value,
    batch_name: batch.value.batch_name || '',
    lat: batch.value.geofence_latitude,
    lng: batch.value.geofence_longitude,
    radius_km: batch.value.geofence_radius_km || 25,
    label: batch.value.geofence_label || ''
  }]
})

// Geofence edit mode
const editingGeofence = ref(false)
const savingGeofence = ref(false)
const geofenceForm = ref({ latitude: null, longitude: null, radius_km: 25, label: '' })

// Zone templates for edit geofence
const zoneTemplates = ref([])
const selectedZoneTemplateId = ref(null)

async function fetchZoneTemplates() {
  try {
    const response = await get('/tenant/geofence/zone-templates')
    if (response.success) {
      zoneTemplates.value = response.data?.zone_templates || []
    }
  } catch (e) { /* ignore - templates are optional */ }
}

function loadZoneTemplate(template) {
  selectedZoneTemplateId.value = template.id
  geofenceForm.value = {
    latitude: template.latitude,
    longitude: template.longitude,
    radius_km: template.radius_km,
    label: template.label || template.template_name,
  }
}

function onGeofenceFormUpdate(val) {
  geofenceForm.value = val
  selectedZoneTemplateId.value = null
}

function startEditGeofence() {
  geofenceForm.value = {
    latitude: batch.value.geofence_latitude,
    longitude: batch.value.geofence_longitude,
    radius_km: batch.value.geofence_radius_km || 25,
    label: batch.value.geofence_label || ''
  }
  selectedZoneTemplateId.value = null
  editingGeofence.value = true
  fetchZoneTemplates()
}

async function saveGeofence() {
  if (heatmapGeofenceViolationCount.value > 0) {
    const ok = confirm(`This batch has ${heatmapGeofenceViolationCount.value} existing violation(s). Changing the zone will only affect future scans. Existing violations will not be recalculated.\n\nContinue?`)
    if (!ok) return
  }
  try {
    savingGeofence.value = true
    const payload = {
      geofence_enabled: true,
      geofence_latitude: geofenceForm.value.latitude,
      geofence_longitude: geofenceForm.value.longitude,
      geofence_radius_km: geofenceForm.value.radius_km,
      geofence_label: geofenceForm.value.label,
    }
    if (selectedZoneTemplateId.value) {
      payload.geofence_zone_template_id = selectedZoneTemplateId.value
    }
    const response = await put(`/tenant/qr-batches/${batchId.value}`, payload)
    if (response.success) {
      editingGeofence.value = false
      fetchBatch()
    }
  } catch (e) {
    console.error('Failed to update geofence:', e)
  } finally {
    savingGeofence.value = false
  }
}

// Tab state
const activeTab = ref('codes')

// Counterfeited QRs tab
const counterfeitCodes = ref([])
const counterfeitPagination = ref({ page: 1, limit: 20, total: 0, total_page: 0 })
const loadingCounterfeitCodes = ref(false)

// Geofence Violations tab
const geoViolations = ref([])
const geoViolationsPagination = ref({ page: 1, limit: 20, total: 0, total_page: 0 })
const loadingGeoViolations = ref(false)

const hasCounterfeitQRs = computed(() => {
  return (heatmapData.value?.summary?.counterfeit_count || 0) > 0
})

const hasGeofenceViolations = computed(() => heatmapGeofenceViolationCount.value > 0)

function getSeverityColor(severity) {
  switch (severity) {
    case 'low':
      return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
    case 'medium':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
    case 'high':
      return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
    case 'critical':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    default:
      return 'bg-gray-100 text-gray-700'
  }
}

const heatmapLocations = computed(() => {
  if (!heatmapData.value?.points) return []
  return heatmapData.value.points.map(p => ({
    latitude: p.lat,
    longitude: p.lng,
    name: batch.value?.product?.product_name || '',
    date: p.created_at ? formatDateTime(p.created_at) : null,
    scanType: p.scan_type,
    counterfeitStatus: p.counterfeit_status,
    qrCodeId: p.qr_code_id,
    batchId: batchId.value
  }))
})

const fetchBatchHeatmap = async () => {
  if (heatmapLoading.value) return
  try {
    heatmapLoading.value = true
    heatmapError.value = false
    const response = await get(`/tenant/qr-batches/${batchId.value}/heatmap`)
    if (response.success) {
      heatmapData.value = response.data
    }
  } catch (error) {
    console.error('Failed to fetch batch heatmap:', error)
    heatmapError.value = true
  } finally {
    heatmapLoading.value = false
  }
}

// Analytics fetch & computed
const fetchBatchAnalytics = async () => {
  if (analyticsData.value || analyticsLoading.value) return
  try {
    analyticsLoading.value = true
    analyticsError.value = false
    const response = await get(`/tenant/qr-batches/${batchId.value}/analytics`)
    if (response.success) {
      analyticsData.value = response.data
    }
  } catch (error) {
    console.error('Failed to fetch batch analytics:', error)
    analyticsError.value = true
  } finally {
    analyticsLoading.value = false
  }
}

// Geofence analytics fetch
const fetchGeofenceAnalytics = async () => {
  if (geofenceAnalytics.value) return
  try {
    const response = await get(`/tenant/qr-batches/${batchId.value}/geofence-analytics`)
    if (response.success) {
      geofenceAnalytics.value = response.data
    }
  } catch (error) {
    // Analytics are supplementary — the page still works without them
  }
}

async function fetchCounterfeitCodes() {
  loadingCounterfeitCodes.value = true
  try {
    const params = {
      page: counterfeitPagination.value.page,
      limit: counterfeitPagination.value.limit,
      counterfeit_status: 'warning,counterfeit'
    }
    const response = await get(`/tenant/qr-batches/${batchId.value}/codes`, params)
    if (response.success && response.data) {
      counterfeitCodes.value = response.data.codes || []
      const p = getPagination(response.data)
      counterfeitPagination.value = { page: p.page, limit: p.limit, total: p.total, total_page: p.totalPages }
    }
  } catch (error) {
    console.error('Failed to fetch counterfeit codes:', error)
  } finally {
    loadingCounterfeitCodes.value = false
  }
}

async function fetchGeoViolations() {
  loadingGeoViolations.value = true
  try {
    const params = {
      page: geoViolationsPagination.value.page,
      limit: geoViolationsPagination.value.limit,
      batch_id: batchId.value
    }
    const response = await get('/tenant/geofence/violations', params)
    if (response.success && response.data) {
      geoViolations.value = response.data.violations || []
      const p = getPagination(response.data)
      geoViolationsPagination.value = {
        page: p.page,
        limit: p.limit,
        total: p.total,
        total_page: p.totalPages
      }
    }
  } catch (error) {
    console.error('Failed to fetch geofence violations:', error)
  } finally {
    loadingGeoViolations.value = false
  }
}

function onTabChange(tab) {
  activeTab.value = tab
  if (tab === 'counterfeit' && counterfeitCodes.value.length === 0 && !loadingCounterfeitCodes.value) {
    fetchCounterfeitCodes()
  }
  if (tab === 'geofence' && geoViolations.value.length === 0 && !loadingGeoViolations.value) {
    fetchGeoViolations()
  }
}

// Pre-fetch tab data once heatmap reveals tabs should be visible
watch(heatmapData, (data) => {
  if (!data?.summary) return
  if (hasCounterfeitQRs.value && counterfeitCodes.value.length === 0 && !loadingCounterfeitCodes.value) {
    fetchCounterfeitCodes()
  }
  if (hasGeofenceViolations.value && geoViolations.value.length === 0 && !loadingGeoViolations.value) {
    fetchGeoViolations()
  }
})

// Reset activeTab if current tab becomes hidden
watch(hasCounterfeitQRs, (has) => {
  if (!has && activeTab.value === 'counterfeit') activeTab.value = 'codes'
})
watch(hasGeofenceViolations, (has) => {
  if (!has && activeTab.value === 'geofence') activeTab.value = 'codes'
})

const trendChartData = computed(() => {
  if (!analyticsData.value?.trends?.length) return null
  const trends = analyticsData.value.trends
  return {
    labels: trends.map(t => formatDate(t.date)),
    datasets: [
      {
        label: 'Validation',
        data: trends.map(t => t.validation),
        borderColor: '#18181b',
        backgroundColor: 'rgba(13,148,136,0.1)',
        fill: true,
        tension: 0.3
      },
      {
        label: 'Warranty',
        data: trends.map(t => t.warranty),
        borderColor: '#6366f1',
        backgroundColor: 'rgba(99,102,241,0.1)',
        fill: true,
        tension: 0.3
      }
    ]
  }
})

const trendChartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    mode: 'index',
    intersect: false
  },
  plugins: {
    legend: {
      position: 'top',
      labels: {
        usePointStyle: true,
        padding: 16
      }
    },
    tooltip: {
      mode: 'index',
      intersect: false
    }
  },
  scales: {
    x: {
      grid: { display: false }
    },
    y: {
      beginAtZero: true,
      ticks: { precision: 0 }
    }
  }
}))

const fetchBatch = async () => {
  try {
    loading.value = true
    const response = await get(`/tenant/qr-batches/${batchId.value}`)
    if (response.success && response.data) {
      batch.value = response.data.batch
      actualQRCount.value = response.data.qr_count
      // Start polling generation status if batch is still in progress
      if (batch.value?.status && !isTerminalStatus(batch.value.status)) {
        generationPolling.startPolling(batchId.value)
      }
      // Fetch geofence analytics if batch has geofencing
      if (batch.value?.geofence_enabled) {
        fetchGeofenceAnalytics()
      }
    }
  } catch (error) {
    console.error('Failed to fetch batch:', error)
    router.push('/tenant/products/dynamic')
  } finally {
    loading.value = false
  }
}

// When polling reports status change to terminal (completed/failed), refresh batch + codes.
// We intentionally fire on first-poll terminal transitions (oldStatus may be undefined on first fetch).
watch(() => generationPolling.status.value?.status, (newStatus, oldStatus) => {
  if (newStatus && newStatus !== oldStatus && isTerminalStatus(newStatus)) {
    fetchBatch()
    if (newStatus === 'completed') {
      fetchCodes()
    }
  }
})

const retryGeneration = async () => {
  if (retrying.value) return
  retrying.value = true
  try {
    const response = await post(`/tenant/qr-batches/${batchId.value}/retry-generation`)
    if (response.success) {
      // fetchBatch() itself restarts polling if the new status is non-terminal,
      // so we don't need to call startPolling explicitly here.
      await fetchBatch()
    }
  } catch (error) {
    console.error('Failed to retry generation:', error)
    alert(error.response?.data?.message || 'Failed to retry batch generation')
  } finally {
    retrying.value = false
  }
}

const fetchCodes = async () => {
  try {
    loadingCodes.value = true
    const params = {
      page: pagination.value.page,
      limit: pagination.value.limit,
    }
    if (statusFilter.value) {
      params.status = statusFilter.value
    }
    if (counterfeitFilter.value) {
      params.counterfeit_status = counterfeitFilter.value
    }
    const response = await get(`/tenant/qr-batches/${batchId.value}/codes`, params)
    if (response.success && response.data) {
      codes.value = response.data.codes || []
      const p = getPagination(response.data)
      pagination.value = { page: p.page, limit: p.limit, total: p.total, total_page: p.totalPages }
    }
  } catch (error) {
    console.error('Failed to fetch codes:', error)
  } finally {
    loadingCodes.value = false
  }
}

const getStatusBadge = (status) => {
  const badges = {
    active: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
    scanned: 'bg-zinc-100 text-zinc-700 dark:bg-zinc-900/30 dark:text-zinc-400',
    blocked: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
    expired: 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-400',
  }
  return badges[status] || 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-400'
}

// Counterfeit status badge
const getCounterfeitBadge = (status) => {
  const badges = {
    valid: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
    warning: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
    counterfeit: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  }
  return badges[status] || badges.valid
}

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
  } catch (error) {
    console.error('Failed to copy:', error)
  }
}

// Export functions - download from backend API with proper format for label printers
const downloadExport = async (format) => {
  if (!batch.value) return

  exporting.value[format] = true
  try {
    const endpoint = `/tenant/qr-batches/${batchId.value}/export/${format}`
    const response = await fetch(`${import.meta.env.VITE_API_URL}${endpoint}`, {
      method: 'GET',
      headers: getAuthHeaders(),
      credentials: 'include',
    })

    if (!response.ok) {
      if (response.status === 503) {
        alert('Server is busy processing another export. Please try again shortly.')
        return
      }
      throw new Error('Export failed')
    }

    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${batch.value.batch_code}_qr_codes.${format === 'csv' ? 'csv' : 'xlsx'}`
    a.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    console.error(`Failed to export ${format}:`, error)
    alert(`Failed to export ${format.toUpperCase()}. Please try again.`)
  } finally {
    exporting.value[format] = false
  }
}

const exportCSV = () => downloadExport('csv')
const exportExcel = () => downloadExport('excel')

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push('/tenant/products/dynamic')
  }
}

const onStatusFilterChange = () => {
  pagination.value.page = 1
  fetchCodes()
}

const onCounterfeitFilterChange = () => {
  pagination.value.page = 1
  fetchCodes()
}

// Template functions
const fetchTemplates = async () => {
  try {
    const response = await get('/tenant/templates', { status: 'active', limit: 100 })
    if (response.success && response.data?.templates) {
      validationTemplates.value = response.data.templates.filter(t => t.template_type === 'validation')
      warrantyTemplates.value = response.data.templates.filter(t => t.template_type === 'warranty')
    }
  } catch (error) {
    console.error('Failed to fetch templates:', error)
  }
}

const startEditingTemplates = () => {
  selectedValidationTemplateId.value = batch.value?.validation_template_id || ''
  selectedWarrantyTemplateId.value = batch.value?.warranty_template_id || ''
  editingTemplates.value = true
}

const cancelEditingTemplates = () => {
  editingTemplates.value = false
}

const saveTemplates = async () => {
  savingTemplates.value = true
  try {
    const response = await put(`/tenant/qr-batches/${batchId.value}`, {
      validation_template_id: selectedValidationTemplateId.value || '',
      warranty_template_id: selectedWarrantyTemplateId.value || '',
    })
    if (response.success) {
      batch.value = response.data
      editingTemplates.value = false
    } else {
      alert(response.message || 'Failed to save templates')
    }
  } catch (error) {
    console.error('Failed to save templates:', error)
    alert('Failed to save templates')
  } finally {
    savingTemplates.value = false
  }
}

const tour = useTour()

function handleTourSetValue(e) {
  if (!isTourActive()) return
  if (e.detail._nonce !== getTourNonce()) return
  // geofence_edit_radius is handled directly by GeofenceMapPicker
  // This handler exists for future tour extensions on this page
}

onMounted(async () => {
  await fetchBatch()
  fetchCodes()
  fetchTemplates()
  fetchBatchHeatmap()
  fetchBatchAnalytics()
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
    <div class="flex justify-between items-center mb-6">
      <div class="flex items-center gap-4">
        <button
          @click="goBack"
          class="p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
        >
          <ArrowLeft class="w-5 h-5" />
        </button>
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ batch?.batch_name || 'QR Batch Details' }}
          </h1>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
            {{ batch?.batch_code || '' }}
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <!-- Retry button for failed generation -->
        <Button
          v-if="isFailed"
          variant="outline"
          @click="retryGeneration"
          :disabled="retrying"
          class="text-zinc-600 border-zinc-200 hover:bg-zinc-50 dark:text-zinc-400 dark:border-zinc-800 dark:hover:bg-zinc-900/30"
        >
          <RefreshCw class="w-4 h-4 mr-2" :class="{ 'animate-spin': retrying }" />
          {{ retrying ? 'Retrying...' : 'Retry Generation' }}
        </Button>
        <!-- Export Buttons (disabled during generation) -->
        <template>
          <Button variant="outline" @click="exportCSV" :disabled="exporting.csv || !batch || isGenerating || isFailed"
            :title="isGenerating ? 'Export available after generation completes' : ''">
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            {{ exporting.csv ? 'Downloading...' : 'CSV' }}
          </Button>
          <Button variant="outline" @click="exportExcel" :disabled="exporting.excel || !batch || isGenerating || isFailed"
            :title="isGenerating ? 'Export available after generation completes' : ''">
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            {{ exporting.excel ? 'Downloading...' : 'Excel' }}
          </Button>
          <BatchPdfDownload
            v-if="batch && isCompleted"
            :batch-id="batchId"
            :qr-count="batch.qr_count || actualQRCount"
            :batch-code="batch.batch_code"
          />
        </template>
      </div>
    </div>

    <!-- Generation Status Card (shown when batch is generating or failed) -->
    <Card v-if="(isGenerating || isFailed) && generationPolling.status.value" class="p-6 mb-6 border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/10">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-3">
        {{ isFailed ? 'Generation Failed' : 'QR Generation In Progress' }}
      </h3>
      <p v-if="isGenerating" class="text-sm text-gray-600 dark:text-gray-400 mb-4">
        Your QR codes are being generated in the background. You can navigate away — we'll notify you when it's ready.
      </p>
      <p v-if="isFailed" class="text-sm text-gray-600 dark:text-gray-400 mb-4">
        Generation stopped unexpectedly. Click "Retry Generation" above to resume from where it left off.
      </p>
      <QRGenerationProgress
        :status="generationPolling.status.value.status"
        :generated-count="generationPolling.status.value.generated_count"
        :total-qr-count="generationPolling.status.value.total_qr_count"
        :progress-percent="generationPolling.status.value.progress_percent"
        :eta-seconds="generationPolling.status.value.eta_seconds"
        :error-message="generationPolling.status.value.error_message"
      />
    </Card>

    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <div v-else-if="batch" class="space-y-6">
      <!-- Batch Info Card -->
      <Card class="p-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Batch Information</h2>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-6">
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">Product</p>
            <p class="text-gray-900 dark:text-white font-medium">
              {{ batch.product?.product_name || '-' }}
            </p>
            <p v-if="batch.product?.product_code" class="text-xs text-gray-500 dark:text-gray-400">
              {{ batch.product.product_code }}
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">QR Type</p>
            <p class="text-gray-900 dark:text-white font-medium">
              {{ actualQRCount.toLocaleString() }} unique codes
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">Production Date</p>
            <p class="text-gray-900 dark:text-white font-medium">
              {{ formatDate(batch.production_date) }}
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">Expiry Date</p>
            <p class="text-gray-900 dark:text-white font-medium">
              {{ formatDate(batch.expiry_date) }}
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">Prefix / Suffix</p>
            <p class="text-gray-900 dark:text-white font-medium">
              {{ batch.prefix || '-' }} / {{ batch.suffix || '-' }}
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">Created At</p>
            <p class="text-gray-900 dark:text-white font-medium">
              {{ formatDateTime(batch.created_at) }}
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">Created By</p>
            <p class="text-gray-900 dark:text-white font-medium">
              {{ batch.created_by_staff?.user?.full_name || '-' }}
            </p>
          </div>
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400 mb-1">Features</p>
            <div class="flex flex-wrap gap-1">
              <!-- Validation/Landing Page is always enabled as entry point -->
              <span class="px-2 py-0.5 text-xs bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400 rounded">
                Landing Page
              </span>
              <span v-if="batch.product?.warranty_enabled" class="px-2 py-0.5 text-xs bg-zinc-100 text-zinc-700 dark:bg-zinc-900/30 dark:text-zinc-400 rounded">
                Warranty
              </span>
            </div>
          </div>
        </div>

        <!-- Templates Section -->
        <div class="mt-6 pt-6 border-t border-gray-200 dark:border-gray-700">
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300">Templates</h3>
            <button
              v-if="!editingTemplates"
              @click="startEditingTemplates"
              class="text-sm text-zinc-600 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300"
            >
              Edit
            </button>
            <div v-else class="flex gap-2">
              <button
                @click="cancelEditingTemplates"
                class="text-sm text-gray-600 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
                :disabled="savingTemplates"
              >
                Cancel
              </button>
              <button
                @click="saveTemplates"
                class="text-sm text-zinc-600 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300 font-medium"
                :disabled="savingTemplates"
              >
                {{ savingTemplates ? 'Saving...' : 'Save' }}
              </button>
            </div>
          </div>

          <!-- View Mode -->
          <div v-if="!editingTemplates" class="grid grid-cols-2 md:grid-cols-3 gap-4">
            <div>
              <p class="text-sm text-gray-500 dark:text-gray-400">Validation / Landing Page</p>
              <p class="text-gray-900 dark:text-white font-medium">
                {{ batch.validation_template?.template_name || 'Tenant Default' }}
              </p>
              <span
                v-if="batch.validation_template_id"
                class="text-xs text-purple-600 dark:text-purple-400"
              >
                (A/B Test Override)
              </span>
              <span
                v-else-if="batch.product?.default_validation_template_id"
                class="text-xs text-gray-500 dark:text-gray-400"
              >
                (Product Default)
              </span>
            </div>
            <div v-if="batch.product?.warranty_enabled">
              <p class="text-sm text-gray-500 dark:text-gray-400">Warranty Activation</p>
              <p class="text-gray-900 dark:text-white font-medium">
                {{ batch.warranty_template?.template_name || 'Tenant Default' }}
              </p>
              <span
                v-if="batch.warranty_template_id"
                class="text-xs text-purple-600 dark:text-purple-400"
              >
                (A/B Test Override)
              </span>
              <span
                v-else-if="batch.product?.default_warranty_template_id"
                class="text-xs text-gray-500 dark:text-gray-400"
              >
                (Product Default)
              </span>
            </div>
          </div>

          <!-- Edit Mode -->
          <div v-else class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label class="block text-sm text-gray-500 dark:text-gray-400 mb-1">Validation / Landing Page</label>
              <select
                v-model="selectedValidationTemplateId"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm"
              >
                <option value="">Use Default (Product/Tenant)</option>
                <option v-for="t in validationTemplates" :key="t.id" :value="t.id">
                  {{ t.template_name }}
                </option>
              </select>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                Override the default template for this batch
              </p>
            </div>
            <div v-if="batch.product?.warranty_enabled">
              <label class="block text-sm text-gray-500 dark:text-gray-400 mb-1">Warranty Activation</label>
              <select
                v-model="selectedWarrantyTemplateId"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm"
              >
                <option value="">Use Default (Product/Tenant)</option>
                <option v-for="t in warrantyTemplates" :key="t.id" :value="t.id">
                  {{ t.template_name }}
                </option>
              </select>
            </div>
          </div>
        </div>
      </Card>

      <!-- Export Info Card -->
      <Card class="p-4 bg-zinc-50 dark:bg-zinc-900/20 border-zinc-200 dark:border-zinc-800">
        <div class="flex items-start gap-3">
          <svg class="w-5 h-5 text-zinc-600 dark:text-zinc-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div>
            <h3 class="text-sm font-medium text-zinc-800 dark:text-zinc-300">Label Printing Guide</h3>
            <p class="text-sm text-zinc-700 dark:text-zinc-400 mt-1">
              Download CSV or Excel, then import to your label printing software (BarTender, NiceLabel, ZebraDesigner).
              The <code class="px-1 py-0.5 bg-zinc-100 dark:bg-zinc-800 rounded text-xs">qr_content</code> column contains the URL to encode as QR code.
              Or use <strong>Download PDF</strong> for ready-to-print label sheets (25 / 38 / 50 mm).
            </p>
          </div>
        </div>
      </Card>

      <!-- Batch Scan Heatmap -->
      <Card class="overflow-hidden">
        <div class="p-4 border-b border-gray-200 dark:border-gray-700">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <svg class="w-5 h-5 text-zinc-600 dark:text-zinc-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Scan Distribution Map</h2>
              <span v-if="heatmapData?.summary" class="text-sm text-gray-500 dark:text-gray-400">
                ({{ heatmapData.summary.total_points.toLocaleString() }} scans)
              </span>
            </div>
            <div v-if="batch.geofence_enabled" class="flex items-center gap-3">
              <button
                v-if="!editingGeofence"
                @click="startEditGeofence"
                class="text-sm text-zinc-600 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300"
                data-tour="edit-geofence-btn"
              >
                Edit Geofence Zone
              </button>
            </div>
          </div>

          <!-- Geofence zone info bar -->
          <div v-if="batch.geofence_enabled && batch.geofence_latitude != null && !editingGeofence" class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-600 dark:text-gray-400">
            <span><MapPinIcon class="w-3.5 h-3.5 inline text-zinc-500" /> {{ batch.geofence_label || 'Distribution Zone' }}</span>
            <span>Radius: {{ batch.geofence_radius_km }} km</span>
            <span>{{ batch.geofence_latitude?.toFixed(4) }}, {{ batch.geofence_longitude?.toFixed(4) }}</span>
          </div>
        </div>

        <!-- Edit geofence mode -->
        <div v-if="editingGeofence && batch.geofence_enabled" class="p-4">
          <!-- Zone Template Selector -->
          <div v-if="zoneTemplates.length > 0" class="flex items-center gap-2 mb-3">
            <label class="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">Load template:</label>
            <select
              class="flex-1 px-2 py-1 text-sm border rounded-md bg-white dark:bg-gray-900 dark:border-gray-700"
              @change="(e) => { if (e.target.value) loadZoneTemplate(zoneTemplates.find(t => t.id === e.target.value)); e.target.value = '' }"
            >
              <option value="">Select a saved zone...</option>
              <option v-for="t in zoneTemplates" :key="t.id" :value="t.id">
                {{ t.template_name }} ({{ t.radius_km }}km)
              </option>
            </select>
          </div>

          <div data-tour="geofence-edit-radius">
            <GeofenceMapPicker :model-value="geofenceForm" @update:model-value="onGeofenceFormUpdate" height="300px" />
          </div>
          <div class="flex gap-2 mt-3">
            <Button @click="saveGeofence" :disabled="savingGeofence || !geofenceForm.latitude" data-tour="save-geofence-btn">
              {{ savingGeofence ? 'Saving...' : 'Save' }}
            </Button>
            <Button variant="outline" @click="editingGeofence = false">Cancel</Button>
          </div>
        </div>

        <!-- Loading -->
        <div v-else-if="heatmapLoading" class="h-[350px] flex items-center justify-center">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
        </div>

        <!-- Error -->
        <div v-else-if="heatmapError" class="p-8 text-center">
          <svg class="w-12 h-12 text-gray-300 dark:text-gray-600 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
          </svg>
          <p class="text-gray-500 dark:text-gray-400">Failed to load heatmap data.</p>
          <button
            @click="heatmapData = null; fetchBatchHeatmap()"
            class="text-zinc-600 dark:text-zinc-400 hover:underline mt-2 text-sm"
          >
            Retry
          </button>
        </div>

        <!-- Empty state -->
        <div v-else-if="heatmapLocations.length === 0" class="p-8 text-center">
          <svg class="w-12 h-12 text-gray-300 dark:text-gray-600 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" />
          </svg>
          <p class="text-gray-500 dark:text-gray-400">No scan locations recorded for this batch yet.</p>
          <p class="text-sm text-gray-400 dark:text-gray-500 mt-1">
            Scan data with geolocation will appear here once end users scan QR codes from this batch.
          </p>
        </div>

        <!-- Heatmap + summary bar -->
        <template v-else>
          <ScanHeatmap
            :locations="heatmapLocations"
            :geofence-violations="heatmapGeofenceViolations"
            :geofence-zones="geofenceZone"
            height="400px"
          />
          <div class="px-4 py-3 bg-gray-50 dark:bg-gray-800 flex items-center gap-6 text-sm border-t border-gray-200 dark:border-gray-700">
            <span class="text-gray-600 dark:text-gray-400">
              Total: <strong class="text-gray-900 dark:text-white">{{ heatmapData.summary.total_points.toLocaleString() }}</strong>
            </span>
            <span class="text-green-600 dark:text-green-400">
              Valid: {{ heatmapData.summary.valid_count.toLocaleString() }}
            </span>
            <span v-if="heatmapData.summary.counterfeit_count > 0" class="text-purple-600 dark:text-purple-400">
              Counterfeit: {{ heatmapData.summary.counterfeit_count.toLocaleString() }}
            </span>
            <span v-if="heatmapData.summary.geofence_violation_count > 0" class="text-orange-600 dark:text-orange-400">
              Geofence: {{ heatmapData.summary.geofence_violation_count.toLocaleString() }}
            </span>
          </div>

          <!-- Geofence analytics -->
          <div v-if="geofenceAnalytics" class="px-4 py-3 border-t border-gray-200 dark:border-gray-700">
            <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">Violation Rate</p>
                <p class="text-xl font-bold" :class="(geofenceAnalytics.violation_rate ?? 0) > 5 ? 'text-red-600' : (geofenceAnalytics.violation_rate ?? 0) > 2 ? 'text-orange-600' : 'text-green-600'">
                  {{ (geofenceAnalytics.violation_rate ?? 0).toFixed(1) }}%
                </p>
              </div>
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">In Zone</p>
                <p class="text-xl font-bold text-green-600">{{ (geofenceAnalytics.in_zone_count ?? 0).toLocaleString() }}</p>
              </div>
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">Out of Zone</p>
                <p class="text-xl font-bold text-orange-600">{{ (geofenceAnalytics.out_of_zone_count ?? 0).toLocaleString() }}</p>
              </div>
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">Total Scans</p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">{{ (geofenceAnalytics.total_scans ?? 0).toLocaleString() }}</p>
              </div>
            </div>
          </div>
        </template>
      </Card>

      <!-- Batch Analytics -->
      <!-- Analytics Loading -->
      <div v-if="analyticsLoading" class="space-y-4">
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card v-for="i in 4" :key="i" class="p-4">
            <div class="animate-pulse space-y-3">
              <div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-24"></div>
              <div class="h-8 bg-gray-200 dark:bg-gray-700 rounded w-16"></div>
              <div class="h-3 bg-gray-200 dark:bg-gray-700 rounded w-32"></div>
            </div>
          </Card>
        </div>
        <Card class="p-4">
          <div class="animate-pulse h-[300px] bg-gray-200 dark:bg-gray-700 rounded"></div>
        </Card>
      </div>

      <!-- Analytics Error -->
      <Card v-else-if="analyticsError" class="p-8 text-center">
        <svg class="w-12 h-12 text-gray-300 dark:text-gray-600 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
        </svg>
        <p class="text-gray-500 dark:text-gray-400">Failed to load analytics data.</p>
        <button
          @click="analyticsData = null; fetchBatchAnalytics()"
          class="text-zinc-600 dark:text-zinc-400 hover:underline mt-2 text-sm"
        >
          Retry
        </button>
      </Card>

      <!-- Analytics Content -->
      <template v-else-if="analyticsData">
        <!-- Stat Cards -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <!-- Total Scans -->
          <Card class="p-4">
            <div class="flex items-center gap-3">
              <div class="p-2 bg-zinc-100 dark:bg-zinc-900/30 rounded-lg">
                <Eye class="w-5 h-5 text-zinc-600 dark:text-zinc-400" />
              </div>
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">Total Scans</p>
                <p class="text-2xl font-bold text-gray-900 dark:text-white">
                  {{ analyticsData.summary.total_scans.toLocaleString() }}
                </p>
                <p class="text-xs text-gray-400 dark:text-gray-500">scans recorded</p>
              </div>
            </div>
          </Card>

          <!-- Scan Rate -->
          <Card class="p-4">
            <div class="flex items-center gap-3">
              <div class="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg">
                <Activity class="w-5 h-5 text-green-600 dark:text-green-400" />
              </div>
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">Scan Rate</p>
                <p class="text-2xl font-bold text-gray-900 dark:text-white">
                  {{ analyticsData.summary.scan_rate }}%
                </p>
                <p class="text-xs text-gray-400 dark:text-gray-500">
                  {{ analyticsData.summary.unique_qr_scanned.toLocaleString() }} of {{ analyticsData.summary.total_qr_codes.toLocaleString() }} QR codes
                </p>
              </div>
            </div>
          </Card>

          <!-- Geographic Reach -->
          <Card class="p-4">
            <div class="flex items-center gap-3">
              <div class="p-2 bg-zinc-100 dark:bg-zinc-900/30 rounded-lg">
                <MapPinIcon class="w-5 h-5 text-zinc-600 dark:text-zinc-400" />
              </div>
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">Geographic Reach</p>
                <p class="text-2xl font-bold text-gray-900 dark:text-white">
                  {{ analyticsData.summary.unique_cities }}
                </p>
                <p class="text-xs text-gray-400 dark:text-gray-500">unique cities</p>
              </div>
            </div>
          </Card>

          <!-- Scan Breakdown -->
          <Card class="p-4">
            <div class="flex items-center gap-3">
              <div class="p-2 bg-amber-100 dark:bg-amber-900/30 rounded-lg">
                <BarChart3 class="w-5 h-5 text-amber-600 dark:text-amber-400" />
              </div>
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">Scan Breakdown</p>
                <div class="flex items-center gap-2 mt-1 text-sm">
                  <span class="text-zinc-600 dark:text-zinc-400 font-medium">
                    {{ analyticsData.summary.validation_scans.toLocaleString() }}
                  </span>
                  <span class="text-gray-300 dark:text-gray-600">/</span>
                  <span class="text-zinc-600 dark:text-zinc-400 font-medium">
                    {{ analyticsData.summary.warranty_scans.toLocaleString() }}
                  </span>
                </div>
                <p class="text-xs text-gray-400 dark:text-gray-500">validation / warranty</p>
              </div>
            </div>
          </Card>
        </div>

        <!-- Scan Trend Chart -->
        <Card v-if="trendChartData" class="overflow-hidden">
          <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Scan Trends</h2>
            <span class="text-xs px-2 py-1 rounded-full bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400">
              {{ analyticsData.trend_granularity === 'week' ? 'Weekly' : 'Daily' }}
            </span>
          </div>
          <div class="p-4">
            <div class="h-[300px]">
              <Line :data="trendChartData" :options="trendChartOptions" />
            </div>
          </div>
        </Card>

        <!-- Top Scan Locations -->
        <Card v-if="analyticsData.top_locations && analyticsData.top_locations.length > 0" class="overflow-hidden">
          <div class="p-4 border-b border-gray-200 dark:border-gray-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Top Scan Locations</h2>
          </div>
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-gray-200 dark:border-gray-700">
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">#</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">City</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Country</th>
                  <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Scans</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider w-48">Distribution</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(loc, index) in analyticsData.top_locations"
                  :key="index"
                  class="border-b border-gray-100 dark:border-gray-800 last:border-0"
                >
                  <td class="px-4 py-3 text-gray-500 dark:text-gray-400">{{ index + 1 }}</td>
                  <td class="px-4 py-3 font-medium text-gray-900 dark:text-white">{{ loc.city }}</td>
                  <td class="px-4 py-3 text-gray-500 dark:text-gray-400">{{ loc.country }}</td>
                  <td class="px-4 py-3 text-right font-medium text-gray-900 dark:text-white">{{ loc.count.toLocaleString() }}</td>
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      <div class="flex-1 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                        <div
                          class="h-full bg-zinc-500 dark:bg-zinc-400 rounded-full"
                          :style="{ width: loc.percentage + '%' }"
                        ></div>
                      </div>
                      <span class="text-xs text-gray-500 dark:text-gray-400 w-12 text-right">{{ loc.percentage }}%</span>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </Card>
      </template>

      <!-- Hide QR codes list while generation is in progress or failed -->
      <!-- Empty state placeholder when generating -->
      <Card v-if="isGenerating || isFailed" class="p-12 text-center">
        <div class="text-gray-400 dark:text-gray-500 mb-3">
          <svg class="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
          </svg>
        </div>
        <p class="text-gray-600 dark:text-gray-300 font-medium">
          {{ isFailed ? 'QR codes are not available' : 'QR codes will be available once generation completes' }}
        </p>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
          {{ isFailed ? 'Please retry generation to resume.' : 'See progress above.' }}
        </p>
      </Card>

      <!-- QR Codes Tabbed Section (for Dynamic QR, completed only) -->
      <Card v-else class="overflow-hidden">
        <!-- Tab Bar -->
        <div class="border-b border-gray-200 dark:border-gray-700">
          <nav class="flex gap-0 px-4 overflow-x-auto">
            <button
              @click="onTabChange('codes')"
              :class="[
                'py-3 px-4 border-b-2 font-medium text-sm transition-colors whitespace-nowrap',
                activeTab === 'codes'
                  ? 'border-zinc-500 text-zinc-600'
                  : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
              ]"
            >
              QR Codes
              <span class="ml-1 text-xs text-gray-400">({{ pagination.total.toLocaleString() }})</span>
            </button>
            <button
              v-if="hasCounterfeitQRs"
              @click="onTabChange('counterfeit')"
              :class="[
                'py-3 px-4 border-b-2 font-medium text-sm transition-colors whitespace-nowrap',
                activeTab === 'counterfeit'
                  ? 'border-red-500 text-red-600'
                  : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
              ]"
            >
              Counterfeited QRs
              <span v-if="counterfeitPagination.total" class="ml-1 text-xs text-gray-400">({{ counterfeitPagination.total.toLocaleString() }})</span>
            </button>
            <button
              v-if="hasGeofenceViolations"
              @click="onTabChange('geofence')"
              :class="[
                'py-3 px-4 border-b-2 font-medium text-sm transition-colors whitespace-nowrap',
                activeTab === 'geofence'
                  ? 'border-orange-500 text-orange-600'
                  : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
              ]"
            >
              Geofence Violated QRs
              <span v-if="geoViolationsPagination.total" class="ml-1 text-xs text-gray-400">({{ geoViolationsPagination.total.toLocaleString() }})</span>
            </button>
          </nav>
        </div>

        <!-- Tab: QR Codes -->
        <template v-if="activeTab === 'codes'">
          <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex justify-end items-center gap-4">
            <div class="flex items-center gap-2">
              <label class="text-sm text-gray-600 dark:text-gray-400">Status:</label>
              <select
                v-model="statusFilter"
                @change="onStatusFilterChange"
                class="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
              >
                <option value="">All</option>
                <option value="active">Active</option>
                <option value="scanned">Scanned</option>
                <option value="blocked">Blocked</option>
                <option value="expired">Expired</option>
              </select>
            </div>
            <div class="flex items-center gap-2">
              <label class="text-sm text-gray-600 dark:text-gray-400">Counterfeit:</label>
              <select
                v-model="counterfeitFilter"
                @change="onCounterfeitFilterChange"
                class="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
              >
                <option value="">All</option>
                <option value="valid">Valid</option>
                <option value="warning">Warning</option>
                <option value="counterfeit">Counterfeit</option>
              </select>
            </div>
          </div>

          <div v-if="loadingCodes" class="flex justify-center py-8">
            <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-zinc-500"></div>
          </div>

          <div v-else-if="codes.length === 0" class="p-8 text-center">
            <svg class="w-12 h-12 text-gray-300 dark:text-gray-600 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
            </svg>
            <p class="text-gray-500 dark:text-gray-400">No QR codes found</p>
          </div>

          <div v-else class="overflow-x-auto">
            <table class="w-full">
              <thead class="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">QR Code</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Status</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Counterfeit</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Scan Count</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">First Scanned</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Created</th>
                  <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
                <tr v-for="code in codes" :key="code.id"
                    @click="$router.push(`/tenant/qr-batches/${batchId}/codes/${code.id}`)"
                    class="hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer">
                  <td class="px-4 py-3">
                    <code class="text-sm font-mono text-gray-900 dark:text-white bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
                      {{ code.qr_code.length > 24 ? code.qr_code.slice(0, 24) + '...' : code.qr_code }}
                    </code>
                  </td>
                  <td class="px-4 py-3">
                    <span :class="['px-2 py-1 text-xs font-medium rounded', getStatusBadge(code.status)]">{{ code.status }}</span>
                  </td>
                  <td class="px-4 py-3">
                    <span :class="['px-2 py-1 text-xs font-medium rounded', getCounterfeitBadge(code.counterfeit_status)]">{{ code.counterfeit_status || 'valid' }}</span>
                  </td>
                  <td class="px-4 py-3 text-sm text-gray-900 dark:text-white">{{ code.scan_count }}</td>
                  <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{{ formatDateTime(code.first_scanned_at) }}</td>
                  <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{{ formatDate(code.created_at) }}</td>
                  <td class="px-4 py-3 text-right">
                    <button @click.stop="copyToClipboard(code.qr_code)" class="text-xs text-zinc-600 dark:text-zinc-400 hover:underline" title="Copy QR code">Copy</button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Pagination -->
          <div v-if="pagination.total_page > 1" class="px-4 py-3 border-t border-gray-200 dark:border-gray-700 grid grid-cols-3 items-center">
            <p class="text-sm text-gray-500 dark:text-gray-400">
              Showing {{ (pagination.page - 1) * pagination.limit + 1 }} - {{ Math.min(pagination.page * pagination.limit, pagination.total) }} of {{ pagination.total }}
            </p>
            <div class="flex gap-2 justify-center">
              <Button variant="outline" size="sm" :disabled="pagination.page === 1" @click="pagination.page--; fetchCodes()">Previous</Button>
              <span class="py-2 px-4 text-sm text-gray-600 dark:text-gray-400">Page {{ pagination.page }} of {{ pagination.total_page }}</span>
              <Button variant="outline" size="sm" :disabled="pagination.page >= pagination.total_page" @click="pagination.page++; fetchCodes()">Next</Button>
            </div>
            <div></div>
          </div>
        </template>

        <!-- Tab: Counterfeited QRs -->
        <template v-else-if="activeTab === 'counterfeit'">
          <div v-if="loadingCounterfeitCodes" class="flex justify-center py-8">
            <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-red-500"></div>
          </div>

          <div v-else-if="counterfeitCodes.length === 0" class="p-8 text-center">
            <svg class="w-12 h-12 text-gray-300 dark:text-gray-600 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
            <p class="text-gray-500 dark:text-gray-400">No counterfeited QR codes found</p>
          </div>

          <div v-else class="overflow-x-auto">
            <table class="w-full">
              <thead class="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">QR Code</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Status</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Counterfeit</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Scan Count</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">First Scanned</th>
                  <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Created</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
                <tr v-for="code in counterfeitCodes" :key="code.id"
                    @click="$router.push(`/tenant/qr-batches/${batchId}/codes/${code.id}`)"
                    class="hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer">
                  <td class="px-4 py-3">
                    <code class="text-sm font-mono text-gray-900 dark:text-white bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
                      {{ code.qr_code.length > 24 ? code.qr_code.slice(0, 24) + '...' : code.qr_code }}
                    </code>
                  </td>
                  <td class="px-4 py-3">
                    <span :class="['px-2 py-1 text-xs font-medium rounded', getStatusBadge(code.status)]">{{ code.status }}</span>
                  </td>
                  <td class="px-4 py-3">
                    <span :class="['px-2 py-1 text-xs font-medium rounded', getCounterfeitBadge(code.counterfeit_status)]">{{ code.counterfeit_status || 'valid' }}</span>
                  </td>
                  <td class="px-4 py-3 text-sm text-gray-900 dark:text-white">{{ code.scan_count }}</td>
                  <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{{ formatDateTime(code.first_scanned_at) }}</td>
                  <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{{ formatDate(code.created_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Pagination -->
          <div v-if="counterfeitPagination.total_page > 1" class="px-4 py-3 border-t border-gray-200 dark:border-gray-700 grid grid-cols-3 items-center">
            <p class="text-sm text-gray-500 dark:text-gray-400">
              Showing {{ (counterfeitPagination.page - 1) * counterfeitPagination.limit + 1 }} - {{ Math.min(counterfeitPagination.page * counterfeitPagination.limit, counterfeitPagination.total) }} of {{ counterfeitPagination.total }}
            </p>
            <div class="flex gap-2 justify-center">
              <Button variant="outline" size="sm" :disabled="counterfeitPagination.page === 1" @click="counterfeitPagination.page--; fetchCounterfeitCodes()">Previous</Button>
              <span class="py-2 px-4 text-sm text-gray-600 dark:text-gray-400">Page {{ counterfeitPagination.page }} of {{ counterfeitPagination.total_page }}</span>
              <Button variant="outline" size="sm" :disabled="counterfeitPagination.page >= counterfeitPagination.total_page" @click="counterfeitPagination.page++; fetchCounterfeitCodes()">Next</Button>
            </div>
            <div></div>
          </div>
        </template>

        <!-- Tab: Geofence Violated QRs -->
        <template v-else-if="activeTab === 'geofence'">
          <div v-if="loadingGeoViolations" class="flex justify-center py-8">
            <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-orange-500"></div>
          </div>

          <div v-else-if="geoViolations.length === 0" class="p-8 text-center">
            <svg class="w-12 h-12 text-gray-300 dark:text-gray-600 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            <p class="text-gray-500 dark:text-gray-400">No geofence violations found</p>
          </div>

          <div v-else class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead class="bg-gray-50 dark:bg-gray-800 text-gray-600 dark:text-gray-400">
                <tr>
                  <th class="px-4 py-3 text-left font-medium">Scan Location</th>
                  <th class="px-4 py-3 text-left font-medium">Distance</th>
                  <th class="px-4 py-3 text-left font-medium">Severity</th>
                  <th class="px-4 py-3 text-left font-medium">Time</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
                <tr
                  v-for="v in geoViolations"
                  :key="v.id"
                  class="hover:bg-gray-50 dark:hover:bg-gray-800/50"
                >
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-1 text-gray-700 dark:text-gray-300">
                      <Navigation class="w-3 h-3 text-gray-400" />
                      {{ v.scan_latitude?.toFixed(4) }}, {{ v.scan_longitude?.toFixed(4) }}
                    </div>
                  </td>
                  <td class="px-4 py-3">
                    <div class="text-gray-900 dark:text-white font-medium">
                      {{ v.distance_from_edge_km?.toFixed(1) }} km
                    </div>
                    <div class="text-xs text-gray-500 dark:text-gray-400">from zone edge</div>
                  </td>
                  <td class="px-4 py-3">
                    <span :class="['px-2 py-1 text-xs font-medium rounded-full', getSeverityColor(v.severity)]">
                      {{ v.severity?.toUpperCase() }}
                    </span>
                  </td>
                  <td class="px-4 py-3 text-gray-500 dark:text-gray-400 whitespace-nowrap">
                    {{ formatDateTime(v.created_at) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Pagination -->
          <div v-if="geoViolationsPagination.total_page > 1" class="px-4 py-3 border-t border-gray-200 dark:border-gray-700 grid grid-cols-3 items-center">
            <p class="text-sm text-gray-500 dark:text-gray-400">
              Showing {{ (geoViolationsPagination.page - 1) * geoViolationsPagination.limit + 1 }} - {{ Math.min(geoViolationsPagination.page * geoViolationsPagination.limit, geoViolationsPagination.total) }} of {{ geoViolationsPagination.total }}
            </p>
            <div class="flex gap-2 justify-center">
              <Button variant="outline" size="sm" :disabled="geoViolationsPagination.page === 1" @click="geoViolationsPagination.page--; fetchGeoViolations()">Previous</Button>
              <span class="py-2 px-4 text-sm text-gray-600 dark:text-gray-400">Page {{ geoViolationsPagination.page }} of {{ geoViolationsPagination.total_page }}</span>
              <Button variant="outline" size="sm" :disabled="geoViolationsPagination.page >= geoViolationsPagination.total_page" @click="geoViolationsPagination.page++; fetchGeoViolations()">Next</Button>
            </div>
            <div></div>
          </div>
        </template>
      </Card>
    </div>
  </div>
</template>
