<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useAPI } from '@/composables/useAPI'
import { useDateTime } from '@/composables/useDateTime'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import ExportTimezoneModal from '@/components/ui/ExportTimezoneModal.vue'
import GeofenceViolationMap from '@/components/GeofenceViolationMap.vue'
import { MapPin, Download, AlertTriangle, Shield, Navigation, TrendingUp, TrendingDown, Percent, Ruler, Package, Search, ChevronDown } from 'lucide-vue-next'
import { onClickOutside } from '@vueuse/core'
import { getPagination } from '@/lib/pagination'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Filler,
  Title,
  Tooltip,
  Legend
} from 'chart.js'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Filler, Title, Tooltip, Legend)

const route = useRoute()
const { get, getAuthHeaders } = useAPI()
const { formatDateTime } = useDateTime()

const loading = ref(false)
const violations = ref([])
const page = ref(1)
const limit = ref(20)
const totalPages = ref(0)

// Date helpers
const formatDateLocal = (date) => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
const today = new Date()
const startOfMonth = new Date(today.getFullYear(), today.getMonth(), 1)

// Filters
const severityFilter = ref('')
const legacyBatchId = ref(route.query.batch_id || '')
const selectedArea = ref(null)
const areas = ref([])
const areaSearchQuery = ref('')
const showAreaDropdown = ref(false)
const areaDropdownRef = ref(null)
onClickOutside(areaDropdownRef, () => { showAreaDropdown.value = false })
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

// Stats
const stats = ref({
  by_severity: [],
  total_violations: 0,
  top_batches: []
})
const loadingStats = ref(false)
const statsDateFilter = ref({ comparison_label: '', previous_from: '', previous_to: '' })

// Area filter computed
const areaGroups = computed(() => {
  const q = areaSearchQuery.value.toLowerCase().trim()
  const grouped = {}
  for (const area of areas.value) {
    if (q && !area.product_name.toLowerCase().includes(q) && !area.geofence_label.toLowerCase().includes(q)) continue
    if (!grouped[area.product_id]) {
      grouped[area.product_id] = { product_id: area.product_id, product_name: area.product_name, areas: [] }
    }
    grouped[area.product_id].areas.push(area)
  }
  return Object.values(grouped).sort((a, b) => a.product_name.localeCompare(b.product_name))
})
const selectedAreaLabel = computed(() => {
  if (!selectedArea.value) return 'All Areas'
  if (!selectedArea.value.geofence_label) return selectedArea.value.product_name
  return `${selectedArea.value.product_name} / ${selectedArea.value.geofence_label}`
})

// Export state
const exporting = ref(false)

// Map data
const mapViolations = ref([])
const mapZones = ref([])
const loadingMap = ref(false)

// Analytics
const analytics = ref(null)
const previousAnalytics = ref(null)
const loadingAnalytics = ref(false)
const dateFilter = ref({
  from: '', to: '', previous_from: '', previous_to: '',
  comparison_label: 'vs last month', preset: 'this_month'
})

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

function getSeverityIcon(severity) {
  switch (severity) {
    case 'critical':
    case 'high':
      return 'text-red-500'
    case 'medium':
      return 'text-amber-500'
    default:
      return 'text-gray-400'
  }
}

function applyAreaFilter(params) {
  if (legacyBatchId.value) {
    params.batch_id = legacyBatchId.value
  } else if (selectedArea.value) {
    params.product_id = selectedArea.value.product_id
    if (selectedArea.value.geofence_label) {
      params.geofence_label = selectedArea.value.geofence_label
    }
  }
}

async function fetchViolations() {
  loading.value = true
  try {
    const params = {
      page: page.value,
      limit: limit.value,
    }
    if (severityFilter.value) params.severity = severityFilter.value
    applyAreaFilter(params)
    if (dateFrom.value) params.from = dateFrom.value
    if (dateTo.value) params.to = dateTo.value

    const response = await get('/tenant/geofence/violations', params)
    if (response.success) {
      violations.value = response.data?.violations || []
      totalPages.value = getPagination(response.data).totalPages
    }
  } catch (error) {
    console.error('Failed to fetch violations:', error)
  } finally {
    loading.value = false
  }
}

