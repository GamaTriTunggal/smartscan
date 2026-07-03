package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GeoResult holds the result of a reverse geocoding lookup.
type GeoResult struct {
	City        string `json:"city"`
	Province    string `json:"province"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
}

// bigDataCloudResponse represents the relevant fields from BigDataCloud API response.
type bigDataCloudResponse struct {
	City                     string `json:"city"`
	Locality                 string `json:"locality"`
	PrincipalSubdivision     string `json:"principalSubdivision"`
	PrincipalSubdivisionCode string `json:"principalSubdivisionCode"`
	CountryName              string `json:"countryName"`
	CountryCode              string `json:"countryCode"`
}

var reverseGeocodeClient = &http.Client{
	Timeout: 3 * time.Second,
}

// ReverseGeocode converts latitude/longitude to city, province, and country
// using the BigDataCloud server-side API.
// Returns nil and error on failure. Callers should handle gracefully (non-blocking).
func ReverseGeocode(lat, lng float64, apiKey string) (*GeoResult, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("reverse geocode skipped: no API key configured")
	}

	url := fmt.Sprintf(
		"https://api-bdc.net/data/reverse-geocode?latitude=%.7f&longitude=%.7f&localityLanguage=en&key=%s",
		lat, lng, apiKey,
	)

	resp, err := reverseGeocodeClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("reverse geocode request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reverse geocode returned status %d", resp.StatusCode)
	}

	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reverse geocode read body failed: %w", err)
	}

	return parseGeoResponse(body)
}

// parseGeoResponse parses a BigDataCloud JSON response into a GeoResult.
func parseGeoResponse(data []byte) (*GeoResult, error) {
	var bdcResp bigDataCloudResponse
	if err := json.Unmarshal(data, &bdcResp); err != nil {
		return nil, fmt.Errorf("reverse geocode decode failed: %w", err)
	}

	city := bdcResp.City
	if city == "" {
		city = bdcResp.Locality
	}

	return &GeoResult{
		City:        city,
		Province:    bdcResp.PrincipalSubdivision,
		Country:     bdcResp.CountryName,
		CountryCode: bdcResp.CountryCode,
	}, nil
}
