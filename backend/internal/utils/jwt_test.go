package utils

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerateTokenPair_Success(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	email := "test@example.com"
	userType := "tenant_staff"
	role := "super_admin"
	secret := "test-secret-key"

	tokenPair, err := GenerateTokenPair(secret, userID, email, userType, role, nil, 24, 168, false)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("Access token is empty")
	}

	if tokenPair.RefreshToken == "" {
		t.Error("Refresh token is empty")
	}

	// ExpiresIn is duration in seconds, not Unix timestamp
	// For 24 hours, expected is 86400 seconds
	expectedDuration := int64(24 * 60 * 60)
	if tokenPair.ExpiresIn != expectedDuration {
		t.Errorf("Expected ExpiresIn %d seconds, got %d", expectedDuration, tokenPair.ExpiresIn)
	}
}

func TestGenerateTokenPair_WithTenantID(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	tenantID := uuid.Must(uuid.NewV7())
	email := "test@example.com"
	userType := "tenant_staff"
	role := "admin"
	secret := "test-secret-key"

	tokenPair, err := GenerateTokenPair(secret, userID, email, userType, role, &tenantID, 24, 168, false)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// Validate the access token includes tenant ID
	claims, err := ValidateToken(tokenPair.AccessToken, secret)
	if err != nil {
		t.Fatalf("Failed to validate access token: %v", err)
	}

	if claims.TenantID == nil || *claims.TenantID != tenantID {
		t.Errorf("Expected tenant ID %s, got %v", tenantID, claims.TenantID)
	}
}

func TestValidateToken_Success(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	email := "test@example.com"
	userType := "tenant_staff"
	role := "super_admin"
	secret := "test-secret-key"

	tokenPair, err := GenerateTokenPair(secret, userID, email, userType, role, nil, 24, 168, false)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	claims, err := ValidateToken(tokenPair.AccessToken, secret)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}

	if claims.Email != email {
		t.Errorf("Expected email %s, got %s", email, claims.Email)
	}

	if claims.UserType != userType {
		t.Errorf("Expected user type %s, got %s", userType, claims.UserType)
	}

	if claims.Role != role {
		t.Errorf("Expected role %s, got %s", role, claims.Role)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	secret := "test-secret-key"

	_, err := ValidateToken("invalid-token", secret)
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	secret := "correct-secret"
	wrongSecret := "wrong-secret"

	tokenPair, err := GenerateTokenPair(secret, userID, "test@example.com", "tenant_staff", "admin", nil, 24, 168, false)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	_, err = ValidateToken(tokenPair.AccessToken, wrongSecret)
	if err == nil {
		t.Error("Expected error for wrong secret, got nil")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	secret := "test-secret-key"

	// Generate token with negative expiration (already expired)
	tokenPair, err := GenerateTokenPair(secret, userID, "test@example.com", "tenant_staff", "admin", nil, -1, -1, false)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	_, err = ValidateToken(tokenPair.AccessToken, secret)
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}
