<script setup>
import { ref, onMounted, computed, watch, nextTick, onUnmounted } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useDateTime } from '@/composables/useDateTime'
import { useEscapeKey } from '@/composables/useEscapeKey'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { getPagination } from '@/lib/pagination'

const { get, post, put } = useAPI()
const { formatDateTime } = useDateTime()

// Helper to escape HTML entities to prevent XSS in Leaflet popups
function escapeHtml(str) {
  if (!str) return ''
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}

// Tabs
const activeTab = ref('detections') // detections, settings

// List state
const detections = ref([])
const loading = ref(false)
const statusFilter = ref('')
const searchQuery = ref('')
const page = ref(1)
const limit = ref(20)
const total = ref(0)
const totalPages = ref(0)

// Stats
const stats = ref({
  active: 0,
  false_positive: 0,
  total: 0,
  recent_7_days: 0
})

// Detail modal state
const showDetailModal = ref(false)
const selectedDetection = ref(null)
const detectionDetail = ref(null)
const interactions = ref([])
const velocityData = ref([])
const geopoints = ref([])
const loadingDetail = ref(false)

// Map state
const mapContainer = ref(null)
let map = null
let markersLayer = null
let pathLayer = null

// False positive modal
const showActionModal = ref(false)
const actionType = ref('')
const actionForm = ref({
  resolution_notes: '',
  reason: '',
  level: 'qr',
  new_threshold: 0
})
const actionError = ref('')
const actionLoading = ref(false)

// Date range filter
const formatDateLocal = (date) => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
const today = new Date()
const startOfMonth = new Date(today.getFullYear(), today.getMonth(), 1)
const dateFrom = ref(formatDateLocal(startOfMonth))
const dateTo = ref(formatDateLocal(today))
const datePreset = ref('this_month')

const applyPreset = (preset) => {
  datePreset.value = preset
  const now = new Date()
  switch (preset) {
    case 'this_month':
      dateFrom.value = formatDateLocal(new Date(now.getFullYear(), now.getMonth(), 1))
      dateTo.value = formatDateLocal(now)
      break
    case 'last_month': {
      const lastMonth = new Date(now.getFullYear(), now.getMonth() - 1, 1)
      const lastDayLastMonth = new Date(now.getFullYear(), now.getMonth(), 0)
      dateFrom.value = formatDateLocal(lastMonth)
      dateTo.value = formatDateLocal(lastDayLastMonth)
      break
    }
    case 'last_7_days': {
      const sevenDaysAgo = new Date(now.getTime() - 6 * 24 * 60 * 60 * 1000)
      dateFrom.value = formatDateLocal(sevenDaysAgo)
      dateTo.value = formatDateLocal(now)
      break
    }
    case 'last_30_days': {
      const thirtyDaysAgo = new Date(now.getTime() - 29 * 24 * 60 * 60 * 1000)
      dateFrom.value = formatDateLocal(thirtyDaysAgo)
      dateTo.value = formatDateLocal(now)
      break
    }
    case 'custom':
      break
  }
}

// Settings state
const settings = ref({
  qc_scan_max: 0,
  warehouse_scan_max: 0,
  end_user_scan_max: 0,
  velocity_check_enabled: false,
  max_speed_kmh: 1000,
  alert_on_detection: true,
  auto_flag_suspicious: true
})
const loadingSettings = ref(false)
const savingSettings = ref(false)
const settingsError = ref('')
const settingsSuccess = ref('')

// Reports tab state
const reports = ref([])
const reportsLoading = ref(false)
const reportsSearchQuery = ref('')
const reportsPage = ref(1)
const reportsLimit = ref(20)
const reportsTotal = ref(0)
const reportsTotalPages = ref(0)
const reportStats = ref({
  total: 0,
  today: 0,
  this_week: 0,
  this_month: 0
})

// Report detail modal
const showReportModal = ref(false)
const selectedReport = ref(null)
const reportDetail = ref(null)
const loadingReportDetail = ref(false)

// Photo lightbox
const lightboxOpen = ref(false)
const lightboxIndex = ref(0)
const lightboxPhotos = ref([])

// Close modals on Escape key
useEscapeKey(() => {
  if (lightboxOpen.value) {
    closeLightbox()
  } else if (showActionModal.value) {
    showActionModal.value = false
  } else if (showReportModal.value) {
    closeReportModal()
  } else if (showDetailModal.value) {
    closeDetailModal()
  }
})

// Fetch functions
async function fetchDetections() {
  loading.value = true
  try {
    let url = `/tenant/counterfeit?page=${page.value}&limit=${limit.value}&from=${dateFrom.value}&to=${dateTo.value}`
    if (statusFilter.value) {
      url += `&status=${statusFilter.value}`
    }
    if (searchQuery.value) {
      url += `&search=${encodeURIComponent(searchQuery.value)}`
    }
    const response = await get(url)
    if (response.success) {
      detections.value = response.data?.detections || []
      const p = getPagination(response.data)
      total.value = p.total
      totalPages.value = p.totalPages
    }
  } catch (error) {
    console.error('Failed to fetch detections:', error)
  } finally {
    loading.value = false
  }
}

async function fetchStats() {
  try {
    const response = await get(`/tenant/counterfeit/stats?from=${dateFrom.value}&to=${dateTo.value}`)
    if (response.success) {
      stats.value = response.data
    }
  } catch (error) {
    console.error('Failed to fetch stats:', error)
  }
}

async function fetchDetectionDetail(id) {
  loadingDetail.value = true
  try {
    const response = await get(`/tenant/counterfeit/${id}`)
    if (response.success) {
      detectionDetail.value = response.data?.detection
      interactions.value = response.data?.interactions || []
      velocityData.value = response.data?.velocity_data || []
    }

    // Also fetch geolocations for map
    const geoResponse = await get(`/tenant/counterfeit/${id}/geolocations`)
    if (geoResponse.success) {
      geopoints.value = geoResponse.data?.geopoints || []
    }
  } catch (error) {
    console.error('Failed to fetch detection detail:', error)
  } finally {
    loadingDetail.value = false
  }
}