async function fetchStats() {
  loadingStats.value = true
  try {
    const params = { preset: datePreset.value }
    if (severityFilter.value) params.severity = severityFilter.value
    applyAreaFilter(params)
    if (dateFrom.value) params.from = dateFrom.value
    if (dateTo.value) params.to = dateTo.value

    const response = await get('/tenant/geofence/stats', params)
    if (response.success) {
      stats.value = response.data || {}
      statsDateFilter.value = response.data?.date_filter || {}
    }
  } catch (error) {
    console.error('Failed to fetch stats:', error)
  } finally {
    loadingStats.value = false
  }
}

async function fetchAreas() {
  try {
    const response = await get('/tenant/geofence/areas')
    if (response.success) {
      areas.value = response.data?.areas || []
    }
  } catch (e) { /* ignore */ }
}

function selectArea(area) {
  selectedArea.value = area
  legacyBatchId.value = ''
  showAreaDropdown.value = false
  areaSearchQuery.value = ''
  onFilterChange()
}

async function fetchMapData() {
  loadingMap.value = true
  try {
    const params = {}
    if (severityFilter.value) params.severity = severityFilter.value
    applyAreaFilter(params)
    if (dateFrom.value) params.from = dateFrom.value
    if (dateTo.value) params.to = dateTo.value

    const response = await get('/tenant/geofence/map-data', params)
    if (response.success) {
      mapViolations.value = response.data?.violations || []
      mapZones.value = response.data?.zones || []
    }
  } catch (error) {
    console.error('Failed to fetch map data:', error)
  } finally {
    loadingMap.value = false
  }
}

async function fetchAnalytics() {
  loadingAnalytics.value = true
  try {
    const params = { preset: datePreset.value }
    if (severityFilter.value) params.severity = severityFilter.value
    applyAreaFilter(params)
    if (dateFrom.value) params.from = dateFrom.value
    if (dateTo.value) params.to = dateTo.value

    const response = await get('/tenant/geofence/analytics', params)
    if (response.success) {
      analytics.value = response.data
      previousAnalytics.value = response.data?.previous_stats || null
      if (response.data?.date_filter) {
        dateFilter.value = response.data.date_filter
      }
    }
  } catch (error) {
    // Analytics are supplementary — the page still works without them
  } finally {
    loadingAnalytics.value = false
  }
}

const showExportModal = ref(false)

async function exportViolations(tz) {
  showExportModal.value = false
  exporting.value = true
  try {
    const apiBase = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
    const params = new URLSearchParams()
    if (severityFilter.value) params.set('severity', severityFilter.value)
    if (legacyBatchId.value) {
      params.set('batch_id', legacyBatchId.value)
    } else if (selectedArea.value) {
      params.set('product_id', selectedArea.value.product_id)
      if (selectedArea.value.geofence_label) params.set('geofence_label', selectedArea.value.geofence_label)
    }
    if (dateFrom.value) params.set('from', dateFrom.value)
    if (dateTo.value) params.set('to', dateTo.value)
    if (tz) params.set('tz', tz)
    const qs = params.toString()
    const response = await fetch(`${apiBase}/tenant/geofence/violations/export${qs ? '?' + qs : ''}`, {
      method: 'GET',
      headers: getAuthHeaders(),
      credentials: 'include',
    })
    if (!response.ok) throw new Error('Export failed')
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'geofence_violations.xlsx'
    a.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Export failed:', error)
  } finally {
    exporting.value = false
  }
}

function onFilterChange() {
  page.value = 1
  fetchViolations()
  fetchStats()
  fetchMapData()
  fetchAnalytics()
}

function goToPage(p) {
  page.value = p
  fetchViolations()
}

const severityCounts = computed(() => {
  const arr = Array.isArray(stats.value.by_severity) ? stats.value.by_severity : []
  const counts = { low: 0, medium: 0, high: 0, critical: 0 }
  arr.forEach(item => {
    if (item.severity in counts) counts[item.severity] = item.count
  })
  return counts
})

const previousSeverityCounts = computed(() => {
  const arr = Array.isArray(stats.value.previous_by_severity) ? stats.value.previous_by_severity : []
  const counts = { low: 0, medium: 0, high: 0, critical: 0 }
  arr.forEach(item => {
    if (item.severity in counts) counts[item.severity] = item.count
  })
  return counts
})

