package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	loadDotEnv(".env")
	key := strings.TrimSpace(os.Getenv("RAZORPAY_KEY_ID"))
	secret := strings.TrimSpace(os.Getenv("RAZORPAY_KEY_SECRET"))
	if key == "" || secret == "" {
		fmt.Fprintln(os.Stderr, "set RAZORPAY_KEY_ID and RAZORPAY_KEY_SECRET")
		os.Exit(1)
	}

	callback := strings.TrimSpace(os.Getenv("NOTES_CALLBACK_URL"))
	if callback == "" {
		callback = "https://www.onkarsawarna.dev/notes/thanks"
	}
	ref := strings.TrimSpace(os.Getenv("NOTES_REFERENCE_ID"))
	if ref == "" {
		ref = "note-computer-networks"
	}

	amount := 100
	if raw := strings.TrimSpace(os.Getenv("NOTES_AMOUNT_PAISE")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err == nil && n > 0 {
			amount = n
		}
	}

	body := map[string]any{
		"amount":          amount,
		"currency":        "INR",
		"accept_partial":  false,
		"description":     "Computer networks, as they show up on a box",
		"reference_id":    ref,
		"callback_url":    callback,
		"callback_method": "get",
		"reminder_enable": true,
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, "https://api.razorpay.com/v1/payment_links", bytes.NewReader(raw))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	req.SetBasicAuth(key, secret)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "razorpay %d: %s\n", res.StatusCode, out)
		os.Exit(1)
	}
	var got struct {
		ID       string `json:"id"`
		ShortURL string `json:"short_url"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		fmt.Fprintln(os.Stderr, string(out))
		os.Exit(1)
	}
	fmt.Printf("id=%s\n", got.ID)
	fmt.Printf("buyUrl=%s\n", got.ShortURL)
	fmt.Printf("set NOTES_PAYMENT_LINK_ID=%s\n", got.ID)
	fmt.Printf("set NOTES_REFERENCE_ID=%s\n", ref)
	fmt.Printf("put buyUrl in src/config.ts\n")
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}
