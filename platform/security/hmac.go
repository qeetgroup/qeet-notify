package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// SignHMAC returns the hex-encoded HMAC-SHA256 signature of payload using secret.
// Used by the outbound webhook worker to sign delivery payloads.
func SignHMAC(secret, payload []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload) //nolint:errcheck
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHMAC reports whether sig is the valid HMAC-SHA256 of payload under secret.
// Comparison is constant-time to prevent timing attacks.
func VerifyHMAC(secret, payload []byte, sig string) bool {
	expected := SignHMAC(secret, payload)
	return hmac.Equal([]byte(expected), []byte(sig))
}
