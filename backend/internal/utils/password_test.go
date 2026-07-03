package utils

import (
	"testing"
)

func TestHashPassword_Success(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if hash == password {
		t.Error("Hash should not equal plain password")
	}
}

func TestHashPassword_DifferentHashes(t *testing.T) {
	password := "testpassword123"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password (1): %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password (2): %v", err)
	}

	// Each hash should be unique due to salt
	if hash1 == hash2 {
		t.Error("Same password should produce different hashes due to salt")
	}
}

func TestCheckPassword_Success(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should return true for correct password")
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	password := "testpassword123"
	wrongPassword := "wrongpassword"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if CheckPassword(wrongPassword, hash) {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestCheckPassword_EmptyPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if CheckPassword("", hash) {
		t.Error("CheckPassword should return false for empty password")
	}
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	if CheckPassword("password", "invalid-hash") {
		t.Error("CheckPassword should return false for invalid hash")
	}
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	// Empty password should still hash (bcrypt doesn't prevent this)
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("Failed to hash empty password: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty even for empty password")
	}

	if !CheckPassword("", hash) {
		t.Error("CheckPassword should work with empty password")
	}
}

func TestHashPassword_LongPassword(t *testing.T) {
	// bcrypt has a max length of 72 bytes, test with exactly 72 bytes
	longPassword := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()"

	hash, err := HashPassword(longPassword)
	if err != nil {
		t.Fatalf("Failed to hash long password: %v", err)
	}

	if !CheckPassword(longPassword, hash) {
		t.Error("CheckPassword should work with long password")
	}
}

func TestHashPassword_TooLongPassword(t *testing.T) {
	// bcrypt has a max length of 72 bytes, passwords longer than that should fail
	tooLongPassword := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=extra"

	_, err := HashPassword(tooLongPassword)
	if err == nil {
		t.Error("Expected error for password exceeding 72 bytes")
	}
}

func TestHashPassword_SpecialCharacters(t *testing.T) {
	password := "p@$$w0rd!#%^&*()_+-=[]{}|;':\",./<>?"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password with special characters: %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should work with special characters")
	}
}

func TestHashPassword_UnicodeCharacters(t *testing.T) {
	password := "密码パスワード"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash unicode password: %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should work with unicode characters")
	}
}
