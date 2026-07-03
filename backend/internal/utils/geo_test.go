package utils

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name     string
		lat1     float64
		lng1     float64
		lat2     float64
		lng2     float64
		wantMin  float64
		wantMax  float64
	}{
		{
			name:    "SamePoint",
			lat1:    -6.2088, lng1: 106.8456,
			lat2:    -6.2088, lng2: 106.8456,
			wantMin: 0, wantMax: 0.01,
		},
		{
			name:    "JakartaToBandung",
			lat1:    -6.2088, lng1: 106.8456,
			lat2:    -6.9175, lng2: 107.6191,
			wantMin: 110_000, wantMax: 130_000,
		},
		{
			name:    "JakartaToSurabaya",
			lat1:    -6.2088, lng1: 106.8456,
			lat2:    -7.2575, lng2: 112.7521,
			wantMin: 640_000, wantMax: 680_000,
		},
		{
			name:    "CrossEquator_SingaporeToJakarta",
			lat1:    1.3521, lng1: 103.8198,
			lat2:    -6.2088, lng2: 106.8456,
			wantMin: 870_000, wantMax: 910_000,
		},
		{
			name:    "DateLineCrossing_FijiToSamoa",
			lat1:    -17.7134, lng1: 178.0650,
			lat2:    -13.8333, lng2: -171.7500,
			wantMin: 1_000_000, wantMax: 1_200_000,
		},
		{
			name:    "OneDegreeLatitudeReference",
			lat1:    0, lng1: 0,
			lat2:    1, lng2: 0,
			wantMin: 110_000, wantMax: 112_000,
		},
		{
			name:    "Antipodal_NorthPoleToSouthPole",
			lat1:    90, lng1: 0,
			lat2:    -90, lng2: 0,
			wantMin: 20_000_000, wantMax: 20_100_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HaversineDistance(tt.lat1, tt.lng1, tt.lat2, tt.lng2)
			assert.GreaterOrEqual(t, got, tt.wantMin, "distance too small")
			assert.LessOrEqual(t, got, tt.wantMax, "distance too large")
		})
	}
}

func TestHaversineDistance_Symmetry(t *testing.T) {
	d1 := HaversineDistance(-6.2088, 106.8456, -7.2575, 112.7521)
	d2 := HaversineDistance(-7.2575, 112.7521, -6.2088, 106.8456)
	assert.InDelta(t, d1, d2, 0.001, "Haversine should be symmetric")
}

func TestHaversineDistance_NonNegative(t *testing.T) {
	d := HaversineDistance(0, 0, 45, 90)
	assert.False(t, math.IsNaN(d), "should not be NaN")
	assert.False(t, math.IsInf(d, 0), "should not be Inf")
	assert.GreaterOrEqual(t, d, 0.0, "distance should be non-negative")
}

func TestGeofenceSeverity(t *testing.T) {
	tests := []struct {
		name     string
		distance float64
		want     string
	}{
		{"ZeroKm", 0, "low"},
		{"FiveKm", 5, "low"},
		{"LowBoundary_10km", 10, "low"},
		{"MediumStart_10.1km", 10.1, "medium"},
		{"MediumMiddle_30km", 30, "medium"},
		{"MediumBoundary_50km", 50, "medium"},
		{"HighStart_50.1km", 50.1, "high"},
		{"HighMiddle_100km", 100, "high"},
		{"HighBoundary_200km", 200, "high"},
		{"CriticalStart_200.1km", 200.1, "critical"},
		{"Critical_500km", 500, "critical"},
		{"Critical_1000km", 1000, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeofenceSeverity(tt.distance)
			assert.Equal(t, tt.want, got)
		})
	}
}
