package rzpsig

import "testing"

func TestVerifyPaymentLinkDocsVector(t *testing.T) {
	// Values from Razorpay's verifyPaymentLink example.
	payload := PaymentLinkPayload(
		"plink_IH3cNucfVEgV68",
		"TSsd1989",
		"paid",
		"pay_IH3d0ara9bSsjQ",
	)
	secret := "EnLs21M47BllR3X8PSFtjtbd"
	sig := "07ae18789e35093e51d0a491eb9922646f3f82773547e5b0f67ee3f2d3bf7d5b"
	if !Verify(payload, sig, secret) {
		t.Fatalf("official vector did not verify; payload=%q computed=%s", payload, Sign(payload, secret))
	}
	if Verify(payload, sig, "wrong-secret") {
		t.Fatal("wrong secret accepted")
	}
}
