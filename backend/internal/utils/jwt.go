package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID   uuid.UUID  `json:"user_id"`
	Email    string     `json:"email"`
	UserType string     `json:"user_type"`
	Role     string     `json:"role"`
	TenantID *uuid.UUID `json:"tenant_id,omitempty"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// JWT issuers to differentiate token types - prevents refresh token from being used as access token
const (
	IssuerAccessToken  = "smartscan"
	IssuerRefreshToken = "smartscan-refresh"
)

func GenerateTokenPair(secret string, userID uuid.UUID, email, userType, role string, tenantID *uuid.UUID, accessHours, refreshHours int) (*TokenPair, error) {
	accessDuration := time.Duration(accessHours) * time.Hour
	accessExpiry := time.Now().UTC().Add(accessDuration)
	refreshExpiry := time.Now().UTC().Add(time.Duration(refreshHours) * time.Hour)

	// Access token - uses IssuerAccessToken to distinguish from refresh token
	accessClaims := JWTClaims{
		UserID:    userID,
		Email:     email,
		UserType:  userType,
		Role:      role,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			Subject:   userID.String(),
			Issuer:    IssuerAccessToken,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}

	// Refresh token - uses IssuerRefreshToken to distinguish from access token
	refreshClaims := JWTClaims{
		UserID:    userID,
		Email:     email,
		UserType:  userType,
		Role:      role,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			Subject:   userID.String(),
			Issuer:    IssuerRefreshToken,
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(accessDuration.Seconds()), // Duration in seconds, not Unix timestamp
	}, nil
}

func ValidateToken(tokenString, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateAccessToken validates an access token and ensures it has the correct issuer
// This prevents refresh tokens from being used as access tokens
func ValidateAccessToken(tokenString, secret string) (*JWTClaims, error) {
	claims, err := ValidateToken(tokenString, secret)
	if err != nil {
		return nil, err
	}

	// Verify this is an access token, not a refresh token
	if claims.Issuer != IssuerAccessToken {
		return nil, errors.New("invalid token type: expected access token")
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token and ensures it has the correct issuer
func ValidateRefreshToken(tokenString, secret string) (*JWTClaims, error) {
	claims, err := ValidateToken(tokenString, secret)
	if err != nil {
		return nil, err
	}

	// Verify this is a refresh token
	if claims.Issuer != IssuerRefreshToken {
		return nil, errors.New("invalid token type: expected refresh token")
	}

	return claims, nil
}




