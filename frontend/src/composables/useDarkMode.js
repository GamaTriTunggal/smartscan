import { ref, watch, onUnmounted } from 'vue'

// Theme mode: 'light' | 'dark' | 'system'
const themeMode = ref('system')
const isDark = ref(false)

// System preference media query and handler (singleton)
let systemPreferenceQuery = null
let systemPreferenceHandler = null
let isInitialized = false
let watcherSetup = false

export function useDarkMode() {
  function getSystemPreference() {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  }

  function applyDarkMode() {
    if (isDark.value) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }

  function updateIsDark() {
    if (themeMode.value === 'system') {
      isDark.value = getSystemPreference()
    } else {
      isDark.value = themeMode.value === 'dark'
    }
    applyDarkMode()
  }

  function initDarkMode() {
    // Prevent double initialization (singleton pattern)
    if (isInitialized) return

    // Check localStorage for theme mode
    const stored = localStorage.getItem('themeMode')
    if (stored && ['light', 'dark', 'system'].includes(stored)) {
      themeMode.value = stored
    } else {
      // Default to system
      themeMode.value = 'system'
    }

    // Set up system preference listener (only once)
    if (typeof window !== 'undefined' && !systemPreferenceQuery) {
      systemPreferenceQuery = window.matchMedia('(prefers-color-scheme: dark)')
      systemPreferenceHandler = (e) => {
        if (themeMode.value === 'system') {
          isDark.value = e.matches
          applyDarkMode()
        }
      }
      systemPreferenceQuery.addEventListener('change', systemPreferenceHandler)
    }

    isInitialized = true
    updateIsDark()
  }

  // Cleanup function (for testing or SPA unmount scenarios)
  function cleanup() {
    if (systemPreferenceQuery && systemPreferenceHandler) {
      systemPreferenceQuery.removeEventListener('change', systemPreferenceHandler)
      systemPreferenceQuery = null
      systemPreferenceHandler = null
      isInitialized = false
    }
  }

  function setThemeMode(mode) {
    if (['light', 'dark', 'system'].includes(mode)) {
      themeMode.value = mode
      localStorage.setItem('themeMode', mode)
      updateIsDark()
    }
  }

  // Cycle through modes: system -> light -> dark -> system
  function toggleDarkMode() {
    const modes = ['system', 'light', 'dark']
    const currentIndex = modes.indexOf(themeMode.value)
    const nextIndex = (currentIndex + 1) % modes.length
    setThemeMode(modes[nextIndex])
  }

  // Legacy support - set dark mode directly
  function setDarkMode(value) {
    setThemeMode(value ? 'dark' : 'light')
  }

  // Watch for theme mode changes - only set up once at module level
  // This avoids "watch called outside of component setup" warnings
  if (!watcherSetup && typeof window !== 'undefined') {
    watch(themeMode, () => {
      updateIsDark()
    })
    watcherSetup = true
  }

  // Initialize on first use
  if (typeof window !== 'undefined') {
    initDarkMode()
  }

  return {
    isDark,
    themeMode,
    toggleDarkMode,
    setDarkMode,
    setThemeMode,
    initDarkMode,
    cleanup
  }
}