async function fetchSettings() {
  loadingSettings.value = true
  try {
    const response = await get('/tenant/counterfeit/settings')
    if (response.success) {
      settings.value = {
        qc_scan_max: response.data?.qc_scan_max || 0,
        warehouse_scan_max: response.data?.warehouse_scan_max || 0,
        end_user_scan_max: response.data?.end_user_scan_max || 0,
        velocity_check_enabled: response.data?.velocity_check_enabled || false,
        max_speed_kmh: response.data?.max_speed_kmh || 1000,
        alert_on_detection: response.data?.alert_on_detection !== false,
        auto_flag_suspicious: response.data?.auto_flag_suspicious !== false
      }
    }
  } catch (error) {
    console.error('Failed to fetch settings:', error)
  } finally {
    loadingSettings.value = false
  }
}

async function fetchReports() {
  reportsLoading.value = true
  try {
    let url = `/tenant/counterfeit/reports?page=${reportsPage.value}&limit=${reportsLimit.value}`
    if (reportsSearchQuery.value) {
      url += `&search=${encodeURIComponent(reportsSearchQuery.value)}`
    }
    const response = await get(url)
    if (response.success) {
      reports.value = response.data?.reports || []
      const p = getPagination(response.data)
      reportsTotal.value = p.total
      reportsTotalPages.value = p.totalPages
    }
  } catch (error) {
    console.error('Failed to fetch reports:', error)
  } finally {
    reportsLoading.value = false
  }
}

async function fetchReportStats() {
  try {
    const response = await get('/tenant/counterfeit/reports/stats')
    if (response.success) {
      reportStats.value = response.data
    }
  } catch (error) {
    console.error('Failed to fetch report stats:', error)
  }
}

async function fetchReportDetail(id) {
  loadingReportDetail.value = true
  try {
    const response = await get(`/tenant/counterfeit/reports/${id}`)
    if (response.success) {
      reportDetail.value = response.data
    }
  } catch (error) {
    console.error('Failed to fetch report detail:', error)
  } finally {
    loadingReportDetail.value = false
  }
}

async function saveSettings() {
  savingSettings.value = true
  settingsError.value = ''
  settingsSuccess.value = ''
  try {
    const response = await put('/tenant/counterfeit/settings', settings.value)
    if (response.success) {
      settingsSuccess.value = 'Settings saved successfully'
      setTimeout(() => { settingsSuccess.value = '' }, 3000)
    } else {
      settingsError.value = response.message || 'Failed to save settings'
    }
  } catch (error) {
    settingsError.value = error.response?.data?.message || 'Failed to save settings'
  } finally {
    savingSettings.value = false
  }
}

// Detail modal functions
function openDetail(detection) {
  selectedDetection.value = detection
  showDetailModal.value = true
  fetchDetectionDetail(detection.id)
}

function closeDetailModal() {
  showDetailModal.value = false
  selectedDetection.value = null
  detectionDetail.value = null
  interactions.value = []
  velocityData.value = []
  geopoints.value = []
  destroyMap()
}

// Map functions
function initMap() {
  if (!mapContainer.value || map) return

  map = L.map(mapContainer.value).setView([-2.5, 118], 5) // Indonesia center

  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
    maxZoom: 19
  }).addTo(map)

  markersLayer = L.layerGroup().addTo(map)
  pathLayer = L.layerGroup().addTo(map)

  updateMapMarkers()
}

function destroyMap() {
  if (map) {
    map.remove()
    map = null
    markersLayer = null
    pathLayer = null
  }
}

function updateMapMarkers() {
  if (!map || !markersLayer) return

  markersLayer.clearLayers()
  pathLayer.clearLayers()

  if (geopoints.value.length === 0) return

  const pathCoords = []

  geopoints.value.forEach((point, index) => {
    const isFirst = index === 0
    const isLast = index === geopoints.value.length - 1

    // Check if this point is part of impossible travel
    const isImpossible = velocityData.value.some(v =>
      v.interaction_index === index && v.is_impossible
    )

    // Determine marker color
    let color = '#F5A623' // Amber for normal
    if (isFirst) color = '#10B981' // Green for first
    else if (isLast) color = '#EF4444' // Red for last
    else if (isImpossible) color = '#3f3f46' // Cyan for impossible travel

    const marker = L.circleMarker([point.lat, point.lng], {
      radius: isFirst || isLast ? 10 : 7,
      fillColor: color,
      color: '#1F2937',
      weight: 2,
      opacity: 1,
      fillOpacity: 0.8
    })

    marker.bindPopup(`
      <div class="text-sm">
        <strong>Scan #${point.index}</strong><br>
        <span class="text-gray-500">Category:</span> ${escapeHtml(point.category)}<br>
        <span class="text-gray-500">Subcategory:</span> ${escapeHtml(point.subcategory)}<br>
        <span class="text-gray-500">Time:</span> ${escapeHtml(formatDateTime(point.timestamp))}<br>
        ${isImpossible ? '<span class="text-orange-600 font-bold">Impossible Travel Detected!</span>' : ''}
      </div>
    `)

    markersLayer.addLayer(marker)
    pathCoords.push([point.lat, point.lng])
  })

  // Draw path between points
  if (pathCoords.length > 1) {
    // Draw segments with different colors for impossible travel
    for (let i = 0; i < pathCoords.length - 1; i++) {
      const isImpossible = velocityData.value.some(v =>
        v.interaction_index === i + 1 && v.is_impossible
      )

      const segment = L.polyline([pathCoords[i], pathCoords[i + 1]], {
        color: isImpossible ? '#3f3f46' : '#F5A623',
        weight: isImpossible ? 4 : 2,
        opacity: 0.8,
        dashArray: isImpossible ? '10, 10' : null
      })

      pathLayer.addLayer(segment)
    }

    // Fit bounds
    const bounds = L.latLngBounds(pathCoords)
    map.fitBounds(bounds, { padding: [50, 50] })
  } else if (pathCoords.length === 1) {
    map.setView(pathCoords[0], 12)
  }
}

