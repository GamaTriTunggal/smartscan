import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useBatchGeofence } from '../useBatchGeofence'

describe('useBatchGeofence', () => {
  let mockApiGet
  let composable

  beforeEach(() => {
    mockApiGet = vi.fn()
    composable = useBatchGeofence(mockApiGet)
  })

  describe('initial state', () => {
    it('returns default geofence data', () => {
      expect(composable.geofenceData.value).toEqual({
        latitude: null,
        longitude: null,
        radius_km: 25,
        label: '',
      })
    })

    it('returns empty zone templates', () => {
      expect(composable.zoneTemplates.value).toEqual([])
    })

    it('returns null selectedZoneTemplateId', () => {
      expect(composable.selectedZoneTemplateId.value).toBeNull()
    })
  })

  describe('getDefaultGeofenceFields', () => {
    it('returns correct defaults', () => {
      const fields = composable.getDefaultGeofenceFields()
      expect(fields).toEqual({
        geofence_enabled: false,
        geofence_latitude: null,
        geofence_longitude: null,
        geofence_radius_km: 25,
        geofence_label: '',
      })
    })
  })

  describe('onGeofenceUpdate', () => {
    it('updates geofence data and batch fields', () => {
      const batch = {
        geofence_latitude: null,
        geofence_longitude: null,
        geofence_radius_km: 25,
        geofence_label: '',
      }

      composable.onGeofenceUpdate({
        latitude: -6.2088,
        longitude: 106.8456,
        radius_km: 50,
        label: 'Jakarta',
      }, batch)

      expect(composable.geofenceData.value.latitude).toBe(-6.2088)
      expect(composable.geofenceData.value.longitude).toBe(106.8456)
      expect(composable.geofenceData.value.radius_km).toBe(50)
      expect(composable.geofenceData.value.label).toBe('Jakarta')

      expect(batch.geofence_latitude).toBe(-6.2088)
      expect(batch.geofence_longitude).toBe(106.8456)
      expect(batch.geofence_radius_km).toBe(50)
      expect(batch.geofence_label).toBe('Jakarta')
    })

    it('clears selectedZoneTemplateId', () => {
      composable.selectedZoneTemplateId.value = 'some-id'

      composable.onGeofenceUpdate({
        latitude: -6.2,
        longitude: 106.8,
        radius_km: 25,
        label: '',
      }, {})

      expect(composable.selectedZoneTemplateId.value).toBeNull()
    })
  })

  describe('loadZoneTemplate', () => {
    it('sets all fields from template', () => {
      const batch = {
        geofence_latitude: null,
        geofence_longitude: null,
        geofence_radius_km: 25,
        geofence_label: '',
      }
      const template = {
        id: 'template-1',
        latitude: -7.0,
        longitude: 110.4,
        radius_km: 100,
        label: 'Semarang',
      }

      composable.loadZoneTemplate(template, batch)

      expect(composable.selectedZoneTemplateId.value).toBe('template-1')
      expect(composable.geofenceData.value.latitude).toBe(-7.0)
      expect(composable.geofenceData.value.longitude).toBe(110.4)
      expect(composable.geofenceData.value.radius_km).toBe(100)
      expect(composable.geofenceData.value.label).toBe('Semarang')
      expect(batch.geofence_latitude).toBe(-7.0)
      expect(batch.geofence_longitude).toBe(110.4)
    })

    it('falls back to template_name when label is empty', () => {
      const batch = {}
      const template = {
        id: 'template-2',
        latitude: -6.5,
        longitude: 107.0,
        radius_km: 30,
        label: '',
        template_name: 'Bandung Metro',
      }

      composable.loadZoneTemplate(template, batch)

      expect(composable.geofenceData.value.label).toBe('Bandung Metro')
    })
  })

  describe('fetchZoneTemplates', () => {
    it('calls apiGet with correct path', async () => {
      mockApiGet.mockResolvedValue({
        success: true,
        data: { zone_templates: [{ id: 'zt1', template_name: 'Zone 1' }] },
      })

      await composable.fetchZoneTemplates()

      expect(mockApiGet).toHaveBeenCalledWith('/tenant/geofence/zone-templates')
      expect(composable.zoneTemplates.value).toHaveLength(1)
      expect(composable.zoneTemplates.value[0].template_name).toBe('Zone 1')
    })

    it('handles API failure gracefully', async () => {
      mockApiGet.mockRejectedValue(new Error('Network error'))

      await composable.fetchZoneTemplates()

      expect(composable.zoneTemplates.value).toEqual([])
    })

    it('handles missing data gracefully', async () => {
      mockApiGet.mockResolvedValue({ success: true, data: {} })

      await composable.fetchZoneTemplates()

      expect(composable.zoneTemplates.value).toEqual([])
    })
  })

  describe('resetGeofence', () => {
    it('clears all geofence fields on batch', () => {
      const batch = {
        geofence_enabled: true,
        geofence_latitude: -6.2,
        geofence_longitude: 106.8,
        geofence_radius_km: 100,
        geofence_label: 'Jakarta',
      }

      composable.resetGeofence(batch)

      expect(batch.geofence_enabled).toBe(false)
      expect(batch.geofence_latitude).toBeNull()
      expect(batch.geofence_longitude).toBeNull()
      expect(batch.geofence_radius_km).toBe(25)
      expect(batch.geofence_label).toBe('')

      expect(composable.geofenceData.value.latitude).toBeNull()
      expect(composable.selectedZoneTemplateId.value).toBeNull()
    })
  })

  describe('buildGeofencePayload', () => {
    it('returns empty object when no template selected', () => {
      expect(composable.buildGeofencePayload()).toEqual({})
    })

    it('returns geofence_zone_template_id when template selected', () => {
      composable.selectedZoneTemplateId.value = 'template-abc'

      expect(composable.buildGeofencePayload()).toEqual({
        geofence_zone_template_id: 'template-abc',
      })
    })
  })
})
