package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/onkar-sawarna/blog/lib/notepdf"
)

func main() {
	in := flag.String("in", "notes/computer-networks.pdf", "plaintext PDF")
	out := flag.String("out", "api/notes-file/computer-networks.pdf.enc", "ciphertext path")
	flag.Parse()

	plain, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	hexKey := strings.TrimSpace(os.Getenv("NOTES_PDF_KEY"))
	var key []byte
	if hexKey == "" {
		hexKey, key, err = notepdf.NewKey()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "generated NOTES_PDF_KEY (put this in .env and Vercel):")
		fmt.Println(hexKey)
	} else {
		key, err = notepdf.ParseKey(hexKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	ct, err := notepdf.Encrypt(plain, key)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, ct, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "wrote", *out)
}
