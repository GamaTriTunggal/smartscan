<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useListFilter } from '@/composables/useListFilter'
import { useDateTime } from '@/composables/useDateTime'
import { useDarkMode } from '@/composables/useDarkMode'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { Line, Doughnut, Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import {
  ChevronDown, ChevronUp, Activity, ShieldAlert, Users, Globe,
  Search, SlidersHorizontal, BarChart3
} from 'lucide-vue-next'
import { getPagination } from '@/lib/pagination'

ChartJS.register(
  CategoryScale, LinearScale, PointElement, LineElement,
  BarElement, ArcElement, Title, Tooltip, Legend, Filler
)

const { get } = useAPI()
const { formatDateTime } = useDateTime()
const { isDark } = useDarkMode()

// --- Summary state ---
const showSummary = ref(localStorage.getItem('auditLogShowSummary') !== 'false')
const statsPeriod = ref('30d')
const stats = ref(null)
const statsLoading = ref(false)

const periodLabels = { '7d': '7 Days', '30d': '30 Days', '90d': '90 Days' }

const chartTextColor = computed(() => isDark.value ? '#9CA3AF' : '#6B7280')
const chartGridColor = computed(() => isDark.value ? 'rgba(55, 65, 81, 0.5)' : 'rgba(229, 231, 235, 0.8)')

const actionColors = {
  login: '#3f3f46',
  logout: '#6B7280',
  create: '#10B981',
  update: '#F59E0B',
  delete: '#EF4444',
  export: '#8B5CF6',
  password_reset: '#F97316',
}

const hasSecurityEvents = computed(() => (stats.value?.summary?.security_events || 0) > 0)

async function fetchStats() {
  statsLoading.value = true
  try {
    const response = await get('/tenant/audit-logs/stats', { period: statsPeriod.value })
    if (response.success) {
      stats.value = response.data
    }
  } catch (error) {
    console.error('Failed to fetch audit log stats:', error)
  } finally {
    statsLoading.value = false
  }
}

function toggleSummary() {
  showSummary.value = !showSummary.value
  localStorage.setItem('auditLogShowSummary', String(showSummary.value))
}

watch(statsPeriod, () => {
  fetchStats()
})

// Chart data
const trendChartData = computed(() => {
  if (!stats.value?.daily_trend?.length) return null
  const trend = stats.value.daily_trend
  return {
    labels: trend.map(d => d.date.substring(5)),
    datasets: [{
      label: 'Events',
      data: trend.map(d => d.count),
      borderColor: '#3f3f46',
      backgroundColor: isDark.value ? 'rgba(6, 182, 212, 0.08)' : 'rgba(6, 182, 212, 0.12)',
      fill: true,
      tension: 0.4,
      pointRadius: 3,
      pointHoverRadius: 6,
      pointBackgroundColor: '#3f3f46',
      pointBorderColor: isDark.value ? '#111827' : '#fff',
      pointBorderWidth: 2,
      borderWidth: 2,
    }]
  }
})

const trendChartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: isDark.value ? '#1F2937' : '#fff',
      titleColor: isDark.value ? '#F3F4F6' : '#111827',
      bodyColor: isDark.value ? '#D1D5DB' : '#4B5563',
      borderColor: isDark.value ? '#374151' : '#E5E7EB',
      borderWidth: 1,
      padding: 10,
      cornerRadius: 8,
      displayColors: false,
      callbacks: {
        title: (items) => items[0]?.label || '',
        label: (ctx) => `${ctx.raw.toLocaleString()} events`
      }
    }
  },
  scales: {
    y: {
      beginAtZero: true,
      ticks: { color: chartTextColor.value, stepSize: 1, padding: 8 },
      grid: { color: chartGridColor.value, drawBorder: false },
      border: { display: false }
    },
    x: {
      ticks: { color: chartTextColor.value, maxRotation: 0, padding: 8 },
      grid: { display: false },
      border: { display: false }
    }
  }
}))

