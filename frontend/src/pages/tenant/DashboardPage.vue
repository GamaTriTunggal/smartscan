<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAPI, isLogoutInProgress } from '@/composables/useAPI'
import { useDateTime } from '@/composables/useDateTime'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import ScanHeatmap from '@/components/ScanHeatmap.vue'
import { Bar, Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  ArcElement,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Filler,
  Title,
  Tooltip,
  Legend
} from 'chart.js'
import { TrendingUp, TrendingDown, Minus, Eye, QrCode, MapPin, AlertTriangle } from 'lucide-vue-next'

// Register Chart.js components
ChartJS.register(ArcElement, CategoryScale, LinearScale, PointElement, LineElement, BarElement, Filler, Title, Tooltip, Legend)

const router = useRouter()
const api = useAPI()
const { formatDate, formatDateTime } = useDateTime()

const loading = ref(true)

const hasDynamicProducts = ref(false)

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
      // Jan 1 - today
      dateFrom.value = formatDateLocal(new Date(now.getFullYear(), now.getMonth(), 1))
      dateTo.value = formatDateLocal(now)
      break
    case 'last_month':
      // Dec 1 - Dec 31 (previous full month)
      const lastMonth = new Date(now.getFullYear(), now.getMonth() - 1, 1)
      const lastDayLastMonth = new Date(now.getFullYear(), now.getMonth(), 0)
      dateFrom.value = formatDateLocal(lastMonth)
      dateTo.value = formatDateLocal(lastDayLastMonth)
      break
    case 'last_7_days':
      const sevenDaysAgo = new Date(now.getTime() - 6 * 24 * 60 * 60 * 1000)
      dateFrom.value = formatDateLocal(sevenDaysAgo)
      dateTo.value = formatDateLocal(now)
      break
    case 'last_30_days':
      const thirtyDaysAgo = new Date(now.getTime() - 29 * 24 * 60 * 60 * 1000)
      dateFrom.value = formatDateLocal(thirtyDaysAgo)
      dateTo.value = formatDateLocal(now)
      break
    case 'custom':
      // Keep current values, let user pick
      break
  }
}

// Enhanced stats with new metrics
const stats = ref({
  total_scans: 0,
  total_qr_codes: 0,
  scan_qr_ratio: 0,
  unique_cities: 0,
  counterfeit_rate: 0,
  counterfeit_count: 0,
})

const previousStats = ref(null)

// Scan trends by type (for stacked area chart)
const scanTrendsByType = ref([])

// Top regions
const topRegions = ref([])

// Counterfeit Intelligence
const counterfeitWidget = ref({
  overall_rate: 0,
  previous_rate: 0,
  per_product: [],
  hotspot_locations: []
})

// Geofence Distribution Alerts
const geofenceWidget = ref(null)

// Heatmap data
const heatmapData = ref(null)
const heatmapLoading = ref(false)

// Template performance data
const templatePerformance = ref({
  validation: [],
  warranty: [],
})

// Date filter info for comparison labels
const dateFilter = ref({
  from: '',
  to: '',
  previous_from: '',
  previous_to: '',
  comparison_label: 'vs last month',
  preset: 'this_month'
})


// Chart colors
const CHART_COLORS = [
  'rgba(99, 102, 241, 0.8)',
  'rgba(16, 185, 129, 0.8)',
  'rgba(6, 182, 212, 0.8)',
  'rgba(239, 68, 68, 0.8)',
  'rgba(139, 92, 246, 0.8)',
  'rgba(59, 130, 246, 0.8)',
  'rgba(236, 72, 153, 0.8)',
  'rgba(20, 184, 166, 0.8)',
  'rgba(251, 146, 60, 0.8)',
  'rgba(168, 162, 158, 0.8)',
  'rgba(156, 163, 175, 0.8)'
]

// Period change computed
const periodChange = computed(() => {
  if (!stats.value || !previousStats.value) {
    return { scans: null, ratio: null, cities: null, counterfeit: null }
  }

  const calcChange = (current, previous) => {
    if (previous === 0) return current > 0 ? 100 : 0
    return ((current - previous) / previous) * 100
  }

  return {
    scans: calcChange(stats.value.total_scans, previousStats.value.total_scans),
    ratio: calcChange(stats.value.scan_qr_ratio, previousStats.value.scan_qr_ratio),
    cities: stats.value.unique_cities - previousStats.value.unique_cities,
    counterfeit: calcChange(stats.value.counterfeit_rate || 0, previousStats.value.counterfeit_rate || 0)
  }
})

