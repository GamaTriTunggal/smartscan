<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import DOMPurify from 'dompurify'
import { useDateTime } from '@/composables/useDateTime'
import { useTranslation } from '@/composables/useTranslation'
import { useBrandingStore } from '@/stores/branding'
import { SOCIAL_ICON_PATHS } from '@/lib/socialIcons'
import CompanyContactCard from '@/components/public/CompanyContactCard.vue'

const route = useRoute()
const brandingStore = useBrandingStore()
const router = useRouter()
const { formatDate } = useDateTime()
const { t, lang } = useTranslation()
const uuid = computed(() => route.params.uuid)

const loading = ref(true)
const error = ref(null)
const validationData = ref(null)
const templateData = ref(null)

// Collapsible section states
const certificationsExpanded = ref(false)
const socialLinksExpanded = ref(false)
const socialAccountsExpanded = ref(false)

// Gallery lightbox state
const lightboxOpen = ref(false)
const lightboxIndex = ref(0)

// Geolocation request lock to prevent race condition
const geoRequestInProgress = ref(false)

// Counterfeit report state
const showReportModal = ref(false)
const reportForm = ref({
  description: '',
  location: ''
})
const reportPhotos = ref([])       // File objects for upload
const photoPreviews = ref([])      // createObjectURL previews
const submittingReport = ref(false)
const uploadProgress = ref(0)      // 0-100 upload progress
const reportSuccess = ref(false)
const reportError = ref('')

// Geolocation permission state (soft-force for scan flow)
// States: 'pending' (show request UI) | 'requesting' (waiting for browser) | 'granted' (allowed) |
//         'denied' (dismissed prompt) | 'blocked' (explicitly blocked, show instructions) | 'not_applicable' (direct URL)
const geoPermissionStatus = ref('pending')
// Check if scan session param is present in URL - will be verified via API
const hasScanSession = computed(() => !!route.query.s)

const MAX_PHOTOS = 5
const MAX_PHOTO_SIZE = 5 * 1024 * 1024 // 5MB

const apiBase = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
const uploadBase = apiBase.replace('/api/v1', '')

// Resolve logo URLs (supports both absolute R2 URLs and relative /uploads/ paths)
function getLogoUrl(url) {
  if (!url) return ''
  if (url.startsWith('http://') || url.startsWith('https://')) return url
  if (url.startsWith('/uploads/') && !url.includes('..')) return uploadBase + url
  return url
}

function copyToClipboard(text) {
  navigator.clipboard.writeText(text).catch(() => {})
}

// Verify scan session via backend API
// Returns true if session is valid and geo is required
const verifyScanSession = async () => {
  const session = route.query.s

  if (!session) {
    return false // No session = direct URL access
  }

  try {
    const response = await axios.get(`${apiBase}/public/verify-scan-session`, {
      params: { code: uuid.value, s: session }
    })
    return response.data?.data?.geo_required === true
  } catch (e) {
    console.log('Session verification failed:', e.message)
    return false
  }
}

// Remove session param from URL after permission granted (prevents Back button showing overlay again)
const removeSessionFromURL = () => {
  const newQuery = { ...route.query }
  delete newQuery.s
  router.replace({
    path: route.path,
    query: Object.keys(newQuery).length ? newQuery : undefined
  })
}

// Send geolocation to backend
const sendLocationToBackend = async (latitude, longitude, accuracy) => {
  try {
    await axios.post(`${apiBase}/public/scan-location`, {
      qr_code: uuid.value,
      latitude,
      longitude,
      accuracy: Math.round(accuracy)
    })
  } catch (err) {
    console.log('Failed to send scan location:', err.message)
  }
}

// Request geolocation permission (soft-force for scan flow)
const requestGeolocation = async () => {
  // Prevent race condition with concurrent requests
  if (geoRequestInProgress.value) return

  if (!navigator.geolocation) {
    console.log('Geolocation not supported')
    geoPermissionStatus.value = 'denied'
    return
  }

  geoRequestInProgress.value = true
  geoPermissionStatus.value = 'requesting'

  try {
    const position = await new Promise((resolve, reject) => {
      navigator.geolocation.getCurrentPosition(resolve, reject, {
        timeout: 15000,
        enableHighAccuracy: false,
        maximumAge: 60000 // Accept cached position up to 1 minute old
      })
    })

    geoPermissionStatus.value = 'granted'

    const { latitude, longitude, accuracy } = position.coords
    await sendLocationToBackend(latitude, longitude, accuracy)
    removeSessionFromURL()
  } catch (err) {
    console.log('Geolocation error:', err.message)
    geoPermissionStatus.value = 'denied'
  } finally {
    geoRequestInProgress.value = false
  }
}

// Retry geolocation - check permission state to determine next action
const retryGeolocation = async () => {
  // Check current permission state via Permissions API
  if (navigator.permissions) {
    try {
      const permissionStatus = await navigator.permissions.query({ name: 'geolocation' })

      if (permissionStatus.state === 'denied') {
        // User has explicitly blocked - show instructions
        geoPermissionStatus.value = 'blocked'
        return
      }

      if (permissionStatus.state === 'prompt') {
        // User dismissed without choosing - can retry
        geoPermissionStatus.value = 'pending'
        return
      }

      if (permissionStatus.state === 'granted') {
        // Permission was granted (maybe from another tab)
        geoPermissionStatus.value = 'granted'
        try {
          const position = await new Promise((resolve, reject) => {
            navigator.geolocation.getCurrentPosition(resolve, reject, {
              timeout: 15000,
              enableHighAccuracy: false,
              maximumAge: 60000
            })
          })
          const { latitude, longitude, accuracy } = position.coords
          await sendLocationToBackend(latitude, longitude, accuracy)
          removeSessionFromURL()
        } catch (err) {
          console.log('Geolocation retry failed:', err.message)
          geoPermissionStatus.value = 'denied'
        }
        return
      }
    } catch (e) {
      // Permissions API not supported
      console.log('Permissions API not supported:', e.message)
    }
  }

  // Fallback: just reset to pending
  geoPermissionStatus.value = 'pending'
}

const fetchValidation = async () => {
  try {
    // Landing page NEVER records - always use GET
    // Recording only happens at /s/:code (ScanRedirect handler in backend)
    // This ensures refresh never increases scan count (me-qr.com pattern)
    const response = await axios.get(`${apiBase}/public/validate-info/${uuid.value}`)

    if (response.data.success) {
      validationData.value = response.data.data
    } else {
      error.value = response.data.message || 'Validation failed'
    }
  } catch (err) {
    error.value = err.response?.data?.message || 'Failed to validate product'
    console.error('Validation error:', err)
  }
}

const fetchTemplate = async () => {
  try {
    const response = await axios.get(`${apiBase}/public/template/${uuid.value}`, {
      params: { type: 'validation' }
    })
    if (response.data.success && response.data.data) {
      const rawData = response.data.data.template || response.data.data
      const customFields = rawData.custom_fields ? (typeof rawData.custom_fields === 'string' ? JSON.parse(rawData.custom_fields) : rawData.custom_fields) : undefined

      templateData.value = {
        html_content: rawData.html_content,
        css_content: rawData.css_content,
        custom_fields: customFields
      }

      if (templateData.value.custom_fields?.certifications_section?.default_expanded) {
        certificationsExpanded.value = true
      }
      if (templateData.value.custom_fields?.social_media_section?.default_expanded) {
        socialLinksExpanded.value = true
      }
    }
  } catch (err) {
    // No custom template or fetch failed - using default template
    // Log for debugging (helps tenant support identify template issues)
    if (err.response?.status !== 404) {
      console.warn('[ValidatePage] Template fetch failed, using default:', err.message || err)
    }
  }
}

// Photo upload helpers
const addPhotos = (event) => {
  const files = Array.from(event.target.files || [])
  const remaining = MAX_PHOTOS - reportPhotos.value.length
  const toAdd = files.slice(0, remaining)

  for (const file of toAdd) {
    if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
      reportError.value = 'Only JPEG, PNG, and WebP images are allowed'
      continue
    }
    if (file.size > MAX_PHOTO_SIZE) {
      reportError.value = 'Each photo must be under 5MB'
      continue
    }
    reportPhotos.value.push(file)
    photoPreviews.value.push(URL.createObjectURL(file))
  }

  // Reset input so same file can be re-selected
  event.target.value = ''
}

const removePhoto = (index) => {
  URL.revokeObjectURL(photoPreviews.value[index])
  reportPhotos.value.splice(index, 1)
  photoPreviews.value.splice(index, 1)
}

