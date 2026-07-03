<script setup>
import { ref, onMounted, computed } from 'vue'
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
const loading = ref(false)
const scanning = ref(false)
const showScanModal = ref(false)
const showCorrectionModal = ref(false)
const statusFilter = ref('')
const locationFilter = ref('')

const pagination = ref({
  page: 1,
  limit: 20,
  total: 0,
  total_pages: 0
})

const scanForm = ref({
  location_id: '',
  qr_code: '',
  status: 'pass',
  latitude: null,
  longitude: null
})

const correctionForm = ref({
  scan_id: '',
  qr_code: '',
  status: 'pass',
  location_id: '',
  latitude: null,
  longitude: null,
  correction_reason: ''
})

const lastScanResult = ref(null)

const statusOptions = [
  { value: '', label: 'All Status' },
  { value: 'pass', label: 'Pass' },
  { value: 'failed', label: 'Failed' }
]

async function fetchLocations() {
  try {
    const response = await get('/tenant/qc/locations')
    if (response.success) {
      locations.value = response.data || []
      if (locations.value.length > 0 && !scanForm.value.location_id) {
        scanForm.value.location_id = locations.value[0].id
      }
    }
  } catch (error) {
    console.error('Failed to fetch QC locations:', error)
  }
}

async function fetchHistory() {
  loading.value = true
  try {
    const params = {
      page: pagination.value.page,
      limit: pagination.value.limit
    }
    if (statusFilter.value) params.status = statusFilter.value
    if (locationFilter.value) params.location_id = locationFilter.value

    const response = await get('/tenant/qc/history', params)
    if (response.success) {
      history.value = response.data.scans || []
      pagination.value = {
        ...pagination.value,
        total: response.data.total,
        total_pages: response.data.total_pages
      }
    }
  } catch (error) {
    console.error('Failed to fetch QC history:', error)
  } finally {
    loading.value = false
  }
}

