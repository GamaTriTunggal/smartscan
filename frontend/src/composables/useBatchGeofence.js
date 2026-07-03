import { ref } from 'vue'

/**
 * Composable for shared geofence state and logic in batch creation modals.
 * Used by DynamicQRPage and ProductBatchHistoryPage.
 */
export function useBatchGeofence(apiGet) {
  const geofenceData = ref({ latitude: null, longitude: null, radius_km: 25, label: '' })
  const zoneTemplates = ref([])
  const selectedZoneTemplateId = ref(null)

  function getDefaultGeofenceFields() {
    return {
      geofence_enabled: false,
      geofence_latitude: null,
      geofence_longitude: null,
      geofence_radius_km: 25,
      geofence_label: '',
    }
  }

  function onGeofenceUpdate(val, batch) {
    geofenceData.value = val
    batch.geofence_latitude = val.latitude
    batch.geofence_longitude = val.longitude
    batch.geofence_radius_km = val.radius_km
    batch.geofence_label = val.label
    selectedZoneTemplateId.value = null
  }

  function loadZoneTemplate(template, batch) {
    selectedZoneTemplateId.value = template.id
    geofenceData.value = {
      latitude: template.latitude,
      longitude: template.longitude,
      radius_km: template.radius_km,
      label: template.label || template.template_name,
    }
    batch.geofence_latitude = geofenceData.value.latitude
    batch.geofence_longitude = geofenceData.value.longitude
    batch.geofence_radius_km = geofenceData.value.radius_km
    batch.geofence_label = geofenceData.value.label
  }

  async function fetchZoneTemplates() {
    try {
      const response = await apiGet('/tenant/geofence/zone-templates')
      if (response.success) {
        zoneTemplates.value = response.data?.zone_templates || []
      }
    } catch (e) { /* ignore */ }
  }

  function resetGeofence(batch) {
    Object.assign(batch, getDefaultGeofenceFields())
    geofenceData.value = { latitude: null, longitude: null, radius_km: 25, label: '' }
    selectedZoneTemplateId.value = null
  }

  function buildGeofencePayload() {
    if (!selectedZoneTemplateId.value) return {}
    return { geofence_zone_template_id: selectedZoneTemplateId.value }
  }

  return {
    geofenceData,
    zoneTemplates,
    selectedZoneTemplateId,
    getDefaultGeofenceFields,
    onGeofenceUpdate,
    loadZoneTemplate,
    fetchZoneTemplates,
    resetGeofence,
    buildGeofencePayload,
  }
}
