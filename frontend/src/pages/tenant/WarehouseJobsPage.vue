<script setup>
import { ref, onMounted } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useDateTime } from '@/composables/useDateTime'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'

const { get, post } = useAPI()
const { formatDate, formatDateTime } = useDateTime()

// State
const locations = ref([])
const history = ref([])
const stock = ref([])
const loading = ref(false)
const scanning = ref(false)
const showScanModal = ref(false)
const showStockModal = ref(false)
const typeFilter = ref('')
const locationFilter = ref('')
const activeTab = ref('history')

const pagination = ref({
  page: 1,
  limit: 20,
  total: 0,
  total_page: 0
})

const scanForm = ref({
  location_id: '',
  qr_code: '',
  movement_type: 'in',
  latitude: null,
  longitude: null
})

const lastScanResult = ref(null)
const selectedLocationForStock = ref('')

const typeOptions = [
  { value: '', label: 'All Types' },
  { value: 'in', label: 'Stock In' },
  { value: 'out', label: 'Stock Out' }
]

async function fetchLocations() {
  try {
    const response = await get('/tenant/warehouse/locations')
    if (response.success) {
      locations.value = response.data || []
      if (locations.value.length > 0 && !scanForm.value.location_id) {
        scanForm.value.location_id = locations.value[0].id
      }
    }
  } catch (error) {
    console.error('Failed to fetch warehouse locations:', error)
  }
}

async function fetchHistory() {
  loading.value = true
  try {
    const params = {
      page: pagination.value.page,
      limit: pagination.value.limit
    }
    if (typeFilter.value) params.type = typeFilter.value
    if (locationFilter.value) params.location_id = locationFilter.value

    const response = await get('/tenant/warehouse/history', params)
    if (response.success) {
      history.value = response.data.movements || []
      pagination.value = {
        ...pagination.value,
        total: response.data?.pagination?.total || 0,
        total_page: response.data?.pagination?.total_page || 0
      }
    }
  } catch (error) {
    console.error('Failed to fetch warehouse history:', error)
  } finally {
    loading.value = false
  }
}

async function fetchStock(locationId) {
  if (!locationId) return
  loading.value = true
  try {
    const response = await get(`/tenant/warehouse/stock?location_id=${locationId}`)
    if (response.success) {
      stock.value = response.data.items || []
    }
  } catch (error) {
    console.error('Failed to fetch stock:', error)
  } finally {
    loading.value = false
  }
}

function openScanModal() {
  scanForm.value = {
    location_id: locations.value.length > 0 ? locations.value[0].id : '',
    qr_code: '',
    movement_type: 'in',
    latitude: null,
    longitude: null
  }
  lastScanResult.value = null
  showScanModal.value = true

  // Try to get current location
  if (navigator.geolocation) {
    navigator.geolocation.getCurrentPosition(
      (position) => {
        scanForm.value.latitude = position.coords.latitude
        scanForm.value.longitude = position.coords.longitude
      },
      (error) => {
        console.warn('Could not get location:', error)
      }
    )
  }
}

async function submitScan() {
  if (!scanForm.value.location_id || !scanForm.value.qr_code) {
    alert('Please select a warehouse and enter QR code')
    return
  }

  scanning.value = true
  try {
    const response = await post('/tenant/warehouse/scan', scanForm.value)
    if (response.success) {
      lastScanResult.value = response.data
      scanForm.value.qr_code = '' // Clear for next scan
      fetchHistory() // Refresh history
    }
  } catch (error) {
    console.error('Failed to submit scan:', error)
    alert(error.response?.data?.message || 'Failed to submit scan')
  } finally {
    scanning.value = false
  }
}

function openStockModal() {
  selectedLocationForStock.value = locations.value.length > 0 ? locations.value[0].id : ''
  stock.value = []
  showStockModal.value = true
  if (selectedLocationForStock.value) {
    fetchStock(selectedLocationForStock.value)
  }
}

