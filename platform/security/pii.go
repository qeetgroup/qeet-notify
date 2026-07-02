package security

import "strings"

// MaskEmail masks an email address for safe logging: "user@example.com" → "us**@example.com".
func MaskEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 0 {
		return "***"
	}
	local := email[:at]
	domain := email[at:]
	if len(local) <= 2 {
		return "**" + domain
	}
	return local[:2] + strings.Repeat("*", len(local)-2) + domain
}

// MaskPhone masks a phone number for safe logging: "+919876543210" → "+91*****3210".
func MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	// Preserve country-code prefix (up to 3 chars) and last 4 digits.
	prefix := phone[:3]
	suffix := phone[len(phone)-4:]
	middle := strings.Repeat("*", len(phone)-7)
	if len(phone) <= 7 {
		return prefix + "****"
	}
	return prefix + middle + suffix
}
