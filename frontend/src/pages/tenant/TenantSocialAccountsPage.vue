<script setup>
import { ref, onMounted, computed } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useEscapeKey } from '@/composables/useEscapeKey'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { Plus, Pencil, Trash2, Link as LinkIcon, ExternalLink, Package } from 'lucide-vue-next'

const { get, post, put, del } = useAPI()

const accounts = ref([])
const platforms = ref([])
const loading = ref(true)
const error = ref('')

// Modal state
const showModal = ref(false)
const modalMode = ref('create') // 'create' or 'edit'
const saving = ref(false)
const editingAccount = ref(null)

const form = ref({
  platform_id: '',
  account_handle: '',
  account_url: ''
})

const formErrors = ref({})
const modalWarning = ref('')
const modalError = ref('')

// Close modal on Escape key
useEscapeKey(() => { showModal.value = false }, showModal)

async function fetchAccounts() {
  loading.value = true
  error.value = ''
  try {
    const response = await get('/tenant/social-accounts')
    if (response.success) {
      accounts.value = response.data?.accounts || []
    } else {
      error.value = response.message || 'Failed to load social accounts'
    }
  } catch (err) {
    console.error('Failed to fetch social accounts:', err)
    error.value = 'Failed to load social accounts'
  } finally {
    loading.value = false
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
  modalMode.value = 'create'
  editingAccount.value = null
  form.value = {
    platform_id: '',
    account_handle: '',
    account_url: ''
  }
  formErrors.value = {}
  modalWarning.value = ''
  modalError.value = ''
  showModal.value = true
}

function openEditModal(account) {
  modalMode.value = 'edit'
  editingAccount.value = account
  form.value = {
    platform_id: account.platform_id,
    account_handle: account.account_handle,
    account_url: account.account_url || ''
  }
  formErrors.value = {}
  modalWarning.value = ''
  modalError.value = ''
  showModal.value = true
}

function validateForm() {
  formErrors.value = {}

  if (!form.value.platform_id) {
    formErrors.value.platform_id = 'Please select a platform'
  }
  if (!form.value.account_handle?.trim()) {
    formErrors.value.account_handle = 'Account handle is required'
  } else if (form.value.account_handle.length > 255) {
    formErrors.value.account_handle = 'Handle must be 255 characters or less'
  }
  if (form.value.account_url && form.value.account_url.length > 500) {
    formErrors.value.account_url = 'URL must be 500 characters or less'
  }

  return Object.keys(formErrors.value).length === 0
}

async function saveAccount() {
  if (!validateForm()) return

  modalWarning.value = ''
  modalError.value = ''
  saving.value = true
  try {
    let response
    if (modalMode.value === 'create') {
      response = await post('/tenant/social-accounts', form.value)
    } else {
      response = await put(`/tenant/social-accounts/${editingAccount.value.id}`, {
        account_handle: form.value.account_handle,
        account_url: form.value.account_url
      })
    }

    if (response.success) {
      // Detect find-or-create: backend returns "already exists" for duplicates
      if (response.message && response.message.toLowerCase().includes('already exists')) {
        modalWarning.value = 'This account already exists.'
        return
      }
      showModal.value = false
      await fetchAccounts()
    } else {
      modalError.value = response.message || 'Failed to save account'
    }
  } catch (err) {
    console.error('Failed to save account:', err)
    modalError.value = 'Failed to save account'
  } finally {
    saving.value = false
  }
}

async function deleteAccount(account) {
  if (account.product_count > 0) {
    alert(`This account is linked to ${account.product_count} product(s). Unlink from all products first.`)
    return
  }

  if (!confirm(`Delete ${account.platform?.name} account "${account.account_handle}"?`)) return

  try {
    const response = await del(`/tenant/social-accounts/${account.id}`)
    if (response.success) {
      await fetchAccounts()
    } else {
      alert(response.message || 'Failed to delete account')
    }
  } catch (err) {
    console.error('Failed to delete account:', err)
    alert('Failed to delete account')
  }
}

async function toggleActive(account) {
  try {
    const response = await put(`/tenant/social-accounts/${account.id}`, {
      is_active: !account.is_active
    })
    if (response.success) {
      await fetchAccounts()
    }
  } catch (err) {
    console.error('Failed to toggle account:', err)
  }
}

function getAccountUrl(account) {
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
  return platform?.icon || null
}

onMounted(() => {
  fetchAccounts()
  fetchPlatforms()
})
</script>

<template>
  <div class="p-6 max-w-4xl mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">Social Media Accounts</h1>
        <p class="text-gray-500 dark:text-gray-400 mt-1">
          Manage your social media accounts. Link them to products to show on landing pages.
        </p>
      </div>
      <Button @click="openCreateModal">
        <Plus class="w-4 h-4 mr-2" />
        Add Account
      </Button>
    </div>

    <!-- Error -->
    <div v-if="error" class="mb-6 p-4 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300 rounded-lg">
      {{ error }}
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="animate-spin rounded-full h-10 w-10 border-b-2 border-zinc-600"></div>
    </div>

    <!-- Accounts Grid -->
    <div v-else-if="accounts.length > 0" class="grid gap-4 sm:grid-cols-2">
      <Card v-for="account in accounts" :key="account.id" class="p-4">
        <div class="flex items-start gap-4">
          <!-- Platform icon -->
          <div class="w-12 h-12 rounded-full bg-gray-100 dark:bg-gray-700 flex items-center justify-center flex-shrink-0">
            <svg
              v-if="getPlatformIcon(account.platform)"
              class="w-6 h-6 text-gray-600 dark:text-gray-300"
              viewBox="0 0 24 24"
              fill="currentColor"
              v-html="getPlatformIcon(account.platform)"
            ></svg>
            <LinkIcon v-else class="w-6 h-6 text-gray-400" />
          </div>

          <!-- Account info -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2">
              <h3 class="font-medium text-gray-900 dark:text-gray-100 truncate">
                {{ account.platform?.name || 'Unknown' }}
              </h3>
              <span
                v-if="!account.is_active"
                class="px-2 py-0.5 text-xs rounded-full bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-300"
              >
                Inactive
              </span>
            </div>
            <p class="text-sm text-gray-600 dark:text-gray-400 truncate">
              {{ account.account_handle }}
            </p>
            <div class="flex items-center gap-3 mt-2 text-sm text-gray-500">
              <span class="flex items-center gap-1">
                <Package class="w-3.5 h-3.5" />
                {{ account.product_count || 0 }} products
              </span>
              <a
                :href="getAccountUrl(account)"
                target="_blank"
                class="flex items-center gap-1 text-zinc-600 hover:underline"
              >
                <ExternalLink class="w-3.5 h-3.5" />
                Open
              </a>
            </div>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-1">
            <button
              @click="openEditModal(account)"
              class="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
              title="Edit"
            >
              <Pencil class="w-4 h-4" />
            </button>
            <button
              @click="deleteAccount(account)"
              class="p-2 text-gray-400 hover:text-red-500 transition-colors"
              :class="{ 'opacity-50 cursor-not-allowed': account.product_count > 0 }"
              :title="account.product_count > 0 ? 'Unlink from all products first' : 'Delete'"
            >
              <Trash2 class="w-4 h-4" />
            </button>
          </div>
        </div>
      </Card>
    </div>

    <!-- Empty state -->
    <Card v-else class="p-12 text-center">
      <LinkIcon class="w-16 h-16 mx-auto mb-4 text-gray-300" />
      <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">No Social Accounts</h3>
      <p class="text-gray-500 dark:text-gray-400 mb-4">
        Add your social media accounts to display them on product landing pages.
      </p>
      <Button @click="openCreateModal">
        <Plus class="w-4 h-4 mr-2" />
        Add Your First Account
      </Button>
    </Card>

    <!-- Create/Edit Modal -->
    <Teleport to="body">
      <div
        v-if="showModal"
        class="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
        @click.self="showModal = false"
      >
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md p-6">
          <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">
            {{ modalMode === 'create' ? 'Add Social Account' : 'Edit Social Account' }}
          </h3>

          <div v-if="modalWarning" class="p-3 mb-4 bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-300 rounded-lg text-sm">
            {{ modalWarning }}
          </div>

          <div v-if="modalError" class="p-3 mb-4 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300 rounded-lg text-sm">
            {{ modalError }}
          </div>

          <div class="space-y-4">
            <!-- Platform select (only for create) -->
            <div v-if="modalMode === 'create'">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Platform *
              </label>
              <select
                v-model="form.platform_id"
                class="w-full px-3 py-2 border rounded-md bg-white dark:bg-gray-800"
                :class="formErrors.platform_id ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'"
              >
                <option value="">Select platform...</option>
                <option v-for="platform in platforms" :key="platform.id" :value="platform.id">
                  {{ platform.name }}
                </option>
              </select>
              <p v-if="formErrors.platform_id" class="mt-1 text-sm text-red-500">
                {{ formErrors.platform_id }}
              </p>
            </div>

            <!-- Platform display (for edit) -->
            <div v-else>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Platform
              </label>
              <p class="text-gray-900 dark:text-gray-100">
                {{ editingAccount?.platform?.name }}
              </p>
            </div>

            <!-- Account handle -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Account Handle *
              </label>
              <Input
                v-model="form.account_handle"
                placeholder="e.g., @yourbrand or yourbrand_official"
                :class="{ 'border-red-500': formErrors.account_handle }"
              />
              <p v-if="formErrors.account_handle" class="mt-1 text-sm text-red-500">
                {{ formErrors.account_handle }}
              </p>
            </div>

            <!-- Custom URL (optional) -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Custom URL (optional)
              </label>
              <Input
                v-model="form.account_url"
                placeholder="https://..."
                :class="{ 'border-red-500': formErrors.account_url }"
              />
              <p class="mt-1 text-xs text-gray-500">
                Leave empty to use platform's default URL pattern
              </p>
              <p v-if="formErrors.account_url" class="mt-1 text-sm text-red-500">
                {{ formErrors.account_url }}
              </p>
            </div>
          </div>

          <div class="flex justify-end gap-3 mt-6">
            <Button variant="ghost" @click="showModal = false">Cancel</Button>
            <Button @click="saveAccount" :disabled="saving">
              {{ saving ? 'Saving...' : (modalMode === 'create' ? 'Add Account' : 'Save Changes') }}
            </Button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
