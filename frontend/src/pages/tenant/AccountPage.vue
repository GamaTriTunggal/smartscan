<script setup>
import { ref, onMounted, computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useAPI } from '@/composables/useAPI'
import { useToast } from '@/composables/useToast'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import PhoneInput from '@/components/PhoneInput.vue'

const authStore = useAuthStore()
const { get, put, post } = useAPI()
const toast = useToast()

const activeTab = ref('profile')
const loading = ref(false)
const saving = ref(false)

// Profile form
const profileForm = ref({
  full_name: '',
  phone_number: '',
  address: ''
})

// Company form (Admin only)
const companyForm = ref({
  company_name: '',
  company_address: '',
  phone_number: '',
  business_field: ''
})

// Password form
const passwordForm = ref({
  current_password: '',
  new_password: '',
  confirm_password: ''
})

const roleLabels = {
  admin: 'Administrator',
  qc_staff: 'QC Staff',
  warehouse_staff: 'Warehouse Staff'
}

const isAdmin = computed(() => authStore.user?.role === 'admin')

function getRoleClass(role) {
  const classes = {
    admin: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
    qc_staff: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
    warehouse_staff: 'bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400'
  }
  return classes[role] || 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
}

function loadProfile() {
  if (authStore.user) {
    profileForm.value = {
      full_name: authStore.user.full_name || '',
      phone_number: authStore.user.phone_number || '',
      address: authStore.user.address || ''
    }
  }
}

async function loadCompanyInfo() {
  if (!isAdmin.value) return

  loading.value = true
  try {
    const response = await get('/tenant/info')
    if (response.success && response.data) {
      companyForm.value = {
        company_name: response.data.company_name || '',
        company_address: response.data.company_address || '',
        phone_number: response.data.phone_number || '',
        business_field: response.data.business_field || ''
      }
    }
  } catch (error) {
    console.error('Failed to load company info:', error)
  } finally {
    loading.value = false
  }
}

async function saveProfile() {
  saving.value = true

  try {
    const response = await put('/me', profileForm.value)
    if (response.success) {
      toast.success('Profile updated successfully')
      // Update local auth store
      if (authStore.user) {
        authStore.user.full_name = profileForm.value.full_name
        authStore.user.phone_number = profileForm.value.phone_number
        authStore.user.address = profileForm.value.address
      }
    } else {
      toast.error(response.message || 'Failed to update profile')
    }
  } catch (error) {
    toast.error('Failed to update profile')
  } finally {
    saving.value = false
  }
}

async function saveCompanyInfo() {
  saving.value = true

  try {
    const response = await put('/tenant/info', companyForm.value)
    if (response.success) {
      toast.success('Company info updated successfully')
      // Update tenant name in auth store
      if (authStore.user && companyForm.value.company_name) {
        authStore.user.tenant_name = companyForm.value.company_name
      }
    } else {
      toast.error(response.message || 'Failed to update company info')
    }
  } catch (error) {
    toast.error('Failed to update company info')
  } finally {
    saving.value = false
  }
}

async function changePassword() {
  if (passwordForm.value.new_password !== passwordForm.value.confirm_password) {
    toast.error('Passwords do not match')
    return
  }

  if (passwordForm.value.new_password.length < 8) {
    toast.error('Password must be at least 8 characters')
    return
  }

  saving.value = true

  try {
    const response = await post('/auth/change-password', {
      current_password: passwordForm.value.current_password,
      new_password: passwordForm.value.new_password
    })
    if (response.success) {
      toast.success('Password changed successfully')
      passwordForm.value = {
        current_password: '',
        new_password: '',
        confirm_password: ''
      }
    } else {
      toast.error(response.message || 'Failed to change password')
    }
  } catch (error) {
    toast.error(error.response?.data?.message || 'Failed to change password')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadProfile()
  loadCompanyInfo()
})
</script>

