import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
const routes = [
  // Public routes
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/pages/auth/LoginPage.vue'),
    meta: { guest: true },
  },

  // First-run setup wizard
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('@/pages/auth/SetupPage.vue'),
    meta: { guest: true },
  },

  // Forced password change interstitial
  {
    path: '/change-password',
    name: 'ChangePassword',
    component: () => import('@/pages/auth/ChangePasswordPage.vue'),
    meta: { requiresAuth: true },
  },

  // Tenant Dashboard
  {
    path: '/tenant',
    component: () => import('@/layouts/TenantLayout.vue'),
    meta: { requiresAuth: true, tenantOnly: true },
    children: [
      {
        path: '',
        redirect: '/tenant/dashboard',
      },
      {
        path: 'dashboard',
        name: 'TenantDashboard',
        component: () => import('@/pages/tenant/DashboardPage.vue'),
      },
      {
        path: 'products',
        redirect: '/tenant/products/dynamic',
      },
      {
        path: 'products/:id',
        name: 'TenantProductDetail',
        component: () => import('@/pages/tenant/ProductDetailPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'products/dynamic',
        name: 'TenantDynamicQR',
        component: () => import('@/pages/tenant/products/DynamicQRPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'products/:productId/batches',
        name: 'TenantProductBatches',
        component: () => import('@/pages/tenant/products/ProductBatchHistoryPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'qr-batches/:id',
        name: 'TenantQRBatchDetail',
        component: () => import('@/pages/tenant/QRBatchDetailPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'qr-batches/:batchId/codes/:codeId',
        name: 'TenantQRCodeDetail',
        component: () => import('@/pages/tenant/QRCodeDetailPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'templates',
        name: 'TenantTemplates',
        component: () => import('@/pages/tenant/TemplatesPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'templates/:id',
        name: 'TenantTemplateEditor',
        component: () => import('@/pages/tenant/TemplateEditorPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'locations',
        name: 'TenantLocations',
        component: () => import('@/pages/tenant/LocationsPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'analytics',
        name: 'TenantAnalytics',
        component: () => import('@/pages/tenant/AnalyticsPage.vue'),
        meta: { adminOnly: true },
      },
      // Warranty Activations (Intermediate+ tier)
      {
        path: 'warranties',
        name: 'TenantWarrantyList',
        component: () => import('@/pages/tenant/warranty/WarrantyListPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'counterfeit',
        name: 'TenantCounterfeit',
        component: () => import('@/pages/tenant/CounterfeitPage.vue'),
        meta: {},
      },
      {
        path: 'geofence',
        name: 'TenantGeofenceViolations',
        component: () => import('@/pages/tenant/GeofenceViolationsPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'geofence/zone-templates',
        name: 'TenantGeofenceZoneTemplates',
        component: () => import('@/pages/tenant/GeofenceZoneTemplatesPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'staff',
        name: 'TenantStaff',
        component: () => import('@/pages/tenant/StaffPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'social-accounts',
        name: 'TenantSocialAccounts',
        component: () => import('@/pages/tenant/TenantSocialAccountsPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'settings',
        name: 'TenantSettings',
        component: () => import('@/pages/tenant/SettingsPage.vue'),
        meta: { adminOnly: true },
      },
      // Admin master data & platform management
      {
        path: 'admin/certification-types',
        name: 'TenantAdminCertificationTypes',
        component: () => import('@/pages/tenant/admin/CertificationTypesPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'admin/social-platforms',
        name: 'TenantAdminSocialPlatforms',
        component: () => import('@/pages/tenant/admin/SocialMediaPlatformsPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'admin/theme-presets',
        name: 'TenantAdminThemePresets',
        component: () => import('@/pages/tenant/admin/ThemePresetsPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'admin/regions',
        name: 'TenantAdminRegions',
        component: () => import('@/pages/tenant/admin/RegionsPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'admin/appearance',
        name: 'TenantAdminAppearance',
        component: () => import('@/pages/tenant/admin/AppearancePage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'admin/audit-logs',
        name: 'TenantAdminAuditLogs',
        component: () => import('@/pages/tenant/admin/AuditLogsPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'admin/integrations',
        name: 'TenantAdminIntegrations',
        component: () => import('@/pages/tenant/admin/IntegrationsPage.vue'),
        meta: { adminOnly: true },
      },
      {
        path: 'admin/company-contact',
        name: 'TenantAdminCompanyContact',
        component: () => import('@/pages/tenant/admin/CompanyContactPage.vue'),
        meta: { adminOnly: true },
      },
      // Role-specific pages
      {
        path: 'qc-jobs',
        name: 'TenantQCJobs',
        component: () => import('@/pages/tenant/QCJobsPage.vue'),
        meta: { qcOrAdmin: true },
      },
      {
        path: 'warehouse-jobs',
        name: 'TenantWarehouseJobs',
        component: () => import('@/pages/tenant/WarehouseJobsPage.vue'),
        meta: { warehouseOrAdmin: true },
      },
      {
        path: 'account',
        name: 'TenantAccount',
        component: () => import('@/pages/tenant/AccountPage.vue'),
      },
    ],
  },



  // Public QR validation pages
  // NOTE: /s/:code is handled by backend via Vite proxy (generates signed URL)
  {
    path: '/v/:uuid',
    name: 'ValidateProduct',
    component: () => import('@/pages/public/ValidatePage.vue'),
    meta: { public: true },
  },
  {
    path: '/w/:uuid',
    name: 'WarrantyActivation',
    component: () => import('@/pages/public/WarrantyPage.vue'),
    meta: { public: true },
  },

  // Root redirect
  {
    path: '/',
    redirect: '/login',
  },

  // 404
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/pages/NotFoundPage.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// First-run setup status, checked at most once per app load (module-level
// cached promise). On failure we resolve false so /login proceeds as normal.
let setupStatusPromise = null
function checkNeedsSetup() {
  if (!setupStatusPromise) {
    const apiBase = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
    setupStatusPromise = fetch(`${apiBase}/setup/status`)
      .then((res) => res.json())
      .then((body) => body?.data?.needs_setup === true)
      .catch(() => false)
  }
  return setupStatusPromise
}

// Called by SetupPage after a successful setup so a later logout doesn't
// bounce /login back to /setup based on the stale cached status.
export function setSetupComplete() {
  setupStatusPromise = Promise.resolve(false)
}

// Navigation guard
router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore()

  // Public pages
  if (to.meta.public) {
    next()
    return
  }

  // Guest only pages (login, setup)
  if (to.meta.guest) {
    if (authStore.isAuthenticated) {
      // If authenticated but no user data, clear stale auth state
      if (!authStore.user) {
        await authStore.logout()
        next()
        return
      }
      // Redirect to dashboard
      next('/tenant/dashboard')
      return
    }
    // First-run redirects: /login → /setup when setup is needed,
    // /setup → /login once setup is done.
    if (to.path === '/login' && (await checkNeedsSetup())) {
      next('/setup')
      return
    }
    if (to.path === '/setup' && !(await checkNeedsSetup())) {
      next('/login')
      return
    }
    next()
    return
  }

  // Protected routes
  if (to.meta.requiresAuth) {
    // Check if token is expired (safety net - initFromStorage should have cleared this)
    if (authStore.isTokenExpired) {
      await authStore.logout()
      next('/login')
      return
    }

    if (!authStore.isAuthenticated) {
      next('/login')
      return
    }

    // Fetch user if not loaded
    if (!authStore.user) {
      const success = await authStore.fetchUser()
      if (!success) {
        authStore.logout()
        next('/login')
        return
      }
    }

    // Password change required - force the interstitial before anything else
    if (authStore.mustChangePassword && to.path !== '/change-password') {
      next('/change-password')
      return
    }

    // Tenant only (unknown user_type — send back to login to avoid redirect loop)
    if (to.meta.tenantOnly && !authStore.isTenant) {
      next('/login')
      return
    }

    // Admin only
    if (to.meta.adminOnly && !authStore.isAdmin) {
      next('/tenant/dashboard')
      return
    }

    // QC Staff or Admin
    if (to.meta.qcOrAdmin && !authStore.canAccessQC) {
      next('/tenant/dashboard')
      return
    }

    // Warehouse Staff or Admin
    if (to.meta.warehouseOrAdmin && !authStore.canAccessWarehouse) {
      next('/tenant/dashboard')
      return
    }

    // Redirect non-admin staff away from dashboard to their landing page
    if (to.path === '/tenant/dashboard' && authStore.isTenant && !authStore.isAdmin) {
      if (authStore.isQCStaff) {
        next('/tenant/qc-jobs')
        return
      }
      if (authStore.isWarehouseStaff) {
        next('/tenant/warehouse-jobs')
        return
      }
    }

    // Redirect Admin from /tenant/account to /tenant/settings
    if (to.path === '/tenant/account' && authStore.isAdmin) {
      next('/tenant/settings')
      return
    }
  }

  next()
})

export default router