const actionChartData = computed(() => {
  if (!stats.value?.by_action?.length) return null
  const actions = stats.value.by_action
  return {
    labels: actions.map(a => a.action_type),
    datasets: [{
      data: actions.map(a => a.count),
      backgroundColor: actions.map(a => actionColors[a.action_type] || '#6B7280'),
      borderWidth: 2,
      borderColor: isDark.value ? '#111827' : '#fff',
      hoverOffset: 6,
    }]
  }
})

const actionChartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: '60%',
  plugins: {
    legend: {
      position: 'right',
      labels: {
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 14,
        font: { size: 11, weight: '500' },
        color: chartTextColor.value
      }
    },
    tooltip: {
      backgroundColor: isDark.value ? '#1F2937' : '#fff',
      titleColor: isDark.value ? '#F3F4F6' : '#111827',
      bodyColor: isDark.value ? '#D1D5DB' : '#4B5563',
      borderColor: isDark.value ? '#374151' : '#E5E7EB',
      borderWidth: 1,
      padding: 10,
      cornerRadius: 8,
      callbacks: {
        label: (ctx) => {
          const total = ctx.dataset.data.reduce((a, b) => a + b, 0)
          const pct = ((ctx.raw / total) * 100).toFixed(1)
          return ` ${ctx.label}: ${ctx.raw.toLocaleString()} (${pct}%)`
        }
      }
    }
  }
}))

const entityChartData = computed(() => {
  if (!stats.value?.by_entity?.length) return null
  const entities = stats.value.by_entity
  const colors = [
    'rgba(6, 182, 212, 0.8)', 'rgba(59, 130, 246, 0.8)', 'rgba(139, 92, 246, 0.8)',
    'rgba(16, 185, 129, 0.8)', 'rgba(245, 158, 11, 0.8)', 'rgba(239, 68, 68, 0.8)',
    'rgba(249, 115, 22, 0.8)', 'rgba(236, 72, 153, 0.8)', 'rgba(20, 184, 166, 0.8)',
    'rgba(107, 114, 128, 0.8)'
  ]
  return {
    labels: entities.map(e => e.entity_type),
    datasets: [{
      label: 'Events',
      data: entities.map(e => e.count),
      backgroundColor: entities.map((_, i) => colors[i % colors.length]),
      borderRadius: 4,
      borderSkipped: false,
      barThickness: 20,
    }]
  }
})

const entityChartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  indexAxis: 'y',
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: isDark.value ? '#1F2937' : '#fff',
      titleColor: isDark.value ? '#F3F4F6' : '#111827',
      bodyColor: isDark.value ? '#D1D5DB' : '#4B5563',
      borderColor: isDark.value ? '#374151' : '#E5E7EB',
      borderWidth: 1,
      padding: 10,
      cornerRadius: 8,
      displayColors: false,
      callbacks: {
        label: (ctx) => `${ctx.raw.toLocaleString()} events`
      }
    }
  },
  scales: {
    x: {
      beginAtZero: true,
      ticks: { color: chartTextColor.value, stepSize: 1, padding: 8 },
      grid: { color: chartGridColor.value, drawBorder: false },
      border: { display: false }
    },
    y: {
      ticks: { color: chartTextColor.value, padding: 8, font: { size: 11 } },
      grid: { display: false },
      border: { display: false }
    }
  }
}))

// --- Table state ---
const logs = ref([])
const loading = ref(false)
const expandedLog = ref(null)

// Filters
const actionFilter = ref('')
const entityFilter = ref('')
const dateFrom = ref('')
const dateTo = ref('')

const { search, pagination, watchFilter, prevPage, nextPage } = useListFilter(fetchLogs)
watchFilter(actionFilter, entityFilter, dateFrom, dateTo)

