<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useBrandingStore } from '@/stores/branding'
import { useQRGenerationStore } from '@/stores/qrGeneration'
import { useDarkMode } from '@/composables/useDarkMode'
import Button from '@/components/ui/Button.vue'
import ThemeSwitcher from '@/components/ui/ThemeSwitcher.vue'
import NotificationBell from '@/components/NotificationBell.vue'
import TutorialPanel from '@/components/TutorialPanel.vue'
import { useTour } from '@/composables/useTour.js'
import { allTours } from '@/lib/tours/index.js'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const brandingStore = useBrandingStore()
const qrGenerationStore = useQRGenerationStore()
useDarkMode() // Initialize dark mode

// Tour system
const showTutorialPanel = ref(false)
const tour = useTour()
allTours.forEach(t => tour.registerTour(t))

function onStartTour(tourId) {
  showTutorialPanel.value = false
  tour.startTour(tourId)
}

// Note: Inactivity timeout is now handled by the API interceptor in useAPI.js
// Backend returns 401 INACTIVITY_TIMEOUT which triggers redirect to login

// Fetch branding on mount
onMounted(() => {
  brandingStore.fetchBranding()
  // Check for active QR generations (e.g., from a previous session) and start polling if any
  qrGenerationStore.checkAndStartPolling()
})

onUnmounted(() => {
  // Stop polling when leaving the tenant layout (e.g., logout)
  qrGenerationStore.stopPolling()
})

const sidebarOpen = ref(true)

// Icons
const icons = {
  dashboard: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6',
  products: 'M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4',
  qrBatches: 'M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z',
  templates: 'M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z',
  locations: 'M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z M15 11a3 3 0 11-6 0 3 3 0 016 0z',
  warranty: 'M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z',
  counterfeit: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z',
  staff: 'M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z',
  settings: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z',
  qcJobs: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z',
  warehouseJobs: 'M8 14v3m4-3v3m4-3v3M3 21h18M3 10h18M3 7l9-4 9 4M4 10h16v11H4V10z',
  account: 'M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z',
  socialAccounts: 'M18 8h1a4 4 0 010 8h-1M2 8h16v9a4 4 0 01-4 4H6a4 4 0 01-4-4V8z M6 1v3M10 1v3M14 1v3',
  printer: 'M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4H7v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z',
  geofence: 'M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z M15 11a3 3 0 11-6 0 3 3 0 016 0z',
  certifications: 'M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z',
  socialPlatforms: 'M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z',
  themePresets: 'M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01',
  regions: 'M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z',
  appearance: 'M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z',
  auditLogs: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z',
  integrations: 'M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1',
  companyContact: 'M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z',
}

const menuItems = computed(() => {
  const items = []

  // Dashboard - standalone, visible to all
  items.push({ type: 'item', name: 'Dashboard', path: '/tenant/dashboard', icon: icons.dashboard })

  // Role-based menus
  if (authStore.isAdmin) {
    // ── PRODUCTS ──
    items.push(
      { type: 'header', name: 'PRODUCTS' },
      { type: 'item', name: 'Products', path: '/tenant/products/dynamic', icon: icons.qrBatches, tourId: 'sidebar-dynamic-qr' },
    )

    // ── MONITORING ──
    items.push(
      { type: 'header', name: 'MONITORING' },
      { type: 'item', name: 'Warranties', path: '/tenant/warranties', icon: icons.warranty },
      { type: 'item', name: 'Counterfeit', path: '/tenant/counterfeit', icon: icons.counterfeit },
      { type: 'item', name: 'Geofence', path: '/tenant/geofence', icon: icons.geofence },
      { type: 'item', name: 'Zone Templates', path: '/tenant/geofence/zone-templates', icon: icons.geofence },
    )

    // ── TEMPLATES ──
    items.push(
      { type: 'header', name: 'TEMPLATES' },
      { type: 'item', name: 'Landing', path: '/tenant/templates?type=validation', icon: icons.templates, tourId: 'sidebar-landing' },
      { type: 'item', name: 'Warranty', path: '/tenant/templates?type=warranty', icon: icons.templates },
    )

    // ── ADMIN ──
    items.push(
      { type: 'header', name: 'ADMIN' },
      { type: 'item', name: 'Staff', path: '/tenant/staff', icon: icons.staff },
      { type: 'item', name: 'Social Accounts', path: '/tenant/social-accounts', icon: icons.socialAccounts },
      { type: 'item', name: 'Locations', path: '/tenant/locations', icon: icons.locations },
      { type: 'item', name: 'Certification Types', path: '/tenant/admin/certification-types', icon: icons.certifications },
      { type: 'item', name: 'Social Platforms', path: '/tenant/admin/social-platforms', icon: icons.socialPlatforms },
      { type: 'item', name: 'Theme Presets', path: '/tenant/admin/theme-presets', icon: icons.themePresets },
      { type: 'item', name: 'Regions', path: '/tenant/admin/regions', icon: icons.regions },
      { type: 'item', name: 'Appearance', path: '/tenant/admin/appearance', icon: icons.appearance },
      { type: 'item', name: 'Audit Logs', path: '/tenant/admin/audit-logs', icon: icons.auditLogs },
      { type: 'item', name: 'Integrations', path: '/tenant/admin/integrations', icon: icons.integrations },
      { type: 'item', name: 'Company Contact', path: '/tenant/admin/company-contact', icon: icons.companyContact },
      { type: 'item', name: 'Settings', path: '/tenant/settings', icon: icons.settings },
    )
  } else if (authStore.isQCStaff) {
    // QC Staff sees QC Jobs
    items.push({ type: 'item', name: 'QC Jobs', path: '/tenant/qc-jobs', icon: icons.qcJobs })
  } else if (authStore.isWarehouseStaff) {
    // Warehouse Staff sees Warehouse Jobs
    items.push({ type: 'item', name: 'Warehouse Jobs', path: '/tenant/warehouse-jobs', icon: icons.warehouseJobs })
  }

  // Account - visible to non-admin roles only (Admin uses Settings page)
  if (!authStore.isAdmin) {
    items.push({ type: 'item', name: 'Account', path: '/tenant/account', icon: icons.account })
  }

  return items
})

