package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gamatritunggal/smartscan/backend/internal/models"
)

// checkVelocityAnomalyShared is the single implementation of the
// impossible-travel check. It compares the current scan position against the
// most recent geolocated interaction for the same QR code; if covering that
// distance would require a speed above the tenant's configured maximum, the
// scan is flagged as a counterfeit signal (the same physical label cannot be
// in two distant places at once — a clone can).
//
// interactions.geolocation is stored as {"lat": ..., "lng": ...} — every
// writer uses that shape (see UpdateScanLocation and the QC/warehouse scan
// paths). Historical note: an earlier duplicate of this check parsed
// {"latitude","longitude"} and therefore never fired.
func checkVelocityAnomalyShared(db *gorm.DB, tenantID, qrCodeID uuid.UUID, currentLat, currentLng float64) (bool, string) {
	if currentLat == 0 || currentLng == 0 {
		return false, "" // No location data
	}

	var setting models.TenantSettings
	if err := db.Where("tenant_id = ? AND setting_key = ?", tenantID, "counterfeit_thresholds").First(&setting).Error; err != nil {
		return false, "" // No settings, skip check
	}

	var thresholds map[string]interface{}
	if err := json.Unmarshal(setting.SettingValue, &thresholds); err != nil {
		return false, ""
	}

	velocityEnabled, ok := thresholds["velocity_check_enabled"].(bool)
	if !ok || !velocityEnabled {
		return false, ""
	}

	maxSpeedKmh, ok := thresholds["max_speed_kmh"].(float64)
	if !ok || maxSpeedKmh == 0 {
		maxSpeedKmh = 1000 // Default: 1000 km/h (commercial flight speed)
	}

	var lastInteraction models.Interaction
	if err := db.Where("qr_code_id = ? AND geolocation IS NOT NULL", qrCodeID).
		Order("created_at DESC").
		First(&lastInteraction).Error; err != nil {
		return false, "" // No previous geolocated interaction
	}

	var lastGeo struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	if err := json.Unmarshal(lastInteraction.Geolocation, &lastGeo); err != nil {
		return false, ""
	}
	if lastGeo.Lat == 0 || lastGeo.Lng == 0 {
		return false, ""
	}

	distance := calculateDistance(lastGeo.Lat, lastGeo.Lng, currentLat, currentLng)
	timeDiff := time.Since(lastInteraction.CreatedAt).Seconds()
	if timeDiff <= 0 {
		return false, ""
	}

	speedKmh := (distance / 1000) / (timeDiff / 3600)
	if speedKmh > maxSpeedKmh {
		return true, fmt.Sprintf(
			"Impossible velocity detected: %.0f km/h (max: %.0f km/h), distance: %.0f km, time: %.0f sec",
			speedKmh, maxSpeedKmh, distance/1000, timeDiff)
	}
	return false, ""
}
