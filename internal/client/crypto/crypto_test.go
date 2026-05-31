package crypto

import (
	"bytes"
	"testing"
)

func TestDeriveKey_Deterministic(t *testing.T) {
	a := DeriveKey("pw", "alice")
	b := DeriveKey("pw", "alice")
	if !bytes.Equal(a, b) {
		t.Fatal("same (password, login) should provide same key")
	}
	if len(a) != KeySize {
		t.Fatalf("expected %d-byte key, got %d", KeySize, len(a))
	}

	if bytes.Equal(a, DeriveKey("pw", "bob")) {
		t.Fatal("different login should provide different key")
	}
	if bytes.Equal(a, DeriveKey("other", "alice")) {
		t.Fatal("different password should provide different key")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := DeriveKey("pw", "alice")
	plain := []byte("hello, GophKeeper")

	ct, err := Encrypt(key, plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if bytes.Equal(ct, plain) {
		t.Fatal("ciphertext must not equal plaintext")
	}

	got, err := Decrypt(key, ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("round-trip mismatch: got %q", got)
	}
}

func TestEncrypt_FreshNonce(t *testing.T) {
	key := DeriveKey("pw", "alice")
	a, _ := Encrypt(key, []byte("same"))
	b, _ := Encrypt(key, []byte("same"))
	if bytes.Equal(a, b) {
		t.Fatal("two encryptions of the same plaintext must differ")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	ct, _ := Encrypt(DeriveKey("right", "alice"), []byte("secret"))
	if _, err := Decrypt(DeriveKey("wrong", "alice"), ct); err == nil {
		t.Fatal("decrypt with wrong key must fail")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	key := DeriveKey("pw", "alice")
	if _, err := Decrypt(key, []byte("x")); err == nil {
		t.Fatal("decrypt of tiny input must fail")
	}
}