const severityChange = computed(() => {
  const calc = (current, previous) => {
    if (current === 0 && previous === 0) return null
    if (previous === 0) return 100
    return ((current - previous) / previous) * 100
  }
  return {
    low: calc(severityCounts.value.low, previousSeverityCounts.value.low),
    medium: calc(severityCounts.value.medium, previousSeverityCounts.value.medium),
    high: calc(severityCounts.value.high, previousSeverityCounts.value.high),
    critical: calc(severityCounts.value.critical, previousSeverityCounts.value.critical),
  }
})

const trendChartData = computed(() => {
  if (!analytics.value?.trends?.length) return null
  const labels = analytics.value.trends.map(t => {
    const d = new Date(t.week)
    return `${d.getMonth() + 1}/${d.getDate()}`
  })
  return {
    labels,
    datasets: [{
      label: 'Violations',
      data: analytics.value.trends.map(t => t.count),
      borderColor: '#f97316',
      backgroundColor: 'rgba(249, 115, 22, 0.1)',
      fill: true,
      tension: 0.3,
      pointRadius: 3,
      pointHoverRadius: 5,
    }]
  }
})

const trendChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        title: (items) => `Week of ${items[0].label}`,
      }
    }
  },
  scales: {
    y: {
      beginAtZero: true,
      ticks: { precision: 0 },
      grid: { color: 'rgba(156, 163, 175, 0.15)' },
    },
    x: {
      grid: { display: false },
    }
  }
}

const topCityMax = computed(() => {
  if (!analytics.value?.top_cities?.length) return 1
  return analytics.value.top_cities[0]?.count || 1
})

const byProductMax = computed(() => {
  if (!analytics.value?.by_product?.length) return 1
  return analytics.value.by_product[0]?.count || 1
})

// Period comparison for analytics cards
const periodChange = computed(() => {
  if (!analytics.value || !previousAnalytics.value) {
    return { rate: null, avgKm: null, maxKm: null }
  }
  const calcChange = (current, previous) => {
    if (current === 0 && previous === 0) return null
    if (previous === 0) return 100
    return ((current - previous) / previous) * 100
  }
  return {
    rate: calcChange(analytics.value.violation_rate?.rate || 0, previousAnalytics.value.violation_rate?.rate || 0),
    avgKm: calcChange(analytics.value.distance_stats?.avg_km || 0, previousAnalytics.value.distance_stats?.avg_km || 0),
    maxKm: calcChange(analytics.value.distance_stats?.max_km || 0, previousAnalytics.value.distance_stats?.max_km || 0),
  }
})

