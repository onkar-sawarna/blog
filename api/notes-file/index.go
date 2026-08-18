package handler

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

	ok := false
	if status == "paid" && sig != "" && payID != "" {
		payload := rzpsig.PaymentLinkPayload(linkID, ref, status, payID)
		if rzpsig.Verify(payload, sig, secret) {
			if !allowedValue(env("NOTES_PAYMENT_LINK_ID"), linkID) {
				http.Error(w, "Unknown payment link", http.StatusForbidden)
				return
			}
			if ref != "" && !allowedValue(env("NOTES_REFERENCE_ID"), ref) {
				http.Error(w, "Unknown note", http.StatusForbidden)
				return
			}
			ok = true
		}
	}
	if !ok && payID != "" {
		if err := paymentCaptured(payID); err != nil {
			if errors.Is(err, errNotConfigured) {
				http.Error(w, "Download is not configured", http.StatusServiceUnavailable)
				return
			}
			http.Error(w, "Payment is not complete", http.StatusForbidden)
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

var errNotConfigured = errors.New("not_configured")

func allowedValue(wantList, got string) bool {
	if strings.TrimSpace(wantList) == "" {
		return true
	}
	for _, part := range strings.Split(wantList, ",") {
		if strings.TrimSpace(part) == got {
			return true
		}
	}
	return false
}

func allowedPaise() []int64 {
	raw := env("NOTES_AMOUNT_PAISE")
	if raw == "" {
		return []int64{100, 14900}
	}
	var out []int64
	for _, part := range strings.Split(raw, ",") {
		n, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err == nil && n > 0 {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return []int64{100, 14900}
	}
	return out
}

func amountAllowed(amount int64) bool {
	for _, n := range allowedPaise() {
		if amount == n {
			return true
		}
	}
	return false
}

func paymentCaptured(id string) error {
	if !strings.HasPrefix(id, "pay_") || len(id) < 8 || len(id) > 40 {
		return errors.New("bad id")
	}
	key := env("RAZORPAY_KEY_ID")
	secret := env("RAZORPAY_KEY_SECRET")
	if key == "" || secret == "" {
		return errNotConfigured
	}
	req, err := http.NewRequest(http.MethodGet, "https://api.razorpay.com/v1/payments/"+id, nil)
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
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.New("razorpay " + strconv.Itoa(res.StatusCode))
	}
	var got struct {
		Status string `json:"status"`
		Amount int64  `json:"amount"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		return err
	}
	if got.Status != "captured" && got.Status != "authorized" {
		return errors.New("not captured")
	}
	if !amountAllowed(got.Amount) {
		return errors.New("wrong amount")
	}
	return nil
}
