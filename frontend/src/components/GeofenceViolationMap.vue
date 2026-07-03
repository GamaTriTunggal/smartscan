<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

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
  violations: {
    type: Array,
    default: () => []
  },
  zones: {
    type: Array,
    default: () => []
  },
  height: {
    type: String,
    default: '400px'
  }
})

const mapContainer = ref(null)
let map = null
let violationMarkers = []
let zoneCircles = []
let zoneCenterMarkers = []
let initTimeoutId = null

function getGeofenceSeverityColor(severity) {
  switch (severity) {
    case 'critical': return '#DC2626'
    case 'high': return '#EA580C'
    case 'medium': return '#D97706'
    case 'low': return '#EAB308'
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

function clearMarkers() {
  violationMarkers.forEach(m => { if (map) map.removeLayer(m) })
  violationMarkers = []
  zoneCircles.forEach(c => { if (map) map.removeLayer(c) })
  zoneCircles = []
  zoneCenterMarkers.forEach(m => { if (map) map.removeLayer(m) })
  zoneCenterMarkers = []
}

function renderData() {
  if (!map) return
  clearMarkers()

  // 1. Draw zone circles first (underneath violations)
  props.zones.forEach(zone => {
    if (zone.lat == null || zone.lng == null || !zone.radius_km) return

    const circle = L.circle([zone.lat, zone.lng], {
      radius: zone.radius_km * 1000,
      color: '#18181b',
      fillColor: '#18181b',
      fillOpacity: 0.10,
      weight: 2,
      dashArray: '8, 6'
    })

    const zonePopup = `
      <div style="min-width: 160px; font-family: system-ui, sans-serif;">
        <div style="font-weight: 600; font-size: 13px; margin-bottom: 6px;">Distribution Zone</div>
        <div style="font-size: 12px; color: #6B7280; margin-bottom: 4px;">Batch: ${escapeHtml(zone.batch_name)}</div>
        ${zone.label ? `<div style="font-size: 12px; color: #6B7280; margin-bottom: 4px;">Label: ${escapeHtml(zone.label)}</div>` : ''}
        <div style="font-size: 12px; color: #6B7280;">Radius: ${zone.radius_km} km</div>
      </div>
    `
    circle.bindPopup(zonePopup)
    circle.addTo(map)
    zoneCircles.push(circle)

    // Small teal dot at zone center
    const centerMarker = L.circleMarker([zone.lat, zone.lng], {
      radius: 5,
      fillColor: '#18181b',
      color: '#27272a',
      weight: 2,
      fillOpacity: 0.8
    })
    centerMarker.bindPopup(zonePopup)
    centerMarker.addTo(map)
    zoneCenterMarkers.push(centerMarker)
  })

  // 2. Draw violation triangle markers (on top)
  props.violations.forEach(v => {
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
        ${v.batch_id ? `<div style="margin-top: 8px; padding-top: 8px; border-top: 1px solid #E5E7EB;"><a href="/tenant/qr-batches/${escapeHtml(v.batch_id)}" style="color: #EA580C; font-size: 12px; text-decoration: none;">View Batch &rarr;</a></div>` : ''}
      </div>
    `
    marker.bindPopup(popupContent)
    marker.addTo(map)
    violationMarkers.push(marker)
  })

  // 3. Auto-fit bounds
  const allPoints = []
  props.violations.forEach(v => {
    if (v.lat != null && v.lng != null) allPoints.push([v.lat, v.lng])
  })
  props.zones.forEach(z => {
    if (z.lat != null && z.lng != null) allPoints.push([z.lat, z.lng])
  })

  if (allPoints.length > 1) {
    map.fitBounds(L.latLngBounds(allPoints), { padding: [50, 50] })
  } else if (allPoints.length === 1) {
    map.setView(allPoints[0], 10)
  }
}

function initMap() {
  if (!mapContainer.value || map) return

  const worldBounds = L.latLngBounds(
    L.latLng(-85, -180),
    L.latLng(85, 180)
  )

  map = L.map(mapContainer.value, {
    minZoom: 2,
    maxBounds: worldBounds,
    maxBoundsViscosity: 1.0
  }).setView([-2.5, 118], 5)

  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
    maxZoom: 19,
    noWrap: true
  }).addTo(map)

  renderData()
}

watch([() => props.violations, () => props.zones], () => renderData(), { deep: true })

onMounted(() => {
  initTimeoutId = setTimeout(initMap, 100)
})

onUnmounted(() => {
  if (initTimeoutId) {
    clearTimeout(initTimeoutId)
    initTimeoutId = null
  }
  clearMarkers()
  if (map) {
    map.remove()
    map = null
  }
})
</script>

<template>
  <div class="geofence-violation-map">
    <div
      ref="mapContainer"
      :style="{ height: height, width: '100%' }"
      class="rounded-lg overflow-hidden border border-gray-200 dark:border-gray-700"
    />

    <!-- Legend -->
    <div class="flex flex-wrap items-center justify-center gap-4 mt-3 text-xs text-gray-600 dark:text-gray-400">
      <div class="flex items-center gap-1">
        <span class="w-3 h-3 rounded-full border-2 border-dashed" style="border-color: #18181b; background: rgba(13,148,136,0.15);"></span>
        <span>Distribution Zone</span>
      </div>
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
  </div>
</template>

<style>
/* Dark mode support for map */
.dark .leaflet-tile {
  filter: invert(1) hue-rotate(180deg) brightness(0.95) contrast(0.9);
}

.dark .leaflet-container {
  background: #1f2937;
}
</style>
