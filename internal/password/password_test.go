package password

import "testing"

func TestHashAndCheck(t *testing.T) {
	hash, err := Hash("secret123")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}
	if hash == "secret123" {
		t.Fatal("hash equals plaintext")
	}
	if !CheckHash("secret123", hash) {
		t.Fatal("CheckHash returned false for correct password")
	}
	if CheckHash("wrong", hash) {
		t.Fatal("CheckHash returned true for wrong password")
	}
}
