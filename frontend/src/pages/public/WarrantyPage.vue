<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'
import DOMPurify from 'dompurify'
import { useDateTime } from '@/composables/useDateTime'
import { useBrandingStore } from '@/stores/branding'
import { usePublicCompanyContact } from '@/composables/usePublicCompanyContact'
import PhoneInput from '@/components/PhoneInput.vue'
import Alert from '@/components/ui/Alert.vue'
import CompanyContactCard from '@/components/public/CompanyContactCard.vue'
import { getPagination } from '@/lib/pagination'

const route = useRoute()
const brandingStore = useBrandingStore()
const { formatDate } = useDateTime()
const { companyName, fetchOnce: fetchCompanyContact } = usePublicCompanyContact()
const uuid = computed(() => route.params.uuid)

// Consumer-facing company name for the consent label (falls back when unset).
const consentCompanyName = computed(() => companyName.value?.trim() || 'the brand owner')

const loading = ref(true)
const error = ref(null)
const productData = ref(null)
const templateData = ref(null)
const submitting = ref(false)
const submitted = ref(false)
const registrationResult = ref(null)
const formError = ref(null)

// Terms acceptance state
const termsAccepted = ref(false)

// Location data
const countries = ref([])
const provinces = ref([])
const cities = ref([])
const loadingCountries = ref(false)
const loadingProvinces = ref(false)
const loadingCities = ref(false)

// Form data
const formData = ref({
  customer_name: '',
  email: '',
  phone: '',
  purchase_date: '',
  purchase_store: '',
  // Customer address fields
  address: '',
  country_code: '',
  province_id: '',
  city_id: '',
})

// Custom field values (keyed by field.id)
const customFieldValues = ref({})

const apiBase = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'

const fetchProductInfo = async () => {
  try {
    // Use warranty status endpoint to get product info
    const response = await axios.get(`${apiBase}/public/warranty/${uuid.value}`)
    if (response.data.success) {
      productData.value = response.data.data
    } else {
      error.value = response.data.message || 'Failed to load product info'
    }
  } catch (err) {
    error.value = err.response?.data?.message || 'Failed to load product information'
    console.error('Product info error:', err)
  }
}

const fetchTemplate = async () => {
  try {
    const response = await axios.get(`${apiBase}/public/template/${uuid.value}`, {
      params: { type: 'warranty' }
    })
    if (response.data.success && response.data.data) {
      templateData.value = response.data.data
    }
  } catch (err) {
    console.log('No custom template, using default')
  }
}

// Location fetching functions
const fetchCountries = async () => {
  try {
    loadingCountries.value = true
    // The backend caps limit at 100, so walk every page to collect all countries.
    const all = []
    let page = 1
    let totalPage = 1
    do {
      const response = await axios.get(`${apiBase}/public/locations/countries`, {
        params: { limit: 100, page }
      })
      if (!response.data.success) break
      all.push(...(response.data.data?.countries || []))
      totalPage = getPagination(response.data.data).totalPages || 1
      page++
    } while (page <= totalPage)
    countries.value = all
  } catch (err) {
    console.error('Failed to fetch countries:', err)
  } finally {
    loadingCountries.value = false
  }
}

const fetchProvinces = async (countryCode) => {
  if (!countryCode) {
    provinces.value = []
    cities.value = []
    return
  }
  try {
    loadingProvinces.value = true
    const response = await axios.get(`${apiBase}/public/locations/provinces`, {
      params: { country_code: countryCode, limit: 100 }
    })
    if (response.data.success && response.data.data?.provinces) {
      provinces.value = response.data.data.provinces
    }
  } catch (err) {
    console.error('Failed to fetch provinces:', err)
  } finally {
    loadingProvinces.value = false
  }
}

const fetchCities = async (provinceId) => {
  if (!provinceId) {
    cities.value = []
    return
  }
  try {
    loadingCities.value = true
    // The backend caps limit at 100, so walk every page to collect all cities.
    const all = []
    let page = 1
    let totalPage = 1
    do {
      const response = await axios.get(`${apiBase}/public/locations/cities`, {
        params: { province_id: provinceId, limit: 100, page }
      })
      if (!response.data.success) break
      all.push(...(response.data.data?.cities || []))
      totalPage = getPagination(response.data.data).totalPages || 1
      page++
    } while (page <= totalPage)
    cities.value = all
  } catch (err) {
    console.error('Failed to fetch cities:', err)
  } finally {
    loadingCities.value = false
  }
}

