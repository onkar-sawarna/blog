package handler

import (
	_ "embed"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/onkar-sawarna/blog/lib/notepdf"
	"github.com/onkar-sawarna/blog/lib/rzpsig"
)

//go:embed computer-networks.pdf.enc
var embeddedEnc []byte

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	linkID := strings.TrimSpace(q.Get("razorpay_payment_link_id"))
	ref := strings.TrimSpace(q.Get("razorpay_payment_link_reference_id"))
	status := strings.TrimSpace(q.Get("razorpay_payment_link_status"))
	payID := strings.TrimSpace(q.Get("razorpay_payment_id"))
	sig := strings.TrimSpace(q.Get("razorpay_signature"))

	secret := env("RAZORPAY_KEY_SECRET")
	if secret == "" {
		http.Error(w, "Download is not configured", http.StatusServiceUnavailable)
		return
	}
	if status != "paid" {
		http.Error(w, "Payment is not complete", http.StatusForbidden)
		return
	}
	payload := rzpsig.PaymentLinkPayload(linkID, ref, status, payID)
	if !rzpsig.Verify(payload, sig, secret) {
		http.Error(w, "Invalid payment signature", http.StatusForbidden)
		return
	}
	if want := env("NOTES_PAYMENT_LINK_ID"); want != "" && linkID != want {
		http.Error(w, "Unknown payment link", http.StatusForbidden)
		return
	}
	if want := env("NOTES_REFERENCE_ID"); want != "" && ref != want {
		http.Error(w, "Unknown note", http.StatusForbidden)
		return
	}

	pdf, name, err := loadPDF()
	if err != nil {
		http.Error(w, "The file is not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	_, _ = w.Write(pdf)
}

func loadPDF() ([]byte, string, error) {
	name := "computer-networks.pdf"
	if p := env("NOTES_PDF_PATH"); p != "" {
		b, err := os.ReadFile(p)
		return b, filepath.Base(p), err
	}
	if b, err := os.ReadFile(filepath.Join("notes", "computer-networks.pdf")); err == nil && env("VERCEL") == "" {
		return b, name, nil
	}

	key, err := notepdf.ParseKey(env("NOTES_PDF_KEY"))
	if err != nil {
		return nil, "", err
	}
	raw := embeddedEnc
	if p := env("NOTES_PDF_ENC"); p != "" {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, "", err
		}
		raw = b
	}
	plain, err := notepdf.Decrypt(raw, key)
	if err != nil {
		return nil, "", err
	}
	return plain, name, nil
}

func env(name string) string {
	v := strings.TrimSpace(os.Getenv(name))
	return strings.Trim(v, `"'`)
}
