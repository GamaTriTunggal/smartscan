package utils

// courierTrackingURLs maps courier names to their public tracking page URLs.
// These are free URL redirects (no API integration needed).
var courierTrackingURLs = map[string]string{
	"JNE":           "https://www.jne.co.id/id/tracking/trace",
	"J&T Express":   "https://www.jet.co.id/track",
	"SiCepat":       "https://www.sicepat.com/checkAwb",
	"AnterAja":      "https://anteraja.id/tracking",
	"Pos Indonesia":  "https://www.posindonesia.co.id/id/tracking",
	"GoSend":        "",
	"GrabExpress":   "",
	"Paxel":         "https://paxel.co/id/tracking",
	"Deliveree":     "",
	"Lalamove":      "",
}

// GetTrackingURL returns the public tracking page URL for a courier.
// Returns empty string if no public tracking page exists.
func GetTrackingURL(courier string) string {
	return courierTrackingURLs[courier]
}

// GetCourierList returns the list of supported couriers.
func GetCourierList() []string {
	return []string{
		"JNE", "J&T Express", "SiCepat", "AnterAja", "Pos Indonesia",
		"GoSend", "GrabExpress", "Paxel", "Deliveree", "Lalamove", "Other",
	}
}
