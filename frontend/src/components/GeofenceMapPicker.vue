<script setup>
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { onClickOutside } from '@vueuse/core'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { MapPin, Search, X } from 'lucide-vue-next'
import { isTourActive, getTourNonce } from '@/composables/useTour'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({ latitude: null, longitude: null, radius_km: 25, label: '' })
  },
  height: {
    type: String,
    default: '350px'
  },
  disabled: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue'])

const mapContainer = ref(null)
const searchContainer = ref(null)
let map = null
let marker = null
let circle = null

const searchQuery = ref('')
const searchResults = ref([])
const showResults = ref(false)
const searching = ref(false)
const radiusKm = ref(props.modelValue.radius_km || 25)
const label = ref(props.modelValue.label || '')

let searchTimeout = null

function initMap() {
  if (!mapContainer.value || map) return

  const lat = props.modelValue.latitude ?? -2.5
  const lng = props.modelValue.longitude ?? 118
  const zoom = props.modelValue.latitude != null ? 10 : 5

  map = L.map(mapContainer.value).setView([lat, lng], zoom)
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
    maxZoom: 19
  }).addTo(map)

  // If initial value has coordinates, place marker and circle
  if (props.modelValue.latitude != null && props.modelValue.longitude != null) {
    placeMarker(props.modelValue.latitude, props.modelValue.longitude)
  }

  // Click on map to place marker
  if (!props.disabled) {
    map.on('click', (e) => {
      placeMarker(e.latlng.lat, e.latlng.lng)
      emitUpdate()
    })
  }
}

function placeMarker(lat, lng) {
  if (marker) {
    marker.setLatLng([lat, lng])
  } else {
    marker = L.marker([lat, lng], {
      draggable: !props.disabled
    }).addTo(map)

    if (!props.disabled) {
      marker.on('dragend', () => {
        const pos = marker.getLatLng()
        updateCircle(pos.lat, pos.lng)
        emitUpdate()
      })
    }
  }

  updateCircle(lat, lng)
}

function updateCircle(lat, lng) {
  const radiusMeters = radiusKm.value * 1000

  if (circle) {
    circle.setLatLng([lat, lng])
    circle.setRadius(radiusMeters)
  } else {
    circle = L.circle([lat, lng], {
      radius: radiusMeters,
      color: '#18181b',
      fillColor: '#18181b',
      fillOpacity: 0.15,
      weight: 2.5,
      dashArray: '8, 6'
    }).addTo(map)
  }
}

function emitUpdate() {
  if (!marker) return
  const pos = marker.getLatLng()
  emit('update:modelValue', {
    latitude: Math.round(pos.lat * 10000000) / 10000000,
    longitude: Math.round(pos.lng * 10000000) / 10000000,
    radius_km: radiusKm.value,
    label: label.value
  })
}

// City search via OpenStreetMap Nominatim API
function onSearchInput() {
  if (searchTimeout) clearTimeout(searchTimeout)
  const q = searchQuery.value.trim()
  if (q.length < 2) {
    searchResults.value = []
    showResults.value = false
    return
  }
  searching.value = true
  searchTimeout = setTimeout(() => {
    fetchSearchResults(q)
  }, 500)
}

async function fetchSearchResults(query) {
  try {
    const response = await fetch(
      `https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(query)}&format=json&limit=5&addressdetails=1`,
      { headers: { 'User-Agent': 'smartscan/1.0' } }
    )
    const data = await response.json()
    searchResults.value = data.map(r => ({
      name: r.display_name,
      lat: parseFloat(r.lat),
      lng: parseFloat(r.lon),
      shortName: r.address?.city || r.address?.town || r.address?.village || r.name || query
    }))
    showResults.value = searchResults.value.length > 0
  } catch (err) {
    searchResults.value = []
  } finally {
    searching.value = false
  }
}

function selectResult(result) {
  searchQuery.value = ''
  showResults.value = false
  label.value = result.shortName

  if (map) {
    map.flyTo([result.lat, result.lng], 11, { duration: 1 })
  }
  placeMarker(result.lat, result.lng)
  emitUpdate()
}

function clearSearch() {
  searchQuery.value = ''
  searchResults.value = []
  showResults.value = false
}

watch(radiusKm, () => {
  if (marker) {
    const pos = marker.getLatLng()
    updateCircle(pos.lat, pos.lng)
    emitUpdate()
    // Auto-fit map to show full circle
    if (circle && map) {
      map.fitBounds(circle.getBounds(), { padding: [20, 20], animate: true, duration: 0.3 })
    }
  }
})

watch(label, () => {
  emitUpdate()
})

