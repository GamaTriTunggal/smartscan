<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAPI } from '@/composables/useAPI'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { Chart as ChartJS, ArcElement, CategoryScale, LinearScale, PointElement, LineElement, BarElement, Filler, Title, Tooltip, Legend } from 'chart.js'
import { Pie, Line, Bar } from 'vue-chartjs'

ChartJS.register(ArcElement, CategoryScale, LinearScale, PointElement, LineElement, BarElement, Filler, Title, Tooltip, Legend)

const route = useRoute()
const router = useRouter()
const { get } = useAPI()

// State
const loading = ref(true)
const analytics = ref(null)

// Date range filter (same as Dashboard)
const formatDateLocal = (date) => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const today = new Date()
const startOfMonth = new Date(today.getFullYear(), today.getMonth(), 1)

// Initialize from URL params or default to this month
const dateFrom = ref(route.query.from || formatDateLocal(startOfMonth))
const dateTo = ref(route.query.to || formatDateLocal(today))
const datePreset = ref(route.query.preset || 'this_month')

const applyPreset = (preset) => {
  datePreset.value = preset
  const now = new Date()

  switch (preset) {
    case 'this_month':
      dateFrom.value = formatDateLocal(new Date(now.getFullYear(), now.getMonth(), 1))
      dateTo.value = formatDateLocal(now)
      break
    case 'last_3_months':
      const threeMonthsAgo = new Date(now.getFullYear(), now.getMonth() - 3, now.getDate())
      dateFrom.value = formatDateLocal(threeMonthsAgo)
      dateTo.value = formatDateLocal(now)
      break
    case 'this_year':
      dateFrom.value = formatDateLocal(new Date(now.getFullYear(), 0, 1))
      dateTo.value = formatDateLocal(now)
      break
    case 'custom':
      // Keep current dates, just show inputs
      break
  }

  // Update URL without navigation
  router.replace({ query: { from: dateFrom.value, to: dateTo.value, preset: preset } })
}

// Watch date changes and refetch
watch([dateFrom, dateTo], () => {
  fetchAnalytics()
})

// Fetch analytics
async function fetchAnalytics() {
  loading.value = true
  try {
    const response = await get(`/tenant/analytics?from=${dateFrom.value}&to=${dateTo.value}`)
    if (response.success) {
      analytics.value = response.data
    }
  } catch (error) {
    console.error('Failed to fetch analytics:', error)
  } finally {
    loading.value = false
  }
}

// Chart colors
const chartColors = {
  product_validation: '#F5A623',
  warranty_activation: '#10B981',
  qc_scan: '#8B5CF6',
  warehouse_scan: '#EC4899'
}

// Scan type pie chart data
const scanTypeChartData = ref({
  labels: [],
  datasets: [{
    data: [],
    backgroundColor: []
  }]
})

// Scan type pie chart options
const scanTypeChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      position: 'right'
    }
  }
}

// Counterfeit trend chart data
const counterfeitChartData = ref({
  labels: [],
  datasets: [{
    label: 'Counterfeit Detections',
    data: [],
    borderColor: '#EF4444',
    backgroundColor: 'rgba(239, 68, 68, 0.1)',
    fill: true,
    tension: 0.3
  }]
})

// Counterfeit trend chart options
const counterfeitChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false
    }
  },
  scales: {
    y: {
      beginAtZero: true
    }
  }
}

// Top products chart data
const topProductsChartData = ref({
  labels: [],
  datasets: [{
    label: 'Scans',
    data: [],
    backgroundColor: '#F5A623'
  }]
})

// Top products chart options
const topProductsChartOptions = {
  indexAxis: 'y',
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false
    }
  },
  scales: {
    x: {
      beginAtZero: true
    }
  }
}