// Stacked area chart for scan trends by type
const scanTrendsStackedData = computed(() => {
  if (!scanTrendsByType.value?.length) return null

  return {
    labels: scanTrendsByType.value.map(t => t.date.slice(5)),
    datasets: [
      {
        label: 'Validation',
        data: scanTrendsByType.value.map(t => t.validation),
        borderColor: '#8B5CF6',
        backgroundColor: 'rgba(139, 92, 246, 0.5)',
        fill: true,
        tension: 0.3
      },
      {
        label: 'Warranty',
        data: scanTrendsByType.value.map(t => t.warranty),
        borderColor: '#3f3f46',
        backgroundColor: 'rgba(59, 130, 246, 0.5)',
        fill: true,
        tension: 0.3
      }
    ]
  }
})

const stackedAreaOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { position: 'bottom', labels: { boxWidth: 12, padding: 8, font: { size: 11 } } },
    tooltip: { mode: 'index', intersect: false }
  },
  scales: {
    x: { grid: { display: false } },
    y: { stacked: true, beginAtZero: true, grid: { color: 'rgba(156, 163, 175, 0.2)' } }
  },
  interaction: { mode: 'nearest', axis: 'x', intersect: false }
}

// Top regions horizontal bar chart
const topRegionsChartData = computed(() => {
  if (!topRegions.value?.length) return null

  return {
    labels: topRegions.value.slice(0, 5).map(r => `${r.city || 'Unknown'}, ${r.country || ''}`),
    datasets: [{
      data: topRegions.value.slice(0, 5).map(r => r.count),
      backgroundColor: 'rgba(20, 184, 166, 0.8)',
      borderRadius: 4
    }]
  }
})

const horizontalBarOptions = {
  indexAxis: 'y',
  responsive: true,
  maintainAspectRatio: false,
  plugins: { legend: { display: false } },
  scales: {
    x: { beginAtZero: true, grid: { display: false } },
    y: { grid: { display: false } }
  }
}



// Risk level helpers
const getRiskBadgeClass = (level) => {
  const classes = {
    high: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
    medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
    low: 'bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400',
    safe: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
  }
  return classes[level] || classes.safe
}

const getRiskLabel = (level) => {
  const labels = { high: 'High Risk', medium: 'Medium', low: 'Low', safe: 'Safe' }
  return labels[level] || 'Safe'
}

// Template performance helpers
const getMaxTemplateCount = (templates) => {
  if (!templates?.length) return 1
  return Math.max(...templates.map(t => t.count), 1)
}

const getTemplateBarWidth = (count, max) => {
  return Math.max((count / max) * 100, 2)
}

const getTrendClass = (change) => {
  if (change > 0) return 'text-green-600 dark:text-green-400'
  if (change < 0) return 'text-red-600 dark:text-red-400'
  return 'text-gray-500 dark:text-gray-400'
}

const formatTrend = (change) => {
  if (change === 0 || change === null || change === undefined) return '0%'
  const sign = change > 0 ? '+' : ''
  return `${sign}${change.toFixed(1)}%`
}

const fetchDashboard = async () => {
  try {
    loading.value = true
    const response = await api.get(`/tenant/dashboard?from=${dateFrom.value}&to=${dateTo.value}&preset=${datePreset.value}`)
    if (response.success && response.data) {
      stats.value = response.data.stats
      previousStats.value = response.data.previous_stats
      scanTrendsByType.value = response.data.scan_trends_by_type || []
      topRegions.value = response.data.top_regions || []

      hasDynamicProducts.value = response.data.has_dynamic_products || false

      // Store date filter info for comparison labels
      if (response.data.date_filter) {
        dateFilter.value = response.data.date_filter
      }

      // Counterfeit intelligence
      if (response.data.counterfeit) {
        counterfeitWidget.value = response.data.counterfeit
      }

      // Geofence distribution alerts
      geofenceWidget.value = response.data.geofence || null

      // Template performance data
      if (response.data.template_performance) {
        templatePerformance.value = {
          validation: response.data.template_performance.validation || [],
          warranty: response.data.template_performance.warranty || [],
        }
      }

      fetchHeatmap()
    }
  } catch (error) {
    console.error('Failed to fetch dashboard:', error)
  } finally {
    // Don't show empty state during logout redirect
    if (!isLogoutInProgress()) {
      loading.value = false
    }
  }
}

