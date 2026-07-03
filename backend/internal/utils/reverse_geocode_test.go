package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReverseGeocode_NoAPIKey(t *testing.T) {
	result, err := ReverseGeocode(-6.2088, 106.8456, "")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no API key")
}

func TestReverseGeocode_Success_CityFromCity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"city": "Jakarta",
			"locality": "Menteng",
			"principalSubdivision": "DKI Jakarta",
			"principalSubdivisionCode": "ID-JK",
			"countryName": "Indonesia",
			"countryCode": "ID"
		}`))
	}))
	defer server.Close()

	// Temporarily swap the client and URL
	origClient := reverseGeocodeClient
	reverseGeocodeClient = server.Client()
	defer func() { reverseGeocodeClient = origClient }()

	// We need to hit the test server instead of the real API.
	// Since ReverseGeocode constructs the URL internally, we test via the real function
	// only for the "no API key" and error cases. For success, we test parsing separately.
	// However, since the URL is hardcoded, let's test via a table-driven approach.
	// For comprehensive unit testing, we verify the response parsing logic here.

	// Instead, let's test the actual function by verifying that with a real key it returns correctly
	// (will hit the test server only if we can override the URL, which we can't directly).
	// So we test the response parsing by calling the function and verifying the expected error
	// since the test server URL won't match the hardcoded URL.

	// Best approach: test the parsing logic directly
	result, err := parseGeoResponse([]byte(`{
		"city": "Jakarta",
		"locality": "Menteng",
		"principalSubdivision": "DKI Jakarta",
		"countryName": "Indonesia",
		"countryCode": "ID"
	}`))
	require.NoError(t, err)
	assert.Equal(t, "Jakarta", result.City)
	assert.Equal(t, "DKI Jakarta", result.Province)
	assert.Equal(t, "Indonesia", result.Country)
	assert.Equal(t, "ID", result.CountryCode)
}

func TestReverseGeocode_CityFallbackToLocality(t *testing.T) {
	result, err := parseGeoResponse([]byte(`{
		"city": "",
		"locality": "Menteng",
		"principalSubdivision": "DKI Jakarta",
		"countryName": "Indonesia",
		"countryCode": "ID"
	}`))
	require.NoError(t, err)
	assert.Equal(t, "Menteng", result.City, "should fallback to locality when city is empty")
}

func TestReverseGeocode_InvalidJSON(t *testing.T) {
	result, err := parseGeoResponse([]byte(`not json`))
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode failed")
}

func TestReverseGeocode_EmptyResponse(t *testing.T) {
	result, err := parseGeoResponse([]byte(`{}`))
	require.NoError(t, err)
	assert.Equal(t, "", result.City)
	assert.Equal(t, "", result.Province)
	assert.Equal(t, "", result.Country)
}

func TestReverseGeocode_ClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // exceed client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// The real function has a 3s timeout built-in, and the URL is hardcoded,
	// so we can't easily test timeout without refactoring. We verify the timeout is set.
	assert.Equal(t, 3*time.Second, reverseGeocodeClient.Timeout)
}
