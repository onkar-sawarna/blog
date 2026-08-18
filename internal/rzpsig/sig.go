package rzpsig

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// PaymentLinkPayload is the HMAC body Razorpay documents for payment-link callbacks.
func PaymentLinkPayload(linkID, referenceID, status, paymentID string) string {
	return strings.Join([]string{linkID, referenceID, status, paymentID}, "|")
}

func Verify(payload, signatureHex, secret string) bool {
	if secret == "" || signatureHex == "" {
		return false
	}
	sig, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hmac.Equal(mac.Sum(nil), sig)
}

func Sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
