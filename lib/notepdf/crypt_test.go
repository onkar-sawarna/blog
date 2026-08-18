package notepdf

import "testing"

func TestEncryptRoundTrip(t *testing.T) {
	_, key, err := NewKey()
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("%PDF-1.4 test")
	ct, err := Encrypt(plain, key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decrypt(ct, key)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plain) {
		t.Fatalf("got %q", got)
	}
	if _, err := Decrypt(ct, make([]byte, 32)); err == nil {
		t.Fatal("wrong key should fail")
	}
}