onMounted(() => {
  fetchViolations()
  fetchStats()
  fetchAreas()
  fetchAnalytics()
  fetchMapData()
})
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Geofence Violations</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Monitor out-of-zone product scans</p>
      </div>
      <Button
        variant="outline"
        @click="showExportModal = true"
        :disabled="exporting"
      >
        <Download class="w-4 h-4 mr-2" />
        {{ exporting ? 'Exporting...' : 'Export Excel' }}
      </Button>
    </div>

    <!-- Filters -->
    <div class="flex flex-wrap items-center gap-4 mb-6">
      <select
        v-model="severityFilter"
        @change="onFilterChange"
        class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-zinc-500"
      >
        <option value="">All Severities</option>
        <option value="low">Low</option>
        <option value="medium">Medium</option>
        <option value="high">High</option>
        <option value="critical">Critical</option>
      </select>

      <!-- Area Selector (Product → Geofence Label) -->
      <div class="relative flex-1" ref="areaDropdownRef">
        <button
          @click="showAreaDropdown = !showAreaDropdown"
          class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm text-left flex items-center justify-between focus:ring-2 focus:ring-zinc-500"
        >
          <span class="truncate" :class="selectedArea ? '' : 'text-gray-500 dark:text-gray-400'">
            {{ selectedAreaLabel }}
          </span>
          <ChevronDown class="w-4 h-4 text-gray-400 shrink-0 ml-2" />
        </button>

        <div
          v-if="showAreaDropdown"
          class="absolute z-[1000] mt-1 w-full min-w-[280px] bg-white dark:bg-gray-800 rounded-md shadow-lg border border-gray-200 dark:border-gray-600 max-h-80 flex flex-col"
        >
          <div class="p-2 border-b border-gray-200 dark:border-gray-700 shrink-0">
            <div class="relative">
              <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                v-model="areaSearchQuery"
                type="text"
                placeholder="Search product or area..."
                class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white pl-8 pr-3 py-1.5 text-sm focus:ring-2 focus:ring-zinc-500 outline-none"
              />
            </div>
          </div>

          <div class="overflow-y-auto">
            <button
              @click="selectArea(null)"
              class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors"
              :class="!selectedArea ? 'bg-zinc-50 dark:bg-zinc-900/20 font-medium text-zinc-700 dark:text-zinc-300' : 'text-gray-900 dark:text-white'"
            >
              All Areas
            </button>

            <template v-for="group in areaGroups" :key="group.product_id">
              <button
                @click="selectArea({ product_id: group.product_id, product_name: group.product_name, geofence_label: null })"
                class="sticky top-0 w-full text-left px-3 py-1.5 text-xs font-semibold uppercase tracking-wider transition-colors cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-900/20"
                :class="selectedArea?.product_id === group.product_id && !selectedArea?.geofence_label ? 'bg-zinc-50 dark:bg-zinc-900/20 text-zinc-700 dark:text-zinc-300' : 'text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-700/50'"
              >
                {{ group.product_name }}
              </button>
              <button
                v-for="area in group.areas"
                :key="area.product_id + area.geofence_label"
                @click="selectArea(area)"
                class="w-full text-left px-3 py-2 pl-5 text-sm hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors flex items-center justify-between"
                :class="selectedArea?.product_id === area.product_id && selectedArea?.geofence_label === area.geofence_label ? 'bg-zinc-50 dark:bg-zinc-900/20 text-zinc-700 dark:text-zinc-300' : 'text-gray-900 dark:text-white'"
              >
                <span class="truncate">{{ area.geofence_label }}</span>
                <span v-if="area.total_violations > 0" class="ml-2 text-xs text-gray-400 dark:text-gray-500 tabular-nums shrink-0">
                  {{ area.total_violations }}
                </span>
              </button>
            </template>

            <div v-if="areaGroups.length === 0 && areaSearchQuery" class="p-4 text-sm text-gray-500 dark:text-gray-400 text-center">
              No areas found
            </div>
            <div v-if="areas.length === 0 && !areaSearchQuery" class="p-4 text-sm text-gray-500 dark:text-gray-400 text-center">
              No geofence areas configured
            </div>
          </div>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <div class="flex rounded-lg border border-gray-300 dark:border-gray-600 overflow-hidden">
          <button
            v-for="p in [
              { key: 'this_month', label: 'This Month' },
              { key: 'last_month', label: 'Last Month' },
              { key: 'last_7_days', label: '7 Days' },
              { key: 'last_30_days', label: '30 Days' },
              { key: 'custom', label: 'Custom' }
            ]"
            :key="p.key"
            @click="applyPreset(p.key); onFilterChange()"
            :class="[
              'px-3 py-1.5 text-sm font-medium transition-colors',
              datePreset === p.key
                ? 'bg-[#18181b] dark:bg-[#27272a] text-white'
                : 'bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
            ]"
          >
            {{ p.label }}
          </button>
        </div>

        <template v-if="datePreset === 'custom'">
          <input
            v-model="dateFrom"
            type="date"
            @change="onFilterChange"
            class="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
          />
          <span class="text-gray-400">-</span>
          <input
            v-model="dateTo"
            type="date"
            @change="onFilterChange"
            class="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
          />
        </template>
      </div>
    </div>

    <!-- Stats Cards -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
      <Card v-for="s in [
        { key: 'low', label: 'Low', desc: '0-10 km from zone', color: 'text-gray-600 dark:text-gray-400' },
        { key: 'medium', label: 'Medium', desc: '10-50 km from zone', color: 'text-amber-600 dark:text-amber-400' },
        { key: 'high', label: 'High', desc: '50-200 km from zone', color: 'text-orange-600 dark:text-orange-400' },
        { key: 'critical', label: 'Critical', desc: '200+ km from zone', color: 'text-red-600 dark:text-red-400' }
      ]" :key="s.key" class="p-4">
        <div class="flex items-start justify-between">
          <div>
            <div class="text-sm text-gray-500 dark:text-gray-400">{{ s.label }}</div>
            <div :class="['text-2xl font-bold', s.color]">{{ severityCounts[s.key] }}</div>
            <div class="text-xs text-gray-400 mt-1">{{ s.desc }}</div>
          </div>
          <div v-if="severityChange[s.key] !== null" class="text-right relative group">
            <div
              :class="['flex items-center justify-end text-sm font-medium cursor-help',
                severityChange[s.key] <= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400']"
            >
              <component :is="severityChange[s.key] <= 0 ? TrendingDown : TrendingUp" class="w-4 h-4 mr-1" />
              {{ Math.abs(severityChange[s.key]).toFixed(1) }}%
            </div>
            <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{{ statsDateFilter.comparison_label }}</p>
            <div class="absolute right-0 top-full mt-1 z-20 hidden group-hover:block bg-gray-900 text-white text-xs rounded-lg py-2 px-3 whitespace-nowrap shadow-lg">
              <div>Current: {{ severityCounts[s.key] }} violations</div>
              <div>Previous: {{ previousSeverityCounts[s.key] }} violations</div>
              <div class="text-gray-400 mt-1 text-[10px]">{{ statsDateFilter.previous_from }} to {{ statsDateFilter.previous_to }}</div>
            </div>
          </div>
        </div>
      </Card>
    </div>

    <!-- Violation Map (conditional: only when violations exist) -->
    <Card v-if="stats.total_violations > 0" class="p-4 mb-6">
      <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">Violation Map</h3>
      <div v-if="loadingMap" class="flex justify-center py-8">
        <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-zinc-500"></div>
      </div>
      <template v-else-if="mapViolations.length > 0 || mapZones.length > 0">
        <GeofenceViolationMap
          :violations="mapViolations"
          :zones="mapZones"
          height="400px"
        />
        <p v-if="mapViolations.length >= 500" class="text-xs text-gray-400 dark:text-gray-500 mt-2 text-center">
          Showing latest 500 violations. Use filters to narrow results.
        </p>
      </template>
      <div v-else class="text-center py-8 text-sm text-gray-500 dark:text-gray-400">
        No violations match the current filters.
      </div>
    </Card>

    <!-- Top Batches (from stats) -->
    <Card v-if="stats.top_batches && stats.top_batches.length > 0" class="p-4 mb-6">
      <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">Most Affected Batches</h3>
      <div class="space-y-1">
        <router-link
          v-for="tb in stats.top_batches"
          :key="tb.batch_id"
          :to="`/tenant/qr-batches/${tb.batch_id}`"
          class="flex items-center justify-between text-sm hover:bg-zinc-50 dark:hover:bg-zinc-900/30 rounded-lg px-2 py-2 -mx-2 transition-colors cursor-pointer"
        >
          <div class="flex items-center gap-2">
            <MapPin class="w-4 h-4 text-zinc-500" />
            <span class="text-gray-900 dark:text-white font-medium">{{ tb.product_name || '-' }}</span>
            <span class="text-gray-500 dark:text-gray-400">{{ tb.batch_name }}</span>
          </div>
          <span class="font-medium text-gray-900 dark:text-white">{{ tb.violation_count }} violations</span>
        </router-link>
      </div>
    </Card>

    <!-- Analytics -->
    <template v-if="analytics">
      <!-- Summary Cards with Period Comparison -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <!-- Violation Rate -->
        <Card class="p-4">
          <div class="flex items-center justify-between">
            <div>
              <div class="flex items-center gap-2 mb-1">
                <Percent class="w-4 h-4 text-orange-500" />
                <span class="text-sm text-gray-500 dark:text-gray-400">Violation Rate</span>
              </div>
              <div class="text-2xl font-bold text-gray-900 dark:text-white">
                {{ analytics.violation_rate?.rate?.toFixed(1) || '0.0' }}%
              </div>
              <div class="text-xs text-gray-400 mt-1">
                {{ analytics.violation_rate?.total_violations || 0 }} of {{ analytics.violation_rate?.total_scans || 0 }} geofenced scans
              </div>
            </div>
            <div v-if="periodChange.rate !== null" class="text-right relative group">
              <div
                :class="['flex items-center justify-end text-sm font-medium cursor-help', periodChange.rate <= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400']"
              >
                <component :is="periodChange.rate <= 0 ? TrendingDown : TrendingUp" class="w-4 h-4 mr-1" />
                {{ Math.abs(periodChange.rate).toFixed(1) }}%
              </div>
              <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{{ dateFilter.comparison_label }}</p>
              <div class="absolute right-0 top-full mt-1 z-20 hidden group-hover:block bg-gray-900 text-white text-xs rounded-lg py-2 px-3 whitespace-nowrap shadow-lg">
                <div>Current: {{ (analytics.violation_rate?.rate || 0).toFixed(1) }}%</div>
                <div>Previous: {{ (previousAnalytics?.violation_rate?.rate || 0).toFixed(1) }}%</div>
                <div class="text-gray-400 mt-1 text-[10px]">{{ dateFilter.previous_from }} to {{ dateFilter.previous_to }}</div>
              </div>
            </div>
          </div>
        </Card>

        <!-- Avg Distance -->
        <Card class="p-4">
          <div class="flex items-center justify-between">
            <div>
              <div class="flex items-center gap-2 mb-1">
                <Ruler class="w-4 h-4 text-amber-500" />
                <span class="text-sm text-gray-500 dark:text-gray-400">Avg Distance from Zone</span>
              </div>
              <div class="text-2xl font-bold text-gray-900 dark:text-white">
                {{ analytics.distance_stats?.avg_km || 0 }} km
              </div>
              <div class="text-xs text-gray-400 mt-1">Average distance from zone edge</div>
            </div>
            <div v-if="periodChange.avgKm !== null" class="text-right relative group">
              <div
                :class="['flex items-center justify-end text-sm font-medium cursor-help', periodChange.avgKm <= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400']"
              >
                <component :is="periodChange.avgKm <= 0 ? TrendingDown : TrendingUp" class="w-4 h-4 mr-1" />
                {{ Math.abs(periodChange.avgKm).toFixed(1) }}%
              </div>
              <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{{ dateFilter.comparison_label }}</p>
              <div class="absolute right-0 top-full mt-1 z-20 hidden group-hover:block bg-gray-900 text-white text-xs rounded-lg py-2 px-3 whitespace-nowrap shadow-lg">
                <div>Current: {{ analytics.distance_stats?.avg_km || 0 }} km</div>
                <div>Previous: {{ previousAnalytics?.distance_stats?.avg_km || 0 }} km</div>
                <div class="text-gray-400 mt-1 text-[10px]">{{ dateFilter.previous_from }} to {{ dateFilter.previous_to }}</div>
              </div>
            </div>
          </div>
        </Card>

        <!-- Max Distance -->
        <Card class="p-4">
          <div class="flex items-center justify-between">
            <div>
              <div class="flex items-center gap-2 mb-1">
                <AlertTriangle class="w-4 h-4 text-red-500" />
                <span class="text-sm text-gray-500 dark:text-gray-400">Max Distance</span>
              </div>
              <div class="text-2xl font-bold text-gray-900 dark:text-white">
                {{ analytics.distance_stats?.max_km || 0 }} km
              </div>
              <div class="text-xs text-gray-400 mt-1">Farthest scan from zone edge</div>
            </div>
            <div v-if="periodChange.maxKm !== null" class="text-right relative group">
              <div
                :class="['flex items-center justify-end text-sm font-medium cursor-help', periodChange.maxKm <= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400']"
              >
                <component :is="periodChange.maxKm <= 0 ? TrendingDown : TrendingUp" class="w-4 h-4 mr-1" />
                {{ Math.abs(periodChange.maxKm).toFixed(1) }}%
              </div>
              <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{{ dateFilter.comparison_label }}</p>
              <div class="absolute right-0 top-full mt-1 z-20 hidden group-hover:block bg-gray-900 text-white text-xs rounded-lg py-2 px-3 whitespace-nowrap shadow-lg">
                <div>Current: {{ analytics.distance_stats?.max_km || 0 }} km</div>
                <div>Previous: {{ previousAnalytics?.distance_stats?.max_km || 0 }} km</div>
                <div class="text-gray-400 mt-1 text-[10px]">{{ dateFilter.previous_from }} to {{ dateFilter.previous_to }}</div>
              </div>
            </div>
          </div>
        </Card>
      </div>

      <!-- Trend Chart -->
      <Card v-if="trendChartData" class="p-4 mb-6">
        <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
          <TrendingUp class="w-4 h-4 text-orange-500" />
          Violation Trends
        </h3>
        <div style="height: 250px;">
          <Line :data="trendChartData" :options="trendChartOptions" />
        </div>
      </Card>

      <!-- Two tables side-by-side: Top Cities | By Product -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        <!-- Top Violation Cities -->
        <Card v-if="analytics.top_cities && analytics.top_cities.length > 0" class="p-4">
          <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
            <MapPin class="w-4 h-4 text-orange-500" />
            Top Violation Cities
          </h3>
          <div class="space-y-2">
            <div v-for="city in analytics.top_cities" :key="city.city + city.province" class="text-sm">
              <div class="flex justify-between mb-1">
                <div>
                  <span class="text-gray-900 dark:text-white font-medium">{{ city.city }}</span>
                  <span class="text-gray-500 dark:text-gray-400 ml-1">{{ city.province }}</span>
                </div>
                <span class="font-medium text-gray-900 dark:text-white">{{ city.count }}</span>
              </div>
              <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                <div
                  class="bg-orange-500 h-1.5 rounded-full"
                  :style="{ width: (city.count / topCityMax * 100) + '%' }"
                ></div>
              </div>
            </div>
          </div>
        </Card>

        <!-- Violations by Product -->
        <Card v-if="analytics.by_product && analytics.by_product.length > 0" class="p-4">
          <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
            <Package class="w-4 h-4 text-orange-500" />
            Violations by Product
          </h3>
          <div class="space-y-2">
            <div v-for="prod in analytics.by_product" :key="prod.product_id" class="text-sm">
              <div class="flex justify-between mb-1">
                <span class="text-gray-900 dark:text-white font-medium">{{ prod.product_name }}</span>
                <span class="font-medium text-gray-900 dark:text-white">{{ prod.count }}</span>
              </div>
              <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                <div
                  class="bg-orange-500 h-1.5 rounded-full"
                  :style="{ width: (prod.count / byProductMax * 100) + '%' }"
                ></div>
              </div>
            </div>
          </div>
        </Card>
      </div>
    </template>

    <!-- Loading Analytics -->
    <div v-else-if="loadingAnalytics" class="flex justify-center py-8 mb-6">
      <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-orange-500"></div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <!-- Violations Table -->
    <Card v-else-if="violations.length > 0" class="overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 dark:bg-gray-800 text-gray-600 dark:text-gray-400">
            <tr>
              <th class="px-4 py-3 text-left font-medium">Batch / Product</th>
              <th class="px-4 py-3 text-left font-medium">Scan Location</th>
              <th class="px-4 py-3 text-left font-medium">Distance</th>
              <th class="px-4 py-3 text-left font-medium">Severity</th>
              <th class="px-4 py-3 text-left font-medium">Time</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
            <tr
              v-for="v in violations"
              :key="v.id"
              class=""
            >
              <td class="px-4 py-3">
                <div class="font-medium text-gray-900 dark:text-white">
                  {{ v.batch?.batch_name || '-' }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ v.batch?.product?.product_name || '-' }}
                </div>
              </td>
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
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  from zone edge
                </div>
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
    </Card>

    <!-- Empty state -->
    <div v-else class="text-center py-12">
      <Shield class="w-16 h-16 text-gray-300 dark:text-gray-600 mx-auto mb-4" />
      <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No violations found</h3>
      <p class="text-gray-500 dark:text-gray-400">All scans are within distribution zones, or no geofence-enabled batches exist.</p>
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
      <span class="flex items-center text-sm text-gray-600 dark:text-gray-400">
        Page {{ page }} of {{ totalPages }}
      </span>
      <Button
        variant="outline"
        size="sm"
        :disabled="page >= totalPages"
        @click="goToPage(page + 1)"
      >
        Next
      </Button>
    </div>

    <ExportTimezoneModal
      :open="showExportModal"
      title="Export Geofence Violations"
      :loading="exporting"
      @confirm="exportViolations"
      @cancel="showExportModal = false"
    />
  </div>
</template>
