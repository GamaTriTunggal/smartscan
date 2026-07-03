package handlers

import (
	"encoding/json"
	"log"

	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"gorm.io/gorm"
)

// ResolveCounterfeitThreshold resolves the effective counterfeit scan threshold
// using a 4-level hierarchy: QR > Batch > Product > Tenant setting > Default(3).
// Returns 0 if counterfeit detection is disabled at any level.
// Requires qrCode.Batch.Product to be preloaded.
func ResolveCounterfeitThreshold(db *gorm.DB, qrCode *models.QRCode) int {
	const defaultThreshold = 3

	// Priority 1: QR-level override
	if qrCode.CounterfeitScanMax != nil {
		return *qrCode.CounterfeitScanMax
	}

	// Priority 2: Batch-level override
	if qrCode.Batch != nil && qrCode.Batch.CounterfeitScanMax != nil {
		return *qrCode.Batch.CounterfeitScanMax
	}

	// Priority 3: Product-level override
	if qrCode.Batch != nil && qrCode.Batch.Product != nil && qrCode.Batch.Product.CounterfeitScanMax != nil {
		return *qrCode.Batch.Product.CounterfeitScanMax
	}

	// Priority 4: Tenant setting
	if qrCode.Batch != nil {
		var settings models.TenantSettings
		if err := db.Where("tenant_id = ? AND setting_key = ?", qrCode.Batch.TenantID, "counterfeit_thresholds").
			First(&settings).Error; err == nil {
			var thresholds map[string]int
			if err := json.Unmarshal(settings.SettingValue, &thresholds); err != nil {
				log.Printf("Warning: failed to parse counterfeit_thresholds for tenant %s: %v", qrCode.Batch.TenantID, err)
			} else if t, ok := thresholds["end_user_scan_max"]; ok {
				return t
			}
		}
	}

	// Priority 5: Default
	return defaultThreshold
}
