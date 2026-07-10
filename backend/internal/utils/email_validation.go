package utils

import (
	"errors"
	"strings"
)

// disposableEmailDomains contains known disposable/temporary email domains
var disposableEmailDomains = map[string]bool{
	// Popular temp mail services
	"tempmail.com":        true,
	"temp-mail.org":       true,
	"guerrillamail.com":   true,
	"guerrillamail.org":   true,
	"guerrillamail.net":   true,
	"sharklasers.com":     true,
	"10minutemail.com":    true,
	"10minutemail.net":    true,
	"mailinator.com":      true,
	"mailinator.net":      true,
	"yopmail.com":         true,
	"yopmail.fr":          true,
	"maildrop.cc":         true,
	"throwaway.email":     true,
	"getnada.com":         true,
	"mohmal.com":          true,
	"tempail.com":         true,
	"fakeinbox.com":       true,
	"trashmail.com":       true,
	"trashmail.net":       true,
	"dispostable.com":     true,
	"mailnesia.com":       true,
	"mintemail.com":       true,
	"spamgourmet.com":     true,
	"mytrashmail.com":     true,
	"getairmail.com":      true,
	"mailcatch.com":       true,
	"tempr.email":         true,
	"discard.email":       true,
	"mailsac.com":         true,
	"inboxalias.com":      true,
	"burnermail.io":       true,
	"emailondeck.com":     true,
	"fakemail.net":        true,
	"dropmail.me":         true,
	"crazymailing.com":    true,
	"tempinbox.com":       true,
	"instantemailaddress.com": true,
}

// NormalizeEmail normalizes an email address for duplicate detection.
// ALL providers: removes +suffix, converts to lowercase
// Gmail only: also removes dots (Gmail treats dots as insignificant)
//
// Examples:
//   - john+spam@outlook.com -> john@outlook.com (+suffix removed)
//   - john.doe@outlook.com -> john.doe@outlook.com (dots kept for non-Gmail)
//   - j.o.h.n+spam@gmail.com -> john@gmail.com (both removed for Gmail)
//   - user@googlemail.com -> user@gmail.com (domain normalized)
func NormalizeEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}

	// Convert to lowercase
	email = strings.ToLower(email)

	// Split into local and domain parts
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return email // Invalid format, return as-is
	}

	local := parts[0]
	domain := parts[1]

	// Normalize googlemail.com to gmail.com
	if domain == "googlemail.com" {
		domain = "gmail.com"
	}

	// Remove +suffix for ALL providers (subaddressing)
	// Most providers support this: Gmail, Outlook, Protonmail, iCloud, FastMail
	// john+spam@outlook.com -> john@outlook.com
	if plusIndex := strings.Index(local, "+"); plusIndex != -1 {
		local = local[:plusIndex]
	}

	// Remove dots for Gmail ONLY
	// Gmail is unique in treating dots as insignificant
	// Other providers: dots are significant (different accounts)
	if domain == "gmail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}

	return local + "@" + domain
}

// IsDisposableEmail checks if the email domain is a known disposable email provider
func IsDisposableEmail(email string) bool {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return false
	}

	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return false
	}

	domain := parts[1]
	return disposableEmailDomains[domain]
}

// ValidateAndNormalizeEmail validates and normalizes an email address.
// Returns the normalized email or an error if the email is from a disposable provider.
func ValidateAndNormalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", nil // Empty email is allowed (may not be required)
	}

	// Check for disposable email
	if IsDisposableEmail(email) {
		return "", errors.New("disposable email addresses are not allowed")
	}

	// Normalize the email
	normalized := NormalizeEmail(email)

	return normalized, nil
}