// Submit counterfeit report
const submitReport = async () => {
  submittingReport.value = true
  uploadProgress.value = 0
  reportError.value = ''

  try {
    const formData = new FormData()
    formData.append('qr_code', uuid.value)
    if (reportForm.value.description) formData.append('description', reportForm.value.description)
    if (reportForm.value.location) formData.append('store_name', reportForm.value.location)
    for (const photo of reportPhotos.value) {
      formData.append('photos', photo)
    }

    const response = await axios.post(`${apiBase}/public/counterfeit-report`, formData, {
      onUploadProgress: (progressEvent) => {
        if (progressEvent.total) {
          uploadProgress.value = Math.round((progressEvent.loaded * 100) / progressEvent.total)
        }
      }
    })

    if (response.data.success) {
      reportSuccess.value = true
      resetReportForm()
    } else {
      reportError.value = response.data.message || 'Failed to submit report'
    }
  } catch (err) {
    reportError.value = err.response?.data?.message || 'Failed to submit report'
    console.error('Report error:', err)
  } finally {
    submittingReport.value = false
    uploadProgress.value = 0
  }
}

const resetReportForm = () => {
  reportForm.value = { description: '', location: '' }
  photoPreviews.value.forEach(url => URL.revokeObjectURL(url))
  reportPhotos.value = []
  photoPreviews.value = []
}

const closeReportModal = () => {
  showReportModal.value = false
  reportSuccess.value = false
  reportError.value = ''
  resetReportForm()
}

// Computed styles from template config
const certConfig = computed(() => {
  return templateData.value?.custom_fields?.certifications_section || {
    header_text: 'Certifications',
    icon_color: '#3f3f46',
    bg_color: '#eff6ff',
    default_expanded: false
  }
})

const socialConfig = computed(() => {
  return templateData.value?.custom_fields?.social_media_section || {
    header_text: 'Social Media',
    icon_color: '#9333ea',
    bg_color: '#f3e8ff',
    default_expanded: false
  }
})

// Sticky social media bar — default ON (sticky_enabled !== false)
const isSocialMediaSticky = computed(() => {
  return templateData.value?.custom_fields?.social_media_section?.sticky_enabled !== false
})

// Unified template config — reads custom_fields with validation-state-aware defaults
// When counterfeit: header appearance is system-forced (red) and cannot be overridden by template
const templateConfig = computed(() => {
  const cf = templateData.value?.custom_fields || {}
  const isCounterfeit = validationData.value?.is_counterfeit === true

  // Shared sections (not affected by counterfeit state)
  const styling = {
    card_bg_color: cf.styling?.card_bg_color || '#ffffff',
    field_bg_color: cf.styling?.field_bg_color || '#f9fafb',
    text_color: cf.styling?.text_color || '#111827',
    main_image_size: cf.styling?.main_image_size || 96,
  }
  const warrantyButton = {
    text: cf.warranty_button?.text || 'Activate Warranty',
    bg_color: cf.warranty_button?.bg_color || '#9333ea',
    text_color: cf.warranty_button?.text_color || '#ffffff',
  }

  // Counterfeit: forced red appearance (not overridable by template customization)
  if (isCounterfeit) {
    return {
      header: {
        bg_color: '#dc2626',
        badge_text: 'Suspected Counterfeit',
        badge_bg_color: '#b91c1c',
        badge_text_color: '#ffffff',
        logo_enabled: cf.header?.logo_enabled || false,
        logo_url: cf.header?.logo_url || null,
        logo_max_height: cf.header?.logo_max_height || 60,
      },
      styling,
      warranty_button: warrantyButton,
    }
  }

  // Valid: template values apply
  return {
    header: {
      bg_color: cf.header?.bg_color || '#16a34a',
      badge_text: cf.header?.badge_text || 'Authentic Product',
      badge_bg_color: cf.header?.badge_bg_color || '#22c55e',
      badge_text_color: cf.header?.badge_text_color || '#ffffff',
      logo_enabled: cf.header?.logo_enabled || false,
      logo_url: cf.header?.logo_url || null,
      logo_max_height: cf.header?.logo_max_height || 60,
    },
    styling,
    warranty_button: warrantyButton,
  }
})

// Helper to lighten color for hover state
const lightenColor = (hex, percent) => {
  const num = parseInt(hex.replace('#', ''), 16)
  const amt = Math.round(2.55 * percent)
  const R = Math.min(255, (num >> 16) + amt)
  const G = Math.min(255, ((num >> 8) & 0x00FF) + amt)
  const B = Math.min(255, (num & 0x0000FF) + amt)
  return `#${(0x1000000 + R * 0x10000 + G * 0x100 + B).toString(16).slice(1)}`
}

// Helper to darken color for text
const darkenColor = (hex, percent) => {
  const num = parseInt(hex.replace('#', ''), 16)
  const amt = Math.round(2.55 * percent)
  const R = Math.max(0, (num >> 16) - amt)
  const G = Math.max(0, ((num >> 8) & 0x00FF) - amt)
  const B = Math.max(0, (num & 0x0000FF) - amt)
  return `#${(0x1000000 + R * 0x10000 + G * 0x100 + B).toString(16).slice(1)}`
}

// HTML entity escaping for safe interpolation in v-html templates
const escapeHtml = (str) => {
  if (!str) return ''
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#x27;')
}

