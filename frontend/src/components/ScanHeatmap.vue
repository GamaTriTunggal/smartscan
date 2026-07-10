<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import 'leaflet.heat'

// Patch leaflet.heat: set willReadFrequently on canvas for better performance
const _origInitCanvas = L.HeatLayer.prototype._initCanvas
L.HeatLayer.prototype._initCanvas = function () {
  var canvas = this._canvas = L.DomUtil.create('canvas', 'leaflet-heatmap-layer leaflet-layer')
  var originProp = L.DomUtil.testProp(['transformOrigin', 'WebkitTransformOrigin', 'msTransformOrigin'])
  canvas.style[originProp] = '50% 50%'
  var size = this._map.getSize()
  canvas.width = size.x
  canvas.height = size.y
  var animated = this._map.options.zoomAnimation && L.Browser.any3d
  L.DomUtil.addClass(canvas, 'leaflet-zoom-' + (animated ? 'animated' : 'hide'))
  canvas.getContext('2d', { willReadFrequently: true })
  this._heat = window.simpleheat(canvas)
  this._updateOptions()
}

import { ChevronDown, Globe, MapPin, Building, Map } from 'lucide-vue-next'

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

const props = defineProps({
  locations: {
    type: Array,
    default: () => []
  },
  height: {
    type: String,
    default: '400px'
  },
  center: {
    type: Array,
    default: () => [-2.5, 118] // Indonesia center
  },
  zoom: {
    type: Number,
    default: 5
  },
  // Advanced map options
  countryFilter: {
    type: String,
    default: null
  },
  aggregateMode: {
    type: String,
    default: 'points',
    validator: v => ['points', 'country', 'province', 'city'].includes(v)
  },
  showCountrySelector: {
    type: Boolean,
    default: false
  },
  availableCountries: {
    type: Array,
    default: () => []
  },
  countryAggregates: {
    type: Array,
    default: () => []
  },
  provinceAggregates: {
    type: Array,
    default: () => []
  },
  geofenceViolations: {
    type: Array,
    default: () => []
  },
  geofenceZones: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:countryFilter', 'update:aggregateMode', 'countryClick', 'provinceClick'])

const mapContainer = ref(null)
let map = null
let heatLayer = null
let aggregateMarkers = [] // Track aggregate markers for cleanup
let pointMarkers = [] // Track counterfeit + geofence markers for cleanup
let zoneCircles = [] // Track geofence zone circles for cleanup
let zoneCenterMarkers = [] // Track geofence zone center dots for cleanup
let initTimeoutId = null // Track init timeout for cleanup

// Local state for controls
const selectedCountry = ref(props.countryFilter)
const selectedAggregate = ref(props.aggregateMode)
const showCountryDropdown = ref(false)
const showAggregateDropdown = ref(false)

// Aggregate mode options
const aggregateModes = [
  { value: 'points', label: 'Heat Points', icon: MapPin },
  { value: 'country', label: 'By Country', icon: Globe },
  { value: 'province', label: 'By Province', icon: Map },
  { value: 'city', label: 'By City', icon: Building }
]

// Watch for prop changes
watch(() => props.countryFilter, (val) => {
  selectedCountry.value = val
})

watch(() => props.aggregateMode, (val) => {
  selectedAggregate.value = val
})

// Emit changes to parent
function updateCountryFilter(code) {
  selectedCountry.value = code
  showCountryDropdown.value = false
  emit('update:countryFilter', code)
}

function updateAggregateMode(mode) {
  selectedAggregate.value = mode
  showAggregateDropdown.value = false
  emit('update:aggregateMode', mode)
}

// Handle click on aggregate region
function handleAggregateClick(type, data) {
  if (type === 'country') {
    emit('countryClick', data)
  } else if (type === 'province') {
    emit('provinceClick', data)
  }
}

// Check if any locations have counterfeit status
const hasCounterfeitLocations = computed(() => {
  return props.locations.some(loc => loc.counterfeitStatus === 'counterfeit')
})

// Check if there are geofence violations
const hasGeofenceViolations = computed(() => {
  return props.geofenceViolations.length > 0
})

// Check if there are geofence zones
const hasGeofenceZones = computed(() => {
  return props.geofenceZones.length > 0
})

// Geofence severity color mapping
function getGeofenceSeverityColor(severity) {
  switch (severity) {
    case 'critical': return '#DC2626' // red-600
    case 'high': return '#EA580C' // orange-600
    case 'medium': return '#D97706' // amber-600
    case 'low': return '#EAB308' // yellow-500
    default: return '#EAB308'
  }
}

function getGeofenceSeverityBadge(severity) {
  switch (severity) {
    case 'critical': return { bg: '#FEE2E2', text: '#991B1B', label: 'Critical' }
    case 'high': return { bg: '#FFEDD5', text: '#9A3412', label: 'High' }
    case 'medium': return { bg: '#FEF3C7', text: '#92400E', label: 'Medium' }
    case 'low': return { bg: '#FEF9C3', text: '#854D0E', label: 'Low' }
    default: return { bg: '#FEF9C3', text: '#854D0E', label: severity || 'Unknown' }
  }
}

// Get marker color based on counterfeit status
function getStatusColor(status) {
  switch (status) {
    case 'valid':
      return { fill: '#22C55E', border: '#16A34A' } // Green
    case 'warning':
      return { fill: '#3f3f46', border: '#18181b' } // Yellow/Amber
    case 'counterfeit':
      return { fill: '#A855F7', border: '#9333EA' } // Purple
    default:
      return { fill: '#3f3f46', border: '#1E40AF' } // Blue (default)
  }
}

// Get status badge class for popup
function getStatusBadge(status) {
  switch (status) {
    case 'valid':
      return { bg: '#DCFCE7', text: '#166534', label: 'Valid' }
    case 'warning':
      return { bg: '#FEF3C7', text: '#92400E', label: 'Warning' }
    case 'counterfeit':
      return { bg: '#F3E8FF', text: '#6B21A8', label: 'Counterfeit' }
    default:
      return { bg: '#E5E7EB', text: '#374151', label: status || 'Unknown' }
  }
}

// Format scan type for display
function formatScanType(scanType) {
  switch (scanType) {
    case 'product_validation':
      return 'Product Validation'
    case 'warranty_activation':
      return 'Warranty Activation'
    default:
      return scanType || 'Scan'
  }
}

// Initialize map
function initMap() {
  if (!mapContainer.value || map) return

  // Define world bounds to prevent horizontal repeating
  const worldBounds = L.latLngBounds(
    L.latLng(-85, -180), // Southwest corner
    L.latLng(85, 180)    // Northeast corner
  )

  map = L.map(mapContainer.value, {
    minZoom: 2,                    // Prevent zooming out too far
    maxBounds: worldBounds,        // Restrict panning to world bounds
    maxBoundsViscosity: 1.0        // Make bounds "sticky" (no overscroll)
  }).setView(props.center, props.zoom)

  // Add OpenStreetMap tiles
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
    maxZoom: 19,
    noWrap: true                   // Prevent tile layer from wrapping horizontally
  }).addTo(map)

  // Initialize empty heat layer
  updateHeatmap()
}