async function fetchLogs() {
  loading.value = true
  try {
    const params = {
      page: pagination.value.page,
      limit: pagination.value.limit,
    }
    if (search.value) params.search = search.value
    if (actionFilter.value) params.action_type = actionFilter.value
    if (entityFilter.value) params.entity_type = entityFilter.value
    if (dateFrom.value) params.date_from = dateFrom.value
    if (dateTo.value) params.date_to = dateTo.value

    const response = await get('/tenant/audit-logs', params)
    if (response.success) {
      logs.value = response.data.logs || []
      const p = getPagination(response.data)
      pagination.value = {
        ...pagination.value,
        page: p.page,
        total: p.total,
        total_page: p.totalPages,
      }
    }
  } catch (error) {
    console.error('Failed to fetch audit logs:', error)
  } finally {
    loading.value = false
  }
}

function toggleExpand(logId) {
  expandedLog.value = expandedLog.value === logId ? null : logId
}

function parseJsonValues(val) {
  if (!val) return null
  if (typeof val === 'object') return val
  try {
    return JSON.parse(val)
  } catch {
    return null
  }
}

function formatKey(key) {
  return key
    .split('_')
    .map(w => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ')
}

function formatValue(value) {
  if (value === null || value === undefined) return 'null'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function getActionClass(action) {
  const classes = {
    login: 'bg-zinc-50 text-zinc-700 ring-zinc-600/20 dark:bg-zinc-500/10 dark:text-zinc-400 dark:ring-zinc-500/30',
    logout: 'bg-gray-50 text-gray-700 ring-gray-600/20 dark:bg-gray-500/10 dark:text-gray-400 dark:ring-gray-500/30',
    create: 'bg-emerald-50 text-emerald-700 ring-emerald-600/20 dark:bg-emerald-500/10 dark:text-emerald-400 dark:ring-emerald-500/30',
    update: 'bg-amber-50 text-amber-700 ring-amber-600/20 dark:bg-amber-500/10 dark:text-amber-400 dark:ring-amber-500/30',
    delete: 'bg-red-50 text-red-700 ring-red-600/20 dark:bg-red-500/10 dark:text-red-400 dark:ring-red-500/30',
    export: 'bg-violet-50 text-violet-700 ring-violet-600/20 dark:bg-violet-500/10 dark:text-violet-400 dark:ring-violet-500/30',
    password_reset: 'bg-orange-50 text-orange-700 ring-orange-600/20 dark:bg-orange-500/10 dark:text-orange-400 dark:ring-orange-500/30',
  }
  return classes[action] || 'bg-gray-50 text-gray-700 ring-gray-600/20 dark:bg-gray-500/10 dark:text-gray-400 dark:ring-gray-500/30'
}

function getOldValuesHeader(action) {
  return action === 'delete' ? 'Deleted Values' : 'Previous Values'
}

function getNewValuesHeader(action) {
  return action === 'create' ? 'Created Values' : 'New Values'
}

onMounted(() => {
  fetchLogs()
  fetchStats()
})
</script>

<template>
  <div class="audit-page">
    <!-- ═══════════════════════════════════════════ -->
    <!-- OBSERVATORY HEADER                          -->
    <!-- ═══════════════════════════════════════════ -->
    <div class="observatory-panel relative overflow-hidden rounded-2xl mb-6 border border-gray-200/80 dark:border-gray-700/50 bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-900 dark:via-gray-900 dark:to-gray-800">
      <!-- Dot-grid background texture -->
      <div class="absolute inset-0 dot-grid-bg opacity-[0.04] dark:opacity-[0.06]"></div>
      <!-- Subtle gradient overlay -->
      <div class="absolute top-0 right-0 w-1/2 h-full bg-gradient-to-l from-zinc-500/[0.03] to-transparent dark:from-zinc-500/[0.05]"></div>

      <div class="relative p-6">
        <!-- Title Row -->
        <div class="flex items-start justify-between mb-1">
          <div class="flex items-center gap-3">
            <div class="flex items-center gap-2">
              <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">Audit Logs</h1>
              <span class="live-dot relative flex h-2.5 w-2.5 mt-0.5">
                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-zinc-400 opacity-75"></span>
                <span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-zinc-500"></span>
              </span>
            </div>
          </div>
          <button
            @click="toggleSummary"
            class="group inline-flex items-center gap-2 rounded-lg px-3 py-2 text-xs font-medium transition-all duration-200 border border-gray-200 dark:border-gray-700 bg-white/80 dark:bg-gray-800/80 text-gray-600 dark:text-gray-400 hover:border-zinc-300 dark:hover:border-zinc-700 hover:text-zinc-600 dark:hover:text-zinc-400 backdrop-blur-sm"
          >
            <BarChart3 class="w-3.5 h-3.5" />
            {{ showSummary ? 'Hide Analytics' : 'Show Analytics' }}
            <ChevronUp v-if="showSummary" class="w-3 h-3 transition-transform" />
            <ChevronDown v-else class="w-3 h-3 transition-transform" />
          </button>
        </div>
        <p class="text-sm text-gray-500 dark:text-gray-500 mb-0">Security operations activity trail</p>

        <!-- ─── Collapsible Analytics ─── -->
        <div v-if="showSummary" class="summary-enter mt-6">
          <!-- Period Selector -->
          <div class="flex items-center justify-between mb-5">
            <div class="inline-flex items-center rounded-lg bg-gray-100 dark:bg-gray-800 p-0.5">
              <button
                v-for="p in ['7d', '30d', '90d']" :key="p"
                @click="statsPeriod = p"
                :class="[
                  'rounded-md px-3.5 py-1.5 text-xs font-semibold transition-all duration-200',
                  statsPeriod === p
                    ? 'bg-white dark:bg-zinc-600 text-gray-900 dark:text-white shadow-sm'
                    : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                ]"
              >
                {{ periodLabels[p] }}
              </button>
            </div>
            <span v-if="statsLoading" class="inline-flex items-center gap-1.5 text-xs text-gray-400">
              <span class="h-1.5 w-1.5 rounded-full bg-zinc-500 animate-pulse"></span>
              Fetching...
            </span>
          </div>

          <!-- Stat Cards -->
          <div class="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-5">
            <!-- Total Events -->
            <div class="stat-card relative overflow-hidden rounded-xl border-l-[3px] border-zinc-500 bg-white dark:bg-gray-800/60 p-4">
              <div class="absolute top-2 right-2 rounded-lg bg-zinc-500/8 dark:bg-zinc-400/10 p-1.5">
                <Activity class="w-4 h-4 text-zinc-500 dark:text-zinc-400" />
              </div>
              <p class="text-[10px] font-semibold uppercase tracking-[0.1em] text-gray-400 dark:text-gray-500">Total Events</p>
              <p class="mt-1 text-2xl font-bold font-mono tabular-nums tracking-tight text-gray-900 dark:text-white">
                {{ stats?.summary?.total_events?.toLocaleString() || '0' }}
              </p>
            </div>

            <!-- Security Events -->
            <div
              :class="[
                'stat-card relative overflow-hidden rounded-xl border-l-[3px] p-4',
                hasSecurityEvents
                  ? 'border-red-500 bg-red-50/80 dark:bg-red-950/30 security-pulse'
                  : 'border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800/60'
              ]"
            >
              <div :class="['absolute top-2 right-2 rounded-lg p-1.5', hasSecurityEvents ? 'bg-red-500/10 dark:bg-red-400/15' : 'bg-gray-100 dark:bg-gray-700/50']">
                <ShieldAlert class="w-4 h-4" :class="hasSecurityEvents ? 'text-red-500 dark:text-red-400' : 'text-gray-400 dark:text-gray-500'" />
              </div>
              <p class="text-[10px] font-semibold uppercase tracking-[0.1em]" :class="hasSecurityEvents ? 'text-red-500/70 dark:text-red-400/70' : 'text-gray-400 dark:text-gray-500'">Security Events</p>
              <p class="mt-1 text-2xl font-bold font-mono tabular-nums tracking-tight" :class="hasSecurityEvents ? 'text-red-600 dark:text-red-400' : 'text-gray-900 dark:text-white'">
                {{ stats?.summary?.security_events?.toLocaleString() || '0' }}
              </p>
            </div>

            <!-- Unique Users -->
            <div class="stat-card relative overflow-hidden rounded-xl border-l-[3px] border-zinc-500 bg-white dark:bg-gray-800/60 p-4">
              <div class="absolute top-2 right-2 rounded-lg bg-zinc-500/8 dark:bg-zinc-400/10 p-1.5">
                <Users class="w-4 h-4 text-zinc-500 dark:text-zinc-400" />
              </div>
              <p class="text-[10px] font-semibold uppercase tracking-[0.1em] text-gray-400 dark:text-gray-500">Unique Users</p>
              <p class="mt-1 text-2xl font-bold font-mono tabular-nums tracking-tight text-gray-900 dark:text-white">
                {{ stats?.summary?.unique_users?.toLocaleString() || '0' }}
              </p>
            </div>

            <!-- Unique IPs -->
            <div class="stat-card relative overflow-hidden rounded-xl border-l-[3px] border-violet-500 bg-white dark:bg-gray-800/60 p-4">
              <div class="absolute top-2 right-2 rounded-lg bg-violet-500/8 dark:bg-violet-400/10 p-1.5">
                <Globe class="w-4 h-4 text-violet-500 dark:text-violet-400" />
              </div>
              <p class="text-[10px] font-semibold uppercase tracking-[0.1em] text-gray-400 dark:text-gray-500">Unique IPs</p>
              <p class="mt-1 text-2xl font-bold font-mono tabular-nums tracking-tight text-gray-900 dark:text-white">
                {{ stats?.summary?.unique_ips?.toLocaleString() || '0' }}
              </p>
            </div>
          </div>

          <!-- Charts -->
          <div class="grid grid-cols-1 lg:grid-cols-5 gap-3 mb-3">
            <!-- Activity Trend (wider) -->
            <div class="lg:col-span-3 rounded-xl bg-white dark:bg-gray-800/60 border border-gray-100 dark:border-gray-700/50 p-4">
              <h3 class="text-[10px] font-semibold uppercase tracking-[0.12em] text-gray-400 dark:text-gray-500 mb-3">Activity Trend</h3>
              <div v-if="trendChartData" class="h-52">
                <Line :data="trendChartData" :options="trendChartOptions" />
              </div>
              <div v-else class="h-52 flex items-center justify-center">
                <p class="text-xs text-gray-300 dark:text-gray-600">No trend data for this period</p>
              </div>
            </div>
            <!-- Action Breakdown -->
            <div class="lg:col-span-2 rounded-xl bg-white dark:bg-gray-800/60 border border-gray-100 dark:border-gray-700/50 p-4">
              <h3 class="text-[10px] font-semibold uppercase tracking-[0.12em] text-gray-400 dark:text-gray-500 mb-3">Action Breakdown</h3>
              <div v-if="actionChartData" class="h-52">
                <Doughnut :data="actionChartData" :options="actionChartOptions" />
              </div>
              <div v-else class="h-52 flex items-center justify-center">
                <p class="text-xs text-gray-300 dark:text-gray-600">No action data for this period</p>
              </div>
            </div>
          </div>

          <!-- Entity Types (full width) -->
          <div class="rounded-xl bg-white dark:bg-gray-800/60 border border-gray-100 dark:border-gray-700/50 p-4">
            <h3 class="text-[10px] font-semibold uppercase tracking-[0.12em] text-gray-400 dark:text-gray-500 mb-3">Top Entity Types</h3>
            <div v-if="entityChartData" class="h-48">
              <Bar :data="entityChartData" :options="entityChartOptions" />
            </div>
            <div v-else class="h-48 flex items-center justify-center">
              <p class="text-xs text-gray-300 dark:text-gray-600">No entity data for this period</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ═══════════════════════════════════════════ -->
    <!-- FILTERS                                     -->
    <!-- ═══════════════════════════════════════════ -->
    <div class="rounded-xl border border-gray-200/80 dark:border-gray-700/50 bg-white dark:bg-gray-900/80 p-4 mb-4">
      <div class="flex items-center gap-2 mb-3">
        <SlidersHorizontal class="w-3.5 h-3.5 text-gray-400" />
        <span class="text-[10px] font-semibold uppercase tracking-[0.12em] text-gray-400 dark:text-gray-500">Filters</span>
      </div>
      <div class="flex flex-wrap gap-3 items-end">
        <div class="flex-1 min-w-[200px]">
          <label class="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">Search</label>
          <div class="relative">
            <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" />
            <input
              v-model="search"
              placeholder="IP address or entity type..."
              class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-white pl-8 pr-3 py-2 text-sm placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-zinc-500/30 focus:border-zinc-500/50 transition-colors"
            />
          </div>
        </div>
        <div class="w-36">
          <label class="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">Action</label>
          <select v-model="actionFilter"
            class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-zinc-500/30 focus:border-zinc-500/50 transition-colors">
            <option value="">All Actions</option>
            <option value="login">Login</option>
            <option value="logout">Logout</option>
            <option value="create">Create</option>
            <option value="update">Update</option>
            <option value="delete">Delete</option>
            <option value="export">Export</option>
            <option value="password_reset">Password Reset</option>
          </select>
        </div>
        <div class="w-36">
          <label class="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">Entity</label>
          <select v-model="entityFilter"
            class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-zinc-500/30 focus:border-zinc-500/50 transition-colors">
            <option value="">All Entities</option>
            <option value="user">User</option>
            <option value="password">Password</option>
            <option value="tenant_staff">Staff</option>
            <option value="subscription">Subscription</option>
            <option value="subscription_plan">Subscription Plan</option>
            <option value="campaign">Campaign</option>
            <option value="qr_batch">QR Batch</option>
          </select>
        </div>
        <div class="w-36">
          <label class="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">From</label>
          <input v-model="dateFrom" type="date"
            class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-zinc-500/30 focus:border-zinc-500/50 transition-colors" />
        </div>
        <div class="w-36">
          <label class="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">To</label>
          <input v-model="dateTo" type="date"
            class="w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-zinc-500/30 focus:border-zinc-500/50 transition-colors" />
        </div>
      </div>
    </div>

    <!-- ═══════════════════════════════════════════ -->
    <!-- DATA TABLE                                  -->
    <!-- ═══════════════════════════════════════════ -->
    <div class="rounded-xl border border-gray-200/80 dark:border-gray-700/50 bg-white dark:bg-gray-900/80 overflow-hidden">
      <div class="overflow-x-auto">
        <table class="min-w-full">
          <thead>
            <tr class="border-b border-gray-100 dark:border-gray-800">
              <th class="px-4 py-3 text-left text-[10px] font-semibold uppercase tracking-[0.1em] text-gray-400 dark:text-gray-500">Time</th>
              <th class="px-4 py-3 text-left text-[10px] font-semibold uppercase tracking-[0.1em] text-gray-400 dark:text-gray-500">User</th>
              <th class="px-4 py-3 text-left text-[10px] font-semibold uppercase tracking-[0.1em] text-gray-400 dark:text-gray-500">Action</th>
              <th class="px-4 py-3 text-left text-[10px] font-semibold uppercase tracking-[0.1em] text-gray-400 dark:text-gray-500">Entity</th>
              <th class="px-4 py-3 text-left text-[10px] font-semibold uppercase tracking-[0.1em] text-gray-400 dark:text-gray-500">Entity ID</th>
              <th class="px-4 py-3 text-left text-[10px] font-semibold uppercase tracking-[0.1em] text-gray-400 dark:text-gray-500">IP Address</th>
              <th class="w-10 px-2 py-3"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="7" class="px-4 py-16 text-center">
                <div class="flex flex-col items-center gap-2">
                  <div class="h-1.5 w-1.5 rounded-full bg-zinc-500 animate-pulse"></div>
                  <span class="text-xs text-gray-400">Loading audit trail...</span>
                </div>
              </td>
            </tr>
            <tr v-else-if="logs.length === 0">
              <td colspan="7" class="px-4 py-16 text-center">
                <span class="text-sm text-gray-400 dark:text-gray-500">No audit logs match your filters</span>
              </td>
            </tr>
            <template v-else v-for="log in logs" :key="log.id">
              <!-- Data Row -->
              <tr
                class="log-row group border-b border-gray-50 dark:border-gray-800/50 cursor-pointer transition-colors duration-150 hover:bg-gray-50/80 dark:hover:bg-gray-800/40"
                :class="expandedLog === log.id ? 'bg-gray-50/60 dark:bg-gray-800/30' : ''"
                @click="toggleExpand(log.id)"
              >
                <td class="px-4 py-3 text-xs font-mono text-gray-500 dark:text-gray-400 whitespace-nowrap">
                  {{ formatDateTime(log.created_at) }}
                </td>
                <td class="px-4 py-3 text-sm text-gray-800 dark:text-gray-200 font-medium">
                  {{ log.user_email || '-' }}
                </td>
                <td class="px-4 py-3">
                  <span :class="['inline-flex items-center rounded-md px-2 py-0.5 text-[11px] font-semibold ring-1 ring-inset', getActionClass(log.action_type)]">
                    {{ log.action_type }}
                  </span>
                </td>
                <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                  {{ log.entity_type }}
                </td>
                <td class="px-4 py-3 text-xs font-mono text-gray-400 dark:text-gray-500 tracking-wide">
                  {{ log.entity_id ? log.entity_id.substring(0, 8) + '...' : '-' }}
                </td>
                <td class="px-4 py-3 text-xs font-mono text-gray-400 dark:text-gray-500 tracking-wide">
                  {{ log.ip_address }}
                </td>
                <td class="px-2 py-3 text-center">
                  <ChevronDown
                    class="w-3.5 h-3.5 text-gray-300 dark:text-gray-600 transition-transform duration-200"
                    :class="expandedLog === log.id ? 'rotate-180 text-zinc-500 dark:text-zinc-400' : 'group-hover:text-gray-500 dark:group-hover:text-gray-400'"
                  />
                </td>
              </tr>

              <!-- Expansion Row -->
              <tr v-if="expandedLog === log.id" class="expand-row">
                <td colspan="7" class="p-0">
                  <div class="px-5 py-4 bg-gray-50/80 dark:bg-gray-800/40 border-b border-gray-100 dark:border-gray-800/80">
                    <div class="grid gap-4 md:grid-cols-3">
                      <!-- Details Column -->
                      <div class="space-y-3">
                        <h4 class="text-[10px] font-semibold uppercase tracking-[0.12em] text-gray-400 dark:text-gray-500">Details</h4>
                        <dl class="space-y-2.5">
                          <div>
                            <dt class="text-[10px] font-medium uppercase tracking-wider text-gray-400 dark:text-gray-500 mb-0.5">Entity ID</dt>
                            <dd class="text-xs font-mono text-gray-700 dark:text-gray-300 break-all leading-relaxed">
                              {{ log.entity_id || '-' }}
                            </dd>
                          </div>
                          <div>
                            <dt class="text-[10px] font-medium uppercase tracking-wider text-gray-400 dark:text-gray-500 mb-0.5">User Agent</dt>
                            <dd class="text-xs text-gray-700 dark:text-gray-300 break-all leading-relaxed">
                              {{ log.user_agent || '-' }}
                            </dd>
                          </div>
                        </dl>
                      </div>

                      <!-- Old Values Column -->
                      <div v-if="parseJsonValues(log.old_values)" class="space-y-3">
                        <h4 class="text-[10px] font-semibold uppercase tracking-[0.12em] text-red-500 dark:text-red-400">
                          {{ getOldValuesHeader(log.action_type) }}
                        </h4>
                        <div class="rounded-lg bg-red-50/80 dark:bg-red-950/20 border border-red-100 dark:border-red-900/30 p-3">
                          <dl class="space-y-1.5">
                            <div
                              v-for="(value, key) in parseJsonValues(log.old_values)"
                              :key="key"
                              class="flex justify-between gap-3"
                            >
                              <dt class="text-red-600/70 dark:text-red-400/70 text-xs font-medium shrink-0">{{ formatKey(key) }}</dt>
                              <dd class="text-red-700 dark:text-red-300 text-xs text-right break-all font-mono">{{ formatValue(value) }}</dd>
                            </div>
                          </dl>
                        </div>
                      </div>

                      <!-- New Values Column -->
                      <div v-if="parseJsonValues(log.new_values)" class="space-y-3">
                        <h4 class="text-[10px] font-semibold uppercase tracking-[0.12em] text-emerald-500 dark:text-emerald-400">
                          {{ getNewValuesHeader(log.action_type) }}
                        </h4>
                        <div class="rounded-lg bg-emerald-50/80 dark:bg-emerald-950/20 border border-emerald-100 dark:border-emerald-900/30 p-3">
                          <dl class="space-y-1.5">
                            <div
                              v-for="(value, key) in parseJsonValues(log.new_values)"
                              :key="key"
                              class="flex justify-between gap-3"
                            >
                              <dt class="text-emerald-600/70 dark:text-emerald-400/70 text-xs font-medium shrink-0">{{ formatKey(key) }}</dt>
                              <dd class="text-emerald-700 dark:text-emerald-300 text-xs text-right break-all font-mono">{{ formatValue(value) }}</dd>
                            </div>
                          </dl>
                        </div>
                      </div>

                      <!-- No values message -->
                      <div
                        v-if="!parseJsonValues(log.old_values) && !parseJsonValues(log.new_values)"
                        class="md:col-span-2 flex items-center"
                      >
                        <p class="text-xs text-gray-400 dark:text-gray-600">No value changes recorded for this event</p>
                      </div>
                    </div>
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div class="px-4 py-3 border-t border-gray-100 dark:border-gray-800 flex items-center justify-between">
        <p class="text-xs text-gray-400 dark:text-gray-500 font-mono tabular-nums">
          {{ logs.length }} of {{ pagination.total }} entries
        </p>
        <div class="flex gap-1.5 items-center">
          <button
            :disabled="pagination.page <= 1"
            @click="prevPage"
            class="rounded-md px-3 py-1.5 text-xs font-medium border border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
          >
            Prev
          </button>
          <span class="px-2 text-xs text-gray-400 dark:text-gray-500 font-mono tabular-nums">
            {{ pagination.page }}/{{ pagination.total_page || 1 }}
          </span>
          <button
            :disabled="pagination.page >= pagination.total_page"
            @click="nextPage"
            class="rounded-md px-3 py-1.5 text-xs font-medium border border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Dot-grid background pattern */
