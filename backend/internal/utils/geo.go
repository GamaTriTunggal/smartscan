package utils

import "math"

// HaversineDistance calculates distance between two lat/lng points in meters
// using the Haversine formula (great-circle distance on Earth's surface).
func HaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000 // meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// GeofenceSeverity returns severity level based on distance from zone edge in kilometers.
// Distances: 0-10km=low, 10-50km=medium, 50-200km=high, >200km=critical
func GeofenceSeverity(distanceFromEdgeKm float64) string {
	switch {
	case distanceFromEdgeKm <= 10:
		return "low"
	case distanceFromEdgeKm <= 50:
		return "medium"
	case distanceFromEdgeKm <= 200:
		return "high"
	default:
		return "critical"
	}
}