// Watch for country change to load provinces
watch(() => formData.value.country_code, (newVal) => {
  formData.value.province_id = ''
  formData.value.city_id = ''
  cities.value = []
  if (newVal) {
    fetchProvinces(newVal)
  } else {
    provinces.value = []
  }
})

// Watch for province change to load cities
watch(() => formData.value.province_id, (newVal) => {
  formData.value.city_id = ''
  if (newVal) {
    fetchCities(newVal)
  } else {
    cities.value = []
  }
})

const submitWarranty = async () => {
  // Clear previous error
  formError.value = null

  // Validate terms acceptance
  if (!termsAccepted.value) {
    formError.value = 'Please accept the Terms & Privacy Notice to continue.'
    return
  }

  // Validate fixed required fields
  if (!formData.value.customer_name || !formData.value.email || !formData.value.phone || !formData.value.purchase_date) {
    formError.value = 'Please fill in all required fields'
    return
  }

  // Validate purchase date not in future
  if (formData.value.purchase_date > maxPurchaseDate.value) {
    formError.value = 'Purchase date cannot be in the future'
    return
  }

  // Validate customizable fields based on config
  if (isFieldRequired('store_name') && !formData.value.purchase_store) {
    formError.value = 'Store name is required'
    return
  }
  if (isFieldRequired('country') && !formData.value.country_code) {
    formError.value = 'Country is required'
    return
  }
  if (isFieldRequired('province') && !formData.value.province_id) {
    formError.value = 'Province is required'
    return
  }
  if (isFieldRequired('city') && !formData.value.city_id) {
    formError.value = 'City is required'
    return
  }
  if (isFieldRequired('address') && !formData.value.address) {
    formError.value = 'Full address is required'
    return
  }

  try {
    submitting.value = true
    // Prepare payload with proper types - only include visible fields
    const payload = {
      customer_name: formData.value.customer_name,
      email: formData.value.email,
      phone: formData.value.phone,
      purchase_date: formData.value.purchase_date,
    }

    // Include customizable fields only if visible
    if (isFieldVisible('store_name') && formData.value.purchase_store) {
      payload.purchase_store = formData.value.purchase_store
    }
    if (isFieldVisible('country') && formData.value.country_code) {
      payload.country_code = formData.value.country_code
    }
    if (isFieldVisible('province') && formData.value.province_id) {
      payload.province_id = parseInt(formData.value.province_id)
    }
    if (isFieldVisible('city') && formData.value.city_id) {
      payload.city_id = parseInt(formData.value.city_id)
    }
    if (isFieldVisible('address') && formData.value.address) {
      payload.address = formData.value.address
    }

    // Include custom fields if any
    if (customFieldsConfig.value.length > 0) {
      payload.custom_fields = customFieldValues.value
    }

    const response = await axios.post(`${apiBase}/public/warranty/${uuid.value}`, payload)
    if (response.data.success && response.data.data?.success) {
      registrationResult.value = response.data.data
      submitted.value = true
    } else {
      formError.value = response.data.data?.message || response.data.message || 'Failed to register warranty'
    }
  } catch (err) {
    formError.value = err.response?.data?.message || 'Failed to register warranty'
    console.error('Warranty registration error:', err)
  } finally {
    submitting.value = false
  }
}

// Field configuration helpers
const fieldsConfig = computed(() => {
  // Default config if not provided by API
  const defaultConfig = {
    store_name: 'optional',
    country: 'required',
    province: 'required',
    city: 'required',
    address: 'required'
  }
  return productData.value?.warranty_fields_config?.fields || defaultConfig
})

const isFieldVisible = (field) => {
  return fieldsConfig.value[field] !== 'hidden'
}

const isFieldRequired = (field) => {
  return fieldsConfig.value[field] === 'required'
}

// Custom fields config from product
const customFieldsConfig = computed(() => {
  return productData.value?.warranty_fields_config?.custom_fields || []
})

// Check if any address field is visible
const showAddressSection = computed(() => {
  return isFieldVisible('country') || isFieldVisible('province') || isFieldVisible('city') || isFieldVisible('address')
})

// Max purchase date (today)
const maxPurchaseDate = computed(() => {
  return new Date().toISOString().split('T')[0]
})

