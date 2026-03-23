package messenger

import (
	"testing"
)

func TestHashPasswordAndCheckPassword(t *testing.T) {
	password := "mysecretpassword"

	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword() error = %v", err)
	}

	if hash == "" {
		t.Fatal("hashPassword() returned empty hash")
	}

	if !CheckPassword(password, hash) {
		t.Fatal("CheckPassword() returned false for correct password")
	}

	if CheckPassword("wrongpassword", hash) {
		t.Fatal("CheckPassword() returned true for wrong password")
	}
}

func TestCheckPassword_DifferentHashes(t *testing.T) {
	password := "samepassword"

	hash1, _ := hashPassword(password)
	hash2, _ := hashPassword(password)

	if hash1 == hash2 {
		t.Fatal("hashPassword() should produce different hashes each time")
	}

	if !CheckPassword(password, hash1) || !CheckPassword(password, hash2) {
		t.Fatal("CheckPassword() should work for both hashes")
	}
}
