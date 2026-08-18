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
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id != "computer-networks" {
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
	ref := "n-cn-" + hex.EncodeToString(nonce)

	body, _ := json.Marshal(map[string]any{
		"amount":          amountPaise(),
		"currency":        "INR",
		"accept_partial":  false,
		"description":     "Computer networks, as they show up on a box",
		"reference_id":    ref,
		"callback_url":    callback,
		"callback_method": "get",
		"reminder_enable": false,
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

func amountPaise() int {
	raw := env("NOTES_AMOUNT_PAISE")
	if raw == "" {
		return 9900
	}
	first, _, _ := strings.Cut(raw, ",")
	n, err := strconv.Atoi(strings.TrimSpace(first))
	if err != nil || n <= 0 {
		return 9900
	}
	return n
}

func env(name string) string {
	v := strings.TrimSpace(os.Getenv(name))
	return strings.Trim(v, `"'`)
}
