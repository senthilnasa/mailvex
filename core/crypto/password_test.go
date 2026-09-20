package crypto

import (
	"testing"
)

func TestArgon2idPassword(t *testing.T) {
	password := "CorrectHorseBatteryStaple!987"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	// Verify with correct password
	match, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("failed to verify password: %v", err)
	}
	if !match {
		t.Fatalf("expected password verification to succeed")
	}

	// Verify with incorrect password
	match, err = VerifyPassword("WrongPassword123", hash)
	if err != nil {
		t.Fatalf("unexpected error on wrong password: %v", err)
	}
	if match {
		t.Fatalf("expected wrong password to fail verification")
	}
}