// Warranty status helpers. Uses the top-level (non-PII) warranty_expiry the public
// endpoint returns — registrant PII is no longer exposed publicly.
const isWarrantyActive = computed(() => {
  if (!productData.value?.warranty_expiry) return false
  return new Date(productData.value.warranty_expiry) > new Date()
})

const printCertificate = () => {
  window.print()
}

const replacePlaceholders = (html) => {
  if (!productData.value) return html

  const data = productData.value
  const expiryDate = data.warranty_expiry
    ? formatDate(data.warranty_expiry)
    : 'N/A'

  const replacements = {
    '{{product_name}}': data.product?.product_name || 'Unknown Product',
    '{{product_code}}': data.product?.product_code || '',
    '{{brand_name}}': data.tenant?.brand_name || data.tenant?.company_name || '',
    '{{logo_url}}': data.tenant?.logo_url || '/placeholder-logo.png',
    '{{expiry_date}}': expiryDate,
    '{{warranty_status}}': data.warranty_registered ? 'Registered' : 'Not Registered',
    '{{warranty_status_class}}': data.warranty_registered ? 'status-registered' : 'status-pending',
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
    // Sanitize HTML to prevent XSS attacks
    return DOMPurify.sanitize(html, {
      ALLOWED_TAGS: ['div', 'span', 'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'img', 'a', 'ul', 'ol', 'li', 'br', 'strong', 'em', 'b', 'i', 'u', 'table', 'tr', 'td', 'th', 'thead', 'tbody', 'section', 'article', 'header', 'footer', 'nav', 'main', 'label'],
      ALLOWED_ATTR: ['class', 'id', 'style', 'href', 'src', 'alt', 'title', 'target', 'rel', 'width', 'height', 'for'],
      FORBID_TAGS: ['script', 'iframe', 'object', 'embed', 'form', 'input', 'button'],
      FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'onfocus', 'onblur'],
      ALLOW_DATA_ATTR: false
    })
  }
  return null
})

