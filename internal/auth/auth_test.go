package auth

import (
	"crypto/rand"
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Error("expected error for short password")
	}
	hash, err := HashPassword("a-strong-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "a-strong-password-123") {
		t.Error("valid password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("invalid password accepted")
	}
}

func TestEncryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	secret := []byte("smtp-pa$$word")

	ct, err := Encrypt(key, secret)
	if err != nil {
		t.Fatal(err)
	}
	if string(ct) == string(secret) {
		t.Error("ciphertext equals plaintext")
	}
	pt, err := Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if string(pt) != string(secret) {
		t.Errorf("roundtrip mismatch: %q", pt)
	}

	// Wrong key must fail.
	badKey := make([]byte, 32)
	_, _ = rand.Read(badKey)
	if _, err := Decrypt(badKey, ct); err == nil {
		t.Error("decrypt with wrong key should fail")
	}
}

func TestNewTokenUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		tok := NewToken()
		if tok == "" || seen[tok] {
			t.Fatalf("empty or duplicate token: %q", tok)
		}
		seen[tok] = true
	}
}