// Watch date changes to refetch detections and stats
watch([dateFrom, dateTo], () => {
  page.value = 1
  fetchDetections()
  fetchStats()
})

// Watch geopoints changes to update map
watch(geopoints, () => {
  if (showDetailModal.value && geopoints.value.length > 0) {
    nextTick(() => {
      if (!map) {
        setTimeout(initMap, 100)
      } else {
        updateMapMarkers()
      }
    })
  }
}, { deep: true })

// Action modal functions
function openFalsePositiveModal() {
  actionType.value = 'false_positive'
  const scanCount = detectionDetail.value?.total_interactions_count || 0
  actionForm.value = {
    resolution_notes: '',
    reason: '',
    level: 'qr',
    new_threshold: scanCount + 10
  }
  actionError.value = ''
  showActionModal.value = true
}

async function submitAction() {
  actionError.value = ''
  actionLoading.value = true

  try {
    if (!actionForm.value.reason) {
      actionError.value = 'Reason is required'
      actionLoading.value = false
      return
    }
    if (!actionForm.value.new_threshold || actionForm.value.new_threshold < 1) {
      actionError.value = 'New threshold must be at least 1'
      actionLoading.value = false
      return
    }
    const currentScanCount = detectionDetail.value?.total_interactions_count || 0
    if (actionForm.value.new_threshold <= currentScanCount) {
      actionError.value = `New threshold (${actionForm.value.new_threshold}) must be greater than current scan count (${currentScanCount})`
      actionLoading.value = false
      return
    }
    const endpoint = `/tenant/counterfeit/${selectedDetection.value.id}/override-threshold`
    const payload = {
      level: actionForm.value.level,
      new_threshold: parseInt(actionForm.value.new_threshold),
      reason: actionForm.value.reason
    }

    const response = await post(endpoint, payload)
    if (response.success) {
      showActionModal.value = false
      closeDetailModal()
      fetchDetections()
      fetchStats()
    } else {
      actionError.value = response.message || 'Action failed'
    }
  } catch (error) {
    actionError.value = error.response?.data?.message || 'Action failed'
  } finally {
    actionLoading.value = false
  }
}

// Utility functions
function getStatusColor(status) {
  switch (status) {
    case 'active':
      return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
    case 'false_positive':
      return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    default:
      return 'bg-gray-100 text-gray-800'
  }
}

function getStatusLabel(status) {
  switch (status) {
    case 'active': return 'Counterfeit'
    case 'false_positive': return 'False Positive'
    default: return status
  }
}

function formatSpeed(speedKmh) {
  if (speedKmh > 1000) return speedKmh.toFixed(0) + ' km/h'
  return speedKmh.toFixed(1) + ' km/h'
}

function formatDistance(meters) {
  if (meters > 1000) return (meters / 1000).toFixed(2) + ' km'
  return meters.toFixed(0) + ' m'
}

function formatDuration(seconds) {
  if (seconds < 60) return seconds.toFixed(0) + ' sec'
  if (seconds < 3600) return (seconds / 60).toFixed(1) + ' min'
  return (seconds / 3600).toFixed(1) + ' hours'
}

// Pagination
function goToPage(p) {
  page.value = p
  fetchDetections()
}

// Report modal functions
function openReportDetail(report) {
  selectedReport.value = report
  showReportModal.value = true
  fetchReportDetail(report.id)
}

function closeReportModal() {
  showReportModal.value = false
  selectedReport.value = null
  reportDetail.value = null
  lightboxOpen.value = false
}

function getPhotoUrls(report) {
  if (!report) return []
  const photos = report.photos
  if (!photos) return []
  if (Array.isArray(photos)) return photos
  try {
    const parsed = JSON.parse(photos)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

// Lightbox
function openLightbox(photos, index) {
  lightboxPhotos.value = photos
  lightboxIndex.value = index
  lightboxOpen.value = true
}

function closeLightbox() {
  lightboxOpen.value = false
}

function lightboxPrev() {
  if (lightboxIndex.value > 0) lightboxIndex.value--
}

function lightboxNext() {
  if (lightboxIndex.value < lightboxPhotos.value.length - 1) lightboxIndex.value++
}

// Reports search debounce
let reportsSearchTimeout = null
function onReportsSearch() {
  clearTimeout(reportsSearchTimeout)
  reportsSearchTimeout = setTimeout(() => {
    reportsPage.value = 1
    fetchReports()
  }, 300)
}

function goToReportsPage(p) {
  reportsPage.value = p
  fetchReports()
}

// Filter change handlers
function onFilterChange() {
  page.value = 1
  fetchDetections()
}

// Search debounce
let searchTimeout = null
function onSearch() {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    page.value = 1
    fetchDetections()
  }, 300)
}

// Tab change handler
function onTabChange(tab) {
  activeTab.value = tab
  if (tab === 'settings') {
    fetchSettings()
  } else if (tab === 'reports') {
    fetchReports()
    fetchReportStats()
  }
}

// Initialize
onMounted(() => {
  fetchDetections()
  fetchStats()
})

onUnmounted(() => {
  destroyMap()
  if (searchTimeout) clearTimeout(searchTimeout)
  if (reportsSearchTimeout) clearTimeout(reportsSearchTimeout)
})
</script>

