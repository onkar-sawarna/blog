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
	"github.com/onkar-sawarna/blog/lib/notespec"
	"github.com/onkar-sawarna/blog/lib/rzpsig"
)

//go:embed computer-networks.pdf.enc
var networksEnc []byte

//go:embed objects-as-they-show-up-in-a-request.pdf.enc
var objectsEnc []byte

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
	noteID := strings.TrimSpace(q.Get("id"))
	description := ""
	if status == "paid" && sig != "" && payID != "" {
		payload := rzpsig.PaymentLinkPayload(linkID, ref, status, payID)
		if rzpsig.Verify(payload, sig, secret) {
			ok = true
		}
	}
	if !ok && payID != "" {
		pay, err := lookupPayment(payID)
		if err != nil {
			http.Error(w, err.Error(), statusFor(err))
			return
		}
		ok = true
		if noteID == "" {
			noteID = pay.NoteID
		}
		description = pay.Description
		if ref == "" {
			ref = pay.Ref
		}
	}
	if !ok {
		http.Error(w, "Payment is not complete", http.StatusForbidden)
		return
	}

	spec, found := notespec.ByHint(noteID, description, ref)
	if !found {
		spec, found = notespec.ByID("computer-networks")
	}
	if !found {
		http.Error(w, "Unknown note", http.StatusNotFound)
		return
	}

	pdf, name, err := loadPDF(spec)
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

func loadPDF(spec notespec.Spec) ([]byte, string, error) {
	if p := env("NOTES_PDF_PATH"); p != "" {
		b, err := os.ReadFile(p)
		return b, filepath.Base(p), err
	}
	if env("VERCEL") == "" {
		if b, err := os.ReadFile(filepath.Join("notes", spec.Filename)); err == nil {
			return b, spec.Filename, nil
		}
	}

	key, err := notepdf.ParseKey(env("NOTES_PDF_KEY"))
	if err != nil {
		return nil, "", err
	}
	raw := encFor(spec.ID)
	if p := env("NOTES_PDF_ENC"); p != "" {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, "", err
		}
		raw = b
	}
	if len(raw) == 0 {
		return nil, "", errors.New("no ciphertext")
	}
	plain, err := notepdf.Decrypt(raw, key)
	if err != nil {
		return nil, "", err
	}
	return plain, spec.Filename, nil
}

func encFor(id string) []byte {
	switch id {
	case "objects-as-they-show-up-in-a-request":
		return objectsEnc
	default:
		return networksEnc
	}
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

type paymentInfo struct {
	NoteID      string
	Description string
	Ref         string
}

func lookupPayment(id string) (paymentInfo, error) {
	if !strings.HasPrefix(id, "pay_") || len(id) < 8 || len(id) > 64 {
		return paymentInfo{}, errBadPayID
	}
	key := env("RAZORPAY_KEY_ID")
	secret := env("RAZORPAY_KEY_SECRET")
	if key == "" || secret == "" {
		return paymentInfo{}, errNotConfigured
	}
	req, err := http.NewRequest(http.MethodGet, "https://api.razorpay.com/v1/payments/"+urlPathEscape(id), nil)
	if err != nil {
		return paymentInfo{}, err
	}
	req.SetBasicAuth(key, secret)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return paymentInfo{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return paymentInfo{}, err
	}
	if res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusBadRequest {
		return paymentInfo{}, errUnknownPay
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return paymentInfo{}, errors.New("Razorpay rejected the lookup")
	}
	var got struct {
		Status      string            `json:"status"`
		Description string            `json:"description"`
		Notes       map[string]string `json:"notes"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		return paymentInfo{}, err
	}
	switch got.Status {
	case "captured", "authorized", "refunded":
		info := paymentInfo{Description: got.Description}
		if got.Notes != nil {
			info.NoteID = strings.TrimSpace(got.Notes["note"])
		}
		return info, nil
	default:
		return paymentInfo{}, errNotPaid
	}
}

func urlPathEscape(id string) string {
	return strings.ReplaceAll(id, " ", "")
}
