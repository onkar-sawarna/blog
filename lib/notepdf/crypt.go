package notepdf

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

const magic = "ONK1"

func ParseKey(s string) ([]byte, error) {
	s = strings.TrimSpace(strings.Trim(s, `"'`))
	if s == "" {
		return nil, errors.New("empty key")
	}
	key, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("key must be hex: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}
	return key, nil
}

func NewKey() (hexKey string, key []byte, err error) {
	key = make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		return "", nil, err
	}
	return hex.EncodeToString(key), key, nil
}

func Encrypt(plain, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(magic)+len(nonce)+len(plain)+gcm.Overhead())
	out = append(out, magic...)
	out = append(out, nonce...)
	return gcm.Seal(out, nonce, plain, []byte(magic)), nil
}

func Decrypt(blob, key []byte) ([]byte, error) {
	if len(blob) < len(magic)+12+16 {
		return nil, errors.New("ciphertext too short")
	}
	if string(blob[:len(magic)]) != magic {
		return nil, errors.New("not a note ciphertext")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	rest := blob[len(magic):]
	if len(rest) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := rest[:nonceSize], rest[nonceSize:]
	return gcm.Open(nil, nonce, ct, []byte(magic))
}
