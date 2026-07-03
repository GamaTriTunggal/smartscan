package utils

import (
	"golang.org/x/crypto/bcrypt"
)

// BcryptCost is the cost factor for password hashing
// OWASP recommends minimum cost of 12 for production systems
// Higher cost = more secure but slower hashing
const BcryptCost = 12

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
