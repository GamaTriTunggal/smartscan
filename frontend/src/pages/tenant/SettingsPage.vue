<script setup>
import { ref, onMounted, computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useAPI } from '@/composables/useAPI'
import { useToast } from '@/composables/useToast'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import PhoneInput from '@/components/PhoneInput.vue'
import Label from '@/components/ui/Label.vue'

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

// Company form
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

// Counterfeit Settings
const counterfeitSettings = ref({
  qc_scan_max: 0,
  warehouse_scan_max: 0,
  end_user_scan_max: 0,
  velocity_check_enabled: false,
  max_speed_kmh: 1000,
  auto_flag_suspicious: true,
})

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

async function loadCounterfeitSettings() {
  loading.value = true
  try {
    const response = await get('/tenant/counterfeit/settings')
    if (response.success && response.data) {
      counterfeitSettings.value = {
        qc_scan_max: response.data.qc_scan_max || 0,
        warehouse_scan_max: response.data.warehouse_scan_max || 0,
        end_user_scan_max: response.data.end_user_scan_max || 0,
        velocity_check_enabled: response.data.velocity_check_enabled || false,
        max_speed_kmh: response.data.max_speed_kmh || 1000,
        auto_flag_suspicious: response.data.auto_flag_suspicious !== false,
      }
    }
  } catch (error) {
    console.error('Failed to load counterfeit settings:', error)
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

async function saveCounterfeitSettings() {
  saving.value = true

  try {
    const response = await put('/tenant/counterfeit/settings', counterfeitSettings.value)
    if (response.success) {
      toast.success('Counterfeit settings saved successfully')
    } else {
      toast.error(response.message || 'Failed to save settings')
    }
  } catch (error) {
    toast.error('Failed to save settings')
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  loadProfile()
  loadCompanyInfo()
  loadCounterfeitSettings()
})
</script>

<template>
  <div>
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Settings</h1>
      <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Manage your profile, company, and security settings</p>
    </div>

    <!-- Tabs -->
    <div class="border-b border-gray-200 dark:border-gray-700 mb-6">
      <nav class="flex gap-4 overflow-x-auto">
        <button
          @click="activeTab = 'profile'"
          :class="[
            'py-2 px-1 border-b-2 font-medium text-sm transition-colors whitespace-nowrap',
            activeTab === 'profile'
              ? 'border-zinc-500 text-zinc-600'
              : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
          ]"
        >
          Profile
        </button>
        <button
          @click="activeTab = 'company'"
          :class="[
            'py-2 px-1 border-b-2 font-medium text-sm transition-colors whitespace-nowrap',
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
            'py-2 px-1 border-b-2 font-medium text-sm transition-colors whitespace-nowrap',
            activeTab === 'security'
              ? 'border-zinc-500 text-zinc-600'
              : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
          ]"
        >
          Security
        </button>
        <button
          @click="activeTab = 'counterfeit'"
          :class="[
            'py-2 px-1 border-b-2 font-medium text-sm transition-colors whitespace-nowrap',
            activeTab === 'counterfeit'
              ? 'border-zinc-500 text-zinc-600'
              : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
          ]"
        >
          Counterfeit Detection
        </button>
      </nav>
    </div>

    <!-- Profile Tab -->
    <div v-if="activeTab === 'profile'" class="grid grid-cols-1 lg:grid-cols-2 gap-6">
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

      <Card class="p-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Account Summary</h2>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-500 dark:text-gray-400">Role</label>
            <div class="mt-1">
              <span class="px-2 py-1 text-xs font-medium rounded-full bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400">
                Administrator
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

    <!-- Company Tab -->
    <div v-if="activeTab === 'company'">
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

    <!-- Counterfeit Detection Tab -->
    <div v-if="activeTab === 'counterfeit'">
      <Card class="p-6 dark:bg-gray-800 dark:border-gray-700">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          Counterfeit Detection Settings
        </h2>
        <p class="text-sm text-gray-600 dark:text-gray-400 mb-1">
          These are <strong class="text-gray-700 dark:text-gray-300">global default settings</strong> applied to all products.
          A value of 0 means unlimited (no detection).
        </p>
        <p class="text-sm text-gray-600 dark:text-gray-400 mb-6">
          Each product may have its own threshold override depending on your needs — configure per-product settings in the product detail page.
        </p>

        <div v-if="loading" class="flex justify-center py-8">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500 dark:border-zinc-400"></div>
        </div>

        <div v-else>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div class="space-y-2">
              <Label class="dark:text-gray-300">QC Staff Scan Limit</Label>
              <Input
                v-model.number="counterfeitSettings.qc_scan_max"
                type="number"
                min="0"
                placeholder="0 = unlimited"
              />
              <p class="text-xs text-gray-500 dark:text-gray-400">
                Max scans by QC Staff before flagging as suspicious
              </p>
            </div>

            <div class="space-y-2">
              <Label class="dark:text-gray-300">Warehouse Staff Scan Limit</Label>
              <Input
                v-model.number="counterfeitSettings.warehouse_scan_max"
                type="number"
                min="0"
                placeholder="0 = unlimited"
              />
              <p class="text-xs text-gray-500 dark:text-gray-400">
                Max scans by Warehouse Staff before flagging as suspicious
              </p>
            </div>

            <div class="space-y-2">
              <Label class="dark:text-gray-300">End User Scan Limit</Label>
              <Input
                v-model.number="counterfeitSettings.end_user_scan_max"
                type="number"
                min="0"
                placeholder="0 = unlimited"
              />
              <p class="text-xs text-gray-500 dark:text-gray-400">
                Max scans by end users before flagging as suspicious
              </p>
            </div>

            <div class="space-y-2">
              <Label class="dark:text-gray-300">Max Travel Speed (km/h)</Label>
              <Input
                v-model.number="counterfeitSettings.max_speed_kmh"
                type="number"
                min="0"
                placeholder="1000"
                :disabled="!counterfeitSettings.velocity_check_enabled"
              />
              <p class="text-xs text-gray-500 dark:text-gray-400">
                Maximum reasonable travel speed between scans
              </p>
            </div>
          </div>

          <div class="mt-6 space-y-4">
            <label class="flex items-center space-x-3 cursor-pointer">
              <input
                type="checkbox"
                v-model="counterfeitSettings.velocity_check_enabled"
                class="w-4 h-4 text-zinc-600 bg-gray-100 border-gray-300 rounded focus:ring-[#27272a] dark:focus:ring-[#27272a] dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
              />
              <div>
                <span class="text-sm font-medium text-gray-900 dark:text-white">
                  Velocity Detection
                </span>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  Detect impossible travel (geolocation anomaly)
                </p>
              </div>
            </label>

            <label class="flex items-center space-x-3 cursor-pointer">
              <input
                type="checkbox"
                v-model="counterfeitSettings.auto_flag_suspicious"
                class="w-4 h-4 text-zinc-600 bg-gray-100 border-gray-300 rounded focus:ring-[#27272a] dark:focus:ring-[#27272a] dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
              />
              <div>
                <span class="text-sm font-medium text-gray-900 dark:text-white">
                  Auto-Flag Suspicious
                </span>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  Automatically flag when threshold is exceeded
                </p>
              </div>
            </label>
          </div>

          <div class="mt-6 pt-6 border-t border-gray-200 dark:border-gray-700">
            <Button @click="saveCounterfeitSettings" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save Settings' }}
            </Button>
          </div>
        </div>
      </Card>

      <!-- Info Card -->
      <Card class="p-6 mt-6 bg-zinc-50 dark:bg-zinc-900/20 border-zinc-200 dark:border-zinc-800 max-w-2xl">
        <div class="flex items-start space-x-3">
          <svg class="w-6 h-6 text-zinc-600 dark:text-zinc-400 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div>
            <h3 class="text-sm font-semibold text-zinc-800 dark:text-zinc-300">
              About Counterfeit Detection
            </h3>
            <p class="mt-1 text-sm text-zinc-700 dark:text-zinc-400">
              The system detects potential counterfeiting based on:
            </p>
            <ul class="mt-2 text-sm text-zinc-700 dark:text-zinc-400 list-disc list-inside space-y-1">
              <li><strong>Scan Threshold:</strong> QR code scanned more than the limit</li>
              <li><strong>Geolocation Anomaly:</strong> QR code scanned from distant locations in short time</li>
            </ul>
            <p class="mt-2 text-sm text-zinc-700 dark:text-zinc-400">
              View detection results in the <strong>Counterfeit</strong> menu.
            </p>
          </div>
        </div>
      </Card>
    </div>

  </div>
</template>