// Watch external model changes (e.g. zone template loaded)
watch(() => props.modelValue, (val, oldVal) => {
  if (val.latitude != null && val.longitude != null && map) {
    // Only setView when coordinates actually changed (not on radius/label change)
    const latChanged = val.latitude !== oldVal?.latitude
    const lngChanged = val.longitude !== oldVal?.longitude
    if (latChanged || lngChanged) {
      placeMarker(val.latitude, val.longitude)
      map.setView([val.latitude, val.longitude], 11)
    }
  }
  if (val.radius_km !== radiusKm.value) radiusKm.value = val.radius_km
  if (val.label !== label.value) label.value = val.label
}, { deep: true })

onClickOutside(searchContainer, () => {
  showResults.value = false
})

// ── Tour support ──
async function handleTourSetValue(e) {
  if (!isTourActive()) return
  if (e.detail._nonce !== getTourNonce()) return
  const { field, value } = e.detail
  switch (field) {
    case 'geofence_search':
      searchQuery.value = value
      await fetchSearchResults(value)
      // Auto-select first result matching the query
      if (searchResults.value.length > 0) {
        selectResult(searchResults.value[0])
      }
      break
    case 'geofence_radius':
    case 'geofence_edit_radius':
      radiusKm.value = value
      break
    case 'geofence_label':
      label.value = value
      break
  }
}

onMounted(() => {
  setTimeout(initMap, 100)
  window.addEventListener('tour-set-value', handleTourSetValue)
})

onUnmounted(() => {
  if (searchTimeout) clearTimeout(searchTimeout)
  window.removeEventListener('tour-set-value', handleTourSetValue)
  if (map) {
    map.remove()
    map = null
    marker = null
    circle = null
  }
})
</script>

<template>
  <div class="space-y-3">
    <!-- City Search -->
    <div ref="searchContainer" class="relative">
      <div class="flex items-center gap-2">
        <div class="relative flex-1">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search city or location..."
            class="w-full pl-9 pr-8 py-2 border rounded-md text-sm bg-white dark:bg-gray-900 dark:border-gray-700 focus:outline-none focus:ring-2 focus:ring-zinc-500"
            :disabled="disabled"
            data-tour="geofence-search-input"
            @input="onSearchInput"
            @focus="showResults = searchResults.length > 0"
          />
          <button
            v-if="searchQuery"
            class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
            @click="clearSearch"
          >
            <X class="w-4 h-4" />
          </button>
        </div>
      </div>

      <!-- Search Results Dropdown -->
      <div
        v-if="showResults"
        class="absolute z-[1000] mt-1 w-full bg-white dark:bg-gray-800 border dark:border-gray-700 rounded-md shadow-lg max-h-48 overflow-y-auto"
      >
        <button
          v-for="result in searchResults"
          :key="result.name"
          class="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700 border-b dark:border-gray-700 last:border-0"
          @click="selectResult(result)"
        >
          <div class="flex items-center gap-2">
            <MapPin class="w-4 h-4 text-zinc-500 shrink-0" />
            <span class="truncate">{{ result.name }}</span>
          </div>
        </button>
      </div>
    </div>

    <!-- Map -->
    <div
      ref="mapContainer"
      :style="{ height: height }"
      class="rounded-lg border dark:border-gray-700 z-0"
    />

    <!-- Controls -->
    <div class="grid grid-cols-2 gap-3">
      <!-- Radius -->
      <div data-tour="geofence-radius">
        <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">
          Radius: {{ radiusKm }} km
        </label>
        <input
          v-model.number="radiusKm"
          type="range"
          min="1"
          max="500"
          step="1"
          class="w-full accent-zinc-600"
          :disabled="disabled"
        />
        <div class="flex justify-between text-xs text-gray-400">
          <span>1 km</span>
          <span>500 km</span>
        </div>
      </div>

      <!-- Label -->
      <div data-tour="geofence-zone-label">
        <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">
          Zone Label
        </label>
        <input
          v-model="label"
          type="text"
          placeholder="e.g. Semarang Metro"
          maxlength="255"
          class="w-full px-3 py-1.5 border rounded-md text-sm bg-white dark:bg-gray-900 dark:border-gray-700 focus:outline-none focus:ring-2 focus:ring-zinc-500"
          :disabled="disabled"
        />
      </div>
    </div>

    <!-- Coordinates Display -->
    <div v-if="modelValue.latitude" class="text-xs text-gray-500 dark:text-gray-400">
      Center: {{ modelValue.latitude?.toFixed(4) }}, {{ modelValue.longitude?.toFixed(4) }}
      | Radius: {{ radiusKm }} km
    </div>
  </div>
</template>
