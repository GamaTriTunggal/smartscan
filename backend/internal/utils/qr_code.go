package utils

import (
	"fmt"

	"github.com/google/uuid"
)

// QRCodeLookup contains parsed QR code lookup info
type QRCodeLookup struct {
	QRUUID       uuid.UUID // Parsed UUID (for qr_uuid lookup)
	LookupByCode bool      // true = use qr_code field, false = use qr_uuid field
	OriginalCode string    // Original input (for qr_code lookup)
}

// ParseQRCodeParam parses a QR code parameter supporting multiple formats.
// Returns lookup info for database queries or error if format is invalid.
//
// Supported formats:
//   - Base58 (21-22 chars): New compact format, decoded to UUID
//   - UUID (36 chars): Standard UUID format with dashes
//   - Hex (32 chars): Legacy format, uses qr_code field directly
//   - Internal code (any other non-empty string): Looked up by qr_code field
//
// Usage:
//
//	lookup, err := utils.ParseQRCodeParam(code)
//	if err != nil {
//	    return error
//	}
//	if lookup.LookupByCode {
//	    db.First(&qr, "qr_code = ?", lookup.OriginalCode)
//	} else {
//	    db.First(&qr, "qr_uuid = ?", lookup.QRUUID)
//	}
func ParseQRCodeParam(code string) (*QRCodeLookup, error) {
	if code == "" {
		return nil, fmt.Errorf("empty QR code")
	}

	inputLen := len(code)

	switch {
	case inputLen >= 21 && inputLen <= 22 && IsBase58UUID(code):
		// Base58 encoded UUID (new compact format)
		qrUUID, err := Base58ToUUID(code)
		if err != nil {
			return nil, fmt.Errorf("invalid Base58 format: %w", err)
		}
		return &QRCodeLookup{QRUUID: qrUUID, LookupByCode: false, OriginalCode: code}, nil

	case inputLen == 36:
		// Standard UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
		qrUUID, err := uuid.Parse(code)
		if err != nil {
			// Not a valid UUID — fall through to qr_code lookup
			return &QRCodeLookup{LookupByCode: true, OriginalCode: code}, nil
		}
		return &QRCodeLookup{QRUUID: qrUUID, LookupByCode: false, OriginalCode: code}, nil

	case inputLen == 32:
		// Legacy hex format (qr_code string without dashes)
		return &QRCodeLookup{LookupByCode: true, OriginalCode: code}, nil

	default:
		// Internal qr_code format (e.g. QR-ADV-PB14-0018, DEMO-DYN-001)
		return &QRCodeLookup{LookupByCode: true, OriginalCode: code}, nil
	}
}