// Clear aggregate markers
function clearAggregateMarkers() {
  aggregateMarkers.forEach(marker => {
    if (map) map.removeLayer(marker)
  })
  aggregateMarkers = []
}

// Clear point markers (counterfeit + geofence)
function clearPointMarkers() {
  pointMarkers.forEach(marker => {
    if (map) map.removeLayer(marker)
  })
  pointMarkers = []
}

// Clear zone circles and center dots
function clearZoneMarkers() {
  zoneCircles.forEach(c => { if (map) map.removeLayer(c) })
  zoneCircles = []
  zoneCenterMarkers.forEach(m => { if (map) map.removeLayer(m) })
  zoneCenterMarkers = []
}

// Get color based on scan count (for aggregate markers)
function getAggregateColor(count, maxCount) {
  const ratio = count / maxCount
  if (ratio > 0.75) return { fill: '#EF4444', border: '#DC2626' } // Red
  if (ratio > 0.5) return { fill: '#F59E0B', border: '#D97706' } // Orange
  if (ratio > 0.25) return { fill: '#22C55E', border: '#16A34A' } // Green
  return { fill: '#3f3f46', border: '#1E40AF' } // Blue
}

// Get marker size based on scan count
function getAggregateSize(count, maxCount) {
  const minSize = 15
  const maxSize = 40
  const ratio = count / maxCount
  return minSize + (maxSize - minSize) * Math.sqrt(ratio)
}

