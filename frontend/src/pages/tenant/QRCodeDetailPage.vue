<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAPI } from '@/composables/useAPI'
import { useDateTime } from '@/composables/useDateTime'
import QRCode from 'qrcode'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import { ArrowLeft } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const { get } = useAPI()
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

const batchId = computed(() => route.params.batchId)
const codeId = computed(() => route.params.codeId)

const loading = ref(true)
const qrCode = ref(null)
const batch = ref(null)
const product = ref(null)
const interactions = ref([])
const qrImage = ref('')

// QR URL from backend (with /s/{base58} format for scan redirect)
const qrUrl = ref('')

// Map refs
const mapContainer = ref(null)
let map = null

// Fetch QR code details
const fetchQRCodeDetail = async () => {
  try {
    loading.value = true
    const response = await get(`/tenant/qr-batches/${batchId.value}/codes/${codeId.value}`)
    if (response.success && response.data) {
      qrCode.value = response.data.qr_code
      batch.value = response.data.batch
      product.value = response.data.product
      interactions.value = response.data.interactions || []

      // Add scan stats from response to qrCode object (calculated on-the-fly by backend)
      if (qrCode.value) {
        qrCode.value.scan_count = response.data.scan_count || 0
        qrCode.value.first_scanned_at = response.data.first_scanned_at
        qrCode.value.last_scanned_at = response.data.last_scanned_at
      }

      // Store QR URL from backend (uses /s/{base58} format for scan redirect)
      qrUrl.value = response.data.qr_url || ''

      // Generate QR code image using URL from backend
      if (qrCode.value && response.data.qr_url) {
        qrImage.value = await QRCode.toDataURL(response.data.qr_url, {
          width: 192,
          margin: 2,
          color: { dark: '#000000', light: '#ffffff' }
        })
      }
    }
  } catch (error) {
    console.error('Failed to fetch QR code details:', error)
    router.push(`/tenant/qr-batches/${batchId.value}`)
  } finally {
    loading.value = false
  }
}

// Status badges
const getStatusBadge = (status) => {
  const badges = {
    active: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
    scanned: 'bg-zinc-100 text-zinc-700 dark:bg-zinc-900/30 dark:text-zinc-400',
    blocked: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
    expired: 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-400',
  }
  return badges[status] || badges.active
}

const getCounterfeitBadge = (status) => {
  const badges = {
    valid: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
    warning: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
    counterfeit: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  }
  return badges[status] || badges.valid
}

const getTradeBadge = (status) => {
  const badges = {
    available: 'bg-zinc-100 text-zinc-700 dark:bg-zinc-900/30 dark:text-zinc-400',
    redeemed: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  }
  return badges[status] || badges.available
}

// Extract locations from interactions
const scanLocations = computed(() => {
  return interactions.value
    .filter(i => i.geolocation && i.geolocation.lat && i.geolocation.lng)
    .map(i => ({
      lat: i.geolocation.lat,
      lng: i.geolocation.lng,
      city: i.geolocation.city || 'Unknown',
      country: i.geolocation.country || '',
      category: i.interaction_category,
      date: formatDateTime(i.created_at)
    }))
})

// Initialize map
function initMap() {
  if (!mapContainer.value || map) return

  map = L.map(mapContainer.value).setView([-2.5, 118], 5) // Indonesia center

  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
    maxZoom: 19
  }).addTo(map)

  updateMarkers()
}

// Update map markers
function updateMarkers() {
  if (!map) return

  // Clear existing markers
  map.eachLayer(layer => {
    if (layer instanceof L.Marker || layer instanceof L.CircleMarker) {
      map.removeLayer(layer)
    }
  })

  if (scanLocations.value.length === 0) return

  // Add markers
  const markers = scanLocations.value.map(loc => {
    const marker = L.circleMarker([loc.lat, loc.lng], {
      radius: 8,
      fillColor: '#3f3f46',
      color: '#18181b',
      weight: 2,
      opacity: 1,
      fillOpacity: 0.8
    })

    marker.bindPopup(`
      <div class="text-sm p-1">
        <strong>${escapeHtml(loc.city)}${loc.country ? ', ' + escapeHtml(loc.country) : ''}</strong><br>
        <span class="text-gray-600">${escapeHtml(loc.category)}</span><br>
        <span class="text-gray-500 text-xs">${escapeHtml(loc.date)}</span>
      </div>
    `)

    marker.addTo(map)
    return marker
  })

  // Fit bounds to markers
  if (markers.length > 0) {
    const bounds = L.latLngBounds(scanLocations.value.map(loc => [loc.lat, loc.lng]))
    map.fitBounds(bounds, { padding: [30, 30], maxZoom: 12 })
  }
}

