package utils

import (
	"errors"
	"math/big"
	"strings"

	"github.com/google/uuid"
)

// Base58 alphabet - Bitcoin style, excludes confusing characters: 0, O, I, l
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

var (
	base58AlphabetMap = make(map[rune]int64)
	bigRadix          = big.NewInt(58)
	bigZero           = big.NewInt(0)
)

func init() {
	for i, c := range base58Alphabet {
		base58AlphabetMap[c] = int64(i)
	}
}

// UUIDToBase58 encodes a UUID to Base58 string (~22 chars for UUIDv7)
func UUIDToBase58(u uuid.UUID) string {
	bytes := u[:]

	// Convert bytes to big integer
	num := new(big.Int).SetBytes(bytes)

	// Encode to base58
	var result []byte
	mod := new(big.Int)

	for num.Cmp(bigZero) > 0 {
		num.DivMod(num, bigRadix, mod)
		result = append(result, base58Alphabet[mod.Int64()])
	}

	// Handle leading zeros in input
	for _, b := range bytes {
		if b != 0 {
			break
		}
		result = append(result, base58Alphabet[0])
	}

	// Reverse the result
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// Base58ToUUID decodes a Base58 string back to UUID
func Base58ToUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, errors.New("empty base58 string")
	}

	// Decode base58 to big integer
	num := big.NewInt(0)
	for _, c := range s {
		val, ok := base58AlphabetMap[c]
		if !ok {
			return uuid.Nil, errors.New("invalid base58 character")
		}
		num.Mul(num, bigRadix)
		num.Add(num, big.NewInt(val))
	}

	// Convert to bytes
	bytes := num.Bytes()

	// UUID is always 16 bytes
	result := make([]byte, 16)

	// Check if decoded bytes fit in 16 bytes
	if len(bytes) > 16 {
		return uuid.Nil, errors.New("decoded value too large for UUID")
	}

	// Copy decoded bytes to result (right-aligned, zeros are already in result)
	copy(result[16-len(bytes):], bytes)

	return uuid.UUID(result), nil
}

// IsValidBase58 checks if a string contains only valid Base58 characters
func IsValidBase58(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune(base58Alphabet, c) {
			return false
		}
	}
	return true
}

// IsBase58UUID checks if a string is a valid Base58-encoded UUID (22 chars typical for UUIDv7)
func IsBase58UUID(s string) bool {
	if len(s) < 21 || len(s) > 22 {
		return false
	}
	return IsValidBase58(s)
}