<template>
  <div>
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">My Account</h1>
      <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Manage your profile and security settings</p>
    </div>

    <!-- Tabs -->
    <div class="border-b border-gray-200 dark:border-gray-700 mb-6">
      <nav class="flex gap-4">
        <button
          @click="activeTab = 'profile'"
          :class="[
            'py-2 px-1 border-b-2 font-medium text-sm transition-colors',
            activeTab === 'profile'
              ? 'border-zinc-500 text-zinc-600'
              : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
          ]"
        >
          Profile
        </button>
        <button
          v-if="isAdmin"
          @click="activeTab = 'company'"
          :class="[
            'py-2 px-1 border-b-2 font-medium text-sm transition-colors',
            activeTab === 'company'
              ? 'border-zinc-500 text-zinc-600'
              : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
          ]"
        >
          Company
        </button>
        <button
          @click="activeTab = 'security'"
          :class="[
            'py-2 px-1 border-b-2 font-medium text-sm transition-colors',
            activeTab === 'security'
              ? 'border-zinc-500 text-zinc-600'
              : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
          ]"
        >
          Security
        </button>
      </nav>
    </div>


    <!-- Profile Tab -->
    <div v-if="activeTab === 'profile'" class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Profile Info Card -->
      <Card class="p-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Profile Information</h2>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Email</label>
            <Input :value="authStore.user?.email" disabled class="bg-gray-50 dark:bg-gray-800" />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Email cannot be changed</p>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Full Name</label>
            <Input v-model="profileForm.full_name" placeholder="Your full name" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Phone Number</label>
            <PhoneInput v-model="profileForm.phone_number" placeholder="Phone number" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Address</label>
            <textarea
              v-model="profileForm.address"
              rows="3"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
              placeholder="Your address"
            ></textarea>
          </div>

          <div class="pt-2">
            <Button @click="saveProfile" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save Changes' }}
            </Button>
          </div>
        </div>
      </Card>

      <!-- Account Summary Card -->
      <Card class="p-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Account Summary</h2>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-500 dark:text-gray-400">Role</label>
            <div class="mt-1">
              <span :class="['px-2 py-1 text-xs font-medium rounded-full', getRoleClass(authStore.user?.role)]">
                {{ roleLabels[authStore.user?.role] || authStore.user?.role }}
              </span>
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-500 dark:text-gray-400">Company</label>
            <div class="mt-1 text-gray-900 dark:text-white">
              {{ authStore.user?.tenant_name || '-' }}
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-500 dark:text-gray-400">Account Status</label>
            <div class="mt-1">
              <span class="px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
                Active
              </span>
            </div>
          </div>
        </div>
      </Card>
    </div>

    <!-- Company Tab (Admin only) -->
    <div v-if="activeTab === 'company' && isAdmin">
      <Card class="p-6 max-w-2xl">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Company Information</h2>

        <div v-if="loading" class="flex justify-center py-8">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500 dark:border-zinc-400"></div>
        </div>

        <div v-else class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Company Name</label>
            <Input v-model="companyForm.company_name" placeholder="PT Example Indonesia" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Business Field</label>
            <Input v-model="companyForm.business_field" placeholder="e.g., Manufacturing, Retail, F&B" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Company Phone</label>
            <PhoneInput v-model="companyForm.phone_number" placeholder="Company phone" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Company Address</label>
            <textarea
              v-model="companyForm.company_address"
              rows="3"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
              placeholder="Company address"
            ></textarea>
          </div>

          <div class="pt-2">
            <Button @click="saveCompanyInfo" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save Company Info' }}
            </Button>
          </div>
        </div>
      </Card>
    </div>

    <!-- Security Tab -->
    <div v-if="activeTab === 'security'">
      <Card class="p-6 max-w-2xl">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Change Password</h2>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Current Password</label>
            <Input v-model="passwordForm.current_password" type="password" placeholder="Enter current password" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">New Password</label>
            <Input v-model="passwordForm.new_password" type="password" placeholder="Enter new password" />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Minimum 8 characters</p>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Confirm New Password</label>
            <Input v-model="passwordForm.confirm_password" type="password" placeholder="Confirm new password" />
          </div>

          <div class="pt-2">
            <Button @click="changePassword" :disabled="saving">
              {{ saving ? 'Changing...' : 'Change Password' }}
            </Button>
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>