<template>
  <div>
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-4 sm:mb-0">Counterfeit Detection</h1>
      <div class="flex items-center gap-2">
        <div class="flex rounded-lg border border-gray-300 dark:border-gray-600 overflow-hidden">
          <button
            v-for="preset in [
              { key: 'this_month', label: 'This Month' },
              { key: 'last_month', label: 'Last Month' },
              { key: 'last_7_days', label: '7 Days' },
              { key: 'last_30_days', label: '30 Days' },
              { key: 'custom', label: 'Custom' }
            ]"
            :key="preset.key"
            @click="applyPreset(preset.key)"
            :class="[
              'px-3 py-1.5 text-sm font-medium transition-colors',
              datePreset === preset.key
                ? 'bg-[#18181b] dark:bg-[#27272a] text-white'
                : 'bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
            ]"
          >
            {{ preset.label }}
          </button>
        </div>
        <template v-if="datePreset === 'custom'">
          <input
            type="date"
            v-model="dateFrom"
            class="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
          />
          <span class="text-gray-400">-</span>
          <input
            type="date"
            v-model="dateTo"
            class="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
          />
        </template>
      </div>
    </div>

    <!-- Stats Cards -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
      <Card class="p-4">
        <div class="text-sm text-gray-500 dark:text-gray-400">Counterfeit</div>
        <div class="text-2xl font-bold text-red-600 dark:text-red-400">{{ stats.active }}</div>
      </Card>
      <Card class="p-4">
        <div class="text-sm text-gray-500 dark:text-gray-400">False Positive</div>
        <div class="text-2xl font-bold text-gray-600 dark:text-gray-400">{{ stats.false_positive }}</div>
      </Card>
      <Card class="p-4">
        <div class="text-sm text-gray-500 dark:text-gray-400">Total</div>
        <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ stats.total }}</div>
      </Card>
      <Card class="p-4">
        <div class="text-sm text-gray-500 dark:text-gray-400">Last 7 Days</div>
        <div class="text-2xl font-bold text-orange-600 dark:text-orange-400">{{ stats.recent_7_days }}</div>
      </Card>
    </div>

    <!-- Tabs -->
    <div class="flex border-b border-gray-200 dark:border-gray-700 mb-6">
      <button
        @click="onTabChange('detections')"
        :class="[
          'px-4 py-2 font-medium text-sm border-b-2 transition-colors',
          activeTab === 'detections'
            ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
            : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
        ]"
      >
        Detections
        <span v-if="stats.active > 0" class="ml-2 px-2 py-0.5 text-xs rounded-full bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400">
          {{ stats.active }}
        </span>
      </button>
      <button
        @click="onTabChange('reports')"
        :class="[
          'px-4 py-2 font-medium text-sm border-b-2 transition-colors',
          activeTab === 'reports'
            ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
            : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
        ]"
      >
        Reports
        <span v-if="reportStats.total > 0" class="ml-2 px-2 py-0.5 text-xs rounded-full bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400">
          {{ reportStats.total }}
        </span>
      </button>
      <button
        @click="onTabChange('settings')"
        :class="[
          'px-4 py-2 font-medium text-sm border-b-2 transition-colors',
          activeTab === 'settings'
            ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
            : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
        ]"
      >
        Settings
      </button>
    </div>

    <!-- Detections Tab -->
    <div v-if="activeTab === 'detections'">
      <!-- Filters -->
      <div class="flex flex-wrap gap-4 mb-6">
        <select
          v-model="statusFilter"
          @change="onFilterChange"
          class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
        >
          <option value="">All Status</option>
          <option value="active">Counterfeit</option>
          <option value="false_positive">False Positive</option>
        </select>

        <Input
          v-model="searchQuery"
          @input="onSearch"
          placeholder="Search QR code or product..."
          class="w-64"
        />
      </div>

      <!-- Loading -->
      <div v-if="loading" class="text-center py-12 text-gray-500 dark:text-gray-400">
        Loading...
      </div>

      <!-- Detection List -->
      <div v-else class="space-y-4">
        <Card
          v-for="detection in detections"
          :key="detection.id"
          class="p-4 cursor-pointer hover:border-zinc-300 dark:hover:border-zinc-600 transition-colors"
          @click="openDetail(detection)"
        >
          <div class="flex justify-between items-start">
            <div class="flex-1">
              <div class="flex items-center gap-2 mb-2">
                <span :class="['px-2 py-0.5 text-xs font-medium rounded-full', getStatusColor(detection.status)]">
                  {{ getStatusLabel(detection.status) }}
                </span>
                <span class="text-sm text-gray-500 dark:text-gray-400">
                  {{ formatDateTime(detection.created_at) }}
                </span>
              </div>

              <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">
                {{ detection.qr_code?.batch?.product?.product_name || 'Unknown Product' }}
              </h3>

              <div class="text-sm text-gray-600 dark:text-gray-300 space-y-1">
                <div>
                  <span class="text-gray-500">QR Code:</span>
                  {{ detection.qr_code?.qr_code || 'N/A' }}
                </div>
                <div>
                  <span class="text-gray-500">Reason:</span>
                  {{ detection.detection_reason?.split(' | ')[0] || 'N/A' }}
                </div>
                <div>
                  <span class="text-gray-500">Interactions:</span>
                  {{ detection.total_interactions_count }}
                </div>
              </div>
            </div>

            <div class="text-right">
              <div class="text-2xl font-bold text-red-600 dark:text-red-400">
                {{ detection.total_interactions_count }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">scans</div>
            </div>
          </div>
        </Card>
      </div>

      <!-- Empty state -->
      <div v-if="!loading && detections.length === 0" class="text-center py-12 text-gray-500 dark:text-gray-400">
        <svg class="w-16 h-16 mx-auto mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
        </svg>
        <p>No counterfeit detections found</p>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex justify-center gap-2 mt-6">
        <Button
          variant="outline"
          size="sm"
          :disabled="page === 1"
          @click="goToPage(page - 1)"
        >
          Previous
        </Button>
        <span class="px-4 py-2 text-sm text-gray-600 dark:text-gray-400">
          Page {{ page }} of {{ totalPages }}
        </span>
        <Button
          variant="outline"
          size="sm"
          :disabled="page === totalPages"
          @click="goToPage(page + 1)"
        >
          Next
        </Button>
      </div>
    </div>

    <!-- Reports Tab -->
    <div v-if="activeTab === 'reports'">
      <!-- Report Stats Cards -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <Card class="p-4">
          <div class="text-sm text-gray-500 dark:text-gray-400">Total Reports</div>
          <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ reportStats.total }}</div>
        </Card>
        <Card class="p-4">
          <div class="text-sm text-gray-500 dark:text-gray-400">Today</div>
          <div class="text-2xl font-bold text-orange-600 dark:text-orange-400">{{ reportStats.today }}</div>
        </Card>
        <Card class="p-4">
          <div class="text-sm text-gray-500 dark:text-gray-400">This Week</div>
          <div class="text-2xl font-bold text-orange-500 dark:text-orange-400">{{ reportStats.this_week }}</div>
        </Card>
        <Card class="p-4">
          <div class="text-sm text-gray-500 dark:text-gray-400">This Month</div>
          <div class="text-2xl font-bold text-yellow-600 dark:text-yellow-400">{{ reportStats.this_month }}</div>
        </Card>
      </div>

      <!-- Search -->
      <div class="flex flex-wrap gap-4 mb-6">
        <Input
          v-model="reportsSearchQuery"
          @input="onReportsSearch"
          placeholder="Search by store name..."
          class="w-64"
        />
      </div>

      <!-- Loading -->
      <div v-if="reportsLoading" class="text-center py-12 text-gray-500 dark:text-gray-400">
        Loading...
      </div>

      <!-- Report List -->
      <div v-else-if="reports.length > 0" class="space-y-4">
        <Card
          v-for="report in reports"
          :key="report.id"
          class="p-4 cursor-pointer hover:border-zinc-300 dark:hover:border-zinc-600 transition-colors"
          @click="openReportDetail(report)"
        >
          <div class="flex justify-between items-start">
            <div class="flex-1">
              <div class="flex items-center gap-2 mb-2">
                <span class="text-sm text-gray-500 dark:text-gray-400">
                  {{ formatDateTime(report.created_at) }}
                </span>
              </div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white mb-1">
                {{ report.qr_code?.batch?.product?.product_name || 'Unknown Product' }}
              </h3>
              <div class="text-sm text-gray-600 dark:text-gray-300 space-y-1">
                <div v-if="report.store_name">
                  <span class="text-gray-500">Store:</span>
                  {{ report.store_name }}
                </div>
                <div>
                  <span class="text-gray-500">QR Code:</span>
                  {{ report.qr_code?.qr_code || 'N/A' }}
                </div>
                <div v-if="report.description">
                  <span class="text-gray-500">Description:</span>
                  {{ report.description.length > 100 ? report.description.slice(0, 100) + '...' : report.description }}
                </div>
              </div>
            </div>
            <div v-if="getPhotoUrls(report).length > 0" class="text-right ml-4 flex-shrink-0">
              <div class="flex items-center gap-1 text-sm text-gray-500 dark:text-gray-400">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                {{ getPhotoUrls(report).length }}
              </div>
            </div>
          </div>
        </Card>
      </div>

      <!-- Empty state -->
      <div v-else class="text-center py-12 text-gray-500 dark:text-gray-400">
        <svg class="w-16 h-16 mx-auto mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
        </svg>
        <p>No counterfeit reports submitted yet</p>
      </div>

      <!-- Pagination -->
      <div v-if="reportsTotalPages > 1" class="flex justify-center gap-2 mt-6">
        <Button
          variant="outline"
          size="sm"
          :disabled="reportsPage === 1"
          @click="goToReportsPage(reportsPage - 1)"
        >
          Previous
        </Button>
        <span class="px-4 py-2 text-sm text-gray-600 dark:text-gray-400">
          Page {{ reportsPage }} of {{ reportsTotalPages }}
        </span>
        <Button
          variant="outline"
          size="sm"
          :disabled="reportsPage === reportsTotalPages"
          @click="goToReportsPage(reportsPage + 1)"
        >
          Next
        </Button>
      </div>
    </div>

    <!-- Settings Tab -->
    <div v-if="activeTab === 'settings'">
      <Card class="p-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">Counterfeit Detection Settings</h2>
        <p class="text-sm text-gray-500 dark:text-gray-400 mb-1">These are <strong class="text-gray-700 dark:text-gray-300">global default settings</strong> applied to all products.</p>
        <p class="text-sm text-gray-500 dark:text-gray-400 mb-6">Each product may have its own threshold override depending on your needs — configure per-product settings in the product detail page.</p>

        <div v-if="loadingSettings" class="text-center py-8 text-gray-500">Loading...</div>

        <div v-else class="space-y-6">
          <!-- Scan Thresholds -->
          <div>
            <h3 class="text-md font-medium text-gray-900 dark:text-white mb-4">Scan Count Thresholds</h3>
            <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
              Set maximum number of scans before triggering a counterfeit alert. Set to 0 to disable.
            </p>

            <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">QC Scan Max</label>
                <Input v-model.number="settings.qc_scan_max" type="number" min="0" placeholder="0 = disabled" />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Warehouse Scan Max</label>
                <Input v-model.number="settings.warehouse_scan_max" type="number" min="0" placeholder="0 = disabled" />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">End User Scan Max</label>
                <Input v-model.number="settings.end_user_scan_max" type="number" min="0" placeholder="0 = disabled" />
              </div>
            </div>
          </div>

          <!-- Velocity Detection -->
          <div class="border-t border-gray-200 dark:border-gray-700 pt-6">
            <h3 class="text-md font-medium text-gray-900 dark:text-white mb-4">Velocity Detection (Impossible Travel)</h3>
            <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
              Detect when a QR code is scanned at two locations that would require impossible travel speed.
            </p>

            <div class="space-y-4">
              <div class="flex items-center gap-3">
                <input
                  type="checkbox"
                  id="velocity_check"
                  v-model="settings.velocity_check_enabled"
                  class="w-4 h-4 text-zinc-600 border-gray-300 rounded focus:ring-[#27272a]"
                />
                <label for="velocity_check" class="text-sm text-gray-700 dark:text-gray-300">
                  Enable velocity check
                </label>
              </div>

              <div v-if="settings.velocity_check_enabled" class="ml-7">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Max Speed (km/h)</label>
                <Input v-model.number="settings.max_speed_kmh" type="number" min="100" max="5000" class="w-48" />
                <p class="text-xs text-gray-500 mt-1">Scans faster than this speed will be flagged (default: 1000 km/h)</p>
              </div>
            </div>
          </div>

          <!-- Alert Settings -->
          <div class="border-t border-gray-200 dark:border-gray-700 pt-6">
            <h3 class="text-md font-medium text-gray-900 dark:text-white mb-4">Alert Settings</h3>

            <div class="space-y-4">
              <div class="flex items-center gap-3">
                <input
                  type="checkbox"
                  id="alert_on_detection"
                  v-model="settings.alert_on_detection"
                  class="w-4 h-4 text-zinc-600 border-gray-300 rounded focus:ring-[#27272a]"
                />
                <label for="alert_on_detection" class="text-sm text-gray-700 dark:text-gray-300">
                  Show alert notification when counterfeit is detected
                </label>
              </div>

              <div class="flex items-center gap-3">
                <input
                  type="checkbox"
                  id="auto_flag"
                  v-model="settings.auto_flag_suspicious"
                  class="w-4 h-4 text-zinc-600 border-gray-300 rounded focus:ring-[#27272a]"
                />
                <label for="auto_flag" class="text-sm text-gray-700 dark:text-gray-300">
                  Automatically flag suspicious QR codes
                </label>
              </div>
            </div>
          </div>

          <!-- Error/Success Messages -->
          <div v-if="settingsError" class="p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-md text-sm">
            {{ settingsError }}
          </div>
          <div v-if="settingsSuccess" class="p-3 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 rounded-md text-sm">
            {{ settingsSuccess }}
          </div>

          <!-- Save Button -->
          <div class="flex justify-end pt-4">
            <Button @click="saveSettings" :disabled="savingSettings">
              {{ savingSettings ? 'Saving...' : 'Save Settings' }}
            </Button>
          </div>
        </div>
      </Card>
    </div>

    <!-- Detail Modal -->
    <div v-if="showDetailModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-gray-800 rounded-lg w-full max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
        <!-- Header -->
        <div class="flex justify-between items-center p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Detection Detail</h2>
          <button @click="closeDetailModal" class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Content -->
        <div class="flex-1 overflow-y-auto p-4">
          <div v-if="loadingDetail" class="text-center py-12 text-gray-500">Loading...</div>

          <div v-else-if="detectionDetail" class="space-y-6">
            <!-- Detection Info -->
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Card class="p-4">
                <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">Product Information</h3>
                <div class="space-y-2 text-sm">
                  <div>
                    <span class="text-gray-500">Product:</span>
                    <span class="ml-2 text-gray-900 dark:text-white">{{ detectionDetail.qr_code?.batch?.product?.product_name }}</span>
                  </div>
                  <div>
                    <span class="text-gray-500">QR Code:</span>
                    <span class="ml-2 font-mono text-gray-900 dark:text-white">{{ detectionDetail.qr_code?.qr_code }}</span>
                  </div>
                  <div>
                    <span class="text-gray-500">Batch:</span>
                    <span class="ml-2 text-gray-900 dark:text-white">{{ detectionDetail.qr_code?.batch?.batch_code }}</span>
                  </div>
                </div>
              </Card>

              <Card class="p-4">
                <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">Detection Info</h3>
                <div class="space-y-2 text-sm">
                  <div>
                    <span class="text-gray-500">Status:</span>
                    <span :class="['ml-2 px-2 py-0.5 text-xs font-medium rounded-full', getStatusColor(detectionDetail.status)]">
                      {{ getStatusLabel(detectionDetail.status) }}
                    </span>
                  </div>
                  <div>
                    <span class="text-gray-500">Reason:</span>
                    <span class="ml-2 text-gray-900 dark:text-white">{{ detectionDetail.detection_reason?.split(' | ')[0] }}</span>
                  </div>
                  <div>
                    <span class="text-gray-500">Total Scans:</span>
                    <span class="ml-2 text-gray-900 dark:text-white">{{ detectionDetail.total_interactions_count }}</span>
                  </div>
                  <div>
                    <span class="text-gray-500">First Scan:</span>
                    <span class="ml-2 text-gray-900 dark:text-white">{{ formatDateTime(detectionDetail.first_interaction_at) }}</span>
                  </div>
                  <div>
                    <span class="text-gray-500">Last Scan:</span>
                    <span class="ml-2 text-gray-900 dark:text-white">{{ formatDateTime(detectionDetail.last_interaction_at) }}</span>
                  </div>
                </div>
              </Card>
            </div>

            <!-- Map -->
            <Card class="p-4">
              <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">Scan Locations Map</h3>
              <div
                ref="mapContainer"
                class="h-64 rounded-lg border border-gray-200 dark:border-gray-700"
              ></div>
              <div v-if="geopoints.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
                No geolocation data available
              </div>
              <!-- Map Legend -->
              <div v-else class="flex items-center justify-center gap-4 mt-3 text-xs text-gray-600 dark:text-gray-400">
                <div class="flex items-center gap-1">
                  <span class="w-3 h-3 rounded-full bg-green-500"></span>
                  <span>First Scan</span>
                </div>
                <div class="flex items-center gap-1">
                  <span class="w-3 h-3 rounded-full bg-amber-500"></span>
                  <span>Normal</span>
                </div>
                <div class="flex items-center gap-1">
                  <span class="w-3 h-3 rounded-full bg-zinc-500"></span>
                  <span>Impossible Travel</span>
                </div>
                <div class="flex items-center gap-1">
                  <span class="w-3 h-3 rounded-full bg-red-500"></span>
                  <span>Last Scan</span>
                </div>
              </div>
            </Card>

            <!-- Velocity Data (Impossible Travel) -->
            <Card v-if="velocityData.length > 0" class="p-4">
              <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">Velocity Analysis</h3>
              <div class="overflow-x-auto">
                <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                  <thead>
                    <tr>
                      <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Segment</th>
                      <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Distance</th>
                      <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Time</th>
                      <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Speed</th>
                      <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
                    <tr v-for="v in velocityData" :key="`velocity-${v.interaction_index}`" :class="v.is_impossible ? 'bg-orange-50 dark:bg-orange-900/20' : ''">
                      <td class="px-3 py-2 text-sm text-gray-900 dark:text-white">
                        Scan {{ v.interaction_index }} → {{ v.interaction_index + 1 }}
                      </td>
                      <td class="px-3 py-2 text-sm text-gray-600 dark:text-gray-300">
                        {{ formatDistance(v.distance_meters) }}
                      </td>
                      <td class="px-3 py-2 text-sm text-gray-600 dark:text-gray-300">
                        {{ formatDuration(v.time_seconds) }}
                      </td>
                      <td class="px-3 py-2 text-sm text-gray-600 dark:text-gray-300">
                        {{ formatSpeed(v.speed_kmh) }}
                      </td>
                      <td class="px-3 py-2 text-sm">
                        <span v-if="v.is_impossible" class="px-2 py-1 text-xs font-medium rounded-full bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400">
                          IMPOSSIBLE
                        </span>
                        <span v-else class="px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
                          Normal
                        </span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </Card>

            <!-- Interaction History -->
            <Card class="p-4">
              <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">Interaction History</h3>
              <div class="overflow-x-auto">
                <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                  <thead>
                    <tr>
                      <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">#</th>
                      <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Time</th>
                      <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Category</th>
                      <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Subcategory</th>
                      <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
                    <tr v-for="(interaction, index) in interactions" :key="interaction.id">
                      <td class="px-3 py-2 text-sm text-gray-900 dark:text-white">{{ index + 1 }}</td>
                      <td class="px-3 py-2 text-sm text-gray-600 dark:text-gray-300">{{ formatDateTime(interaction.created_at) }}</td>
                      <td class="px-3 py-2 text-sm text-gray-600 dark:text-gray-300">{{ interaction.interaction_category }}</td>
                      <td class="px-3 py-2 text-sm text-gray-600 dark:text-gray-300">{{ interaction.interaction_subcategory }}</td>
                      <td class="px-3 py-2 text-sm">
                        <span :class="[
                          'px-2 py-1 text-xs font-medium rounded-full',
                          interaction.interaction_status === 'success'
                            ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                            : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
                        ]">
                          {{ interaction.interaction_status }}
                        </span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </Card>
          </div>
        </div>

        <!-- Footer Actions -->
        <div class="p-4 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
          <Button variant="outline" @click="closeDetailModal">
            Close
          </Button>
          <template v-if="detectionDetail && detectionDetail.status === 'active'">
            <Button @click="openFalsePositiveModal" class="bg-zinc-600 hover:bg-zinc-700 text-white dark:bg-zinc-500 dark:hover:bg-zinc-600">
              Mark as False Positive
            </Button>
          </template>
        </div>
      </div>
    </div>

    <!-- Action Modal (False Positive / Override Threshold) -->
    <div v-if="showActionModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Mark as False Positive
        </h2>

        <div v-if="actionError" class="mb-4 p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-md text-sm">
          {{ actionError }}
        </div>

        <div class="space-y-4">
          <div class="space-y-4">
            <!-- Current Info -->
            <div v-if="detectionDetail" class="p-3 bg-gray-50 dark:bg-gray-700/50 rounded-md text-sm space-y-1">
              <div class="flex justify-between">
                <span class="text-gray-500 dark:text-gray-400">Scan Count:</span>
                <span class="font-medium text-gray-900 dark:text-white">{{ detectionDetail.total_interactions_count }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-500 dark:text-gray-400">QR Code:</span>
                <span class="font-mono text-xs text-gray-900 dark:text-white">{{ detectionDetail.qr_code?.qr_code || 'N/A' }}</span>
              </div>
              <div v-if="detectionDetail.qr_code?.batch?.product" class="flex justify-between">
                <span class="text-gray-500 dark:text-gray-400">Product:</span>
                <span class="text-gray-900 dark:text-white">{{ detectionDetail.qr_code.batch.product.product_name }}</span>
              </div>
            </div>

            <!-- Override Level -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Apply Override To</label>
              <div class="space-y-2">
                <label class="flex items-center gap-2 cursor-pointer">
                  <input type="radio" v-model="actionForm.level" value="qr" class="text-zinc-500 focus:ring-zinc-400" />
                  <span class="text-sm text-gray-700 dark:text-gray-300">This QR only</span>
                </label>
                <label v-if="detectionDetail?.qr_code?.batch" class="flex items-center gap-2 cursor-pointer">
                  <input type="radio" v-model="actionForm.level" value="batch" class="text-zinc-500 focus:ring-zinc-400" />
                  <span class="text-sm text-gray-700 dark:text-gray-300">
                    Batch: {{ detectionDetail.qr_code.batch.batch_name || detectionDetail.qr_code.batch.batch_code }}
                  </span>
                </label>
                <label v-if="detectionDetail?.qr_code?.batch?.product" class="flex items-center gap-2 cursor-pointer">
                  <input type="radio" v-model="actionForm.level" value="product" class="text-zinc-500 focus:ring-zinc-400" />
                  <span class="text-sm text-gray-700 dark:text-gray-300">
                    Product: {{ detectionDetail.qr_code.batch.product.product_name }}
                  </span>
                </label>
              </div>
            </div>

            <!-- New Threshold -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">New Threshold *</label>
              <input
                v-model.number="actionForm.new_threshold"
                type="number"
                min="1"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
              />
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                Must be greater than current scan count ({{ detectionDetail?.total_interactions_count || 0 }})
              </p>
            </div>

            <!-- Reason -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Reason *</label>
              <textarea
                v-model="actionForm.reason"
                rows="3"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
                placeholder="e.g., Product displayed in store, warehouse scanning test..."
              />
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-2 mt-6">
          <Button variant="outline" @click="showActionModal = false" :disabled="actionLoading">Cancel</Button>
          <Button
            @click="submitAction"
            :disabled="actionLoading"
            class="bg-zinc-600 hover:bg-zinc-700 text-white dark:bg-zinc-500 dark:hover:bg-zinc-600"
          >
            {{ actionLoading ? 'Processing...' : 'Mark as False Positive' }}
          </Button>
        </div>
      </div>
    </div>

    <!-- Report Detail Modal -->
    <div v-if="showReportModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-gray-800 rounded-lg w-full max-w-3xl max-h-[90vh] overflow-hidden flex flex-col">
        <div class="flex justify-between items-center p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Counterfeit Report Detail</h2>
          <button @click="closeReportModal" class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-4">
          <div v-if="loadingReportDetail" class="text-center py-12 text-gray-500">Loading...</div>

          <div v-else-if="reportDetail" class="space-y-6">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Card class="p-4">
                <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">Product Information</h3>
                <div class="space-y-2 text-sm">
                  <div>
                    <span class="text-gray-500">Product:</span>
                    <span class="ml-2 text-gray-900 dark:text-white">{{ reportDetail.qr_code?.batch?.product?.product_name || 'N/A' }}</span>
                  </div>
                  <div>
                    <span class="text-gray-500">QR Code:</span>
                    <span class="ml-2 font-mono text-gray-900 dark:text-white">{{ reportDetail.qr_code?.qr_code || 'N/A' }}</span>
                  </div>
                  <div>
                    <span class="text-gray-500">Batch:</span>
                    <span class="ml-2 text-gray-900 dark:text-white">{{ reportDetail.qr_code?.batch?.batch_code || 'N/A' }}</span>
                  </div>
                </div>
              </Card>

              <Card class="p-4">
                <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">Report Information</h3>
                <div class="space-y-2 text-sm">
                  <div>
                    <span class="text-gray-500">Submitted:</span>
                    <span class="ml-2 text-gray-900 dark:text-white">{{ formatDateTime(reportDetail.created_at) }}</span>
                  </div>
                  <div v-if="reportDetail.store_name">
                    <span class="text-gray-500">Store Name:</span>
                    <span class="ml-2 text-gray-900 dark:text-white">{{ reportDetail.store_name }}</span>
                  </div>
                  <div v-if="reportDetail.province">
                    <span class="text-gray-500">Province:</span>
                    <span class="ml-2 text-gray-900 dark:text-white">{{ reportDetail.province }}</span>
                  </div>
                  <div v-if="reportDetail.city">
                    <span class="text-gray-500">City:</span>
                    <span class="ml-2 text-gray-900 dark:text-white">{{ reportDetail.city }}</span>
                  </div>
                  <div>
                    <span class="text-gray-500">Reporter IP:</span>
                    <span class="ml-2 font-mono text-gray-900 dark:text-white">{{ reportDetail.ip_address || 'N/A' }}</span>
                  </div>
                </div>
              </Card>
            </div>

            <Card v-if="reportDetail.description" class="p-4">
              <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">Description</h3>
              <p class="text-sm text-gray-900 dark:text-white whitespace-pre-wrap">{{ reportDetail.description }}</p>
            </Card>

            <Card v-if="getPhotoUrls(reportDetail).length > 0" class="p-4">
              <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">
                Evidence Photos ({{ getPhotoUrls(reportDetail).length }})
              </h3>
              <div class="grid grid-cols-3 gap-2">
                <div
                  v-for="(photoUrl, index) in getPhotoUrls(reportDetail)"
                  :key="index"
                  class="relative cursor-pointer group"
                  @click="openLightbox(getPhotoUrls(reportDetail), index)"
                >
                  <img
                    :src="photoUrl"
                    :alt="`Evidence photo ${index + 1}`"
                    class="w-full aspect-square object-cover rounded-lg border border-gray-200 dark:border-gray-600 group-hover:opacity-80 transition-opacity"
                  />
                </div>
              </div>
            </Card>

            <Card v-if="reportDetail.counterfeit_detection" class="p-4">
              <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">Linked Detection</h3>
              <div class="space-y-2 text-sm">
                <div>
                  <span class="text-gray-500">Status:</span>
                  <span :class="['ml-2 px-2 py-0.5 text-xs font-medium rounded-full', getStatusColor(reportDetail.counterfeit_detection.status)]">
                    {{ getStatusLabel(reportDetail.counterfeit_detection.status) }}
                  </span>
                </div>
                <div v-if="reportDetail.counterfeit_detection.detection_reason">
                  <span class="text-gray-500">Reason:</span>
                  <span class="ml-2 text-gray-900 dark:text-white">{{ reportDetail.counterfeit_detection.detection_reason.split(' | ')[0] }}</span>
                </div>
              </div>
            </Card>
          </div>
        </div>

        <div class="p-4 border-t border-gray-200 dark:border-gray-700 flex justify-end">
          <Button variant="outline" @click="closeReportModal">Close</Button>
        </div>
      </div>
    </div>

    <!-- Photo Lightbox -->
    <div
      v-if="lightboxOpen"
      class="fixed inset-0 bg-black bg-opacity-90 flex items-center justify-center z-[60]"
      @click.self="closeLightbox"
    >
      <button @click="closeLightbox" class="absolute top-4 right-4 text-white hover:text-gray-300">
        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>

      <button
        v-if="lightboxIndex > 0"
        @click="lightboxPrev"
        class="absolute left-4 text-white hover:text-gray-300 p-2"
      >
        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>

      <img
        :src="lightboxPhotos[lightboxIndex]"
        class="max-h-[85vh] max-w-[90vw] object-contain rounded-lg"
        :alt="`Photo ${lightboxIndex + 1} of ${lightboxPhotos.length}`"
      />

      <button
        v-if="lightboxIndex < lightboxPhotos.length - 1"
        @click="lightboxNext"
        class="absolute right-4 text-white hover:text-gray-300 p-2"
      >
        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
      </button>

      <div class="absolute bottom-4 text-white text-sm">
        {{ lightboxIndex + 1 }} / {{ lightboxPhotos.length }}
      </div>
    </div>
  </div>
</template>

<style>
/* Fix Leaflet icons */
.leaflet-default-icon-path {
  background-image: url(https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon.png);
}

/* Dark mode support for map */
.dark .leaflet-tile {
  filter: invert(1) hue-rotate(180deg) brightness(0.95) contrast(0.9);
}

.dark .leaflet-container {
  background: #1f2937;
}
</style>
