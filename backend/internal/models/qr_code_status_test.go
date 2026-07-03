package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestQRCode_IsScannable locks the contract for the IsScannable() helper.
// IsScannable combines two orthogonal status columns:
//   - Status (admin enable/disable)
//   - CounterfeitStatus (fraud signal)
//
// Rule: scannable iff Status='active' AND CounterfeitStatus != 'counterfeit'.
//
// This matrix locks the rule so future refactors cannot silently change scan eligibility.
func TestQRCode_IsScannable(t *testing.T) {
	adminStatuses := []QRCodeStatus{QRCodeStatusActive, QRCodeStatusInactive}
	cfStatuses := []CounterfeitStatus{
		CounterfeitStatusValid,
		CounterfeitStatusWarning, // Deprecated — treated as valid
		CounterfeitStatusCounterfeit,
	}

	for _, as := range adminStatuses {
		for _, cs := range cfStatuses {
			as, cs := as, cs
			name := string(as) + "/" + string(cs)
			t.Run(name, func(t *testing.T) {
				qr := QRCode{Status: as, CounterfeitStatus: cs}
				expected := as == QRCodeStatusActive && cs != CounterfeitStatusCounterfeit
				assert.Equal(t, expected, qr.IsScannable())
			})
		}
	}
}