// Update chart data when analytics changes
function updateCharts() {
  if (!analytics.value) return

  // Scan type pie chart (scans_by_type is array of {type, count})
  if (analytics.value.scans_by_type?.length > 0) {
    const types = analytics.value.scans_by_type
    scanTypeChartData.value = {
      labels: types.map(t => getScanTypeLabel(t.type)),
      datasets: [{
        data: types.map(t => t.count),
        backgroundColor: types.map(t => chartColors[t.type] || '#6B7280')
      }]
    }
  }

  // Counterfeit trend chart
  if (analytics.value.counterfeit_trend) {
    const trend = analytics.value.counterfeit_trend
    counterfeitChartData.value = {
      labels: trend.map(t => t.date),
      datasets: [{
        label: 'Counterfeit Detections',
        data: trend.map(t => t.count),
        borderColor: '#EF4444',
        backgroundColor: 'rgba(239, 68, 68, 0.1)',
        fill: true,
        tension: 0.3
      }]
    }
  }

  // Top products chart
  if (analytics.value.top_products) {
    const products = analytics.value.top_products.slice(0, 10)
    topProductsChartData.value = {
      labels: products.map(p => p.product_name?.slice(0, 20) || 'Unknown'),
      datasets: [{
        label: 'Scans',
        data: products.map(p => p.scan_count),
        backgroundColor: '#F5A623'
      }]
    }
  }
}

