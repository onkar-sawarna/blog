package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/onkar-sawarna/blog/lib/rzpsig"
)

func TestHandlerRejectsBadSignature(t *testing.T) {
	t.Setenv("RAZORPAY_KEY_SECRET", "test-secret")
	t.Setenv("NOTES_PDF_KEY", "00")
	req := httptest.NewRequest(http.MethodGet, "/api/notes-file?razorpay_payment_link_status=paid&razorpay_signature=00", nil)
	rec := httptest.NewRecorder()
	Handler(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code %d body %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerServesPDF(t *testing.T) {
	secret := "test-secret"
	t.Setenv("RAZORPAY_KEY_SECRET", secret)
	t.Setenv("NOTES_PDF_PATH", filepathJoinNotes(t))
	payload := rzpsig.PaymentLinkPayload("plink_x", "note-computer-networks", "paid", "pay_x")
	sig := rzpsig.Sign(payload, secret)
	u := "/api/notes-file?razorpay_payment_link_id=plink_x&razorpay_payment_link_reference_id=note-computer-networks&razorpay_payment_link_status=paid&razorpay_payment_id=pay_x&razorpay_signature=" + sig
	req := httptest.NewRequest(http.MethodGet, u, nil)
	rec := httptest.NewRecorder()
	Handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d body %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("ctype %s", rec.Header().Get("Content-Type"))
	}
	if rec.Body.Len() < 8 {
		t.Fatal("empty body")
	}
}

func filepathJoinNotes(t *testing.T) string {
	t.Helper()
	p := "../../notes/computer-networks.pdf"
	if _, err := os.Stat(p); err != nil {
		t.Skip("plaintext PDF not on this machine")
	}
	return p
}
