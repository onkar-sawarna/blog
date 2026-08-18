package handler

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io"
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
	payID := extractPayID(q.Get("razorpay_payment_id"))
	sig := strings.TrimSpace(q.Get("razorpay_signature"))

	secret := env("RAZORPAY_KEY_SECRET")
	if secret == "" {
		http.Error(w, "Download is not configured", http.StatusServiceUnavailable)
		return
	}

	ok := false
	if status == "paid" && sig != "" && payID != "" {
		payload := rzpsig.PaymentLinkPayload(linkID, ref, status, payID)
		if rzpsig.Verify(payload, sig, secret) {
			ok = true
		}
	}
	if !ok && payID != "" {
		if err := paymentCaptured(payID); err != nil {
			http.Error(w, err.Error(), statusFor(err))
			return
		}
		ok = true
	}
	if !ok {
		http.Error(w, "Payment is not complete", http.StatusForbidden)
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

var (
	errNotConfigured = errors.New("RAZORPAY_KEY_ID is not set on the server")
	errBadPayID      = errors.New("that is not a payment id (it should start with pay_)")
	errUnknownPay    = errors.New("Razorpay does not know this payment id")
	errNotPaid       = errors.New("that payment is not captured yet")
)

func statusFor(err error) int {
	if errors.Is(err, errNotConfigured) {
		return http.StatusServiceUnavailable
	}
	return http.StatusForbidden
}

func extractPayID(raw string) string {
	raw = strings.TrimSpace(raw)
	for i := 0; i < len(raw)-4; i++ {
		if raw[i:i+4] != "pay_" {
			continue
		}
		if i > 0 {
			c := raw[i-1]
			if c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
				continue
			}
		}
		rest := raw[i:]
		var b strings.Builder
		for _, r := range rest {
			if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
				continue
			}
			break
		}
		return b.String()
	}
	return ""
}

func paymentCaptured(id string) error {
	if !strings.HasPrefix(id, "pay_") || len(id) < 8 || len(id) > 64 {
		return errBadPayID
	}
	key := env("RAZORPAY_KEY_ID")
	secret := env("RAZORPAY_KEY_SECRET")
	if key == "" || secret == "" {
		return errNotConfigured
	}
	req, err := http.NewRequest(http.MethodGet, "https://api.razorpay.com/v1/payments/"+urlPathEscape(id), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(key, secret)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusBadRequest {
		return errUnknownPay
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.New("Razorpay rejected the lookup")
	}
	var got struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		return err
	}
	switch got.Status {
	case "captured", "authorized", "refunded":
		return nil
	default:
		return errNotPaid
	}
}

func urlPathEscape(id string) string {
	return strings.ReplaceAll(id, " ", "")
}
