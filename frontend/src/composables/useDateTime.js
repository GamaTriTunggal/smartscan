/**
 * useDateTime - Centralized date/time formatting composable
 *
 * Strategy: Store UTC, Display Local
 * - All timestamps from API are in UTC (ISO 8601 format)
 * - Browser automatically converts to user's local timezone when parsing
 * - Display uses user's browser locale (auto-detected with Indonesian fallback)
 */

export function useDateTime() {
  /**
   * Get browser locale with fallback to Indonesian
   */
  const getLocale = () => {
    return navigator.language || 'id-ID'
  }

  /**
   * Format date only (e.g., "15 Jan 2025")
   * Automatically converts UTC to local timezone
   */
  const formatDate = (dateStr, options) => {
    if (!dateStr) return '-'
    try {
      const date = new Date(dateStr)
      if (isNaN(date.getTime())) return '-'
      return date.toLocaleDateString(getLocale(), {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        ...options
      })
    } catch {
      return '-'
    }
  }

  /**
   * Format datetime (e.g., "15 Jan 2025, 08:30")
   * Automatically converts UTC to local timezone
   */
  const formatDateTime = (dateStr, options) => {
    if (!dateStr) return '-'
    try {
      const date = new Date(dateStr)
      if (isNaN(date.getTime())) return '-'
      return date.toLocaleString(getLocale(), {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        ...options
      })
    } catch {
      return '-'
    }
  }

  /**
   * Format with full month name (e.g., "15 January 2025")
   */
  const formatDateLong = (dateStr) => {
    if (!dateStr) return '-'
    try {
      const date = new Date(dateStr)
      if (isNaN(date.getTime())) return '-'
      return date.toLocaleDateString(getLocale(), {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      })
    } catch {
      return '-'
    }
  }

  /**
   * Format time only (e.g., "08:30")
   */
  const formatTime = (dateStr) => {
    if (!dateStr) return '-'
    try {
      const date = new Date(dateStr)
      if (isNaN(date.getTime())) return '-'
      return date.toLocaleTimeString(getLocale(), {
        hour: '2-digit',
        minute: '2-digit',
      })
    } catch {
      return '-'
    }
  }

  /**
   * Convert local date input to UTC ISO string (for sending to API)
   * HTML date input gives YYYY-MM-DD in local time
   */
  const toUTCString = (localDateStr) => {
    if (!localDateStr) return null
    try {
      return new Date(localDateStr).toISOString()
    } catch {
      return null
    }
  }

  /**
   * Get user's timezone name (e.g., "Asia/Jakarta", "Asia/Manila")
   */
  const getTimezone = () => {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  }

  /**
   * Format relative time (e.g., "2 hours ago", "3 days ago")
   * Uses Intl.RelativeTimeFormat if available
   */
  const formatRelative = (dateStr) => {
    if (!dateStr) return '-'
    try {
      const date = new Date(dateStr)
      if (isNaN(date.getTime())) return '-'

      const now = new Date()
      const diffMs = now.getTime() - date.getTime()
      const diffSec = Math.floor(diffMs / 1000)
      const diffMin = Math.floor(diffSec / 60)
      const diffHour = Math.floor(diffMin / 60)
      const diffDay = Math.floor(diffHour / 24)

      // Use RelativeTimeFormat if available
      if (typeof Intl !== 'undefined' && Intl.RelativeTimeFormat) {
        const rtf = new Intl.RelativeTimeFormat(getLocale(), { numeric: 'auto' })

        if (diffDay > 30) {
          return formatDate(dateStr)
        } else if (diffDay >= 1) {
          return rtf.format(-diffDay, 'day')
        } else if (diffHour >= 1) {
          return rtf.format(-diffHour, 'hour')
        } else if (diffMin >= 1) {
          return rtf.format(-diffMin, 'minute')
        } else {
          return rtf.format(-diffSec, 'second')
        }
      }

      // Fallback for older browsers
      if (diffDay >= 1) {
        return `${diffDay} day${diffDay > 1 ? 's' : ''} ago`
      } else if (diffHour >= 1) {
        return `${diffHour} hour${diffHour > 1 ? 's' : ''} ago`
      } else if (diffMin >= 1) {
        return `${diffMin} minute${diffMin > 1 ? 's' : ''} ago`
      } else {
        return 'just now'
      }
    } catch {
      return '-'
    }
  }

  return {
    formatDate,
    formatDateTime,
    formatDateLong,
    formatTime,
    formatRelative,
    toUTCString,
    getTimezone,
    getLocale
  }
}
