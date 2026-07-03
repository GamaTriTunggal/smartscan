package utils

import (
	"regexp"
	"strings"
)

// DeviceInfo contains parsed user agent information
type DeviceInfo struct {
	DeviceType string // mobile, desktop, tablet, bot
	OS         string // Windows 10, macOS 14, Android 14, iOS 17
	Browser    string // Chrome 120, Firefox 121, Safari 17
}

// Common bot patterns
var botPatterns = regexp.MustCompile(`(?i)(bot|crawler|spider|slurp|bingpreview|facebookexternalhit|facebot|twitterbot|linkedinbot|whatsapp|telegram|googlebot|baiduspider|yandexbot|sogou|duckduckbot)`)

// Mobile device patterns
var mobilePatterns = regexp.MustCompile(`(?i)(android|iphone|ipod|blackberry|windows phone|opera mini|iemobile|mobile)`)

// Tablet patterns
var tabletPatterns = regexp.MustCompile(`(?i)(ipad|tablet|playbook|silk|kindle)`)

// ParseUserAgent parses a User-Agent string and returns device information
func ParseUserAgent(uaString string) DeviceInfo {
	info := DeviceInfo{
		DeviceType: "unknown",
		OS:         "Unknown",
		Browser:    "Unknown",
	}

	if uaString == "" {
		return info
	}

	// Detect device type
	info.DeviceType = detectDeviceType(uaString)

	// Detect OS
	info.OS = detectOS(uaString)

	// Detect browser
	info.Browser = detectBrowser(uaString)

	return info
}

func detectDeviceType(ua string) string {
	// Check for bots first
	if botPatterns.MatchString(ua) {
		return "bot"
	}

	// Check for tablets (before mobile since iPad contains mobile sometimes)
	if tabletPatterns.MatchString(ua) {
		return "tablet"
	}

	// Check for mobile devices
	if mobilePatterns.MatchString(ua) {
		return "mobile"
	}

	return "desktop"
}

func detectOS(ua string) string {
	// Windows versions
	if strings.Contains(ua, "Windows NT 10.0") {
		if strings.Contains(ua, "Windows NT 10.0; Win64") {
			return "Windows 10/11"
		}
		return "Windows 10"
	}
	if strings.Contains(ua, "Windows NT 6.3") {
		return "Windows 8.1"
	}
	if strings.Contains(ua, "Windows NT 6.2") {
		return "Windows 8"
	}
	if strings.Contains(ua, "Windows NT 6.1") {
		return "Windows 7"
	}
	if strings.Contains(ua, "Windows") {
		return "Windows"
	}

	// macOS versions
	if strings.Contains(ua, "Mac OS X") {
		// Extract version like "Mac OS X 10_15" or "Mac OS X 14_0"
		re := regexp.MustCompile(`Mac OS X (\d+)[_.](\d+)`)
		if matches := re.FindStringSubmatch(ua); len(matches) >= 3 {
			major := matches[1]
			minor := matches[2]
			if major >= "11" || major == "10" && minor >= "16" {
				return "macOS " + major
			}
			return "macOS 10." + minor
		}
		return "macOS"
	}

	// iOS
	if strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad") || strings.Contains(ua, "iPod") {
		re := regexp.MustCompile(`OS (\d+)[_.](\d+)`)
		if matches := re.FindStringSubmatch(ua); len(matches) >= 2 {
			return "iOS " + matches[1]
		}
		return "iOS"
	}

	// Android
	if strings.Contains(ua, "Android") {
		re := regexp.MustCompile(`Android (\d+)`)
		if matches := re.FindStringSubmatch(ua); len(matches) >= 2 {
			return "Android " + matches[1]
		}
		return "Android"
	}

	// Linux
	if strings.Contains(ua, "Linux") {
		if strings.Contains(ua, "Ubuntu") {
			return "Ubuntu Linux"
		}
		return "Linux"
	}

	// Chrome OS
	if strings.Contains(ua, "CrOS") {
		return "Chrome OS"
	}

	return "Unknown"
}

func detectBrowser(ua string) string {
	// Edge (must check before Chrome since Edge contains Chrome)
	if strings.Contains(ua, "Edg/") || strings.Contains(ua, "Edge/") {
		re := regexp.MustCompile(`Edg[e]?/(\d+)`)
		if matches := re.FindStringSubmatch(ua); len(matches) >= 2 {
			return "Edge " + matches[1]
		}
		return "Edge"
	}

	// Opera (must check before Chrome since Opera contains Chrome)
	if strings.Contains(ua, "OPR/") || strings.Contains(ua, "Opera") {
		re := regexp.MustCompile(`OPR/(\d+)`)
		if matches := re.FindStringSubmatch(ua); len(matches) >= 2 {
			return "Opera " + matches[1]
		}
		return "Opera"
	}

	// Samsung Browser (must check before Chrome)
	if strings.Contains(ua, "SamsungBrowser") {
		re := regexp.MustCompile(`SamsungBrowser/(\d+)`)
		if matches := re.FindStringSubmatch(ua); len(matches) >= 2 {
			return "Samsung Browser " + matches[1]
		}
		return "Samsung Browser"
	}

	// Chrome
	if strings.Contains(ua, "Chrome/") && !strings.Contains(ua, "Chromium") {
		re := regexp.MustCompile(`Chrome/(\d+)`)
		if matches := re.FindStringSubmatch(ua); len(matches) >= 2 {
			// Check if mobile
			if strings.Contains(ua, "Mobile") {
				return "Chrome Mobile " + matches[1]
			}
			return "Chrome " + matches[1]
		}
		return "Chrome"
	}

	// Firefox
	if strings.Contains(ua, "Firefox/") {
		re := regexp.MustCompile(`Firefox/(\d+)`)
		if matches := re.FindStringSubmatch(ua); len(matches) >= 2 {
			if strings.Contains(ua, "Mobile") {
				return "Firefox Mobile " + matches[1]
			}
			return "Firefox " + matches[1]
		}
		return "Firefox"
	}

	// Safari (must check after Chrome since Chrome on iOS contains Safari)
	if strings.Contains(ua, "Safari/") && !strings.Contains(ua, "Chrome") {
		re := regexp.MustCompile(`Version/(\d+)`)
		if matches := re.FindStringSubmatch(ua); len(matches) >= 2 {
			if strings.Contains(ua, "Mobile") {
				return "Safari Mobile " + matches[1]
			}
			return "Safari " + matches[1]
		}
		return "Safari"
	}

	// IE
	if strings.Contains(ua, "MSIE") || strings.Contains(ua, "Trident") {
		re := regexp.MustCompile(`MSIE (\d+)|rv:(\d+)`)
		if matches := re.FindStringSubmatch(ua); len(matches) >= 2 {
			version := matches[1]
			if version == "" && len(matches) >= 3 {
				version = matches[2]
			}
			if version != "" {
				return "IE " + version
			}
		}
		return "IE"
	}

	return "Unknown"
}