// Copy to clipboard
const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
  } catch (error) {
    console.error('Failed to copy:', error)
  }
}

// Download QR image
const downloadQR = () => {
  if (!qrImage.value) return
  const link = document.createElement('a')
  link.download = `qr_${qrCode.value.qr_code}.png`
  link.href = qrImage.value
  link.click()
}

// Navigation
const goBack = () => {
  router.push(`/tenant/qr-batches/${batchId.value}`)
}

// Parse user agent for device info
const parseUserAgent = (ua) => {
  if (!ua) return '-'
  if (ua.includes('Android')) return 'Android'
  if (ua.includes('iPhone') || ua.includes('iPad')) return 'iOS'
  if (ua.includes('Windows')) return 'Windows'
  if (ua.includes('Mac')) return 'Mac'
  if (ua.includes('Linux')) return 'Linux'
  return 'Other'
}

// Watch for location changes
watch(scanLocations, () => {
  if (map) updateMarkers()
}, { deep: true })

onMounted(async () => {
  await fetchQRCodeDetail()
  // Initialize map after data is loaded
  setTimeout(initMap, 100)
})

onUnmounted(() => {
  if (map) {
    map.remove()
    map = null
  }
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
            QR Code Details
          </h1>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
            {{ batch?.batch_name || '' }}
          </p>
        </div>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <div v-else-if="qrCode" class="space-y-6">
      <!-- QR Code Info Card -->
      <Card class="p-6">
        <div class="flex flex-col md:flex-row gap-6">
          <!-- QR Code Image -->
          <div class="flex-shrink-0 text-center">
            <div class="w-48 h-48 mx-auto bg-white rounded-lg flex items-center justify-center overflow-hidden border border-gray-200 dark:border-gray-600">
              <img v-if="qrImage" :src="qrImage" alt="QR Code" class="w-full h-full" />
              <div v-else class="animate-pulse bg-gray-200 w-full h-full"></div>
            </div>
            <code class="block mt-2 text-sm font-mono text-gray-900 dark:text-white">
              {{ qrCode.qr_code }}
            </code>
            <div class="flex flex-wrap justify-center gap-2 mt-3">
              <Button variant="outline" size="sm" @click="copyToClipboard(qrCode.qr_code)">
                Copy Code
              </Button>
              <Button variant="outline" size="sm" @click="copyToClipboard(qrUrl)">
                Copy URL
              </Button>
              <Button variant="outline" size="sm" @click="downloadQR">
                Download
              </Button>
            </div>
          </div>

          <!-- QR Info -->
          <div class="flex-1">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">QR Code Information</h2>
            <div class="grid grid-cols-2 md:grid-cols-3 gap-4">
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">Status</p>
                <span :class="['px-2 py-1 text-xs font-medium rounded', getStatusBadge(qrCode.status)]">
                  {{ qrCode.status }}
                </span>
              </div>
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">Counterfeit Status</p>
                <span :class="['px-2 py-1 text-xs font-medium rounded', getCounterfeitBadge(qrCode.counterfeit_status)]">
                  {{ qrCode.counterfeit_status || 'valid' }}
                </span>
              </div>
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">Total Scans</p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">{{ qrCode.scan_count || 0 }}</p>
              </div>
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">First Scanned</p>
                <p class="text-gray-900 dark:text-white">{{ formatDateTime(qrCode.first_scanned_at) }}</p>
              </div>
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">Last Scanned</p>
                <p class="text-gray-900 dark:text-white">{{ formatDateTime(qrCode.last_scanned_at) }}</p>
              </div>
              <div>
                <p class="text-sm text-gray-500 dark:text-gray-400">Created</p>
                <p class="text-gray-900 dark:text-white">{{ formatDateTime(qrCode.created_at) }}</p>
              </div>
              <div v-if="product">
                <p class="text-sm text-gray-500 dark:text-gray-400">Product</p>
                <p class="text-gray-900 dark:text-white font-medium">{{ product.product_name }}</p>
              </div>
              <div v-if="qrCode.embedded_value">
                <p class="text-sm text-gray-500 dark:text-gray-400">Embedded Value</p>
                <p class="text-gray-900 dark:text-white font-medium">{{ qrCode.value_label || qrCode.embedded_value }}</p>
              </div>
              <div v-if="qrCode.trade_status">
                <p class="text-sm text-gray-500 dark:text-gray-400">Trade Status</p>
                <span :class="['px-2 py-1 text-xs font-medium rounded', getTradeBadge(qrCode.trade_status)]">
                  {{ qrCode.trade_status }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </Card>

      <!-- Scan Locations Map -->
      <Card class="p-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Scan Locations
          <span class="text-sm font-normal text-gray-500 dark:text-gray-400 ml-2">
            ({{ scanLocations.length }} location{{ scanLocations.length !== 1 ? 's' : '' }})
          </span>
        </h2>
        <div class="relative">
          <div
            ref="mapContainer"
            style="height: 350px; width: 100%"
            class="rounded-lg overflow-hidden border border-gray-200 dark:border-gray-700"
          ></div>

          <!-- Empty state overlay -->
          <div
            v-if="scanLocations.length === 0"
            class="absolute inset-0 flex items-center justify-center bg-gray-100 dark:bg-gray-800 rounded-lg"
          >
            <div class="text-center text-gray-500 dark:text-gray-400">
              <svg class="w-12 h-12 mx-auto mb-2 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              <p>No location data available</p>
              <p class="text-sm mt-1">Scans without geolocation will not appear on the map</p>
            </div>
          </div>
        </div>
      </Card>

      <!-- Scan History Table -->
      <Card class="overflow-hidden">
        <div class="p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            Scan History
            <span class="text-sm font-normal text-gray-500 dark:text-gray-400 ml-2">
              ({{ interactions.length }} scan{{ interactions.length !== 1 ? 's' : '' }})
            </span>
          </h2>
        </div>

        <div v-if="interactions.length === 0" class="p-8 text-center">
          <svg class="w-12 h-12 text-gray-300 dark:text-gray-600 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
          </svg>
          <p class="text-gray-500 dark:text-gray-400">No scans recorded yet</p>
        </div>

        <div v-else class="overflow-x-auto">
          <table class="w-full">
            <thead class="bg-gray-50 dark:bg-gray-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Time
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Category
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Location
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Device
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  IP Address
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
              <tr v-for="interaction in interactions" :key="interaction.id" class="hover:bg-gray-50 dark:hover:bg-gray-800">
                <td class="px-4 py-3 text-sm text-gray-900 dark:text-white whitespace-nowrap">
                  {{ formatDateTime(interaction.created_at) }}
                </td>
                <td class="px-4 py-3 text-sm">
                  <span class="px-2 py-1 text-xs font-medium rounded bg-zinc-100 text-zinc-700 dark:bg-zinc-900/30 dark:text-zinc-400">
                    {{ interaction.interaction_category }}
                  </span>
                  <span v-if="interaction.interaction_subcategory" class="ml-1 text-xs text-gray-500 dark:text-gray-400">
                    / {{ interaction.interaction_subcategory }}
                  </span>
                </td>
                <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                  <template v-if="interaction.geolocation?.city">
                    {{ interaction.geolocation.city }}{{ interaction.geolocation.country ? ', ' + interaction.geolocation.country : '' }}
                  </template>
                  <template v-else>
                    -
                  </template>
                </td>
                <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                  {{ parseUserAgent(interaction.user_agent) }}
                </td>
                <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400 font-mono">
                  {{ interaction.ip_address || '-' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </Card>
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