function handleLocationChange() {
  if (selectedLocationForStock.value) {
    fetchStock(selectedLocationForStock.value)
  }
}

function handleFilterChange() {
  pagination.value.page = 1
  fetchHistory()
}

function prevPage() {
  if (pagination.value.page > 1) {
    pagination.value.page--
    fetchHistory()
  }
}

function nextPage() {
  if (pagination.value.page < pagination.value.total_page) {
    pagination.value.page++
    fetchHistory()
  }
}


function getTypeClass(type) {
  return type === 'in'
    ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    : 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400'
}

onMounted(() => {
  fetchLocations()
  fetchHistory()
})
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Warehouse Jobs</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Scan QR codes for inventory in/out</p>
      </div>
      <div class="flex gap-2">
        <Button variant="outline" @click="openStockModal">
          View Stock
        </Button>
        <Button @click="openScanModal">
          <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
          </svg>
          Scan QR
        </Button>
      </div>
    </div>

    <!-- Filters -->
    <Card class="p-4 mb-6">
      <div class="flex flex-wrap gap-4 items-end">
        <div class="w-48">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Movement Type</label>
          <select
            v-model="typeFilter"
            @change="handleFilterChange"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
          >
            <option v-for="opt in typeOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>
        <div class="w-48">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Warehouse</label>
          <select
            v-model="locationFilter"
            @change="handleFilterChange"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
          >
            <option value="">All Warehouses</option>
            <option v-for="loc in locations" :key="loc.id" :value="loc.id">
              {{ loc.location_name }}
            </option>
          </select>
        </div>
      </div>
    </Card>

    <!-- History Table -->
    <Card class="overflow-hidden">
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-800">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                QR Code
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Product
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Type
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Warehouse
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Scanned By
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Time
              </th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
            <tr v-if="loading">
              <td colspan="6" class="px-6 py-12 text-center text-gray-500 dark:text-gray-400">
                Loading...
              </td>
            </tr>
            <tr v-else-if="history.length === 0">
              <td colspan="6" class="px-6 py-12 text-center text-gray-500 dark:text-gray-400">
                No movement history found
              </td>
            </tr>
            <tr
              v-else
              v-for="movement in history"
              :key="movement.id"
              class="hover:bg-gray-50 dark:hover:bg-gray-800"
            >
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="font-mono text-sm text-gray-900 dark:text-white">
                  {{ movement.qr_code?.qr_code?.substring(0, 12) || '-' }}...
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {{ movement.qr_code?.batch?.product?.product_name || '-' }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span :class="['px-2 py-1 text-xs font-medium rounded-full', getTypeClass(movement.movement_type)]">
                  {{ movement.movement_type === 'in' ? 'Stock In' : 'Stock Out' }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {{ movement.location?.location_name || '-' }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {{ movement.scanned_by_staff?.full_name || '-' }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {{ formatDate(movement.scanned_at) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div class="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
        <div class="text-sm text-gray-500 dark:text-gray-400">
          Showing {{ history.length }} of {{ pagination.total }} movements
        </div>
        <div class="flex gap-2">
          <Button variant="outline" size="sm" :disabled="pagination.page <= 1" @click="prevPage">
            Previous
          </Button>
          <span class="px-3 py-1 text-sm text-gray-600 dark:text-gray-400">
            Page {{ pagination.page }} of {{ pagination.total_page || 1 }}
          </span>
          <Button variant="outline" size="sm" :disabled="pagination.page >= pagination.total_page" @click="nextPage">
            Next
          </Button>
        </div>
      </div>
    </Card>

    <!-- Scan Modal -->
    <div v-if="showScanModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Warehouse Scan</h2>

        <!-- Last Scan Result -->
        <div v-if="lastScanResult" class="mb-4 p-3 rounded-lg" :class="lastScanResult.warnings?.length ? 'bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700' : 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-700'">
          <div class="text-sm font-medium" :class="lastScanResult.warnings?.length ? 'text-yellow-800 dark:text-yellow-200' : 'text-green-800 dark:text-green-200'">
            {{ scanForm.movement_type === 'in' ? 'Stock In' : 'Stock Out' }} recorded successfully
          </div>
          <div v-if="lastScanResult.warnings?.length" class="mt-1 text-xs text-yellow-700 dark:text-yellow-300">
            <div v-for="(warning, idx) in lastScanResult.warnings" :key="idx">{{ warning }}</div>
          </div>
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Warehouse *</label>
            <select
              v-model="scanForm.location_id"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
            >
              <option v-for="loc in locations" :key="loc.id" :value="loc.id">
                {{ loc.location_name }}
              </option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">QR Code *</label>
            <Input
              v-model="scanForm.qr_code"
              placeholder="Scan or enter QR code"
              @keyup.enter="submitScan"
              autofocus
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Movement Type *</label>
            <div class="flex gap-4">
              <label class="flex items-center">
                <input type="radio" v-model="scanForm.movement_type" value="in" class="mr-2">
                <span class="text-green-600 dark:text-green-400">Stock In</span>
              </label>
              <label class="flex items-center">
                <input type="radio" v-model="scanForm.movement_type" value="out" class="mr-2">
                <span class="text-orange-600 dark:text-orange-400">Stock Out</span>
              </label>
            </div>
          </div>

          <div v-if="scanForm.latitude && scanForm.longitude" class="text-xs text-gray-500 dark:text-gray-400">
            Location: {{ scanForm.latitude.toFixed(4) }}, {{ scanForm.longitude.toFixed(4) }}
          </div>
        </div>

        <div class="flex justify-end gap-2 mt-6">
          <Button variant="outline" @click="showScanModal = false">Close</Button>
          <Button @click="submitScan" :disabled="scanning">
            {{ scanning ? 'Scanning...' : 'Submit Scan' }}
          </Button>
        </div>
      </div>
    </div>

    <!-- Stock Modal -->
    <div v-if="showStockModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-2xl max-h-[80vh] overflow-hidden flex flex-col">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Current Stock</h2>

        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Warehouse</label>
          <select
            v-model="selectedLocationForStock"
            @change="handleLocationChange"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
          >
            <option v-for="loc in locations" :key="loc.id" :value="loc.id">
              {{ loc.location_name }}
            </option>
          </select>
        </div>

        <div class="flex-1 overflow-y-auto">
          <div v-if="loading" class="py-12 text-center text-gray-500 dark:text-gray-400">
            Loading...
          </div>
          <div v-else-if="stock.length === 0" class="py-12 text-center text-gray-500 dark:text-gray-400">
            No stock in this warehouse
          </div>
          <table v-else class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead class="bg-gray-50 dark:bg-gray-800 sticky top-0">
              <tr>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                  QR Code
                </th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                  Product
                </th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                  Batch
                </th>
                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                  Last Movement
                </th>
              </tr>
            </thead>
            <tbody class="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
              <tr v-for="item in stock" :key="item.qr_code_id">
                <td class="px-4 py-2 text-sm font-mono text-gray-900 dark:text-white">
                  {{ item.qr_code?.substring(0, 12) }}...
                </td>
                <td class="px-4 py-2 text-sm text-gray-500 dark:text-gray-400">
                  {{ item.product_name }}
                </td>
                <td class="px-4 py-2 text-sm text-gray-500 dark:text-gray-400">
                  {{ item.batch_code }}
                </td>
                <td class="px-4 py-2 text-sm text-gray-500 dark:text-gray-400">
                  {{ formatDate(item.last_movement_at) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700 flex justify-between items-center">
          <div class="text-sm text-gray-500 dark:text-gray-400">
            Total: {{ stock.length }} items in stock
          </div>
          <Button variant="outline" @click="showStockModal = false">Close</Button>
        </div>
      </div>
    </div>
  </div>
</template>