// Format number with commas
function formatNumber(num) {
  if (!num) return '0'
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

// Get total scans (scans_by_type is an array of {type, count})
function getTotalScans() {
  if (!analytics.value?.scans_by_type) return 0
  return analytics.value.scans_by_type.reduce((sum, item) => sum + (item.count || 0), 0)
}

// Get scan count by type
function getScanCountByType(scanType) {
  if (!analytics.value?.scans_by_type) return 0
  const found = analytics.value.scans_by_type.find(item => item.type === scanType)
  return found?.count || 0
}

// Get scan type label
function getScanTypeLabel(type) {
  const labels = {
    product_validation: 'Product Validation',
    warranty_activation: 'Warranty Activation',
    qc_scan: 'QC Scan',
    warehouse_scan: 'Warehouse Scan'
  }
  return labels[type] || type
}

// Get template performance total
function getTemplateTotal() {
  if (!analytics.value?.template_performance) return 0
  return analytics.value.template_performance.reduce((sum, t) => sum + t.scan_count, 0)
}

// Get template percentage
function getTemplatePercentage(count) {
  const total = getTemplateTotal()
  if (total === 0) return '0.0'
  return ((count / total) * 100).toFixed(1)
}

// Watch analytics changes to update charts
watch(analytics, () => {
  updateCharts()
}, { deep: true })

onMounted(() => {
  fetchAnalytics()
})
</script>

<template>
  <div>
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-4 sm:mb-0">Analytics</h1>

      <!-- Date range presets -->
      <div class="flex items-center gap-2">
        <div class="flex rounded-lg border border-gray-300 dark:border-gray-600 overflow-hidden">
          <button
            v-for="preset in [
              { key: 'this_month', label: 'This Month' },
              { key: 'last_3_months', label: 'Last 3 Months' },
              { key: 'this_year', label: 'This Year' },
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

        <!-- Custom date inputs (only visible when Custom is selected) -->
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

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500 mx-auto"></div>
      <p class="mt-2 text-gray-500">Loading analytics...</p>
    </div>

    <div v-else>
      <!-- Summary Stats -->
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <Card class="p-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">Total Scans</p>
          <p class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ formatNumber(getTotalScans()) }}
          </p>
        </Card>
        <Card class="p-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">Product Validations</p>
          <p class="text-2xl font-bold text-zinc-600 dark:text-zinc-400">
            {{ formatNumber(getScanCountByType('product_validation')) }}
          </p>
        </Card>
        <Card class="p-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">Warranty Activations</p>
          <p class="text-2xl font-bold text-green-600 dark:text-green-400">
            {{ formatNumber(getScanCountByType('warranty_activation')) }}
          </p>
        </Card>
        <Card class="p-4">
        </Card>
      </div>

      <!-- Charts Row -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <!-- Scan Types Pie Chart -->
        <Card class="p-6">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Scans by Type</h3>
          <div v-if="getTotalScans() > 0" class="h-64">
            <Pie :data="scanTypeChartData" :options="scanTypeChartOptions" />
          </div>
          <div v-else class="h-64 flex items-center justify-center text-gray-500 dark:text-gray-400">
            No scan data available
          </div>
        </Card>

        <!-- Counterfeit Trend -->
        <Card class="p-6">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Counterfeit Detection Trend</h3>
          <div v-if="analytics?.counterfeit_trend?.length > 0" class="h-64">
            <Line :data="counterfeitChartData" :options="counterfeitChartOptions" />
          </div>
          <div v-else class="h-64 flex items-center justify-center text-gray-500 dark:text-gray-400">
            No counterfeit detections in this period
          </div>
        </Card>
      </div>

      <!-- Top Products -->
      <Card class="p-6 mb-6">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Top Products by Scan Count</h3>
        <div v-if="analytics?.top_products?.length > 0" class="h-80">
          <Bar :data="topProductsChartData" :options="topProductsChartOptions" />
        </div>
        <div v-else class="h-80 flex items-center justify-center text-gray-500 dark:text-gray-400">
          No product scan data available
        </div>
      </Card>

      <!-- Template Performance (A/B Testing) -->
      <Card v-if="analytics?.template_performance?.length > 0" class="p-6 mb-6">
        <div class="flex items-center gap-2 mb-4">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Template Performance</h3>
          <span class="px-2 py-0.5 text-xs rounded-full bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400">
            A/B Testing
          </span>
        </div>
        <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
          Compare scan engagement across different validation templates. Use this data to optimize your landing page design.
        </p>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead class="bg-gray-50 dark:bg-gray-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Template</th>
                <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Scans</th>
                <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Share</th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Distribution</th>
              </tr>
            </thead>
            <tbody class="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
              <tr
                v-for="(template, index) in analytics?.template_performance || []"
                :key="template.template_id || 'default'"
                class="hover:bg-gray-50 dark:hover:bg-gray-800"
              >
                <td class="px-4 py-3 text-sm text-gray-900 dark:text-white font-medium">
                  {{ template.template_name }}
                </td>
                <td class="px-4 py-3 text-sm text-right text-gray-900 dark:text-white">
                  {{ formatNumber(template.scan_count) }}
                </td>
                <td class="px-4 py-3 text-sm text-right text-gray-500 dark:text-gray-400">
                  {{ getTemplatePercentage(template.scan_count) }}%
                </td>
                <td class="px-4 py-3">
                  <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                    <div
                      class="h-2 rounded-full"
                      :class="index === 0 ? 'bg-purple-500' : 'bg-purple-300 dark:bg-purple-700'"
                      :style="{ width: getTemplatePercentage(template.scan_count) + '%' }"
                    ></div>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </Card>

      <!-- Scan Types Breakdown Table -->
      <Card class="p-6">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Scan Types Breakdown</h3>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead class="bg-gray-50 dark:bg-gray-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Type</th>
                <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Count</th>
                <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Percentage</th>
              </tr>
            </thead>
            <tbody class="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
              <tr
                v-for="item in analytics?.scans_by_type || []"
                :key="item.type"
                class="hover:bg-gray-50 dark:hover:bg-gray-800"
              >
                <td class="px-4 py-3 text-sm text-gray-900 dark:text-white">
                  <div class="flex items-center gap-2">
                    <span
                      class="w-3 h-3 rounded-full"
                      :style="{ backgroundColor: chartColors[item.type] || '#6B7280' }"
                    ></span>
                    {{ getScanTypeLabel(item.type) }}
                  </div>
                </td>
                <td class="px-4 py-3 text-sm text-right text-gray-900 dark:text-white font-medium">
                  {{ formatNumber(item.count) }}
                </td>
                <td class="px-4 py-3 text-sm text-right text-gray-500 dark:text-gray-400">
                  {{ getTotalScans() > 0 ? ((item.count / getTotalScans()) * 100).toFixed(1) : 0 }}%
                </td>
              </tr>
            </tbody>
            <tfoot class="bg-gray-50 dark:bg-gray-800">
              <tr>
                <td class="px-4 py-3 text-sm font-semibold text-gray-900 dark:text-white">Total</td>
                <td class="px-4 py-3 text-sm text-right font-semibold text-gray-900 dark:text-white">
                  {{ formatNumber(getTotalScans()) }}
                </td>
                <td class="px-4 py-3 text-sm text-right font-semibold text-gray-500 dark:text-gray-400">100%</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </Card>
    </div>
  </div>
</template>
