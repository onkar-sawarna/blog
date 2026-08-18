package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strings"

	handler "github.com/onkar-sawarna/blog/api"
	notesfile "github.com/onkar-sawarna/blog/api/notes-file"
)

func main() {
	loadDotEnv(".env")

	addr := "127.0.0.1:8080"
	if p := os.Getenv("LIKES_PORT"); p != "" {
		addr = "127.0.0.1:" + p
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/likes/", handler.Handler)
	mux.HandleFunc("/api/likes", handler.Handler)
	mux.HandleFunc("/api/notes-file/", notesfile.Handler)
	mux.HandleFunc("/api/notes-file", notesfile.Handler)

	log.Printf("likes api on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
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
