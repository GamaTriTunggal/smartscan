/**
 * Social media handle validation composable
 * Provides real-time validation and normalization for social media inputs
 */

// Validation types matching backend
const VALIDATION_TYPE = {
  PHONE: 'phone',
  USERNAME: 'username',
  EMAIL: 'email',
  URL: 'url',
  TEXT: 'text'
}

// Username regex: alphanumeric, underscore, period, 1-30 chars
const USERNAME_REGEX = /^[a-zA-Z0-9_.]{1,30}$/

// Basic E.164 regex: + followed by 7-15 digits
const E164_REGEX = /^\+[1-9]\d{6,14}$/

// Email regex (basic)
const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

/**
 * Validate phone number (E.164 format)
 * @param {string} value - Phone number to validate
 * @returns {{ valid: boolean, normalized: string, error: string|null }}
 */
function validatePhone(value) {
  if (!value || !value.trim()) {
    return { valid: false, normalized: '', error: 'Phone number is required' }
  }

  // Remove formatting characters (spaces, dashes, parentheses, periods)
  let normalized = value.replace(/[\s\-\(\)\.]/g, '')

  // Must start with +
  if (!normalized.startsWith('+')) {
    return {
      valid: false,
      normalized,
      error: 'Phone must start with country code (e.g., +62 for Indonesia)'
    }
  }

  // Check for valid E.164 format
  if (!E164_REGEX.test(normalized)) {
    return {
      valid: false,
      normalized,
      error: 'Invalid phone format. Use +[country code][number] (e.g., +6281234567890)'
    }
  }

  return { valid: true, normalized, error: null }
}

/**
 * Validate social media username
 * @param {string} value - Username to validate
 * @returns {{ valid: boolean, normalized: string, error: string|null }}
 */
function validateUsername(value) {
  if (!value || !value.trim()) {
    return { valid: false, normalized: '', error: 'Username is required' }
  }

  // Strip @ prefix if present
  let normalized = value.trim().replace(/^@/, '')

  if (!normalized) {
    return { valid: false, normalized: '', error: 'Username cannot be empty' }
  }

  // Check length
  if (normalized.length > 30) {
    return {
      valid: false,
      normalized,
      error: 'Username must be 30 characters or less'
    }
  }

  // Check format
  if (!USERNAME_REGEX.test(normalized)) {
    return {
      valid: false,
      normalized,
      error: 'Username can only contain letters, numbers, underscores, and periods'
    }
  }

  return { valid: true, normalized, error: null }
}

/**
 * Validate email address
 * @param {string} value - Email to validate
 * @returns {{ valid: boolean, normalized: string, error: string|null }}
 */
function validateEmail(value) {
  if (!value || !value.trim()) {
    return { valid: false, normalized: '', error: 'Email is required' }
  }

  // Normalize to lowercase
  let normalized = value.trim().toLowerCase()

  // Check format
  if (!EMAIL_REGEX.test(normalized)) {
    return {
      valid: false,
      normalized,
      error: 'Invalid email format'
    }
  }

  return { valid: true, normalized, error: null }
}

/**
 * Validate URL
 * @param {string} value - URL to validate
 * @returns {{ valid: boolean, normalized: string, error: string|null }}
 */
function validateUrl(value) {
  if (!value || !value.trim()) {
    return { valid: false, normalized: '', error: 'URL is required' }
  }

  let normalized = value.trim()

  // Add https:// if no scheme
  if (!normalized.startsWith('http://') && !normalized.startsWith('https://')) {
    normalized = 'https://' + normalized
  }

  // Try to parse URL
  try {
    const url = new URL(normalized)

    // Must have a host
    if (!url.host) {
      return {
        valid: false,
        normalized,
        error: 'URL must include a domain'
      }
    }

    // Only allow http/https
    if (url.protocol !== 'http:' && url.protocol !== 'https:') {
      return {
        valid: false,
        normalized,
        error: 'URL must use http or https'
      }
    }

    return { valid: true, normalized, error: null }
  } catch {
    return {
      valid: false,
      normalized,
      error: 'Invalid URL format'
    }
  }
}

/**
 * Validate text (no validation, just trim)
 * @param {string} value - Text to validate
 * @returns {{ valid: boolean, normalized: string, error: string|null }}
 */
function validateText(value) {
  const normalized = (value || '').trim()
  if (!normalized) {
    return { valid: false, normalized: '', error: 'Value is required' }
  }
  return { valid: true, normalized, error: null }
}

/**
 * Format phone number for display (E.164 → readable format)
 * @param {string} phone - Phone in E.164 format (e.g., +6281234567890)
 * @returns {string} Formatted phone (e.g., +62 812-345-67890)
 */
function formatPhoneDisplay(phone) {
  if (!phone || !phone.startsWith('+')) return phone

  // Remove + and get digits
  const digits = phone.slice(1)

  // Format based on length - most phone numbers have 10+ digits after country code
  if (digits.length >= 10) {
    // Detect country code length (1-3 digits typically)
    // Common: +1 (US), +62 (ID), +65 (SG), +852 (HK), +91 (IN)
    let countryCodeLen = 2 // Default assumption
    if (digits.startsWith('1') && digits.length === 11) {
      countryCodeLen = 1 // US/Canada
    } else if (digits.startsWith('852') || digits.startsWith('853')) {
      countryCodeLen = 3 // HK/Macau
    }

    const countryCode = digits.slice(0, countryCodeLen)
    const rest = digits.slice(countryCodeLen)

    // Group remaining digits in 3-3-4+ pattern
    if (rest.length >= 10) {
      const formatted = rest.replace(/(\d{3})(\d{3})(\d+)/, '$1-$2-$3')
      return `+${countryCode} ${formatted}`
    } else if (rest.length >= 7) {
      const formatted = rest.replace(/(\d{3})(\d+)/, '$1-$2')
      return `+${countryCode} ${formatted}`
    }

    return `+${countryCode} ${rest}`
  }

  return phone
}

/**
 * Main composable function
 */
export function useSocialValidation() {
  /**
   * Validate handle based on validation type
   * @param {string} validationType - 'phone' | 'username' | 'email' | 'url' | 'text'
   * @param {string} value - Value to validate
   * @returns {{ valid: boolean, normalized: string, error: string|null }}
   */
  const validateHandle = (validationType, value) => {
    switch (validationType) {
      case VALIDATION_TYPE.PHONE:
        return validatePhone(value)
      case VALIDATION_TYPE.USERNAME:
        return validateUsername(value)
      case VALIDATION_TYPE.EMAIL:
        return validateEmail(value)
      case VALIDATION_TYPE.URL:
        return validateUrl(value)
      case VALIDATION_TYPE.TEXT:
      default:
        return validateText(value)
    }
  }

  /**
   * Get placeholder hint based on validation type
   * @param {string} validationType
   * @returns {string}
   */
  const getValidationHint = (validationType) => {
    switch (validationType) {
      case VALIDATION_TYPE.PHONE:
        return 'Include country code (e.g., +62 for Indonesia)'
      case VALIDATION_TYPE.USERNAME:
        return 'Letters, numbers, underscores, periods only'
      case VALIDATION_TYPE.EMAIL:
        return 'Enter a valid email address'
      case VALIDATION_TYPE.URL:
        return 'Enter full URL (https:// will be added if missing)'
      default:
        return ''
    }
  }

  return {
    validateHandle,
    getValidationHint,
    formatPhoneDisplay,
    VALIDATION_TYPE
  }
}
