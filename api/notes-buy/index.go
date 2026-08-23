package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/onkar-sawarna/blog/lib/notespec"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	spec, ok := notespec.ByID(r.URL.Query().Get("id"))
	if !ok {
		http.Error(w, "Unknown note", http.StatusNotFound)
		return
	}

	key := env("RAZORPAY_KEY_ID")
	secret := env("RAZORPAY_KEY_SECRET")
	if key == "" || secret == "" {
		http.Error(w, "Checkout is not configured", http.StatusServiceUnavailable)
		return
	}

	callback := env("NOTES_CALLBACK_URL")
	if callback == "" {
		callback = "https://www.onkarsawarna.dev/notes/thanks"
	}

	nonce := make([]byte, 5)
	if _, err := rand.Read(nonce); err != nil {
		http.Error(w, "Could not start checkout", http.StatusInternalServerError)
		return
	}
	ref := spec.RefPrefix + hex.EncodeToString(nonce)

	body, _ := json.Marshal(map[string]any{
		"amount":          amountPaise(spec),
		"currency":        "INR",
		"accept_partial":  false,
		"description":     spec.Title,
		"reference_id":    ref,
		"callback_url":    callback,
		"callback_method": "get",
		"reminder_enable": false,
		"notes":           map[string]string{"note": spec.ID},
	})
	req, err := http.NewRequest(http.MethodPost, "https://api.razorpay.com/v1/payment_links", bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Could not start checkout", http.StatusBadGateway)
		return
	}
	req.SetBasicAuth(key, secret)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Could not start checkout", http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		http.Error(w, "Razorpay did not create a checkout", http.StatusBadGateway)
		return
	}
	var got struct {
		ShortURL string `json:"short_url"`
	}
	if err := json.Unmarshal(raw, &got); err != nil || got.ShortURL == "" {
		http.Error(w, "Razorpay did not create a checkout", http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, got.ShortURL, http.StatusFound)
}

func amountPaise(spec notespec.Spec) int {
	raw := env("NOTES_AMOUNT_PAISE")
	if raw == "" {
		return spec.AmountPaise
	}
	// Bare 4900 applies to every note. id=paise overrides one note.
	fallback := 0
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		id, amt, ok := strings.Cut(part, "=")
		if ok {
			if strings.TrimSpace(id) != spec.ID {
				continue
			}
			n, err := strconv.Atoi(strings.TrimSpace(amt))
			if err == nil && n > 0 {
				return n
			}
			continue
		}
		n, err := strconv.Atoi(part)
		if err == nil && n > 0 {
			fallback = n
		}
	}
	if fallback > 0 {
		return fallback
	}
	return spec.AmountPaise
}

func env(name string) string {
	v := strings.TrimSpace(os.Getenv(name))
	return strings.Trim(v, `"'`)
}
