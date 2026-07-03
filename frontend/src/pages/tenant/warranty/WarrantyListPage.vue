<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAPI } from '@/composables/useAPI'
import { useListFilter } from '@/composables/useListFilter'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import ExportTimezoneModal from '@/components/ui/ExportTimezoneModal.vue'

const router = useRouter()
const { get, getAuthHeaders } = useAPI()

const warranties = ref([])
const products = ref([])
const stats = ref(null)
const loading = ref(true)
const exporting = ref(false)

// Filters
const statusFilter = ref('all')
const productFilter = ref('')
const fromDate = ref('')
const toDate = ref('')

const { search, pagination, watchFilter, prevPage, nextPage } = useListFilter(fetchWarranties)
watchFilter(statusFilter, productFilter, fromDate, toDate)

const fetchStats = async () => {
  try {
    const response = await get('/tenant/warranties/stats')
    if (response.success) {
      stats.value = response.data
    }
  } catch (err) {
    console.error('Failed to fetch stats:', err)
  }
}

const fetchProducts = async () => {
  try {
    const response = await get('/tenant/products')
    if (response.success) {
      products.value = response.data?.products || []
    }
  } catch (err) {
    console.error('Failed to fetch products:', err)
  }
}

async function fetchWarranties() {
  loading.value = true
  try {
    const params = {
      page: pagination.value.page,
      limit: pagination.value.limit
    }
    if (statusFilter.value !== 'all') {
      params.status = statusFilter.value
    }
    if (productFilter.value) {
      params.product_id = productFilter.value
    }
    if (search.value) {
      params.search = search.value
    }
    if (fromDate.value) {
      params.from_date = fromDate.value
    }
    if (toDate.value) {
      params.to_date = toDate.value
    }

    const response = await get('/tenant/warranties', params)
    if (response.success) {
      warranties.value = response.data?.warranties || []
      const p = response.data?.pagination
      if (p) {
        pagination.value = {
          ...pagination.value,
          total: p.total || 0,
          total_page: p.total_pages || p.total_page || 0,
        }
      }
    }
  } catch (err) {
    console.error('Failed to fetch warranties:', err)
  } finally {
    loading.value = false
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

const formatDateTime = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const getStatusBadgeClass = (isExpired) => {
  if (isExpired) {
    return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
  }
  return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
}

const clearFilters = () => {
  statusFilter.value = 'all'
  productFilter.value = ''
  search.value = ''
  fromDate.value = ''
  toDate.value = ''
  pagination.value.page = 1
  fetchWarranties()
}

const hasActiveFilters = computed(() => {
  return statusFilter.value !== 'all' ||
         productFilter.value ||
         search.value ||
         fromDate.value ||
         toDate.value
})

const showExportModal = ref(false)

const exportToCSV = async (tz) => {
  showExportModal.value = false
  exporting.value = true
  try {
    const params = new URLSearchParams()
    if (statusFilter.value !== 'all') {
      params.set('status', statusFilter.value)
    }
    if (productFilter.value) {
      params.set('product_id', productFilter.value)
    }
    if (fromDate.value) {
      params.set('from_date', fromDate.value)
    }
    if (toDate.value) {
      params.set('to_date', toDate.value)
    }
    if (tz) {
      params.set('tz', tz)
    }

    const url = `/api/v1/tenant/warranties/export?${params.toString()}`
    const response = await fetch(url, {
      method: 'GET',
      headers: getAuthHeaders(),
      credentials: 'include'
    })

    if (!response.ok) {
      throw new Error('Export failed')
    }

    const blob = await response.blob()
    const downloadUrl = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = downloadUrl
    a.download = `warranties_export_${new Date().toISOString().split('T')[0]}.csv`
    document.body.appendChild(a)
    a.click()
    a.remove()
    window.URL.revokeObjectURL(downloadUrl)
  } catch (err) {
    console.error('Export failed:', err)
    alert('Failed to export warranties')
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  fetchStats()
  fetchProducts()
  fetchWarranties()
})
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Warranty Activations</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400">View all warranty registrations for your products</p>
      </div>
      <Button
        variant="outline"
        @click="showExportModal = true"
        :disabled="exporting || warranties.length === 0"
      >
        <svg v-if="!exporting" class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
        <svg v-else class="w-4 h-4 mr-2 animate-spin" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        {{ exporting ? 'Exporting...' : 'Export CSV' }}
      </Button>
    </div>

    <!-- Stats Cards -->
    <div v-if="stats" class="grid grid-cols-1 md:grid-cols-5 gap-4 mb-6">
      <Card class="p-4">
        <p class="text-sm text-gray-500 dark:text-gray-400">Total Warranties</p>
        <p class="text-2xl font-bold text-gray-900 dark:text-white">{{ stats.total_warranties }}</p>
      </Card>
      <Card class="p-4">
        <p class="text-sm text-gray-500 dark:text-gray-400">Active</p>
        <p class="text-2xl font-bold text-green-600 dark:text-green-400">{{ stats.active_warranties }}</p>
      </Card>
      <Card class="p-4">
        <p class="text-sm text-gray-500 dark:text-gray-400">Expired</p>
        <p class="text-2xl font-bold text-red-600 dark:text-red-400">{{ stats.expired_warranties }}</p>
      </Card>
      <Card class="p-4">
        <p class="text-sm text-gray-500 dark:text-gray-400">This Month</p>
        <p class="text-2xl font-bold text-zinc-600 dark:text-zinc-400">{{ stats.warranties_this_month }}</p>
      </Card>
      <Card class="p-4">
        <p class="text-sm text-gray-500 dark:text-gray-400">Expiring in 30 Days</p>
        <p class="text-2xl font-bold text-amber-600 dark:text-amber-400">{{ stats.expiring_in_30_days }}</p>
      </Card>
    </div>

    <!-- Filters -->
    <Card class="p-4 mb-6">
      <div class="flex flex-wrap gap-4 items-end">
        <!-- Status Filter -->
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">Status</label>
          <select
            v-model="statusFilter"
            class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
          >
            <option value="all">All</option>
            <option value="active">Active</option>
            <option value="expired">Expired</option>
          </select>
        </div>

        <!-- Product Filter -->
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">Product</label>
          <select
            v-model="productFilter"
            class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
          >
            <option value="">All Products</option>
            <option v-for="product in products" :key="product.id" :value="product.id">
              {{ product.product_name }}
            </option>
          </select>
        </div>

        <!-- Date Range -->
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">From Date</label>
          <input
            v-model="fromDate"
            type="date"
            class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
          />
        </div>
        <div>
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">To Date</label>
          <input
            v-model="toDate"
            type="date"
            class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
          />
        </div>

        <!-- Search -->
        <div class="flex-1 min-w-[200px]">
          <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">Search</label>
          <Input
            v-model="search"
            placeholder="Search by name or email..."
          />
        </div>

        <!-- Clear Filters -->
        <Button
          v-if="hasActiveFilters"
          variant="outline"
          @click="clearFilters"
        >
          Clear Filters
        </Button>
      </div>
    </Card>

    <!-- Loading -->
    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <!-- Warranties Table -->
    <Card v-else class="overflow-hidden">
      <div v-if="warranties.length === 0" class="p-8 text-center text-gray-500 dark:text-gray-400">
        <svg class="w-12 h-12 mx-auto mb-4 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
        </svg>
        <p class="text-lg font-medium">No warranties found</p>
        <p class="text-sm mt-1">Warranty activations will appear here when customers register their products.</p>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Customer</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Product</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Purchase Date</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Registered</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Expires</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Status</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
            <tr
              v-for="warranty in warranties"
              :key="warranty.id"
              class="hover:bg-gray-50 dark:hover:bg-gray-700/50"
            >
              <td class="px-4 py-4">
                <div>
                  <p class="font-medium text-gray-900 dark:text-white">{{ warranty.customer_name }}</p>
                  <p class="text-sm text-gray-500 dark:text-gray-400">{{ warranty.customer_email }}</p>
                  <p class="text-xs text-gray-400">{{ warranty.customer_phone }}</p>
                </div>
              </td>
              <td class="px-4 py-4">
                <div>
                  <p class="font-medium text-gray-900 dark:text-white">
                    {{ warranty.qr_code?.batch?.product?.product_name || '-' }}
                  </p>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ warranty.qr_code?.batch?.batch_name || '-' }}
                  </p>
                </div>
              </td>
              <td class="px-4 py-4 text-sm text-gray-700 dark:text-gray-300">
                {{ formatDate(warranty.purchase_date) }}
              </td>
              <td class="px-4 py-4 text-sm text-gray-700 dark:text-gray-300">
                {{ formatDateTime(warranty.activated_at) }}
              </td>
              <td class="px-4 py-4 text-sm">
                <span :class="warranty.is_expired ? 'text-red-600 dark:text-red-400' : 'text-gray-700 dark:text-gray-300'">
                  {{ formatDate(warranty.warranty_expiry_date) }}
                </span>
              </td>
              <td class="px-4 py-4">
                <span
                  class="px-2 py-1 text-xs font-medium rounded-full"
                  :class="getStatusBadgeClass(warranty.is_expired)"
                >
                  {{ warranty.is_expired ? 'Expired' : 'Active' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div v-if="pagination.total_page > 1" class="px-4 py-3 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
        <div class="text-sm text-gray-500 dark:text-gray-400">
          Showing {{ (pagination.page - 1) * pagination.limit + 1 }} to
          {{ Math.min(pagination.page * pagination.limit, pagination.total) }} of
          {{ pagination.total }} warranties
        </div>
        <div class="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            :disabled="pagination.page === 1"
            @click="prevPage"
          >
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            :disabled="pagination.page >= pagination.total_page"
            @click="nextPage"
          >
            Next
          </Button>
        </div>
      </div>
    </Card>

    <ExportTimezoneModal
      :open="showExportModal"
      title="Export Warranties"
      :loading="exporting"
      @confirm="exportToCSV"
      @cancel="showExportModal = false"
    />
  </div>
</template>