.dot-grid-bg {
  background-image: radial-gradient(circle, currentColor 0.8px, transparent 0.8px);
  background-size: 20px 20px;
}

/* Summary section entrance animation */
.summary-enter {
  animation: summaryReveal 0.35s ease-out;
}

@keyframes summaryReveal {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Security card pulse glow */
.security-pulse {
  animation: secPulse 2.5s ease-in-out infinite;
}

@keyframes secPulse {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(239, 68, 68, 0);
  }
  50% {
    box-shadow: 0 0 16px 0 rgba(239, 68, 68, 0.1);
  }
}

/* Stat card stagger entrance */
.stat-card {
  animation: cardIn 0.3s ease-out backwards;
}
.stat-card:nth-child(1) { animation-delay: 0.05s; }
.stat-card:nth-child(2) { animation-delay: 0.1s; }
.stat-card:nth-child(3) { animation-delay: 0.15s; }
.stat-card:nth-child(4) { animation-delay: 0.2s; }

@keyframes cardIn {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Expansion row animation */
.expand-row td {
  animation: expandReveal 0.2s ease-out;
}

@keyframes expandReveal {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

/* Table row left border accent on hover */
.log-row {
  border-left: 2px solid transparent;
  transition: border-color 0.15s, background-color 0.15s;
}
.log-row:hover {
  border-left-color: rgb(6 182 212 / 0.5);
}
</style>
