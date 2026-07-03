<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import { useAPI } from '@/composables/useAPI'
import { useEscapeKey } from '@/composables/useEscapeKey'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { Plus, Trash2, GripVertical, Link as LinkIcon, ExternalLink } from 'lucide-vue-next'
import { SOCIAL_ICON_PATHS } from '@/lib/socialIcons'

const props = defineProps({
  productId: {
    type: String,
    required: true
  }
})

const { get, post, put, del } = useAPI()

const linkedAccounts = ref([])
const tenantAccounts = ref([])
const loading = ref(false)
const linking = ref(false)
const showAddModal = ref(false)

// Close modals on Escape key
useEscapeKey(() => { showAddModal.value = false }, showAddModal)

const selectedAccountId = ref('')
const error = ref('')

// Create Account modal state
const showCreateModal = ref(false)
useEscapeKey(() => { showCreateModal.value = false }, showCreateModal)
const platforms = ref([])
const savingAccount = ref(false)
const createForm = ref({ platform_id: '', account_handle: '', account_url: '' })
const createFormErrors = ref({})
const createError = ref('')

// Get unlinked accounts (available to add)
const availableAccounts = computed(() => {
  const linkedIds = linkedAccounts.value.map(l => l.social_account?.id)
  return tenantAccounts.value.filter(a => !linkedIds.includes(a.id) && a.is_active)
})

async function fetchLinkedAccounts() {
  loading.value = true
  error.value = ''
  try {
    const response = await get(`/tenant/products/${props.productId}/social-accounts`)
    if (response.success) {
      linkedAccounts.value = response.data?.links || []
    } else {
      error.value = response.message || 'Failed to load linked accounts'
    }
  } catch (err) {
    console.error('Failed to fetch linked accounts:', err)
    error.value = 'Failed to load linked accounts'
  } finally {
    loading.value = false
  }
}

async function fetchTenantAccounts() {
  try {
    const response = await get('/tenant/social-accounts')
    if (response.success) {
      tenantAccounts.value = response.data?.accounts || []
    }
  } catch (err) {
    console.error('Failed to fetch tenant accounts:', err)
  }
}

async function fetchPlatforms() {
  try {
    const response = await get('/tenant/social-media/platforms')
    if (response.success) {
      platforms.value = response.data || []
    }
  } catch (err) {
    console.error('Failed to fetch platforms:', err)
  }
}

function openCreateModal() {
  createForm.value = { platform_id: '', account_handle: '', account_url: '' }
  createFormErrors.value = {}
  createError.value = ''
  showCreateModal.value = true
}

function validateCreateForm() {
  createFormErrors.value = {}
  if (!createForm.value.platform_id) {
    createFormErrors.value.platform_id = 'Please select a platform'
  }
  if (!createForm.value.account_handle?.trim()) {
    createFormErrors.value.account_handle = 'Account handle is required'
  } else if (createForm.value.account_handle.length > 255) {
    createFormErrors.value.account_handle = 'Handle must be 255 characters or less'
  }
  if (createForm.value.account_url && createForm.value.account_url.length > 500) {
    createFormErrors.value.account_url = 'URL must be 500 characters or less'
  }
  return Object.keys(createFormErrors.value).length === 0
}

async function saveNewAccount() {
  if (!validateCreateForm()) return
  createError.value = ''
  savingAccount.value = true
  try {
    const response = await post('/tenant/social-accounts', createForm.value)
    if (response.success) {
      showCreateModal.value = false
      const newAccountId = response.data?.account?.id
      // Auto-link to current product
      if (newAccountId) {
        await post(`/tenant/products/${props.productId}/social-accounts`, {
          social_account_id: newAccountId
        })
      }
      await fetchTenantAccounts()
      await fetchLinkedAccounts()
    } else {
      createError.value = response.message || 'Failed to create account'
    }
  } catch (err) {
    console.error('Failed to create account:', err)
    createError.value = 'Failed to create account'
  } finally {
    savingAccount.value = false
  }
}

async function linkAccount() {
  if (!selectedAccountId.value) return

  linking.value = true
  try {
    const response = await post(`/tenant/products/${props.productId}/social-accounts`, {
      social_account_id: selectedAccountId.value
    })
    if (response.success) {
      showAddModal.value = false
      selectedAccountId.value = ''
      await fetchLinkedAccounts()
    } else {
      alert(response.message || 'Failed to link account')
    }
  } catch (err) {
    console.error('Failed to link account:', err)
    alert('Failed to link account')
  } finally {
    linking.value = false
  }
}