// Update heatmap data
function updateHeatmap() {
  if (!map) return

  // Remove existing heat layer
  if (heatLayer) {
    map.removeLayer(heatLayer)
  }

  // Clear existing markers
  clearAggregateMarkers()
  clearPointMarkers()
  clearZoneMarkers()

  // Draw geofence zone circles (underneath all other layers)
  props.geofenceZones.forEach(zone => {
    if (zone.lat == null || zone.lng == null) return

    const circle = L.circle([zone.lat, zone.lng], {
      radius: zone.radius_km * 1000,
      color: '#18181b',
      fillColor: '#18181b',
      fillOpacity: 0.10,
      weight: 2,
      dashArray: '8, 6'
    })

    const popupHtml = `<div style="min-width:160px;font-family:system-ui,sans-serif;">
      <div style="font-weight:600;font-size:13px;margin-bottom:6px;">Distribution Zone</div>
      ${zone.batch_name ? `<div style="font-size:12px;color:#6B7280;margin-bottom:4px;">Batch: ${escapeHtml(zone.batch_name)}</div>` : ''}
      ${zone.label ? `<div style="font-size:12px;color:#6B7280;margin-bottom:4px;">Label: ${escapeHtml(zone.label)}</div>` : ''}
      <div style="font-size:12px;color:#6B7280;">Radius: ${zone.radius_km} km</div>
    </div>`

    circle.bindPopup(popupHtml)
    circle.addTo(map)
    zoneCircles.push(circle)

    const centerDot = L.circleMarker([zone.lat, zone.lng], {
      radius: 5, fillColor: '#18181b', color: '#27272a', weight: 2, fillOpacity: 0.8
    })
    centerDot.bindPopup(popupHtml)
    centerDot.addTo(map)
    zoneCenterMarkers.push(centerDot)
  })

  // Check if we should display aggregated data
  const showAggregates = selectedAggregate.value !== 'points'

  if (showAggregates) {
    // Display aggregated markers
    let aggregateData = []
    let aggregateType = selectedAggregate.value

    if (selectedAggregate.value === 'country' && props.countryAggregates.length > 0) {
      aggregateData = props.countryAggregates
    } else if ((selectedAggregate.value === 'province' || selectedAggregate.value === 'city') && props.provinceAggregates.length > 0) {
      aggregateData = props.provinceAggregates
    }

    if (aggregateData.length > 0) {
      const maxCount = Math.max(...aggregateData.map(d => d.total_scans || d.count || 1))

      aggregateData.forEach(item => {
        // Skip items without coordinates
        if (!item.latitude || !item.longitude) return

        const count = item.total_scans || item.count || 0
        const colors = getAggregateColor(count, maxCount)
        const size = getAggregateSize(count, maxCount)

        const marker = L.circleMarker([item.latitude, item.longitude], {
          radius: size,
          fillColor: colors.fill,
          color: colors.border,
          weight: 2,
          opacity: 0.9,
          fillOpacity: 0.7
        })

        // Build popup content - escape user data to prevent XSS
        const label = item.country_name || item.province || item.city || item.name || 'Unknown'
        const popupContent = `
          <div style="min-width: 160px; font-family: system-ui, sans-serif;">
            <div style="font-weight: 600; font-size: 14px; margin-bottom: 8px;">${escapeHtml(label)}</div>
            <div style="display: flex; flex-direction: column; gap: 4px;">
              <div style="display: flex; justify-content: space-between; font-size: 12px;">
                <span style="color: #6B7280;">Total Scans</span>
                <span style="font-weight: 500;">${count.toLocaleString()}</span>
              </div>
              ${item.validation !== undefined ? `
              <div style="display: flex; justify-content: space-between; font-size: 12px;">
                <span style="color: #6B7280;">Validation</span>
                <span>${(item.validation || 0).toLocaleString()}</span>
              </div>` : ''}
              ${item.warranty !== undefined ? `
              <div style="display: flex; justify-content: space-between; font-size: 12px;">
                <span style="color: #6B7280;">Warranty</span>
                <span>${(item.warranty || 0).toLocaleString()}</span>
              </div>` : ''}
            </div>
            ${aggregateType === 'country' ? `
            <div style="margin-top: 8px; padding-top: 8px; border-top: 1px solid #E5E7EB;">
              <button onclick="window.dispatchEvent(new CustomEvent('heatmap-drill', {detail: {type: 'country', code: '${item.country_code}', name: '${label}'}}))"
                      style="color: #18181b; font-size: 12px; text-decoration: none; cursor: pointer; background: none; border: none; padding: 0;">
                View Details →
              </button>
            </div>` : ''}
          </div>
        `

        marker.bindPopup(popupContent)
        marker.on('click', () => {
          if (aggregateType === 'country' && item.country_code) {
            handleAggregateClick('country', { code: item.country_code, name: label })
          } else if (aggregateType === 'province' && item.province) {
            handleAggregateClick('province', { province: item.province, name: label })
          }
        })

        marker.addTo(map)
        aggregateMarkers.push(marker)
      })

      // Fit bounds to aggregate markers
      if (aggregateData.length > 1) {
        const validCoords = aggregateData.filter(d => d.latitude && d.longitude)
        if (validCoords.length > 0) {
          const bounds = L.latLngBounds(validCoords.map(d => [d.latitude, d.longitude]))
          map.fitBounds(bounds, { padding: [50, 50] })
        }
      }
    }
    return // Don't show heat layer when in aggregate mode
  }

  // Standard heat layer mode
  // Filter valid locations and create heat data
  const heatData = props.locations
    .filter(loc => loc.latitude && loc.longitude)
    .map(loc => [loc.latitude, loc.longitude, loc.intensity || 1])

  if (heatData.length > 0) {
    // Create heat layer with enhanced visibility
    heatLayer = L.heatLayer(heatData, {
      radius: 20,        // Larger area per point
      blur: 20,          // More spread
      maxZoom: 10,
      max: 1.0,
      minOpacity: 0.4,   // Minimum visibility
      gradient: {
        0.0: 'blue',
        0.25: '#71717a',
        0.5: 'lime',
        0.75: 'yellow',
        1.0: 'red'
      }
    }).addTo(map)

    // Auto-fit bounds if there's data
    if (heatData.length > 1) {
      const bounds = L.latLngBounds(heatData.map(d => [d[0], d[1]]))
      map.fitBounds(bounds, { padding: [50, 50] })
    } else if (heatData.length === 1) {
      map.setView([heatData[0][0], heatData[0][1]], 10)
    }
  }

  // Add markers only for counterfeit scans
  props.locations
    .filter(loc => loc.latitude != null && loc.longitude != null && loc.counterfeitStatus === 'counterfeit')
    .forEach(loc => {
      const colors = getStatusColor(loc.counterfeitStatus)
      const marker = L.circleMarker([loc.latitude, loc.longitude], {
        radius: 5,
        fillColor: colors.fill,
        color: colors.border,
        weight: 2,
        opacity: 0.8,
        fillOpacity: 0.6
      })

      // Build enhanced popup content
      const badge = getStatusBadge(loc.counterfeitStatus)
      const scanTypeLabel = formatScanType(loc.scanType)

      let popupContent = `
        <div style="min-width: 180px; font-family: system-ui, sans-serif;">
          ${loc.name ? `<div style="font-weight: 600; font-size: 14px; margin-bottom: 4px;">${escapeHtml(loc.name)}</div>` : ''}
          <div style="font-size: 12px; color: #6B7280; margin-bottom: 6px;">${escapeHtml(scanTypeLabel)}</div>
          <div style="margin-bottom: 6px;">
            <span style="display: inline-block; padding: 2px 8px; border-radius: 9999px; font-size: 11px; font-weight: 500; background: ${badge.bg}; color: ${badge.text};">
              ${escapeHtml(badge.label)}
            </span>
          </div>
          ${loc.date ? `<div style="font-size: 11px; color: #9CA3AF;">${escapeHtml(loc.date)}</div>` : ''}
          ${loc.batchId && loc.qrCodeId ? `<div style="margin-top: 8px; padding-top: 8px; border-top: 1px solid #E5E7EB;"><a href="/tenant/qr-batches/${escapeHtml(loc.batchId)}/codes/${escapeHtml(loc.qrCodeId)}" style="color: #18181b; font-size: 12px; text-decoration: none;">View QR Details →</a></div>` : ''}
        </div>
      `

      marker.bindPopup(popupContent)
      marker.addTo(map)
      pointMarkers.push(marker)
    })

  // Add geofence violation markers (triangle shape)
  props.geofenceViolations.forEach(v => {
    if (v.lat == null || v.lng == null) return
    const color = getGeofenceSeverityColor(v.severity)
    const icon = L.divIcon({
      html: `<svg width="14" height="14" viewBox="0 0 14 14"><polygon points="7,1 13,13 1,13" fill="${color}" stroke="white" stroke-width="1.5"/></svg>`,
      className: '',
      iconSize: [14, 14],
      iconAnchor: [7, 13]
    })
    const marker = L.marker([v.lat, v.lng], { icon })

    const badge = getGeofenceSeverityBadge(v.severity)
    const popupContent = `
      <div style="min-width: 180px; font-family: system-ui, sans-serif;">
        <div style="font-weight: 600; font-size: 13px; margin-bottom: 6px;">Geofence Violation</div>
        <div style="margin-bottom: 6px;">
          <span style="display: inline-block; padding: 2px 8px; border-radius: 9999px; font-size: 11px; font-weight: 500; background: ${badge.bg}; color: ${badge.text};">
            ${escapeHtml(badge.label)}
          </span>
        </div>
        ${v.batch_name ? `<div style="font-size: 12px; color: #6B7280; margin-bottom: 4px;">Batch: ${escapeHtml(v.batch_name)}</div>` : ''}
        ${v.created_at ? `<div style="font-size: 11px; color: #9CA3AF;">${escapeHtml(new Date(v.created_at).toLocaleString())}</div>` : ''}
        ${v.batch_id ? `<div style="margin-top: 8px; padding-top: 8px; border-top: 1px solid #E5E7EB;"><a href="/tenant/qr-batches/${escapeHtml(v.batch_id)}" style="color: #EA580C; font-size: 12px; text-decoration: none;">View Batch →</a></div>` : ''}
      </div>
    `
    marker.bindPopup(popupContent)
    marker.addTo(map)
    pointMarkers.push(marker)
  })
}