const replacePlaceholders = (html) => {
  if (!validationData.value) return html

  const data = validationData.value
  const dc = data.display_config || {}

  // Generate certifications HTML
  let certificationsHtml = ''
  if (data.certifications && data.certifications.length > 0) {
    certificationsHtml = `<div class="section-card">
      <p class="section-title">Certifications</p>
      <div class="cert-list">${data.certifications.map(cert =>
        `<a href="${sanitizeUrl(cert.website_url) || '#'}" target="_blank" rel="noopener" class="cert-item">
          ${cert.logo_url ? `<img src="${cert.logo_url}" alt="${escapeHtml(cert.name || '')}" class="cert-logo">` : ''}
          <span class="cert-name">${escapeHtml(cert.name || '')}</span>
        </a>`
      ).join('')}</div>
    </div>`
  }

  // Generate social links HTML
  let socialLinksHtml = ''
  if (data.social_links && data.social_links.length > 0) {
    socialLinksHtml = `<div class="section-card">
      <div class="social-list">${data.social_links.map(link => {
        const url = link.handle_or_url.startsWith('http') ? link.handle_or_url : (link.base_url ? link.base_url + link.handle_or_url : link.handle_or_url)
        return `<a href="${sanitizeUrl(url) || '#'}" target="_blank" rel="noopener" class="social-item">
          <span class="social-name">${escapeHtml(link.platform || '')}</span>
        </a>`
      }).join('')}</div>
    </div>`
  }

  // Generate action buttons HTML (using <a> tags since DOMPurify forbids button)
  let actionButtonsHtml = ''
  if (showWarrantyButton.value) {
    actionButtonsHtml = '<div class="action-buttons">'
    actionButtonsHtml += `<a href="/w/${uuid.value}" class="btn-warranty">Activate Warranty</a>`
    actionButtonsHtml += '</div>'
  }

  const replacements = {
    // Existing placeholders (backwards compatible)
    '{{product_name}}': data.product?.name || 'Unknown Product',
    '{{product_code}}': data.product?.code || '',
    '{{brand_name}}': data.tenant?.brand_name || data.tenant?.company_name || '',
    '{{logo_url}}': data.tenant?.logo_url || '/placeholder-logo.png',
    '{{validation_message}}': data.message || '',
    '{{scan_count}}': String(data.scan_count || data.validation_count || 0),
    '{{status_class}}': data.is_valid ? 'status-valid' : 'status-invalid',

    // Raw data placeholders
    '{{batch_code}}': data.batch?.batch_code || '',
    '{{production_date}}': data.batch?.production_date ? formatDate(data.batch.production_date) : '',
    '{{expiry_date}}': data.batch?.expiry_date ? formatDate(data.batch.expiry_date) : '',

    // Section placeholders (respect display_config)
    '{{product_code_section}}': dc.product_code && data.product?.code
      ? `<div class="field-card"><p class="field-label">Product Code</p><p class="field-value">${data.product.code}</p></div>`
      : '',
    '{{brand_name_section}}': dc.brand_name !== false
      ? `<div class="field-card"><p class="field-label">Company</p><p class="field-value">${data.tenant?.brand_name || data.tenant?.company_name || ''}</p></div>`
      : '',
    '{{batch_code_section}}': dc.batch_code && data.batch?.batch_code
      ? `<div class="field-card"><p class="field-label">Batch Code</p><p class="field-value">${data.batch.batch_code}</p></div>`
      : '',
    '{{production_date_section}}': dc.production_date && data.batch?.production_date
      ? `<div class="field-card"><p class="field-label">Production Date</p><p class="field-value">${formatDate(data.batch.production_date)}</p></div>`
      : '',
    '{{expiry_date_section}}': dc.expiry_date && data.batch?.expiry_date
      ? `<div class="field-card"><p class="field-label">Expiry Date</p><p class="field-value">${formatDate(data.batch.expiry_date)}</p></div>`
      : '',
    '{{distribution_zone_section}}': data.distribution_zone
      ? `<div class="field-card"><p class="field-label">Distributed for</p><p class="field-value">${data.distribution_zone}</p></div>`
      : '',
    '{{verification_count_section}}': dc.show_verification_count !== false
      ? `<div class="field-card"><p class="field-label">Verification Count</p><p class="field-value">${data.scan_count || data.validation_count || 0} times</p></div>`
      : '',

    // Dynamic sections
    '{{certifications_section}}': certificationsHtml,
    '{{social_links_section}}': socialLinksHtml,
    '{{action_buttons_section}}': actionButtonsHtml,

    // Gallery images section
    '{{images_section}}': (() => {
      const images = data.images
      if (!images || images.length === 0 || !images.some(img => img && img.image_url)) return ''
      const validImages = images.filter(img => img && img.image_url)
      return `<div class="section-card">
        <p class="section-title">Gallery</p>
        <div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(80px,1fr));gap:6px">
          ${validImages.map(img =>
            `<img src="${img.image_url}" alt="${escapeHtml(img.caption || 'Product image')}" style="width:100%;aspect-ratio:1;object-fit:cover;border-radius:6px">`
          ).join('')}
        </div>
      </div>`
    })(),

    // Videos section (links since iframe is blocked by DOMPurify)
    '{{videos_section}}': (() => {
      const videos = data.videos
      if (!videos || videos.length === 0) return ''
      const videoLinks = videos.map(v => {
        if (!v || !v.video_id || !v.platform) return ''
        if (!/^[\w-]{1,255}$/.test(v.video_id)) return ''
        const urls = {
          youtube: `https://www.youtube.com/watch?v=${v.video_id}`,
          tiktok: `https://www.tiktok.com/@user/video/${v.video_id}`,
          instagram: `https://www.instagram.com/reel/${v.video_id}/`
        }
        const url = urls[v.platform]
        if (!url) return ''
        const label = escapeHtml(v.title || v.caption || `${v.platform} video`)
        return `<a href="${url}" target="_blank" rel="noopener" class="social-item">${label}</a>`
      }).filter(Boolean).join('')
      if (!videoLinks) return ''
      return `<div class="section-card">
        <p class="section-title">Videos</p>
        <div style="display:flex;flex-wrap:wrap;gap:8px">${videoLinks}</div>
      </div>`
    })(),

    // Product description section (auto-show if product has description)
    '{{description_section}}': (() => {
      if (!data.product?.description || !data.product.description.trim()) return ''
      return `<div class="field-card"><p class="field-label">Description</p><p class="field-value" style="font-size:14px;font-weight:400;line-height:1.5">${escapeHtml(data.product.description)}</p></div>`
    })(),

    // Social accounts section (N:M) — hidden when sticky mode is ON
    '{{social_accounts_section}}': (() => {
      if (isSocialMediaSticky.value) return ''
      const accounts = data.social_accounts
      if (!accounts || accounts.length === 0) return ''
      const items = accounts.map(acc => {
        let url = acc.url || acc.account_url || ''
        if (!url && acc.account_handle) {
          // If handle is already a full URL, use it directly
          if (acc.account_handle.startsWith('http://') || acc.account_handle.startsWith('https://')) {
            url = acc.account_handle
          } else if (acc.platform_base_url) {
            // Strip @ from handle to avoid double @@ when base_url already contains @
            url = acc.platform_base_url + acc.account_handle.replace(/^@/, '')
          }
        }
        const safeAccUrl = sanitizeUrl(url) || '#'
        const name = escapeHtml(acc.platform_name || acc.account_handle || acc.platform_code || '')
        return `<a href="${safeAccUrl}" target="_blank" rel="noopener" class="social-item">${name}</a>`
      }).join('')
      return `<div class="section-card">
        <div class="social-list">${items}</div>
      </div>`
    })(),

    // Website link section
    '{{website_link_section}}': (() => {
      const url = data.website_url
      if (!url) return ''
      const safeUrl = sanitizeUrl(url)
      if (!safeUrl) return ''
      const caption = escapeHtml(data.website_caption || 'Visit Website')
      return `<div class="section-card" style="text-align:center">
        <a href="${safeUrl}" target="_blank" rel="noopener" style="color:#3f3f46;text-decoration:none;font-weight:500;font-size:14px">${caption}</a>
      </div>`
    })(),
  }

  let result = html
  for (const [placeholder, value] of Object.entries(replacements)) {
    result = result.replace(new RegExp(placeholder.replace(/[{}]/g, '\\$&'), 'g'), value)
  }
  return result
}

const renderedHtml = computed(() => {
  if (templateData.value?.html_content) {
    const html = replacePlaceholders(templateData.value.html_content)
    return DOMPurify.sanitize(html, {
      ALLOWED_TAGS: ['div', 'span', 'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'img', 'a', 'ul', 'ol', 'li', 'br', 'strong', 'em', 'b', 'i', 'u', 'table', 'tr', 'td', 'th', 'thead', 'tbody', 'section', 'article', 'header', 'footer', 'nav', 'main'],
      ALLOWED_ATTR: ['class', 'id', 'style', 'href', 'src', 'alt', 'title', 'target', 'rel', 'width', 'height'],
      FORBID_TAGS: ['script', 'iframe', 'object', 'embed', 'form', 'input', 'button'],
      FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'onfocus', 'onblur'],
      ALLOW_DATA_ATTR: false
    })
  }
  return null
})