function isActive(path) {
  // Handle paths with query params
  if (path.includes('?')) {
    const [basePath, queryString] = path.split('?')
    if (route.path !== basePath) return false

    // Parse query params from menu path
    const params = new URLSearchParams(queryString)
    for (const [key, value] of params) {
      if (route.query[key] !== value) return false
    }
    return true
  }
  return route.path === path
}

async function handleLogout() {
  await authStore.logout()
  router.push('/login')
}
</script>

<template>
  <div class="min-h-screen bg-gray-100 dark:bg-gray-900">
    <!-- Sidebar -->
    <aside
      :class="[
        'fixed inset-y-0 left-0 z-50 w-64 bg-white dark:bg-gray-800 shadow-lg transform transition-transform duration-300 ease-in-out',
        sidebarOpen ? 'translate-x-0' : '-translate-x-full'
      ]"
    >
      <div class="flex items-center justify-center h-16 border-b border-gray-200 dark:border-gray-700 bg-gradient-to-r from-zinc-600 to-[#27272a] dark:from-zinc-700 dark:to-zinc-600">
        <h1 class="text-xl font-bold text-white">{{ brandingStore.appName }}</h1>
      </div>

      <div class="px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-zinc-50 dark:bg-zinc-900/20">
        <p class="text-sm font-medium text-zinc-800 dark:text-zinc-300">{{ authStore.user?.tenant_name }}</p>
        <p class="text-xs text-zinc-600 dark:text-zinc-400">{{ authStore.user?.role }}</p>
      </div>

      <nav class="mt-4 overflow-y-auto" style="height: calc(100vh - 136px)">
        <template v-for="item in menuItems" :key="item.name + (item.path || '')">
          <!-- Section Header -->
          <div
            v-if="item.type === 'header'"
            class="px-6 py-2 mt-4 first:mt-0 text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider"
          >
            {{ item.name }}
          </div>
          <!-- Menu Item -->
          <router-link
            v-else
            :to="item.path"
            :data-tour="item.tourId || undefined"
            :class="[
              'flex items-center px-6 py-2.5 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-all duration-200',
              isActive(item.path) && 'bg-zinc-50 dark:bg-zinc-900/20 text-zinc-700 dark:text-zinc-400 border-r-4 border-zinc-500'
            ]"
          >
            <svg class="w-5 h-5 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="item.icon" />
            </svg>
            {{ item.name }}
            <span
              v-if="item.badge"
              class="ml-auto text-[10px] font-semibold px-1.5 py-0.5 rounded-full bg-zinc-100 text-zinc-700 dark:bg-zinc-900/30 dark:text-zinc-400"
            >{{ item.badge }}</span>
          </router-link>
        </template>

        <!-- Tutorials Button -->
        <div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
          <button
            @click="showTutorialPanel = true"
            class="flex items-center w-full px-6 py-2.5 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-all duration-200"
          >
            <svg class="w-5 h-5 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
            </svg>
            Tutorials
          </button>
        </div>
      </nav>
    </aside>

    <!-- Main content -->
    <div :class="['min-h-screen transition-all duration-300', sidebarOpen ? 'ml-64' : 'ml-0']">
      <!-- Header -->
      <header class="bg-white dark:bg-gray-800 shadow-sm h-16 flex items-center justify-between px-6">
        <button
          @click="sidebarOpen = !sidebarOpen"
          class="text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 focus:outline-none"
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>

        <div class="flex items-center space-x-4">
          <!-- Notifications (admin-only: the feed carries company-wide security alerts) -->
          <NotificationBell v-if="authStore.isAdmin" />

          <!-- Theme Switcher -->
          <ThemeSwitcher />

          <span class="text-sm text-gray-700 dark:text-gray-300">
            {{ authStore.user?.full_name }}
          </span>
          <Button variant="outline" size="sm" @click="handleLogout">
            Logout
          </Button>
        </div>
      </header>

      <!-- Page content -->
      <main class="p-6">
        <router-view />
      </main>
    </div>

    <!-- Tutorial Panel -->
    <TutorialPanel
      :show="showTutorialPanel"
      @close="showTutorialPanel = false"
      @start-tour="onStartTour"
    />

    <!-- Tour Indicator (shown when tour is active) -->
    <div
      v-if="tour.isActive.value"
      class="fixed bottom-4 left-1/2 -translate-x-1/2 z-[9999] bg-white dark:bg-gray-800 border border-zinc-200 dark:border-zinc-800 rounded-full shadow-lg px-4 py-2 flex items-center gap-3"
    >
      <span class="text-sm text-gray-700 dark:text-gray-300">
        Tour in progress
      </span>
      <button
        @click="tour.cancelTour()"
        class="text-xs text-red-500 hover:text-red-700 dark:text-red-400 font-medium"
      >
        Cancel
      </button>
    </div>
  </div>
</template>