const renderedCss = computed(() => {
  if (!templateData.value?.css_content) return ''
  // Sanitize CSS - remove any potential script injections
  let css = templateData.value.css_content
  css = css.replace(/javascript:/gi, '')
  css = css.replace(/expression\s*\(/gi, '')
  css = css.replace(/behavior\s*:/gi, '')
  css = css.replace(/-moz-binding\s*:/gi, '')
  return css
})

// Dynamically inject CSS into head (Vue 3 doesn't allow <style v-html>)
let styleElement = null

watch(renderedCss, (newCss) => {
  if (newCss) {
    if (!styleElement) {
      styleElement = document.createElement('style')
      styleElement.setAttribute('data-warranty-template', 'true')
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
})

onMounted(async () => {
  fetchCompanyContact()
  try {
    await Promise.all([fetchProductInfo(), fetchTemplate(), fetchCountries()])
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
  productData.value = null
  templateData.value = null
  submitting.value = false
  submitted.value = false
  registrationResult.value = null
  formError.value = null
  termsAccepted.value = false
  provinces.value = []
  cities.value = []
  formData.value = { customer_name: '', email: '', phone: '', purchase_date: '', purchase_store: '', address: '', country_code: '', province_id: '', city_id: '' }
  customFieldValues.value = {}

  // Re-fetch (countries already loaded)
  try {
    await Promise.all([fetchProductInfo(), fetchTemplate()])
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- Loading State -->
    <div v-if="loading" class="min-h-screen flex items-center justify-center">
      <div class="text-center">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-zinc-500 mx-auto mb-4"></div>
        <p class="text-gray-600 dark:text-gray-400">Loading warranty information...</p>
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="min-h-screen flex items-center justify-center p-4">
      <div class="w-full max-w-md bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8 text-center">
        <div class="w-16 h-16 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
          <svg class="w-8 h-8 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </div>
        <h1 class="text-xl font-bold text-gray-900 dark:text-white mb-2">Error</h1>
        <p class="text-gray-600 dark:text-gray-400">{{ error }}</p>
      </div>
    </div>

    <!-- Custom Template Rendering -->
    <div v-else-if="renderedHtml && !submitted" class="custom-template-container">
      <!-- CSS is injected dynamically via JavaScript (Vue 3 doesn't allow <style v-html>) -->
      <div v-html="renderedHtml"></div>
    </div>

    <!-- Success State -->
    <div v-else-if="submitted" class="min-h-screen flex items-center justify-center p-4">
      <div class="w-full max-w-md bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8 text-center">
        <div class="w-16 h-16 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
          <svg class="w-8 h-8 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
        </div>
        <h1 class="text-xl font-bold text-gray-900 dark:text-white mb-2">Warranty Registered!</h1>
        <p class="text-gray-600 dark:text-gray-400 mb-4">Your warranty has been successfully registered.</p>

        <!-- Personalized confirmation built entirely from the buyer's own submitted
             data (local to this session) — safe to show since it is never fetched
             from, nor exposed by, the public API. -->
        <div class="text-left border border-gray-200 dark:border-gray-700 rounded-lg p-4 mb-4 space-y-2 text-sm">
          <div class="flex justify-between gap-3">
            <span class="text-gray-500 dark:text-gray-400">Customer</span>
            <span class="font-medium text-gray-900 dark:text-white">{{ formData.customer_name }}</span>
          </div>
          <div class="flex justify-between gap-3">
            <span class="text-gray-500 dark:text-gray-400">Product</span>
            <span class="font-medium text-gray-900 dark:text-white">{{ productData?.product?.product_name || '-' }}</span>
          </div>
          <div v-if="formData.purchase_store" class="flex justify-between gap-3">
            <span class="text-gray-500 dark:text-gray-400">Purchase Store</span>
            <span class="font-medium text-gray-900 dark:text-white">{{ formData.purchase_store }}</span>
          </div>
          <div v-if="formData.purchase_date" class="flex justify-between gap-3">
            <span class="text-gray-500 dark:text-gray-400">Purchase Date</span>
            <span class="font-medium text-gray-900 dark:text-white">{{ formatDate(formData.purchase_date) }}</span>
          </div>
          <div v-if="registrationResult?.warranty_expiry_date" class="flex justify-between gap-3">
            <span class="text-gray-500 dark:text-gray-400">Valid Until</span>
            <span class="font-medium text-green-600 dark:text-green-400">{{ formatDate(registrationResult.warranty_expiry_date) }}</span>
          </div>
        </div>

        <p class="text-sm text-gray-500 dark:text-gray-400 mb-6">
          A confirmation email has been sent to {{ formData.email }}
        </p>
      </div>
    </div>

    <!-- Default Template -->
    <div v-else-if="productData" class="min-h-screen flex items-center justify-center p-4">
      <div class="w-full max-w-md bg-white dark:bg-gray-800 rounded-lg shadow-lg overflow-hidden">
        <!-- Header -->
        <div class="bg-zinc-600 dark:bg-zinc-700 p-6 text-center text-white">
          <div class="w-16 h-16 bg-white/20 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
          </div>
          <h1 class="text-2xl font-bold mb-2">Warranty Registration</h1>
          <p class="text-white/90">Register your product warranty</p>
        </div>

        <!-- Product Info -->
        <div class="p-6">
          <!-- Brand Logo -->
          <div v-if="productData.tenant?.logo_url" class="flex justify-center mb-4">
            <img
              :src="productData.tenant.logo_url"
              :alt="productData.tenant.brand_name || productData.tenant.company_name"
              class="h-12 object-contain"
            />
          </div>

          <!-- Product Details -->
          <div class="text-center mb-6">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ productData.product?.product_name || 'Unknown Product' }}
            </h2>
            <p v-if="productData.product?.product_code" class="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Code: {{ productData.product.product_code }}
            </p>
            <p v-if="productData.warranty_months" class="text-sm text-zinc-600 dark:text-zinc-400 mt-2">
              {{ productData.warranty_months }} months warranty
            </p>
          </div>

          <!-- Already Registered — Certificate View -->
          <div v-if="productData.warranty_registered" id="warranty-certificate">
            <!-- Certificate Card -->
            <div class="border-2 border-zinc-200 dark:border-zinc-800 rounded-xl overflow-hidden">
              <!-- Certificate Header -->
              <div class="bg-zinc-50 dark:bg-zinc-900/30 px-5 py-4 text-center border-b border-zinc-200 dark:border-zinc-800">
                <svg class="w-8 h-8 text-zinc-600 dark:text-zinc-400 mx-auto mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
                <h3 class="text-lg font-bold text-zinc-800 dark:text-zinc-300 uppercase tracking-wide">Warranty Certificate</h3>
              </div>

              <!-- Certificate Body -->
              <!-- Public status only: registrant PII (name, store, purchase date) is
                   intentionally not shown here, since anyone who scans this product's
                   QR code reaches this page. The buyer sees their own details on the
                   confirmation screen right after registering. -->
              <div class="px-5 py-4 space-y-3">
                <div class="grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <p class="text-xs text-gray-500 dark:text-gray-400 uppercase">Warranty Period</p>
                    <p class="font-medium text-gray-900 dark:text-white">{{ productData.warranty_months }} months</p>
                  </div>
                  <div>
                    <p class="text-xs text-gray-500 dark:text-gray-400 uppercase">Expiry Date</p>
                    <p :class="['font-medium', isWarrantyActive ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400']">
                      {{ productData.warranty_expiry ? formatDate(productData.warranty_expiry) : '-' }}
                    </p>
                  </div>
                </div>

                <!-- Status Badge -->
                <div class="text-center pt-2">
                  <span
                    :class="[
                      'inline-flex items-center gap-1.5 px-3 py-1 text-sm font-semibold rounded-full',
                      isWarrantyActive
                        ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                        : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
                    ]"
                  >
                    <span :class="['w-2 h-2 rounded-full', isWarrantyActive ? 'bg-green-500' : 'bg-red-500']"></span>
                    {{ isWarrantyActive ? 'Active' : 'Expired' }}
                  </span>
                </div>
              </div>
            </div>

            <!-- Action Buttons (hidden in print) -->
            <div class="mt-4 space-y-3 no-print">
              <!-- Print Button -->
              <button
                @click="printCertificate"
                class="w-full py-2.5 px-4 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 font-medium rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors flex items-center justify-center gap-2"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
                </svg>
                Print Certificate
              </button>
            </div>
          </div>

          <!-- Registration Form -->
          <form v-else @submit.prevent="submitWarranty" class="space-y-4">
            <!-- Form Error Alert -->
            <Alert v-if="formError" type="error">
              {{ formError }}
            </Alert>

            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Full Name *</label>
              <input
                v-model="formData.customer_name"
                type="text"
                required
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
                placeholder="Enter your full name"
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Email *</label>
              <input
                v-model="formData.email"
                type="email"
                required
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
                placeholder="Enter your email"
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Phone *</label>
              <PhoneInput
                v-model="formData.phone"
                required
                placeholder="Enter your phone number"
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Purchase Date *</label>
              <input
                v-model="formData.purchase_date"
                type="date"
                required
                :max="maxPurchaseDate"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
              />
            </div>

            <div v-if="isFieldVisible('store_name')">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Store Name {{ isFieldRequired('store_name') ? '*' : '' }}
              </label>
              <input
                v-model="formData.purchase_store"
                type="text"
                :required="isFieldRequired('store_name')"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
                placeholder="Where did you buy this product?"
              />
            </div>

            <!-- Customer Address Section (shown if any address field is visible) -->
            <template v-if="showAddressSection">
              <div class="pt-4 border-t border-gray-200 dark:border-gray-700">
                <h3 class="text-sm font-semibold text-gray-800 dark:text-gray-200 mb-3">Your Address</h3>
                <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">For warranty service and delivery</p>
              </div>

              <div v-if="isFieldVisible('country')">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Country {{ isFieldRequired('country') ? '*' : '' }}
                </label>
                <select
                  v-model="formData.country_code"
                  :required="isFieldRequired('country')"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
                >
                  <option value="">Select country</option>
                  <option v-for="country in countries" :key="country.code" :value="country.code">
                    {{ country.name }}
                  </option>
                </select>
              </div>

              <div v-if="isFieldVisible('province')">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Province {{ isFieldRequired('province') ? '*' : '' }}
                </label>
                <select
                  v-model="formData.province_id"
                  :required="isFieldRequired('province')"
                  :disabled="!formData.country_code || loadingProvinces"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500 disabled:bg-gray-100 dark:disabled:bg-gray-600 disabled:cursor-not-allowed"
                >
                  <option value="">{{ loadingProvinces ? 'Loading...' : 'Select province' }}</option>
                  <option v-for="province in provinces" :key="province.id" :value="province.id">
                    {{ province.name }}
                  </option>
                </select>
              </div>

              <div v-if="isFieldVisible('city')">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  City {{ isFieldRequired('city') ? '*' : '' }}
                </label>
                <select
                  v-model="formData.city_id"
                  :required="isFieldRequired('city')"
                  :disabled="!formData.province_id || loadingCities"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500 disabled:bg-gray-100 dark:disabled:bg-gray-600 disabled:cursor-not-allowed"
                >
                  <option value="">{{ loadingCities ? 'Loading...' : 'Select city' }}</option>
                  <option v-for="city in cities" :key="city.id" :value="city.id">
                    {{ city.name }}
                  </option>
                </select>
              </div>

              <div v-if="isFieldVisible('address')">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Full Address {{ isFieldRequired('address') ? '*' : '' }}
                </label>
                <textarea
                  v-model="formData.address"
                  :required="isFieldRequired('address')"
                  rows="2"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
                  placeholder="Street name, building, house number, etc."
                ></textarea>
              </div>
            </template>

            <!-- Custom Fields Section -->
            <template v-if="customFieldsConfig.length > 0">
              <div class="pt-4 border-t border-gray-200 dark:border-gray-700">
                <h3 class="text-sm font-semibold text-gray-800 dark:text-gray-200 mb-3">Additional Information</h3>
              </div>

              <div v-for="field in customFieldsConfig" :key="field.id">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {{ field.label }} {{ field.required ? '*' : '' }}
                </label>

                <!-- Text input -->
                <input
                  v-if="field.type === 'text' || field.type === 'email' || field.type === 'phone'"
                  v-model="customFieldValues[field.id]"
                  :type="field.type === 'email' ? 'email' : field.type === 'phone' ? 'tel' : 'text'"
                  :required="field.required"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
                  :placeholder="'Enter ' + field.label.toLowerCase()"
                >

                <!-- Textarea -->
                <textarea
                  v-else-if="field.type === 'textarea'"
                  v-model="customFieldValues[field.id]"
                  :required="field.required"
                  rows="2"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
                  :placeholder="'Enter ' + field.label.toLowerCase()"
                ></textarea>

                <!-- Number input -->
                <input
                  v-else-if="field.type === 'number'"
                  v-model="customFieldValues[field.id]"
                  type="number"
                  :required="field.required"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
                  :placeholder="'Enter ' + field.label.toLowerCase()"
                >

                <!-- Date input -->
                <input
                  v-else-if="field.type === 'date'"
                  v-model="customFieldValues[field.id]"
                  type="date"
                  :required="field.required"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
                >

                <!-- Select dropdown -->
                <select
                  v-else-if="field.type === 'select'"
                  v-model="customFieldValues[field.id]"
                  :required="field.required"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
                >
                  <option value="">Select {{ field.label.toLowerCase() }}</option>
                  <option v-for="opt in field.options" :key="opt" :value="opt">
                    {{ opt }}
                  </option>
                </select>
              </div>
            </template>

            <!-- Terms & Privacy Notice Checkbox -->
            <div class="flex items-start gap-2 mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
              <input
                type="checkbox"
                v-model="termsAccepted"
                id="terms-consent"
                class="mt-1 w-4 h-4 text-zinc-600 border-gray-300 dark:border-gray-600 rounded focus:ring-zinc-500"
              />
              <label for="terms-consent" class="text-sm text-gray-600 dark:text-gray-400">
                I consent to this information being stored by {{ consentCompanyName }} for warranty service.
              </label>
            </div>

            <button
              type="submit"
              :disabled="submitting || !termsAccepted"
              class="w-full py-3 px-4 bg-zinc-600 text-white font-semibold rounded-lg hover:bg-zinc-700 focus:ring-4 focus:ring-zinc-300 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {{ submitting ? 'Registering...' : 'Register Warranty' }}
            </button>
          </form>

          <!-- Footer -->
          <div class="mt-6 pt-4 border-t border-gray-200 dark:border-gray-700 text-center">
            <p class="text-xs text-gray-500 dark:text-gray-400">
              Powered by {{ brandingStore.appName }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- Company Contact (shown at the bottom in all states) -->
    <div v-if="!loading" class="max-w-md mx-auto px-4 pb-8">
      <CompanyContactCard />
    </div>
  </div>
</template>

<style>
.custom-template-container {
  min-height: 100vh;
}

.custom-template-container .status-registered {
  background-color: #22c55e;
  color: white;
}

.custom-template-container .status-pending {
  background-color: #f59e0b;
  color: white;
}

/* Print styles for warranty certificate */
@media print {
  body {
    background: white !important;
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
  }
  .min-h-screen {
    min-height: auto !important;
  }
  .no-print,
  nav,
  footer,
  .bg-zinc-600,
  form,
  button {
    display: none !important;
  }
  #warranty-certificate {
    border: none;
    box-shadow: none;
  }
}
</style>