const renderedCss = computed(() => {
  if (!templateData.value?.css_content) return ''
  let css = templateData.value.css_content
  css = css.replace(/javascript:/gi, '')
  css = css.replace(/expression\s*\(/gi, '')
  css = css.replace(/behavior\s*:/gi, '')
  css = css.replace(/-moz-binding\s*:/gi, '')
  return css
})

// Dynamically inject CSS into head
let styleElement = null

watch(renderedCss, (newCss) => {
  if (newCss) {
    if (!styleElement) {
      styleElement = document.createElement('style')
      styleElement.setAttribute('data-validate-template', 'true')
      document.head.appendChild(styleElement)
    }
    styleElement.textContent = newCss
  } else if (styleElement) {
    styleElement.remove()
    styleElement = null
  }
}, { immediate: true })

onUnmounted(() => {
  if (styleElement) {
    styleElement.remove()
    styleElement = null
  }
  // Remove keyboard listener
  document.removeEventListener('keydown', handleLightboxKeydown)
})

// Navigation functions for flow logic
const goToWarranty = () => {
  router.push(`/w/${uuid.value}`)
}

// Button visibility based on flow logic
const showWarrantyButton = computed(() => {
  return validationData.value?.need_warranty === true
})

// Display config helpers
const displayConfig = computed(() => {
  return validationData.value?.display_config || {
    product_name: true,
    product_code: false,
    brand_name: true,
    batch_code: false,
    production_date: false,
    expiry_date: false,
    show_verification_count: true
  }
})

const showProductCode = computed(() => displayConfig.value.product_code && validationData.value?.product?.code)
const showBrandName = computed(() => true) // Always required
const showVerificationCount = computed(() => displayConfig.value.show_verification_count !== false)
const showBatchCode = computed(() => displayConfig.value.batch_code && validationData.value?.batch?.batch_code)
const showProductionDate = computed(() => displayConfig.value.production_date && validationData.value?.batch?.production_date)
const showExpiryDate = computed(() => displayConfig.value.expiry_date && validationData.value?.batch?.expiry_date)

// Whether any optional field in the card is actually visible
const hasVisibleFields = computed(() => {
  return showProductCode.value || showVerificationCount.value || showBatchCode.value || showProductionDate.value || showExpiryDate.value || !!validationData.value?.distribution_zone
})

// Field display order (from display_config.field_order) — includes required fields
const DEFAULT_FIELD_ORDER = ['product_name', 'brand_name', 'product_code', 'show_verification_count', 'batch_code', 'production_date', 'expiry_date']
const REQUIRED_KEYS = ['product_name', 'brand_name']
const fieldOrder = computed(() => {
  const dc = displayConfig.value
  const order = dc.field_order || DEFAULT_FIELD_ORDER
  // Ensure required keys are present (backward compat for old 6-key field_order)
  const missing = REQUIRED_KEYS.filter(k => !order.includes(k))
  return [...new Set([...missing, ...order])]
})

// Field order excluding header fields (product_name, brand_name) rendered separately
const fieldOrderWithoutHeader = computed(() => {
  return fieldOrder.value.filter(k => k !== 'product_name' && k !== 'brand_name')
})

// Certifications and social links
const hasCertifications = computed(() => {
  return validationData.value?.certifications && validationData.value.certifications.length > 0
})

const hasSocialLinks = computed(() => {
  return validationData.value?.social_links && validationData.value.social_links.length > 0
})

// NEW: Social accounts (N:M relationship)
const hasSocialAccounts = computed(() => {
  return validationData.value?.social_accounts && validationData.value.social_accounts.length > 0
})

// Images gallery - only show when there are more than 1 image (main image already shown in header)
const hasImages = computed(() => {
  const images = validationData.value?.images
  if (!images) return false
  const validImages = images.filter(img => img && img.image_url)
  return validImages.length > 1
})

const productImages = computed(() => {
  const images = validationData.value?.images || []
  return images.filter(img => img && img.image_url)
})

// Main image for product header thumbnail
const mainImage = computed(() => {
  const images = productImages.value
  if (!images || images.length === 0) return null
  return images.find(img => img.is_main) || null
})

// Index of main image in productImages array (for lightbox)
const mainImageIndex = computed(() => {
  if (!mainImage.value) return 0
  const idx = productImages.value.findIndex(img => img.image_url === mainImage.value.image_url)
  return idx >= 0 ? idx : 0
})

// NEW: Videos
const hasVideos = computed(() => {
  return validationData.value?.videos && validationData.value.videos.length > 0
})

const productVideos = computed(() => {
  return validationData.value?.videos || []
})

// URL Sanitization utility - prevents XSS via javascript: and other dangerous schemes
const sanitizeUrl = (url) => {
  if (!url || typeof url !== 'string') return null
  const trimmed = url.trim()
  const lower = trimmed.toLowerCase()

  // Security: Block dangerous schemes
  if (lower.startsWith('javascript:') || lower.startsWith('data:') || lower.startsWith('vbscript:')) {
    return null
  }

  // Allow http/https schemes
  if (lower.startsWith('http://') || lower.startsWith('https://')) {
    return trimmed
  }

  // Allow mailto: and tel: for specific use cases
  if (lower.startsWith('mailto:') || lower.startsWith('tel:')) {
    return trimmed
  }

  // Auto-prepend https:// for domain-like URLs (contains . and no dangerous chars)
  if (lower.includes('.') && !lower.includes(' ') && !lower.includes('<') && !lower.includes('>')) {
    return 'https://' + trimmed
  }

  return null
}

// NEW: Website URL - with sanitization
const hasWebsiteUrl = computed(() => {
  return !!sanitizeUrl(validationData.value?.website_url)
})

const sanitizedWebsiteUrl = computed(() => {
  return sanitizeUrl(validationData.value?.website_url)
})

// Section ordering - read from template config
const DEFAULT_SECTION_ORDER = [
  'images', 'description', 'certifications', 'social_accounts',
  'videos', 'website_link', 'warranty_button'
]

const sectionOrder = computed(() => {
  const savedOrder = templateData.value?.custom_fields?.section_order
  if (!savedOrder || !Array.isArray(savedOrder) || savedOrder.length === 0) {
    return DEFAULT_SECTION_ORDER
  }
  // Filter valid sections and add any missing ones
  const validSections = new Set(DEFAULT_SECTION_ORDER)
  const validOrder = savedOrder.filter(id => validSections.has(id))

  // Log warning if invalid sections were removed (for debugging)
  const invalidSections = savedOrder.filter(id => !validSections.has(id))
  if (invalidSections.length > 0) {
    console.warn('[ValidatePage] Invalid section IDs filtered out:', invalidSections)
  }

  const missingNew = DEFAULT_SECTION_ORDER.filter(id => !validOrder.includes(id))
  return [...validOrder, ...missingNew]
})

// Section visibility map
const sectionVisibility = computed(() => ({
  images: hasImages.value,
  videos: hasVideos.value,
  social_accounts: hasSocialAccounts.value,
  certifications: hasCertifications.value,
  website_link: hasWebsiteUrl.value,
  description: showDescription.value,
  warranty_button: showWarrantyButton.value
}))

// Show Description: auto-show if product has description content (no toggle needed)
const showDescription = computed(() => {
  return validationData.value?.product?.description &&
         validationData.value.product.description.trim() !== ''
})

// NEW: Social accounts helpers (show first 6 icons, expand if more)
const visibleSocialAccounts = computed(() => {
  const accounts = validationData.value?.social_accounts || []
  return accounts.slice(0, 6)
})

const hiddenSocialAccounts = computed(() => {
  const accounts = validationData.value?.social_accounts || []
  return accounts.slice(6)
})

const getSocialAccountUrl = (account) => {
  // Try direct URL first
  let url = account.url || account.account_url
  if (url) {
    const sanitized = sanitizeUrl(url)
    return sanitized || '#'
  }
  // Build from base_url + handle
  const baseUrl = account.platform_base_url || ''
  const handle = account.account_handle || ''
  if (handle) {
    // If handle is already a full URL, use it directly
    if (handle.startsWith('http://') || handle.startsWith('https://')) {
      const sanitized = sanitizeUrl(handle)
      return sanitized || '#'
    }
    if (baseUrl) {
      // Validate base_url scheme
      if (!baseUrl.startsWith('http://') && !baseUrl.startsWith('https://') && !baseUrl.startsWith('mailto:')) {
        return '#'
      }
      // Strip @ from handle to avoid double @@ when base_url already contains @
      return baseUrl + handle.replace(/^@/, '')
    }
  }
  return '#'
}

// NEW: Get social account icon
const getSocialAccountIcon = (account) => {
  // Only use platform_icon if it looks like an SVG path (contains space or 'M' for moveto command)
  // API may return just the code string like "instagram" instead of actual SVG path
  if (account.platform_icon && account.platform_icon.includes(' ')) {
    return account.platform_icon
  }
  // Fallback to iconMap using platform_code or platform_icon as code
  const code = (account.platform_code || account.platform_icon || '').toUpperCase()
  return SOCIAL_ICON_PATHS[code] || ''
}

// Sanitize video ID to prevent URL injection
const sanitizeVideoId = (id, platform) => {
  if (!id || typeof id !== 'string') return null
  // Remove any query params or fragments that could inject content
  const cleanId = id.split('?')[0].split('#')[0].split('&')[0]
  // Validate format based on platform
  switch (platform) {
    case 'youtube':
      // YouTube IDs are 11 chars, alphanumeric + dash/underscore
      if (/^[a-zA-Z0-9_-]{11}$/.test(cleanId)) return cleanId
      break
    case 'tiktok':
      // TikTok IDs are numeric, 19 digits
      if (/^\d{15,20}$/.test(cleanId)) return cleanId
      break
    case 'instagram':
      // Instagram reel IDs are alphanumeric
      if (/^[a-zA-Z0-9_-]{5,30}$/.test(cleanId)) return cleanId
      break
  }
  return null
}

// NEW: Video embed URL builder with validation
const getVideoEmbedUrl = (video) => {
  if (!video || !video.video_id || !video.platform) return null
  const validPlatforms = ['youtube', 'tiktok', 'instagram']
  if (!validPlatforms.includes(video.platform)) return null

  const sanitizedId = sanitizeVideoId(video.video_id, video.platform)
  if (!sanitizedId) return null

  switch (video.platform) {
    case 'youtube':
      return `https://www.youtube.com/embed/${sanitizedId}?autoplay=${video.autoplay ? 1 : 0}&mute=1`
    case 'tiktok':
      return `https://www.tiktok.com/embed/v2/${sanitizedId}`
    case 'instagram':
      return `https://www.instagram.com/reel/${sanitizedId}/embed`
    default:
      return null
  }
}

// Video aspect ratio - uses stored setting or falls back to platform default
const getVideoAspectClass = (video) => {
  // Use stored aspect_ratio if available, otherwise use platform default
  const aspectRatio = video.aspect_ratio ||
    (video.platform === 'youtube' ? 'landscape' : 'portrait')

  if (aspectRatio === 'portrait') {
    return 'aspect-[9/20] max-w-[280px] mx-auto'  // Portrait (9:20 modern smartphone), centered
  }
  return 'aspect-video'  // 16:9 landscape
}

// NEW: Lightbox functions with bounds checking
const openLightbox = (index) => {
  const maxIndex = productImages.value.length - 1
  // Ensure index is within bounds
  lightboxIndex.value = Math.max(0, Math.min(index, maxIndex))
  lightboxOpen.value = true
}

const closeLightbox = () => {
  lightboxOpen.value = false
}

const nextImage = () => {
  const maxIndex = productImages.value.length - 1
  if (lightboxIndex.value < maxIndex) {
    lightboxIndex.value = Math.min(lightboxIndex.value + 1, maxIndex)
  }
}

const prevImage = () => {
  if (lightboxIndex.value > 0) {
    lightboxIndex.value = Math.max(lightboxIndex.value - 1, 0)
  }
}

// Keyboard controls for lightbox
const handleLightboxKeydown = (e) => {
  if (!lightboxOpen.value) return
  switch (e.key) {
    case 'Escape':
      closeLightbox()
      break
    case 'ArrowLeft':
      prevImage()
      break
    case 'ArrowRight':
      nextImage()
      break
  }
}

// Image error handler for gallery
const handleImageError = (event) => {
  // Set a fallback or hide the broken image
  event.target.style.display = 'none'
}

// Social link URL builder
const getSocialUrl = (link) => {
  if (link.handle_or_url.startsWith('http://') || link.handle_or_url.startsWith('https://')) {
    return link.handle_or_url
  }
  if (link.base_url) {
    return `${link.base_url}${link.handle_or_url}`
  }
  return link.handle_or_url
}

const openSocialLink = (link) => {
  const url = getSocialUrl(link)
  window.open(url, '_blank', 'noopener,noreferrer')
}

// Get social icon path
const getSocialIcon = (link) => {
  if (link.icon) {
    return link.icon
  }
  return SOCIAL_ICON_PATHS[link.code] || ''
}

// Counterfeit status display
const isCounterfeitWarning = computed(() => {
  return validationData.value?.is_counterfeit === true
})

// Landing appearance config (for background customization)
const landingAppearance = computed(() => validationData.value?.landing_appearance_config || null)

const hasBackground = computed(() => {
  const config = landingAppearance.value
  return config && config.background_type !== 'none' && config.background_url
})

const backgroundStyle = computed(() => {
  if (!hasBackground.value) return {}
  return {
    backgroundImage: `url(${landingAppearance.value.background_url})`,
    backgroundSize: 'cover',
    backgroundPosition: 'center',
    backgroundRepeat: 'no-repeat'
  }
})

const overlayStyle = computed(() => {
  if (!hasBackground.value) return {}
  const config = landingAppearance.value
  return {
    backgroundColor: config.overlay_color || '#000000',
    opacity: (config.overlay_opacity ?? 30) / 100
  }
})

// Glass card style for the main container
const glassCardStyle = computed(() => {
  if (!hasBackground.value) return {}
  const config = landingAppearance.value
  return {
    backgroundColor: `rgba(255, 255, 255, ${(config.card_opacity ?? 90) / 100})`,
    backdropFilter: `blur(${config.card_blur || 0}px)`,
    WebkitBackdropFilter: `blur(${config.card_blur || 0}px)`,
    border: '1px solid rgba(255, 255, 255, 0.3)'
  }
})

// Alias for backward compatibility
const cardStyle = glassCardStyle

// Header style — reads from template custom_fields, falls back to validation state colors
const headerGlassStyle = computed(() => {
  const color = templateConfig.value.header.bg_color
  if (!hasBackground.value) {
    return { backgroundColor: color }
  }
  // Parse hex to rgba with 80% opacity for glass effect
  const hex = color.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  return {
    backgroundColor: `rgba(${r}, ${g}, ${b}, 0.8)`
  }
})

// Field card style for inner elements — reads from template config
const fieldCardStyle = computed(() => {
  if (!hasBackground.value) {
    return {
      backgroundColor: templateConfig.value.styling.field_bg_color,
      boxShadow: 'none'
    }
  }
  return {
    backgroundColor: 'rgba(255, 255, 255, 0.15)',
    border: '1px solid rgba(255, 255, 255, 0.2)'
  }
})

// Text colors — reads from template config, glass mode uses semi-transparent
const textColorStyle = computed(() => {
  if (!hasBackground.value) {
    return { color: templateConfig.value.styling.text_color }
  }
  return { color: 'rgba(0, 0, 0, 0.85)' }
})

const labelColorStyle = computed(() => {
  if (!hasBackground.value) {
    // Derive lighter label color from text_color by mixing with gray
    const hex = templateConfig.value.styling.text_color.replace('#', '')
    const r = Math.min(255, parseInt(hex.substring(0, 2), 16) + 60)
    const g = Math.min(255, parseInt(hex.substring(2, 4), 16) + 60)
    const b = Math.min(255, parseInt(hex.substring(4, 6), 16) + 60)
    return { color: `rgb(${r}, ${g}, ${b})` }
  }
  return { color: 'rgba(0, 0, 0, 0.6)' }
})

onMounted(async () => {
  // Add keyboard listener for lightbox
  document.addEventListener('keydown', handleLightboxKeydown)

  try {
    await Promise.all([fetchValidation(), fetchTemplate()])

    // Verify scan token via backend (replaces ?from=scan check)
    const requiresGeo = await verifyScanSession()

    if (requiresGeo) {
      // Check if browser already has permission granted
      if (navigator.permissions) {
        try {
          const permissionStatus = await navigator.permissions.query({ name: 'geolocation' })

          if (permissionStatus.state === 'granted') {
            // Already granted - collect location silently with proper error handling
            geoPermissionStatus.value = 'granted'
            try {
              const position = await new Promise((resolve, reject) => {
                navigator.geolocation.getCurrentPosition(resolve, reject, {
                  timeout: 15000,
                  enableHighAccuracy: false,
                  maximumAge: 60000
                })
              })
              const { latitude, longitude, accuracy } = position.coords
              await sendLocationToBackend(latitude, longitude, accuracy)
              removeSessionFromURL()
            } catch (err) {
              console.log('Silent geolocation failed:', err.message)
            }
            return
          }

          if (permissionStatus.state === 'denied') {
            // Already blocked - show instructions
            geoPermissionStatus.value = 'blocked'
            return
          }
        } catch (e) {
          // Permissions API not supported, show overlay
          console.log('Permissions API not supported:', e.message)
        }
      }
      // Show permission request overlay
      geoPermissionStatus.value = 'pending'
    } else {
      // Direct URL access or invalid/expired signature - no geolocation requirement
      geoPermissionStatus.value = 'not_applicable'
    }
  } finally {
    loading.value = false
  }
})

// Re-fetch when route param changes (Vue Router reuses component)
watch(uuid, async (newVal, oldVal) => {
  if (!newVal || newVal === oldVal) return

  // Reset all state
  loading.value = true
  error.value = null
  validationData.value = null
  templateData.value = null
  certificationsExpanded.value = false
  socialLinksExpanded.value = false
  socialAccountsExpanded.value = false
  lightboxOpen.value = false
  lightboxIndex.value = 0
  showReportModal.value = false
  reportForm.value = { description: '', location: '' }
  reportPhotos.value = []
  photoPreviews.value = []
  submittingReport.value = false
  uploadProgress.value = 0
  reportSuccess.value = false
  reportError.value = ''

  // Re-fetch
  try {
    await Promise.all([fetchValidation(), fetchTemplate()])
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="min-h-screen relative" :class="{ 'bg-gray-50': !hasBackground }">
    <!-- Background Image Layer -->
    <div v-if="hasBackground" class="absolute inset-0" :style="backgroundStyle"></div>

    <!-- Overlay Layer -->
    <div v-if="hasBackground" class="absolute inset-0" :style="overlayStyle"></div>

    <!-- Loading State -->
    <div v-if="loading" class="min-h-screen flex items-center justify-center relative z-10">
      <div class="text-center">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-zinc-500 mx-auto mb-4"></div>
        <p :class="hasBackground ? 'text-white' : 'text-gray-600'">Validating product...</p>
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="min-h-screen flex items-center justify-center p-4 relative z-10">
      <div class="w-full max-w-md bg-white rounded-lg shadow-lg p-8 text-center" :style="cardStyle">
        <div class="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
          <svg class="w-8 h-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </div>
        <h1 class="text-xl font-bold text-gray-900 mb-2">Validation Failed</h1>
        <p class="text-gray-600">{{ error }}</p>
      </div>
    </div>

    <!-- Geolocation Permission Overlay (Soft Force for Scan Flow) -->
    <div v-else-if="geoPermissionStatus !== 'granted' && geoPermissionStatus !== 'not_applicable'"
         class="min-h-screen flex items-center justify-center p-4 relative z-10 bg-white">

      <!-- PENDING/REQUESTING: Show permission request UI -->
      <div v-if="geoPermissionStatus === 'pending' || geoPermissionStatus === 'requesting'" class="w-full max-w-md text-center">
        <div class="w-20 h-20 bg-zinc-100 rounded-full flex items-center justify-center mx-auto mb-6">
          <svg class="w-10 h-10 text-zinc-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
        </div>

        <!-- Tenant branding -->
        <div v-if="validationData?.tenant?.logo_url" class="mb-4">
          <img :src="validationData.tenant.logo_url"
               :alt="validationData.tenant.company_name"
               class="h-12 mx-auto object-contain" />
        </div>
        <p v-else-if="validationData?.tenant?.company_name" class="text-sm text-gray-500 mb-2">
          {{ validationData.tenant.company_name }}
        </p>

        <h1 class="text-2xl font-bold text-gray-900 mb-4">{{ t('securityVerification') }}</h1>

        <div class="text-gray-600 text-sm space-y-3 mb-6 text-left">
          <p><strong>{{ t('locationPromptTitle') }}</strong>{{ t('locationPromptIntro') }}</p>
          <ul class="list-disc pl-5 space-y-2">
            <li>{{ t('locationReason1') }}</li>
            <li>{{ t('locationReason2') }}</li>
            <li>{{ t('locationReason3') }}</li>
          </ul>
          <p class="text-xs text-gray-500 mt-4">
            {{ t('locationDisclaimer') }}
          </p>
        </div>

        <button
          @click="requestGeolocation"
          :disabled="geoPermissionStatus === 'requesting'"
          class="w-full py-3 px-4 bg-zinc-600 text-white font-semibold rounded-lg hover:bg-zinc-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {{ geoPermissionStatus === 'requesting' ? t('requestingPermission') : t('allowLocation') }}
        </button>
      </div>

      <!-- DENIED: Show verification failed UI (dismissed the prompt) -->
      <div v-else-if="geoPermissionStatus === 'denied'" class="w-full max-w-md text-center">
        <div class="w-20 h-20 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-6">
          <svg class="w-10 h-10 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>

        <h1 class="text-2xl font-bold text-gray-900 mb-4">{{ t('verificationFailed') }}</h1>

        <p class="text-gray-600 mb-6">
          {{ t('deniedMessage') }}
        </p>

        <button
          @click="retryGeolocation"
          class="w-full py-3 px-4 bg-zinc-600 text-white font-semibold rounded-lg hover:bg-zinc-700 transition-colors mb-3"
        >
          {{ t('tryAgain') }}
        </button>

        <p class="text-xs text-gray-500">
          {{ t('deniedHint') }}
        </p>
      </div>

      <!-- BLOCKED: Show instructions how to enable in browser settings -->
      <div v-else-if="geoPermissionStatus === 'blocked'" class="w-full max-w-md text-center">
        <div class="w-20 h-20 bg-amber-100 rounded-full flex items-center justify-center mx-auto mb-6">
          <svg class="w-10 h-10 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        </div>

        <h1 class="text-2xl font-bold text-gray-900 mb-4">{{ t('locationBlocked') }}</h1>

        <p class="text-gray-600 mb-6">
          {{ t('blockedMessage') }}
        </p>

        <div class="text-left bg-gray-50 rounded-lg p-4 mb-6 text-sm">
          <p class="font-medium text-gray-900 mb-2">{{ t('howToEnable') }}</p>
          <ol class="list-decimal pl-5 space-y-1 text-gray-600">
            <li>{{ t('step1') }}</li>
            <li>{{ t('step2') }}</li>
            <li>{{ t('step3') }}</li>
            <li>{{ t('step4') }}</li>
          </ol>
        </div>

        <button
          @click="retryGeolocation"
          class="w-full py-3 px-4 bg-zinc-600 text-white font-semibold rounded-lg hover:bg-zinc-700 transition-colors"
        >
          {{ t('refreshPage') }}
        </button>
      </div>
    </div>

    <!-- Custom Template Rendering -->
    <div v-else-if="renderedHtml" class="custom-template-container relative z-10" :class="{ 'pb-20': isSocialMediaSticky && hasSocialAccounts }">
      <div v-html="renderedHtml"></div>
    </div>

    <!-- Default Template (when no custom template) -->
    <div v-else-if="validationData" class="min-h-screen flex items-center justify-center p-4 relative z-10" :class="{ 'pb-20': isSocialMediaSticky && hasSocialAccounts }">
      <div
        class="w-full max-w-md shadow-lg overflow-hidden"
        :class="hasBackground ? 'rounded-2xl' : 'rounded-lg'"
        :style="{ ...glassCardStyle, ...(!hasBackground ? { backgroundColor: templateConfig.styling.card_bg_color } : {}) }"
      >
        <!-- Header — layout matches template editor preview -->
        <div
          class="p-6 text-center text-white relative"
          :style="headerGlassStyle"
        >
          <!-- Company Logo (only when uploaded) -->
          <div v-if="templateConfig.header.logo_enabled && templateConfig.header.logo_url" class="mb-3">
            <img
              :src="getLogoUrl(templateConfig.header.logo_url)"
              alt="Company Logo"
              class="mx-auto object-contain"
              :style="{ maxHeight: templateConfig.header.logo_max_height + 'px' }"
              @error="$event.target.style.display = 'none'"
            />
          </div>
          <!-- Authenticity Badge (centered, matching editor) -->
          <div
            class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium"
            :style="{ backgroundColor: templateConfig.header.badge_bg_color, color: templateConfig.header.badge_text_color }"
          >
            <!-- Warning triangle for counterfeit, checkmark for valid -->
            <svg v-if="isCounterfeitWarning" class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
            </svg>
            <svg v-else class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
            </svg>
            {{ templateConfig.header.badge_text }}
          </div>
          <!-- Validation message -->
          <p class="text-white/90 mt-2 text-sm">{{ validationData.message }}</p>
        </div>

        <!-- Counterfeit Warning Banner -->
        <div v-if="isCounterfeitWarning" class="mx-4 mt-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-start gap-2">
          <svg class="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
          </svg>
          <div>
            <p class="text-sm font-medium text-red-800">This product may be counterfeit</p>
            <p class="text-xs text-red-600 mt-0.5">Multiple validation attempts detected. If you purchased this product, please verify with the seller.</p>
            <p v-if="validationData.qr_code_ref" class="text-xs text-red-600 mt-1 flex items-center gap-1">
              <span>Ref:</span>
              <code class="font-mono bg-red-100 px-1 rounded cursor-pointer hover:bg-red-200 transition-colors" @click="copyToClipboard(validationData.qr_code_ref)">
                {{ validationData.qr_code_ref }}
              </code>
            </p>
          </div>
        </div>

        <!-- Product Info -->
        <div class="p-6">
          <!-- Brand Logo (only if template doesn't have its own logo in header) -->
          <div v-if="!templateConfig.header.logo_enabled && validationData.tenant?.logo_url" class="flex justify-center mb-4">
            <img
              :src="validationData.tenant.logo_url"
              :alt="validationData.tenant.brand_name || validationData.tenant.company_name"
              class="h-12 object-contain"
            />
          </div>

          <!-- Product Header: image thumbnail + product name + company name -->
          <div class="rounded-lg p-4 mb-3" :style="fieldCardStyle">
            <!-- With main image: flex row layout -->
            <div v-if="mainImage" class="flex items-center gap-4">
              <div
                class="flex-shrink-0 rounded-xl overflow-hidden cursor-pointer hover:opacity-80 transition-opacity bg-gray-100"
                :style="{ width: templateConfig.styling.main_image_size + 'px', height: templateConfig.styling.main_image_size + 'px' }"
                @click="openLightbox(mainImageIndex)"
              >
                <img
                  :src="mainImage.image_url"
                  :alt="validationData.product?.name || 'Product'"
                  class="w-full h-full object-cover"
                  @error="handleImageError"
                />
              </div>
              <div class="min-w-0 flex-1">
                <h2 class="text-xl font-semibold leading-tight" :style="textColorStyle">
                  {{ validationData.product?.name || 'Unknown Product' }}
                </h2>
                <p class="text-sm mt-1" :style="labelColorStyle">
                  {{ validationData.tenant?.brand_name || validationData.tenant?.company_name || 'Unknown' }}
                </p>
              </div>
            </div>
            <!-- Without main image: centered layout -->
            <div v-else class="text-center">
              <h2 class="text-xl font-semibold" :style="textColorStyle">
                {{ validationData.product?.name || 'Unknown Product' }}
              </h2>
              <p class="text-sm mt-1" :style="labelColorStyle">
                {{ validationData.tenant?.brand_name || validationData.tenant?.company_name || 'Unknown' }}
              </p>
            </div>
          </div>

          <!-- Product Fields (ordered by display_config.field_order, excluding header fields) -->
          <div v-if="hasVisibleFields" class="rounded-lg p-4 space-y-3" :style="fieldCardStyle">
            <template v-for="fieldKey in fieldOrderWithoutHeader" :key="fieldKey">
              <div v-if="fieldKey === 'product_code' && showProductCode" class="flex justify-between text-sm">
                <span :style="labelColorStyle">Product Code</span>
                <span class="font-medium" :style="textColorStyle">{{ validationData.product?.code }}</span>
              </div>
              <div v-else-if="fieldKey === 'show_verification_count' && showVerificationCount" class="flex justify-between text-sm">
                <span :style="labelColorStyle">Verification Count</span>
                <span class="font-medium" :style="isCounterfeitWarning ? { color: '#dc2626', fontWeight: 700 } : textColorStyle">{{ validationData.validation_count ?? 0 }} times</span>
              </div>
              <div v-else-if="fieldKey === 'batch_code' && showBatchCode" class="flex justify-between text-sm">
                <span :style="labelColorStyle">Batch Code</span>
                <span class="font-medium" :style="textColorStyle">{{ validationData.batch?.batch_code }}</span>
              </div>
              <div v-else-if="fieldKey === 'production_date' && showProductionDate" class="flex justify-between text-sm">
                <span :style="labelColorStyle">Production Date</span>
                <span class="font-medium" :style="textColorStyle">{{ formatDate(validationData.batch?.production_date || null) }}</span>
              </div>
              <div v-else-if="fieldKey === 'expiry_date' && showExpiryDate" class="flex justify-between text-sm">
                <span :style="labelColorStyle">Expiry Date</span>
                <span class="font-medium" :style="textColorStyle">{{ formatDate(validationData.batch?.expiry_date || null) }}</span>
              </div>
            </template>
            <!-- Distribution zone (always last, not configurable) -->
            <div v-if="validationData.distribution_zone" class="flex justify-between text-sm">
              <span :style="labelColorStyle">Distributed for</span>
              <span class="font-medium" :style="textColorStyle">{{ validationData.distribution_zone }}</span>
            </div>
          </div>

          <!-- Dynamic Sections (ordered by template config) -->
          <template v-for="sectionId in sectionOrder" :key="sectionId">
            <!-- Photo Gallery -->
            <div v-if="sectionId === 'images' && sectionVisibility.images" class="mt-4">
              <p class="text-sm font-medium mb-2" :style="labelColorStyle">Gallery</p>
              <div class="grid grid-cols-5 gap-1.5">
                <div
                  v-for="(img, index) in productImages"
                  :key="img.id || index"
                  @click="openLightbox(index)"
                  class="aspect-square rounded-lg overflow-hidden cursor-pointer hover:opacity-80 transition-opacity bg-gray-100"
                >
                  <img
                    :src="img.image_url"
                    :alt="img.caption || 'Product image'"
                    class="w-full h-full object-cover"
                    @error="handleImageError"
                  />
                </div>
              </div>
            </div>

            <!-- Video Embeds -->
            <div v-if="sectionId === 'videos' && sectionVisibility.videos" class="mt-4 space-y-3">
              <p class="text-sm font-medium mb-2" :style="labelColorStyle">Videos</p>
              <div v-for="(video, index) in productVideos" :key="index" :class="[getVideoAspectClass(video), 'rounded-lg overflow-hidden bg-gray-100']">
                <iframe
                  v-if="getVideoEmbedUrl(video)"
                  :src="getVideoEmbedUrl(video)"
                  class="w-full h-full"
                  frameborder="0"
                  allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                  allowfullscreen
                ></iframe>
                <div v-if="video.caption" class="text-xs text-center mt-1" :style="labelColorStyle">
                  {{ video.caption }}
                </div>
              </div>
            </div>

            <!-- Social Accounts (Horizontal Row - N:M) -->
            <div v-if="sectionId === 'social_accounts' && sectionVisibility.social_accounts && !isSocialMediaSticky" class="mt-4">
              <div class="flex items-center justify-center gap-3 flex-wrap">
                <!-- Visible accounts (first 6) -->
                <a
                  v-for="account in visibleSocialAccounts"
                  :key="account.id || account.platform_code"
                  :href="getSocialAccountUrl(account)"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="w-10 h-10 rounded-full bg-gray-100 flex items-center justify-center hover:bg-gray-200 transition-colors"
                  :title="account.platform_name || account.account_handle"
                >
                  <svg v-if="getSocialAccountIcon(account)" class="w-5 h-5 text-gray-700" viewBox="0 0 24 24" fill="currentColor">
                    <path :d="getSocialAccountIcon(account)" />
                  </svg>
                  <span v-else class="text-xs font-bold text-gray-600">
                    {{ (account.platform_name || account.platform_code || '?').charAt(0) }}
                  </span>
                </a>
                <!-- "+N more" button -->
                <button
                  v-if="hiddenSocialAccounts.length > 0 && !socialAccountsExpanded"
                  @click="socialAccountsExpanded = true"
                  class="text-sm text-gray-500 hover:text-gray-700 underline"
                >
                  +{{ hiddenSocialAccounts.length }} more
                </button>
              </div>
              <!-- Expanded view (show all remaining) -->
              <div v-if="socialAccountsExpanded && hiddenSocialAccounts.length > 0" class="flex items-center justify-center gap-3 flex-wrap mt-2">
                <a
                  v-for="account in hiddenSocialAccounts"
                  :key="account.id || account.platform_code"
                  :href="getSocialAccountUrl(account)"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="w-10 h-10 rounded-full bg-gray-100 flex items-center justify-center hover:bg-gray-200 transition-colors"
                  :title="account.platform_name || account.account_handle"
                >
                  <svg v-if="getSocialAccountIcon(account)" class="w-5 h-5 text-gray-700" viewBox="0 0 24 24" fill="currentColor">
                    <path :d="getSocialAccountIcon(account)" />
                  </svg>
                  <span v-else class="text-xs font-bold text-gray-600">
                    {{ (account.platform_name || account.platform_code || '?').charAt(0) }}
                  </span>
                </a>
              </div>
            </div>

            <!-- Certifications Section (Collapsible) -->
            <div
              v-if="sectionId === 'certifications' && sectionVisibility.certifications"
              class="mt-4 rounded-lg overflow-hidden"
              :style="{ backgroundColor: certConfig.bg_color }"
            >
              <button
                @click="certificationsExpanded = !certificationsExpanded"
                class="w-full flex items-center justify-between py-3 px-4 transition-colors"
              >
                <div class="flex items-center gap-2 min-w-0">
                  <!-- Fallback: show label if no cert has a logo -->
                  <template v-if="!(validationData.certifications || []).some(c => c.logo_url)">
                    <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" :style="{ color: certConfig.icon_color }">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
                    </svg>
                    <span class="font-medium flex-shrink-0" :style="{ color: darkenColor(certConfig.icon_color, 20) }">{{ certConfig.header_text }}</span>
                    <span
                      class="text-xs px-2 py-0.5 rounded-full"
                      :style="{ backgroundColor: darkenColor(certConfig.bg_color, 10), color: darkenColor(certConfig.icon_color, 20) }"
                    >
                      {{ validationData.certifications?.length }}
                    </span>
                  </template>
                  <!-- Logo previews when at least one cert has a logo -->
                  <div v-else class="flex items-center gap-1.5 min-w-0">
                    <template v-for="(cert, idx) in (validationData.certifications || []).slice(0, 7)" :key="idx">
                      <img
                        v-if="cert.logo_url"
                        :src="cert.logo_url"
                        :alt="cert.name"
                        class="w-10 h-10 rounded-full object-contain bg-white border border-white/50 flex-shrink-0"
                      />
                      <span
                        v-else
                        class="w-10 h-10 rounded-full flex items-center justify-center bg-white/60 flex-shrink-0"
                      >
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" :style="{ color: certConfig.icon_color }">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                      </span>
                    </template>
                    <span
                      v-if="(validationData.certifications?.length || 0) > 7"
                      class="text-sm font-semibold flex-shrink-0"
                      :style="{ color: darkenColor(certConfig.icon_color, 20) }"
                    >
                      +{{ validationData.certifications.length - 7 }}
                    </span>
                  </div>
                </div>
                <svg
                  :class="['w-5 h-5 transition-transform', certificationsExpanded ? 'rotate-180' : '']"
                  fill="none" stroke="currentColor" viewBox="0 0 24 24"
                  :style="{ color: certConfig.icon_color }"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              <!-- Certifications List -->
              <div v-if="certificationsExpanded" class="px-3 pb-3 space-y-1.5">
                <div
                  v-for="cert in (validationData.certifications || [])"
                  :key="cert.code"
                  class="flex items-center gap-3 p-2 rounded-lg"
                  style="background: rgba(255,255,255,0.15)"
                >
                  <div v-if="cert.logo_url" class="w-10 h-10 flex-shrink-0">
                    <img :src="cert.logo_url" :alt="cert.name" class="w-full h-full object-contain" />
                  </div>
                  <div v-else class="w-10 h-10 flex-shrink-0 rounded-full flex items-center justify-center" style="background: rgba(255,255,255,0.2)">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" :style="{ color: certConfig.icon_color }">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  </div>
                  <div class="flex-1 min-w-0">
                    <p class="font-medium truncate" style="color: rgba(0,0,0,0.85)">{{ cert.name }}</p>
                    <p class="text-xs" style="color: rgba(0,0,0,0.6)">
                      {{ cert.country }} <span v-if="cert.registration_number">- {{ cert.registration_number }}</span>
                    </p>
                  </div>
                </div>
              </div>
            </div>

            <!-- Website Link Button (with URL sanitization) -->
            <a
              v-if="sectionId === 'website_link' && sectionVisibility.website_link"
              :href="sanitizedWebsiteUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="block w-full py-3 px-4 mt-4 bg-zinc-600 text-white text-center font-semibold rounded-lg hover:bg-zinc-700 transition-colors"
            >
              {{ validationData.website_caption || 'Visit Website' }}
            </a>

            <!-- Product Description (separate section) -->
            <div v-if="sectionId === 'description' && sectionVisibility.description" class="mt-4 rounded-lg p-3" :style="fieldCardStyle">
              <p class="text-xs uppercase tracking-wide mb-1" :style="labelColorStyle">Description</p>
              <p class="text-sm" :style="textColorStyle">{{ validationData.product?.description }}</p>
            </div>

            <!-- Warranty Button -->
            <button
              v-if="sectionId === 'warranty_button' && sectionVisibility.warranty_button"
              @click="goToWarranty"
              class="w-full py-3 px-4 mt-4 font-semibold rounded-lg transition-opacity hover:opacity-90 flex items-center justify-center gap-2"
              :style="{ backgroundColor: templateConfig.warranty_button.bg_color, color: templateConfig.warranty_button.text_color }"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
              </svg>
              {{ templateConfig.warranty_button.text }}
            </button>
          </template>

          <!-- Report Counterfeit Button + Company Contact (only when flagged as counterfeit) -->
          <div v-if="isCounterfeitWarning" class="mt-6 pt-4 border-t border-gray-200 space-y-4">
            <button
              @click="showReportModal = true"
              class="w-full py-2 px-4 text-sm font-medium text-red-600 border border-red-200 rounded-lg hover:bg-red-50 transition-colors flex items-center justify-center gap-2"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              Report Suspected Counterfeit
            </button>
            <CompanyContactCard lead="Believe this is an error? Contact us." />
          </div>

          <!-- Company Contact (valid landing) -->
          <div v-else class="mt-4">
            <CompanyContactCard />
          </div>

          <!-- Footer -->
          <div class="mt-4 pt-4 border-t border-gray-200 text-center">
            <p class="text-xs text-gray-500">
              Powered by {{ brandingStore.appName }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- Sticky Social Media Bar (bottom of screen) -->
    <div
      v-if="isSocialMediaSticky && hasSocialAccounts && !showReportModal && !lightboxOpen"
      class="fixed bottom-0 left-0 right-0 z-40"
    >
      <div class="bg-white/95 backdrop-blur-sm border-t border-gray-200 shadow-lg">
        <div class="max-w-md mx-auto px-4 py-3">
          <div class="flex items-center justify-center gap-3">
            <a
              v-for="account in visibleSocialAccounts"
              :key="account.id || account.platform_code"
              :href="getSocialAccountUrl(account)"
              target="_blank"
              rel="noopener noreferrer"
              class="w-10 h-10 rounded-full bg-gray-100 flex items-center justify-center hover:bg-gray-200 transition-colors"
              :title="account.platform_name || account.account_handle"
            >
              <svg v-if="getSocialAccountIcon(account)" class="w-5 h-5 text-gray-700" viewBox="0 0 24 24" fill="currentColor">
                <path :d="getSocialAccountIcon(account)" />
              </svg>
              <span v-else class="text-xs font-bold text-gray-600">
                {{ (account.platform_name || account.platform_code || '?').charAt(0) }}
              </span>
            </a>
          </div>
        </div>
      </div>
    </div>

    <!-- Counterfeit Report Modal -->
    <div v-if="showReportModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="w-full max-w-md bg-white rounded-lg shadow-xl overflow-hidden">
        <!-- Modal Header -->
        <div class="bg-red-600 px-6 py-4 text-white">
          <h2 class="text-lg font-semibold">Report Suspected Counterfeit</h2>
          <p class="text-sm text-red-100 mt-1">Help us protect consumers by reporting suspicious products</p>
        </div>

        <!-- Modal Content -->
        <div class="p-6">
          <!-- Success State -->
          <div v-if="reportSuccess" class="text-center py-6">
            <div class="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg class="w-8 h-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <h3 class="text-lg font-medium text-gray-900 mb-2">Report Submitted</h3>
            <p class="text-sm text-gray-600 mb-4">Thank you for helping us maintain product authenticity. Our team will investigate this report.</p>
            <button
              @click="closeReportModal"
              class="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
            >
              Close
            </button>
          </div>

          <!-- Form State -->
          <div v-else class="space-y-4">
            <!-- Error Message -->
            <div v-if="reportError" class="p-3 bg-red-50 text-red-700 rounded-lg text-sm">
              {{ reportError }}
            </div>

            <!-- Photo Upload -->
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">
                Photos <span class="text-gray-400 text-xs font-normal">(optional, max {{ MAX_PHOTOS }})</span>
              </label>

              <!-- Photo Preview Grid -->
              <div v-if="photoPreviews.length" class="grid grid-cols-3 gap-2 mb-2">
                <div v-for="(preview, index) in photoPreviews" :key="index" class="relative group aspect-square">
                  <img :src="preview" class="w-full h-full object-cover rounded-lg" />
                  <button
                    @click="removePhoto(index)"
                    :disabled="submittingReport"
                    class="absolute top-1 right-1 w-6 h-6 bg-black/60 text-white rounded-full flex items-center justify-center text-xs opacity-0 group-hover:opacity-100 transition-opacity hover:bg-black/80"
                  >
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
                  </button>
                </div>

                <!-- Add More Button (if under limit) -->
                <label v-if="photoPreviews.length < MAX_PHOTOS && !submittingReport" class="aspect-square border-2 border-dashed border-gray-300 rounded-lg flex items-center justify-center cursor-pointer hover:border-red-400 hover:bg-red-50 transition-colors">
                  <svg class="w-6 h-6 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
                  <input type="file" multiple accept="image/jpeg,image/png,image/webp" class="hidden" @change="addPhotos" />
                </label>
              </div>

              <!-- Empty State: Add Photos Button -->
              <label v-if="!photoPreviews.length && !submittingReport" class="flex flex-col items-center justify-center p-4 border-2 border-dashed border-gray-300 rounded-lg cursor-pointer hover:border-red-400 hover:bg-red-50 transition-colors">
                <svg class="w-8 h-8 text-gray-400 mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
                <span class="text-sm text-gray-500">Tap to add photos</span>
                <span class="text-xs text-gray-400 mt-0.5">JPEG, PNG, WebP up to 5MB each</span>
                <input type="file" multiple accept="image/jpeg,image/png,image/webp" class="hidden" @change="addPhotos" />
              </label>
            </div>

            <!-- Description -->
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">
                Description
              </label>
              <textarea
                v-model="reportForm.description"
                rows="3"
                :disabled="submittingReport"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-red-500 focus:border-red-500 disabled:opacity-50"
                placeholder="Describe why you suspect this product is counterfeit..."
              ></textarea>
            </div>

            <!-- Location -->
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">
                Purchase Location
              </label>
              <input
                v-model="reportForm.location"
                type="text"
                :disabled="submittingReport"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-red-500 focus:border-red-500 disabled:opacity-50"
                placeholder="Store name, city, or online marketplace"
              />
            </div>

            <!-- Upload Progress Bar -->
            <div v-if="submittingReport" class="space-y-2">
              <div class="flex items-center justify-between text-sm">
                <span class="text-gray-600 flex items-center gap-2">
                  <svg class="w-4 h-4 animate-spin text-red-600" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
                  {{ uploadProgress < 100 ? 'Uploading...' : 'Processing...' }}
                </span>
                <span class="text-gray-500 font-medium">{{ uploadProgress }}%</span>
              </div>
              <div class="w-full bg-gray-200 rounded-full h-2 overflow-hidden">
                <div
                  class="bg-red-600 h-full rounded-full transition-all duration-300 ease-out"
                  :style="{ width: uploadProgress + '%' }"
                ></div>
              </div>
            </div>

            <!-- Actions -->
            <div class="flex gap-3 pt-4">
              <button
                @click="closeReportModal"
                :disabled="submittingReport"
                class="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                @click="submitReport"
                :disabled="submittingReport"
                class="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {{ submittingReport ? 'Submitting...' : 'Submit Report' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Gallery Lightbox Modal -->
    <div
      v-if="lightboxOpen && hasImages && productImages[lightboxIndex]"
      class="fixed inset-0 bg-black/90 flex items-center justify-center z-50"
      @click="closeLightbox"
    >
      <button
        class="absolute top-4 right-4 text-white hover:text-gray-300 z-10"
        @click.stop="closeLightbox"
        aria-label="Close lightbox"
      >
        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>

      <!-- Previous button -->
      <button
        v-if="lightboxIndex > 0"
        class="absolute left-4 text-white hover:text-gray-300 z-10"
        @click.stop="prevImage"
        aria-label="Previous image"
      >
        <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>

      <!-- Image -->
      <div class="max-w-4xl max-h-[80vh] px-4" @click.stop>
        <img
          :src="productImages[lightboxIndex]?.image_url"
          :alt="productImages[lightboxIndex]?.caption || 'Product image'"
          class="max-w-full max-h-[75vh] object-contain mx-auto"
          @error="handleImageError"
        />
        <p v-if="productImages[lightboxIndex]?.caption" class="text-white text-center mt-3 text-sm">
          {{ productImages[lightboxIndex].caption }}
        </p>
        <p class="text-gray-400 text-center text-xs mt-1">
          {{ lightboxIndex + 1 }} / {{ productImages.length }}
        </p>
      </div>

      <!-- Next button -->
      <button
        v-if="lightboxIndex < productImages.length - 1"
        class="absolute right-4 text-white hover:text-gray-300 z-10"
        @click.stop="nextImage"
        aria-label="Next image"
      >
        <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
      </button>
    </div>
  </div>
</template>

<style>
/* Default styles for custom templates */
.custom-template-container {
  min-height: 100vh;
}

.custom-template-container .status-valid {
  background-color: #22c55e;
  color: white;
}

.custom-template-container .status-invalid {
  background-color: #ef4444;
  color: white;
}

/* Hover border color using CSS custom properties */
.social-item:hover {
  border-color: var(--hover-border-color, #3f3f46);
}
</style>