// Watch for location and aggregate changes
watch(() => props.locations, () => {
  updateHeatmap()
}, { deep: true })

watch(() => props.countryAggregates, () => {
  updateHeatmap()
}, { deep: true })

watch(() => props.provinceAggregates, () => {
  updateHeatmap()
}, { deep: true })

watch(() => props.geofenceViolations, () => {
  updateHeatmap()
}, { deep: true })

watch(() => props.geofenceZones, () => {
  updateHeatmap()
}, { deep: true })

watch(() => selectedAggregate.value, () => {
  updateHeatmap()
})

onMounted(() => {
  // Small delay to ensure container is rendered
  initTimeoutId = setTimeout(initMap, 100)
})

onUnmounted(() => {
  // Clear init timeout if still pending
  if (initTimeoutId) {
    clearTimeout(initTimeoutId)
    initTimeoutId = null
  }
  clearPointMarkers()
  clearAggregateMarkers()
  clearZoneMarkers()
  if (map) {
    map.remove()
    map = null
  }
})
</script>

<template>
  <div class="scan-heatmap">
    <!-- Aggregate / Country Controls -->
    <div v-if="showCountrySelector || availableCountries.length > 0" class="flex flex-wrap gap-3 mb-4">
      <!-- Country Filter Dropdown -->
      <div v-if="showCountrySelector || availableCountries.length > 0" class="relative">
        <button
          @click="showCountryDropdown = !showCountryDropdown"
          class="flex items-center gap-2 px-3 py-2 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
        >
          <Globe class="w-4 h-4 text-gray-500" />
          <span>{{ selectedCountry ? availableCountries.find(c => c.code === selectedCountry)?.name || selectedCountry : 'All Countries' }}</span>
          <ChevronDown class="w-4 h-4 text-gray-400" />
        </button>
        <div
          v-if="showCountryDropdown"
          class="absolute top-full left-0 mt-1 w-48 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg z-50 max-h-60 overflow-y-auto"
        >
          <button
            @click="updateCountryFilter(null)"
            class="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700"
            :class="{ 'bg-zinc-50 dark:bg-zinc-900/20 text-zinc-700 dark:text-zinc-300': !selectedCountry }"
          >
            All Countries
          </button>
          <button
            v-for="country in availableCountries"
            :key="country.code"
            @click="updateCountryFilter(country.code)"
            class="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700 flex justify-between items-center"
            :class="{ 'bg-zinc-50 dark:bg-zinc-900/20 text-zinc-700 dark:text-zinc-300': selectedCountry === country.code }"
          >
            <span>{{ country.name }}</span>
            <span class="text-xs text-gray-500">{{ country.count?.toLocaleString() }}</span>
          </button>
        </div>
      </div>

      <!-- Aggregate Mode Toggle -->
      <div class="relative">
        <button
          @click="showAggregateDropdown = !showAggregateDropdown"
          class="flex items-center gap-2 px-3 py-2 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
        >
          <component :is="aggregateModes.find(m => m.value === selectedAggregate)?.icon || MapPin" class="w-4 h-4 text-gray-500" />
          <span>{{ aggregateModes.find(m => m.value === selectedAggregate)?.label || 'Heat Points' }}</span>
          <ChevronDown class="w-4 h-4 text-gray-400" />
        </button>
        <div
          v-if="showAggregateDropdown"
          class="absolute top-full left-0 mt-1 w-40 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg z-50"
        >
          <button
            v-for="mode in aggregateModes"
            :key="mode.value"
            @click="updateAggregateMode(mode.value)"
            class="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
            :class="{ 'bg-zinc-50 dark:bg-zinc-900/20 text-zinc-700 dark:text-zinc-300': selectedAggregate === mode.value }"
          >
            <component :is="mode.icon" class="w-4 h-4" />
            <span>{{ mode.label }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Map Container -->
    <div
      ref="mapContainer"
      :style="{ height: height, width: '100%' }"
      class="rounded-lg overflow-hidden border border-gray-200 dark:border-gray-700"
    ></div>

    <!-- Legend (color bar showing density gradient) - only for heat points mode -->
    <div v-if="selectedAggregate === 'points'" class="flex items-center justify-center gap-2 mt-3 text-xs text-gray-600 dark:text-gray-400">
      <span>Low</span>
      <div
        class="h-3 w-32 rounded-full"
        style="background: linear-gradient(to right, #3f3f46, #3f3f46, #84CC16, #FACC15, #EF4444)"
      ></div>
      <span>High</span>
    </div>

    <!-- Aggregate legend - for aggregate modes -->
    <div v-if="selectedAggregate !== 'points'" class="flex items-center justify-center gap-4 mt-3 text-xs text-gray-600 dark:text-gray-400">
      <div class="flex items-center gap-1">
        <span class="w-3 h-3 rounded-full bg-zinc-500"></span>
        <span>Low</span>
      </div>
      <div class="flex items-center gap-1">
        <span class="w-3 h-3 rounded-full bg-green-500"></span>
        <span>Medium</span>
      </div>
      <div class="flex items-center gap-1">
        <span class="w-3 h-3 rounded-full bg-orange-500"></span>
        <span>High</span>
      </div>
      <div class="flex items-center gap-1">
        <span class="w-3 h-3 rounded-full bg-red-500"></span>
        <span>Very High</span>
      </div>
      <span class="text-gray-400 ml-2">(Circle size = scan volume)</span>
    </div>

    <!-- Counterfeit marker legend - only show if there are counterfeit locations -->
    <div
      v-if="hasCounterfeitLocations && selectedAggregate === 'points'"
      class="flex items-center justify-center gap-2 mt-2 text-xs text-gray-600 dark:text-gray-400"
    >
      <span class="w-3 h-3 rounded-full bg-purple-500"></span>
      <span>Counterfeit Detected</span>
    </div>

    <!-- Geofence zone legend -->
    <div
      v-if="hasGeofenceZones && selectedAggregate === 'points'"
      class="flex items-center justify-center gap-2 mt-2 text-xs text-gray-600 dark:text-gray-400"
    >
      <span class="w-3 h-3 rounded-full border-2 border-dashed" style="border-color: #18181b; background: rgba(13,148,136,0.15);"></span>
      <span>Distribution Zone</span>
    </div>

    <!-- Geofence violation marker legend -->
    <div
      v-if="hasGeofenceViolations && selectedAggregate === 'points'"
      class="flex items-center justify-center gap-4 mt-2 text-xs text-gray-600 dark:text-gray-400"
    >
      <span class="text-gray-500">Geofence:</span>
      <div class="flex items-center gap-1">
        <svg width="10" height="10" viewBox="0 0 14 14"><polygon points="7,1 13,13 1,13" fill="#DC2626" stroke="white" stroke-width="1.5"/></svg>
        <span>Critical</span>
      </div>
      <div class="flex items-center gap-1">
        <svg width="10" height="10" viewBox="0 0 14 14"><polygon points="7,1 13,13 1,13" fill="#EA580C" stroke="white" stroke-width="1.5"/></svg>
        <span>High</span>
      </div>
      <div class="flex items-center gap-1">
        <svg width="10" height="10" viewBox="0 0 14 14"><polygon points="7,1 13,13 1,13" fill="#D97706" stroke="white" stroke-width="1.5"/></svg>
        <span>Medium</span>
      </div>
      <div class="flex items-center gap-1">
        <svg width="10" height="10" viewBox="0 0 14 14"><polygon points="7,1 13,13 1,13" fill="#EAB308" stroke="white" stroke-width="1.5"/></svg>
        <span>Low</span>
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-if="locations.length === 0 && countryAggregates.length === 0 && provinceAggregates.length === 0"
      class="absolute inset-0 flex items-center justify-center bg-gray-100 dark:bg-gray-800 rounded-lg"
    >
      <div class="text-center text-gray-500 dark:text-gray-400">
        <svg class="w-12 h-12 mx-auto mb-2 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
        <p>No location data available</p>
      </div>
    </div>

    <!-- Click outside to close dropdowns -->
    <div
      v-if="showCountryDropdown || showAggregateDropdown"
      class="fixed inset-0 z-40"
      @click="showCountryDropdown = false; showAggregateDropdown = false"
    ></div>
  </div>
</template>

<style>
.scan-heatmap {
  position: relative;
}

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