async function unlinkAccount(linkId) {
  if (!confirm('Remove this social account from product?')) return

  try {
    const response = await del(`/tenant/products/${props.productId}/social-accounts/${linkId}`)
    if (response.success) {
      await fetchLinkedAccounts()
    } else {
      alert(response.message || 'Failed to unlink account')
    }
  } catch (err) {
    console.error('Failed to unlink account:', err)
  }
}

async function reorderAccounts(linkIds) {
  const originalOrder = [...linkedAccounts.value]
  try {
    await put(`/tenant/products/${props.productId}/social-accounts/reorder`, {
      link_ids: linkIds
    })
  } catch (err) {
    console.error('Failed to reorder accounts:', err)
    linkedAccounts.value = originalOrder
  }
}

function onDragEnd() {
  reorderAccounts(linkedAccounts.value.map(i => i.id))
}

function getAccountUrl(account) {
  if (!account) return '#'
  if (account.account_url) return account.account_url
  const handle = account.account_handle || ''
  // If handle is already a full URL, use it directly (don't prepend base_url)
  if (handle.startsWith('http://') || handle.startsWith('https://')) {
    return handle
  }
  if (account.platform?.base_url && handle) {
    // Strip @ from handle to avoid double @@ when base_url already contains @
    return account.platform.base_url + handle.replace(/^@/, '')
  }
  return '#'
}

function getPlatformIcon(platform) {
  if (!platform?.code) return null

  // Check if icon looks like SVG path (contains space)
  if (platform.icon && platform.icon.includes(' ')) {
    return `<path d="${platform.icon}" />`
  }

  // Fallback to shared icon map
  const code = (platform.code || platform.icon || '').toUpperCase()
  const path = SOCIAL_ICON_PATHS[code]
  return path ? `<path d="${path}" />` : null
}

onMounted(() => {
  fetchLinkedAccounts()
  fetchTenantAccounts()
  fetchPlatforms()
})

// Re-fetch when product changes
watch(() => props.productId, () => {
  fetchLinkedAccounts()
})
</script>

