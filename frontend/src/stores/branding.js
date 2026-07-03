import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useAPI } from '@/composables/useAPI'

// Get app name from environment variable, fallback to 'smartscan'
export const DEFAULT_APP_NAME = import.meta.env.VITE_APP_NAME || 'smartscan'

export const useBrandingStore = defineStore('branding', () => {
  const branding = ref({
    app_name: DEFAULT_APP_NAME,
    logo_url: '',
    header_gradient_start: '#18181b',
    header_gradient_end: '#FFAB2E',
    header_text_color: '#ffffff',
    button_bg_color: '#F5A623',
    button_text_color: '#ffffff'
  })

  const loading = ref(false)
  const error = ref(null)
  const loaded = ref(false)

  const appName = computed(() => branding.value.app_name || DEFAULT_APP_NAME)
  const logoUrl = computed(() => branding.value.logo_url)
  const headerGradient = computed(() =>
    `linear-gradient(135deg, ${branding.value.header_gradient_start} 0%, ${branding.value.header_gradient_end} 100%)`
  )
  const headerTextColor = computed(() => branding.value.header_text_color)
  const buttonBgColor = computed(() => branding.value.button_bg_color)
  const buttonTextColor = computed(() => branding.value.button_text_color)

  async function fetchBranding() {
    // Only fetch once unless forced
    if (loaded.value) return true

    loading.value = true
    error.value = null

    const { get } = useAPI()
    try {
      // Use public endpoint (no auth required)
      const response = await get('/public/branding')
      if (response.success && response.data) {
        branding.value = response.data
        loaded.value = true
        // Also save to localStorage for faster initial load
        localStorage.setItem('branding', JSON.stringify(response.data))
        return true
      }
      return false
    } catch (e) {
      error.value = e.message || 'Failed to fetch branding'
      // Try to load from localStorage as fallback
      const cached = localStorage.getItem('branding')
      if (cached) {
        try {
          branding.value = JSON.parse(cached)
          loaded.value = true
        } catch {}
      }
      return false
    } finally {
      loading.value = false
    }
  }

  async function updateBranding(data) {
    loading.value = true
    error.value = null

    const { put } = useAPI()
    try {
      const response = await put('/tenant/app-settings/branding', data)
      if (response.success && response.data) {
        branding.value = response.data
        localStorage.setItem('branding', JSON.stringify(response.data))
        return true
      }
      return false
    } catch (e) {
      error.value = e.response?.data?.message || e.message || 'Failed to update branding'
      return false
    } finally {
      loading.value = false
    }
  }

  function initFromStorage() {
    const cached = localStorage.getItem('branding')
    if (cached) {
      try {
        branding.value = JSON.parse(cached)
      } catch {}
    }
  }

  // Initialize from storage on load
  initFromStorage()

  return {
    branding,
    loading,
    error,
    loaded,
    appName,
    logoUrl,
    headerGradient,
    headerTextColor,
    buttonBgColor,
    buttonTextColor,
    fetchBranding,
    updateBranding,
    initFromStorage
  }
})