function openScanModal() {
  scanForm.value = {
    location_id: locations.value.length > 0 ? locations.value[0].id : '',
    qr_code: '',
    status: 'pass',
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
    alert('Please select a location and enter QR code')
    return
  }

  scanning.value = true
  try {
    const response = await post('/tenant/qc/scan', scanForm.value)
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

function openCorrectionModal(scan) {
  correctionForm.value = {
    scan_id: scan.id,
    qr_code: scan.qr_code?.qr_code || '',
    status: scan.qc_status === 'pass' ? 'failed' : 'pass', // Toggle status
    location_id: scan.location_id,
    latitude: null,
    longitude: null,
    correction_reason: ''
  }
  showCorrectionModal.value = true

  // Try to get current location
  if (navigator.geolocation) {
    navigator.geolocation.getCurrentPosition(
      (position) => {
        correctionForm.value.latitude = position.coords.latitude
        correctionForm.value.longitude = position.coords.longitude
      },
      (error) => {
        console.warn('Could not get location:', error)
      }
    )
  }
}

async function submitCorrection() {
  scanning.value = true
  try {
    const response = await post('/tenant/qc/scan', {
      location_id: correctionForm.value.location_id,
      qr_code: correctionForm.value.qr_code,
      status: correctionForm.value.status,
      latitude: correctionForm.value.latitude,
      longitude: correctionForm.value.longitude,
      is_correction: true,
      corrects_scan_id: correctionForm.value.scan_id,
      correction_reason: correctionForm.value.correction_reason
    })
    if (response.success) {
      showCorrectionModal.value = false
      fetchHistory()
    }
  } catch (error) {
    console.error('Failed to submit correction:', error)
    alert(error.response?.data?.message || 'Failed to submit correction')
  } finally {
    scanning.value = false
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
  if (pagination.value.page < pagination.value.total_pages) {
    pagination.value.page++
    fetchHistory()
  }
}


function getStatusClass(status) {
  return status === 'pass'
    ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
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
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">QC Jobs</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Scan QR codes for quality control</p>
      </div>
      <Button @click="openScanModal">
        <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
        </svg>
        Scan QR
      </Button>
    </div>

    <!-- Filters -->
    <Card class="p-4 mb-6">
      <div class="flex flex-wrap gap-4 items-end">
        <div class="w-48">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Status</label>
          <select
            v-model="statusFilter"
            @change="handleFilterChange"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
          >
            <option v-for="opt in statusOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>
        <div class="w-48">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Location</label>
          <select
            v-model="locationFilter"
            @change="handleFilterChange"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
          >
            <option value="">All Locations</option>
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
                Status
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Location
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Scanned By
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Time
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
            <tr v-if="loading">
              <td colspan="7" class="px-6 py-12 text-center text-gray-500 dark:text-gray-400">
                Loading...
              </td>
            </tr>
            <tr v-else-if="history.length === 0">
              <td colspan="7" class="px-6 py-12 text-center text-gray-500 dark:text-gray-400">
                No scan history found
              </td>
            </tr>
            <tr
              v-else
              v-for="scan in history"
              :key="scan.id"
              :class="[
                scan.is_correction ? 'bg-yellow-50 dark:bg-yellow-900/10' : '',
                'hover:bg-gray-50 dark:hover:bg-gray-800'
              ]"
            >
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="font-mono text-sm text-gray-900 dark:text-white">
                  {{ scan.qr_code?.qr_code?.substring(0, 12) || '-' }}...
                </div>
                <div v-if="scan.is_correction" class="text-xs text-yellow-600 dark:text-yellow-400">
                  Correction
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {{ scan.qr_code?.batch?.product?.product_name || '-' }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span :class="['px-2 py-1 text-xs font-medium rounded-full', getStatusClass(scan.qc_status)]">
                  {{ scan.qc_status === 'pass' ? 'Pass' : 'Failed' }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {{ scan.location?.location_name || '-' }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {{ scan.scanned_by_staff?.full_name || '-' }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {{ formatDate(scan.scanned_at) }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm">
                <button
                  v-if="!scan.is_correction"
                  @click="openCorrectionModal(scan)"
                  class="text-xs text-zinc-600 dark:text-zinc-400 hover:underline"
                >
                  Correct
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div class="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
        <div class="text-sm text-gray-500 dark:text-gray-400">
          Showing {{ history.length }} of {{ pagination.total }} scans
        </div>
        <div class="flex gap-2">
          <Button variant="outline" size="sm" :disabled="pagination.page <= 1" @click="prevPage">
            Previous
          </Button>
          <span class="px-3 py-1 text-sm text-gray-600 dark:text-gray-400">
            Page {{ pagination.page }} of {{ pagination.total_pages || 1 }}
          </span>
          <Button variant="outline" size="sm" :disabled="pagination.page >= pagination.total_pages" @click="nextPage">
            Next
          </Button>
        </div>
      </div>
    </Card>

    <!-- Scan Modal -->
    <div v-if="showScanModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">QC Scan</h2>

        <!-- Last Scan Result -->
        <div v-if="lastScanResult" class="mb-4 p-3 rounded-lg" :class="lastScanResult.warnings?.length ? 'bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700' : 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-700'">
          <div class="text-sm font-medium" :class="lastScanResult.warnings?.length ? 'text-yellow-800 dark:text-yellow-200' : 'text-green-800 dark:text-green-200'">
            Scan recorded successfully
          </div>
          <div v-if="lastScanResult.warnings?.length" class="mt-1 text-xs text-yellow-700 dark:text-yellow-300">
            <div v-for="(warning, idx) in lastScanResult.warnings" :key="idx">{{ warning }}</div>
          </div>
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">QC Area *</label>
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
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Status *</label>
            <div class="flex gap-4">
              <label class="flex items-center">
                <input type="radio" v-model="scanForm.status" value="pass" class="mr-2">
                <span class="text-green-600 dark:text-green-400">Pass</span>
              </label>
              <label class="flex items-center">
                <input type="radio" v-model="scanForm.status" value="failed" class="mr-2">
                <span class="text-red-600 dark:text-red-400">Failed</span>
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

    <!-- Correction Modal -->
    <div v-if="showCorrectionModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Correct QC Scan</h2>

        <div class="space-y-4">
          <div class="p-3 bg-gray-100 dark:bg-gray-700 rounded-lg">
            <div class="text-sm text-gray-600 dark:text-gray-400">QR Code:</div>
            <div class="font-mono text-sm text-gray-900 dark:text-white">{{ correctionForm.qr_code }}</div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">New Status *</label>
            <div class="flex gap-4">
              <label class="flex items-center">
                <input type="radio" v-model="correctionForm.status" value="pass" class="mr-2">
                <span class="text-green-600 dark:text-green-400">Pass</span>
              </label>
              <label class="flex items-center">
                <input type="radio" v-model="correctionForm.status" value="failed" class="mr-2">
                <span class="text-red-600 dark:text-red-400">Failed</span>
              </label>
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Reason for Correction
              <span class="text-xs text-gray-500">(required after 5 minutes)</span>
            </label>
            <textarea
              v-model="correctionForm.correction_reason"
              rows="3"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
              placeholder="Explain why this correction is needed..."
            ></textarea>
          </div>
        </div>

        <div class="flex justify-end gap-2 mt-6">
          <Button variant="outline" @click="showCorrectionModal = false">Cancel</Button>
          <Button @click="submitCorrection" :disabled="scanning">
            {{ scanning ? 'Submitting...' : 'Submit Correction' }}
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>