<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100">Social Media Accounts</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          Link your social accounts to show on the product landing page.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          @click="openCreateModal"
        >
          <Plus class="w-4 h-4 mr-1" />
          Add Account
        </Button>
        <Button
          v-if="availableAccounts.length > 0"
          size="sm"
          @click="showAddModal = true"
        >
          <Plus class="w-4 h-4 mr-1" />
          Link Account
        </Button>
      </div>
    </div>

    <!-- Error -->
    <div v-if="error" class="p-3 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300 rounded-lg text-sm">
      {{ error }}
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-8">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-600"></div>
    </div>

    <!-- Linked Accounts List -->
    <VueDraggable
      v-if="!loading && linkedAccounts.length > 0"
      v-model="linkedAccounts"
      :animation="150"
      handle=".drag-handle"
      ghost-class="opacity-50"
      class="space-y-2"
      @end="onDragEnd"
    >
      <div
        v-for="link in linkedAccounts"
        :key="link.id"
        class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
      >
        <!-- Drag handle -->
        <div class="drag-handle cursor-grab active:cursor-grabbing p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
          <GripVertical class="w-4 h-4" />
        </div>

        <!-- Platform icon -->
        <div class="w-10 h-10 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center flex-shrink-0">
          <svg
            v-if="getPlatformIcon(link.social_account?.platform)"
            class="w-5 h-5 text-gray-600 dark:text-gray-300"
            viewBox="0 0 24 24"
            fill="currentColor"
            v-html="getPlatformIcon(link.social_account?.platform)"
          ></svg>
          <LinkIcon v-else class="w-5 h-5 text-gray-400" />
        </div>

        <!-- Account info -->
        <div class="flex-1 min-w-0">
          <p class="font-medium text-gray-900 dark:text-gray-100 truncate">
            {{ link.social_account?.platform?.name || 'Unknown Platform' }}
          </p>
          <p class="text-sm text-gray-500 dark:text-gray-400 truncate">
            {{ link.social_account?.account_handle || '-' }}
          </p>
        </div>

        <!-- Actions -->
        <div class="flex items-center gap-2">
          <a
            :href="getAccountUrl(link.social_account)"
            target="_blank"
            class="p-2 text-gray-400 hover:text-zinc-600 transition-colors"
            title="Open link"
          >
            <ExternalLink class="w-4 h-4" />
          </a>
          <button
            @click="unlinkAccount(link.id)"
            class="p-2 text-gray-400 hover:text-red-500 transition-colors"
            title="Unlink"
          >
            <Trash2 class="w-4 h-4" />
          </button>
        </div>
      </div>
    </VueDraggable>

    <!-- Empty state -->
    <div v-if="!loading && linkedAccounts.length === 0" class="text-center py-8 text-gray-500 border-2 border-dashed border-gray-200 dark:border-gray-700 rounded-lg">
      <LinkIcon class="w-12 h-12 mx-auto mb-2 text-gray-300" />
      <p>No social accounts linked</p>
      <p class="text-sm mt-1">
        <router-link to="/tenant/social-accounts" class="text-zinc-600 hover:underline">
          Manage your social accounts
        </router-link>
        to add them here.
      </p>
      <Button
        v-if="availableAccounts.length > 0"
        @click="showAddModal = true"
        class="mt-3"
      >
        <Plus class="w-4 h-4 mr-2" />
        Link Account
      </Button>
    </div>

    <!-- No accounts hint -->
    <div v-if="tenantAccounts.length === 0 && !loading" class="p-4 bg-zinc-50 dark:bg-zinc-900/20 rounded-lg">
      <p class="text-sm text-zinc-700 dark:text-zinc-300">
        You haven't added any social accounts yet.
        <router-link to="/tenant/social-accounts" class="underline font-medium">
          Add social accounts
        </router-link>
        first, then link them to products.
      </p>
    </div>

    <!-- Add Modal -->
    <Teleport to="body">
      <div
        v-if="showAddModal"
        class="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
        @click.self="showAddModal = false"
      >
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md p-6">
          <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Link Social Account</h3>

          <div v-if="availableAccounts.length > 0">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Select Account
            </label>
            <select
              v-model="selectedAccountId"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800"
            >
              <option value="">Select an account...</option>
              <option v-for="account in availableAccounts" :key="account.id" :value="account.id">
                {{ account.platform?.name }} - {{ account.account_handle }}
              </option>
            </select>
          </div>
          <div v-else class="text-center py-4 text-gray-500">
            <p>All accounts are already linked to this product.</p>
          </div>

          <div class="flex justify-end gap-3 mt-6">
            <Button variant="ghost" @click="showAddModal = false">Cancel</Button>
            <Button
              @click="linkAccount"
              :disabled="!selectedAccountId || linking"
            >
              {{ linking ? 'Linking...' : 'Link Account' }}
            </Button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Create Account Modal -->
    <Teleport to="body">
      <div
        v-if="showCreateModal"
        class="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
        @click.self="showCreateModal = false"
      >
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md p-6">
          <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">
            Add Social Account
          </h3>

          <div v-if="createError" class="p-3 mb-4 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300 rounded-lg text-sm">
            {{ createError }}
          </div>

          <div class="space-y-4">
            <!-- Platform select -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Platform *
              </label>
              <select
                v-model="createForm.platform_id"
                class="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800"
                :class="createFormErrors.platform_id ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'"
              >
                <option value="">Select platform...</option>
                <option v-for="platform in platforms" :key="platform.id" :value="platform.id">
                  {{ platform.name }}
                </option>
              </select>
              <p v-if="createFormErrors.platform_id" class="mt-1 text-sm text-red-500">
                {{ createFormErrors.platform_id }}
              </p>
            </div>

            <!-- Account handle -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Account Handle *
              </label>
              <Input
                v-model="createForm.account_handle"
                placeholder="e.g., @yourbrand or yourbrand_official"
                :class="{ 'border-red-500': createFormErrors.account_handle }"
              />
              <p v-if="createFormErrors.account_handle" class="mt-1 text-sm text-red-500">
                {{ createFormErrors.account_handle }}
              </p>
            </div>

            <!-- Custom URL (optional) -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Custom URL (optional)
              </label>
              <Input
                v-model="createForm.account_url"
                placeholder="https://..."
                :class="{ 'border-red-500': createFormErrors.account_url }"
              />
              <p class="mt-1 text-xs text-gray-500">
                Leave empty to use platform's default URL pattern
              </p>
              <p v-if="createFormErrors.account_url" class="mt-1 text-sm text-red-500">
                {{ createFormErrors.account_url }}
              </p>
            </div>
          </div>

          <div class="flex justify-end gap-3 mt-6">
            <Button variant="ghost" @click="showCreateModal = false">Cancel</Button>
            <Button @click="saveNewAccount" :disabled="savingAccount">
              {{ savingAccount ? 'Adding...' : 'Add Account' }}
            </Button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
