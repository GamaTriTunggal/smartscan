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

// dummyBcryptHash is a valid bcrypt hash (cost 12) used purely to normalize the
// response timing of a failed login when the account does not exist. Computed
// once at startup.
var dummyBcryptHash = func() string {
	h, err := bcrypt.GenerateFromPassword([]byte("smartscan-timing-normalizer"), BcryptCost)
	if err != nil {
		return ""
	}
	return string(h)
}()

// CheckDummyPassword performs a bcrypt comparison against a fixed dummy hash so
// that the login handler spends comparable CPU time whether or not the submitted
// email maps to a real account. This closes the timing side-channel that would
// otherwise let an attacker enumerate valid accounts. It always effectively
// fails; the result is intentionally discarded.
func CheckDummyPassword(password string) {
	if dummyBcryptHash == "" {
		return
	}
	_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
}