watch([dateFrom, dateTo], () => {
  fetchDashboard()
})

const fetchHeatmap = async () => {
  try {
    heatmapLoading.value = true
    const response = await api.get(`/tenant/heatmap?source=all&from=${dateFrom.value}&to=${dateTo.value}`)
    if (response.success && response.data) {
      heatmapData.value = response.data
    }
  } catch (error) {
    console.error('Failed to fetch heatmap:', error)
  } finally {
    // Don't show empty state during logout redirect
    if (!isLogoutInProgress()) {
      heatmapLoading.value = false
    }
  }
}

const heatmapLocations = computed(() => {
  if (!heatmapData.value?.points) return []
  return heatmapData.value.points.map(p => ({
    latitude: p.lat,
    longitude: p.lng,
    name: p.product_name,
    date: p.created_at ? formatDateTime(p.created_at) : null,
    scanType: p.scan_type,
    counterfeitStatus: p.counterfeit_status,
    qrCodeId: p.qr_code_id,
    batchId: p.batch_id,
    intensity: 1
  }))
})

const heatmapGeofenceViolations = computed(() => {
  return heatmapData.value?.geofence_violations || []
})


onMounted(() => {
  fetchDashboard()
})

</script>

<template>
  <div>
    <!-- Header with Date Range Filter -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6">
      <div class="flex items-center gap-3 mb-4 sm:mb-0">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
      </div>
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

    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <!-- Getting started (no dynamic products yet) -->
    <Card v-else-if="!hasDynamicProducts" class="p-6 sm:p-8 max-w-2xl mx-auto">
      <div class="flex items-center gap-3 mb-2">
        <div class="p-2.5 bg-zinc-100 dark:bg-zinc-900/30 rounded-lg">
          <QrCode class="w-6 h-6 text-zinc-600 dark:text-zinc-400" />
        </div>
        <h2 class="text-xl font-bold text-gray-900 dark:text-white">Get started</h2>
      </div>
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-6">
        Set up your first product and start protecting it with dynamic QR codes.
      </p>

      <ol class="space-y-4">
        <li class="flex items-start gap-4">
          <span class="flex-shrink-0 w-7 h-7 rounded-full bg-zinc-600 text-white text-sm font-semibold flex items-center justify-center">1</span>
          <div class="flex-1">
            <p class="font-medium text-gray-900 dark:text-white">Create your first product</p>
            <p class="text-sm text-gray-500 dark:text-gray-400 mb-2">Give it a name and (optionally) a code and description.</p>
            <Button @click="router.push('/tenant/products/dynamic')">
              Create a product
            </Button>
          </div>
        </li>
        <li class="flex items-start gap-4">
          <span class="flex-shrink-0 w-7 h-7 rounded-full bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 text-sm font-semibold flex items-center justify-center">2</span>
          <div class="flex-1">
            <p class="font-medium text-gray-900 dark:text-white">Generate a batch of QR codes</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">Choose how many unique codes you need for this run.</p>
          </div>
        </li>
        <li class="flex items-start gap-4">
          <span class="flex-shrink-0 w-7 h-7 rounded-full bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 text-sm font-semibold flex items-center justify-center">3</span>
          <div class="flex-1">
            <p class="font-medium text-gray-900 dark:text-white">Download &amp; print the codes</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">Export ready-to-print label PDFs, or CSV for your print vendor.</p>
          </div>
        </li>
        <li class="flex items-start gap-4">
          <span class="flex-shrink-0 w-7 h-7 rounded-full bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 text-sm font-semibold flex items-center justify-center">4</span>
          <div class="flex-1">
            <p class="font-medium text-gray-900 dark:text-white">Scan one with your phone</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">See exactly what your customers see on the consumer page.</p>
          </div>
        </li>
      </ol>
    </Card>

    <div v-else>
      <!-- KPI Stats Cards with Period Comparison -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <!-- Total Scans -->
        <Card class="p-6">
          <div class="flex items-center justify-between">
            <div class="flex items-center">
              <div class="flex-shrink-0 p-3 bg-green-100 dark:bg-green-900/30 rounded-lg">
                <Eye class="w-6 h-6 text-green-600 dark:text-green-400" />
              </div>
              <div class="ml-4">
                <p class="text-sm text-gray-500 dark:text-gray-400">Total Scans</p>
                <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ stats.total_scans?.toLocaleString() || 0 }}</p>
              </div>
            </div>
            <div v-if="periodChange.scans !== null" class="text-right relative group">
              <div
                :class="['flex items-center justify-end text-sm font-medium cursor-help', periodChange.scans >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400']"
              >
                <component :is="periodChange.scans >= 0 ? TrendingUp : TrendingDown" class="w-4 h-4 mr-1" />
                {{ Math.abs(periodChange.scans).toFixed(1) }}%
              </div>
              <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{{ dateFilter.comparison_label }}</p>
              <!-- Tooltip -->
              <div class="absolute right-0 top-full mt-1 z-20 hidden group-hover:block bg-gray-900 text-white text-xs rounded-lg py-2 px-3 whitespace-nowrap shadow-lg">
                <div>Current: {{ stats.total_scans?.toLocaleString() || 0 }} scans</div>
                <div>Previous: {{ previousStats?.total_scans?.toLocaleString() || 0 }} scans</div>
                <div class="text-gray-400 mt-1 text-[10px]">{{ dateFilter.previous_from }} to {{ dateFilter.previous_to }}</div>
              </div>
            </div>
          </div>
        </Card>

        <!-- Scan-QR Ratio -->
        <Card class="p-6">
          <div class="flex items-center justify-between">
            <div class="flex items-center">
              <div class="flex-shrink-0 p-3 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
                <QrCode class="w-6 h-6 text-purple-600 dark:text-purple-400" />
              </div>
              <div class="ml-4">
                <p class="text-sm text-gray-500 dark:text-gray-400">Scan-QR Ratio</p>
                <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ (stats.scan_qr_ratio || 0).toFixed(1) }}%</p>
              </div>
            </div>
            <div v-if="periodChange.ratio !== null" class="text-right relative group">
              <div
                :class="['flex items-center justify-end text-sm font-medium cursor-help', periodChange.ratio >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400']"
              >
                <component :is="periodChange.ratio >= 0 ? TrendingUp : TrendingDown" class="w-4 h-4 mr-1" />
                {{ Math.abs(periodChange.ratio).toFixed(1) }}%
              </div>
              <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{{ dateFilter.comparison_label }}</p>
              <!-- Tooltip -->
              <div class="absolute right-0 top-full mt-1 z-20 hidden group-hover:block bg-gray-900 text-white text-xs rounded-lg py-2 px-3 whitespace-nowrap shadow-lg">
                <div>Current: {{ (stats.scan_qr_ratio || 0).toFixed(1) }}%</div>
                <div>Previous: {{ (previousStats?.scan_qr_ratio || 0).toFixed(1) }}%</div>
                <div class="text-gray-400 mt-1 text-[10px]">{{ dateFilter.previous_from }} to {{ dateFilter.previous_to }}</div>
              </div>
            </div>
          </div>
        </Card>

        <!-- Geographic Coverage -->
        <Card class="p-6">
          <div class="flex items-center justify-between">
            <div class="flex items-center">
              <div class="flex-shrink-0 p-3 bg-zinc-100 dark:bg-zinc-900/30 rounded-lg">
                <MapPin class="w-6 h-6 text-zinc-600 dark:text-zinc-400" />
              </div>
              <div class="ml-4">
                <p class="text-sm text-gray-500 dark:text-gray-400">Coverage</p>
                <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ stats.unique_cities || 0 }} cities</p>
              </div>
            </div>
            <div v-if="periodChange.cities !== null" class="text-right relative group">
              <div
                :class="['flex items-center justify-end text-sm font-medium cursor-help', periodChange.cities >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400']"
              >
                <component :is="periodChange.cities >= 0 ? TrendingUp : TrendingDown" class="w-4 h-4 mr-1" />
                {{ periodChange.cities >= 0 ? '+' : '' }}{{ periodChange.cities }}
              </div>
              <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{{ dateFilter.comparison_label }}</p>
              <!-- Tooltip -->
              <div class="absolute right-0 top-full mt-1 z-20 hidden group-hover:block bg-gray-900 text-white text-xs rounded-lg py-2 px-3 whitespace-nowrap shadow-lg">
                <div>Current: {{ stats.unique_cities || 0 }} cities</div>
                <div>Previous: {{ previousStats?.unique_cities || 0 }} cities</div>
                <div class="text-gray-400 mt-1 text-[10px]">{{ dateFilter.previous_from }} to {{ dateFilter.previous_to }}</div>
              </div>
            </div>
          </div>
        </Card>

        <!-- Counterfeit Rate -->
        <Card class="p-6">
          <div class="flex items-center justify-between">
            <div class="flex items-center">
              <div class="flex-shrink-0 p-3 bg-red-100 dark:bg-red-900/30 rounded-lg">
                <AlertTriangle class="w-6 h-6 text-red-600 dark:text-red-400" />
              </div>
              <div class="ml-4">
                <p class="text-sm text-gray-500 dark:text-gray-400">Counterfeit Rate</p>
                <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ (stats.counterfeit_rate || 0).toFixed(2) }}%</p>
              </div>
            </div>
            <div v-if="periodChange.counterfeit !== null" class="text-right relative group">
              <div
                :class="['flex items-center justify-end text-sm font-medium cursor-help', periodChange.counterfeit <= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400']"
              >
                <component :is="periodChange.counterfeit <= 0 ? TrendingDown : TrendingUp" class="w-4 h-4 mr-1" />
                {{ Math.abs(periodChange.counterfeit).toFixed(1) }}%
              </div>
              <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{{ dateFilter.comparison_label }}</p>
              <!-- Tooltip -->
              <div class="absolute right-0 top-full mt-1 z-20 hidden group-hover:block bg-gray-900 text-white text-xs rounded-lg py-2 px-3 whitespace-nowrap shadow-lg">
                <div>Current: {{ (stats.counterfeit_rate || 0).toFixed(2) }}%</div>
                <div>Previous: {{ (previousStats?.counterfeit_rate || 0).toFixed(2) }}%</div>
                <div class="text-gray-400 mt-1 text-[10px]">{{ dateFilter.previous_from }} to {{ dateFilter.previous_to }}</div>
              </div>
            </div>
          </div>
        </Card>

      </div>

      <!-- Scan Distribution / Heatmap - Placed high for visibility -->
      <Card class="p-6 mb-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Scan Distribution</h2>
          <router-link :to="{ path: '/tenant/analytics', query: { from: dateFrom, to: dateTo, preset: datePreset } }" class="text-sm text-zinc-600 dark:text-zinc-400 hover:underline">
            View Details &rarr;
          </router-link>
        </div>

        <div v-if="heatmapLoading" class="h-64 flex items-center justify-center">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
        </div>

        <template v-else-if="heatmapLocations.length > 0">
          <ScanHeatmap :locations="heatmapLocations" :geofence-violations="heatmapGeofenceViolations" height="300px" />
        </template>

        <div v-else class="h-64 flex items-center justify-center text-gray-500 dark:text-gray-400">
          <div class="text-center">
            <MapPin class="w-12 h-12 mx-auto mb-2 opacity-50" />
            <p class="text-sm">No scan data for this period</p>
          </div>
        </div>
      </Card>

      <!-- Row 2: Scan Trends + Top Regions -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <!-- Scan Trends by Type (Stacked Area) -->
        <Card class="p-6">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Scan Trends by Type</h2>
          <div v-if="scanTrendsByType.length > 0 && scanTrendsStackedData" class="h-64">
            <Line :data="scanTrendsStackedData" :options="stackedAreaOptions" />
          </div>
          <div v-else class="h-64 flex items-center justify-center text-gray-500 dark:text-gray-400">
            <p class="text-sm">No scan data for this period</p>
          </div>
        </Card>

        <!-- Top Performing Regions -->
        <Card class="p-6">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Top Performing Regions</h2>
          <div v-if="topRegions.length > 0 && topRegionsChartData" class="h-64">
            <Bar :data="topRegionsChartData" :options="horizontalBarOptions" />
          </div>
          <div v-else class="h-64 flex items-center justify-center text-gray-500 dark:text-gray-400">
            <p class="text-sm">No regional data for this period</p>
          </div>
        </Card>
      </div>

      <!-- Counterfeit Intelligence (Dynamic QR only) -->
      <Card class="p-6 mb-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Counterfeit Intelligence</h2>
          <router-link to="/tenant/counterfeit" class="text-sm text-zinc-600 dark:text-zinc-400 hover:underline">
            View All &rarr;
          </router-link>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <!-- Overall Rate -->
          <div class="text-center p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <p class="text-sm text-gray-500 dark:text-gray-400 mb-1">Overall Rate</p>
            <p class="text-3xl font-bold" :class="counterfeitWidget.overall_rate > 2 ? 'text-red-600' : counterfeitWidget.overall_rate > 0.5 ? 'text-yellow-600' : 'text-green-600'">
              {{ (counterfeitWidget.overall_rate || 0).toFixed(2) }}%
            </p>
            <div v-if="counterfeitWidget.previous_rate !== undefined" class="flex items-center justify-center mt-1 text-xs" :class="getTrendClass(-(counterfeitWidget.overall_rate - counterfeitWidget.previous_rate))">
              <component :is="counterfeitWidget.overall_rate <= counterfeitWidget.previous_rate ? TrendingDown : TrendingUp" class="w-3 h-3 mr-1" />
              vs previous period
            </div>
          </div>

          <!-- Per-Product Risk Table -->
          <div class="lg:col-span-2">
            <p class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Product Risk Assessment</p>
            <div class="space-y-2 max-h-40 overflow-y-auto">
              <div v-for="product in counterfeitWidget.per_product" :key="product.product_id"
                   class="flex items-center justify-between p-2 rounded-lg bg-gray-50 dark:bg-gray-800">
                <span class="text-sm text-gray-900 dark:text-white truncate flex-1">
                  {{ product.product_name }}
                </span>
                <div class="flex items-center gap-2 ml-2">
                  <span class="text-xs text-gray-500">{{ product.counterfeit_count }}/{{ product.total_scans }}</span>
                  <span :class="['px-2 py-0.5 text-xs font-medium rounded-full', getRiskBadgeClass(product.risk_level)]">
                    {{ product.rate.toFixed(2) }}%
                  </span>
                </div>
              </div>
              <p v-if="!counterfeitWidget.per_product?.length" class="text-sm text-gray-500 dark:text-gray-400 text-center py-2">
                No counterfeit data for this period
              </p>
            </div>

            <!-- Hotspot Locations -->
            <div v-if="counterfeitWidget.hotspot_locations?.length" class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Hotspot Locations</p>
              <div class="flex flex-wrap gap-2">
                <span v-for="loc in counterfeitWidget.hotspot_locations" :key="loc.city"
                      class="px-2 py-1 text-xs rounded-full bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400">
                  {{ loc.city }}, {{ loc.country }} ({{ loc.count }})
                </span>
              </div>
            </div>
          </div>
        </div>
      </Card>

      <!-- Distribution Alerts / Geofence Intelligence (Dynamic QR only) -->
      <Card v-if="geofenceWidget && geofenceWidget.total_violations > 0" class="p-6 mb-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Distribution Alerts</h2>
          <router-link to="/tenant/geofence" class="text-sm text-orange-600 dark:text-orange-400 hover:underline">
            View All &rarr;
          </router-link>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <!-- Overall Stats -->
          <div class="text-center p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <p class="text-sm text-gray-500 dark:text-gray-400 mb-1">Total Violations</p>
            <p class="text-3xl font-bold text-orange-600">
              {{ geofenceWidget.total_violations.toLocaleString() }}
            </p>
            <!-- violation rate with trend -->
            <template v-if="geofenceWidget.violation_rate !== undefined">
              <p class="text-sm text-gray-500 dark:text-gray-400 mt-2">
                Rate: <span class="font-semibold" :class="(geofenceWidget.violation_rate ?? 0) > 5 ? 'text-red-600' : (geofenceWidget.violation_rate ?? 0) > 2 ? 'text-orange-600' : 'text-green-600'">{{ (geofenceWidget.violation_rate ?? 0).toFixed(2) }}%</span>
              </p>
              <div v-if="geofenceWidget.previous_violation_rate !== undefined" class="flex items-center justify-center mt-1 text-xs" :class="getTrendClass(-(geofenceWidget.violation_rate - geofenceWidget.previous_violation_rate))">
                <component :is="geofenceWidget.violation_rate <= geofenceWidget.previous_violation_rate ? TrendingDown : TrendingUp" class="w-3 h-3 mr-1" />
                vs previous period
              </div>
            </template>
          </div>

          <!-- Severity Breakdown + Top Batches -->
          <div class="lg:col-span-2">
            <!-- Severity breakdown -->
            <p class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Severity Breakdown</p>
            <div class="flex flex-wrap gap-2 mb-4">
              <span v-for="s in (geofenceWidget.by_severity || [])" :key="s.severity"
                    :class="[
                      'px-3 py-1.5 text-sm font-medium rounded-full',
                      s.severity === 'critical' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' :
                      s.severity === 'high' ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400' :
                      'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                    ]">
                {{ s.count }} {{ s.severity.charAt(0).toUpperCase() + s.severity.slice(1) }}
              </span>
            </div>

            <!-- Top batches -->
            <p class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Top Batches</p>
            <div class="space-y-2 max-h-32 overflow-y-auto">
              <div v-for="batch in (geofenceWidget.top_batches || [])" :key="batch.batch_id"
                   class="flex items-center justify-between p-2 rounded-lg bg-gray-50 dark:bg-gray-800">
                <router-link :to="`/tenant/qr-batches/${batch.batch_id}`" class="text-sm text-gray-900 dark:text-white truncate flex-1 hover:text-orange-600 dark:hover:text-orange-400">
                  {{ batch.batch_name }}
                </router-link>
                <span class="text-xs font-medium text-orange-600 dark:text-orange-400 ml-2">
                  {{ batch.count }} violations
                </span>
              </div>
              <p v-if="!geofenceWidget.top_batches?.length" class="text-sm text-gray-500 dark:text-gray-400 text-center py-2">
                No batch data for this period
              </p>
            </div>
          </div>
        </div>
      </Card>

      <!-- Template Performance (Horizontal Bars with Trend) -->
      <div class="mb-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Template Performance</h2>
        <div class="grid gap-6 grid-cols-1 md:grid-cols-3">
          <!-- Landing Page Templates -->
          <Card class="p-6">
            <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-4">Landing Page Templates</h3>
            <div v-if="templatePerformance.validation?.length" class="space-y-3">
              <div v-for="item in templatePerformance.validation" :key="item.template_id || item.template_name" class="flex items-center gap-2">
                <span class="text-sm text-gray-900 dark:text-white truncate flex-1 min-w-0">{{ item.template_name }}</span>
                <div class="w-20 bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                  <div class="bg-zinc-500 h-2 rounded-full" :style="{ width: `${getTemplateBarWidth(item.count, getMaxTemplateCount(templatePerformance.validation))}%` }"></div>
                </div>
                <span class="text-sm font-medium w-10 text-right text-gray-700 dark:text-gray-300">{{ item.count }}</span>
                <span :class="['text-xs w-14 text-right', getTrendClass(item.change_percent)]">
                  {{ formatTrend(item.change_percent) }}
                </span>
              </div>
            </div>
            <div v-else class="h-32 flex items-center justify-center">
              <p class="text-sm text-gray-500 dark:text-gray-400">No data available</p>
            </div>
          </Card>

          <!-- Warranty Templates -->
          <Card class="p-6">
            <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-4">Warranty Templates</h3>
            <div v-if="templatePerformance.warranty?.length" class="space-y-3">
              <div v-for="item in templatePerformance.warranty" :key="item.template_id || item.template_name" class="flex items-center gap-2">
                <span class="text-sm text-gray-900 dark:text-white truncate flex-1 min-w-0">{{ item.template_name }}</span>
                <div class="w-20 bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                  <div class="bg-zinc-500 h-2 rounded-full" :style="{ width: `${getTemplateBarWidth(item.count, getMaxTemplateCount(templatePerformance.warranty))}%` }"></div>
                </div>
                <span class="text-sm font-medium w-10 text-right text-gray-700 dark:text-gray-300">{{ item.count }}</span>
                <span :class="['text-xs w-14 text-right', getTrendClass(item.change_percent)]">
                  {{ formatTrend(item.change_percent) }}
                </span>
              </div>
            </div>
            <div v-else class="h-32 flex items-center justify-center">
              <p class="text-sm text-gray-500 dark:text-gray-400">No data available</p>
            </div>
          </Card>

        </div>
      </div>
    </div>
  </div>
</template>
