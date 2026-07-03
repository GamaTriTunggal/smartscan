<script setup>
import { ref, onMounted } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useDateTime } from '@/composables/useDateTime'
import { useToast } from '@/composables/useToast'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import PhoneInput from '@/components/PhoneInput.vue'
import { Copy, Check } from 'lucide-vue-next'

const { get, post, put, del } = useAPI()
const toast = useToast()
const { formatDate } = useDateTime()

const loading = ref(true)
const staff = ref([])
const pagination = ref({ page: 1, limit: 20, total: 0, total_page: 0 })
const showCreateModal = ref(false)
const showEditModal = ref(false)
const creating = ref(false)
const updating = ref(false)

const roleLabels = {
  admin: 'Administrator',
  qc_staff: 'QC Staff',
  warehouse_staff: 'Warehouse Staff'
}

const allRoleOptions = [
  { value: 'admin', label: 'Administrator' },
  { value: 'qc_staff', label: 'QC Staff' },
  { value: 'warehouse_staff', label: 'Warehouse Staff' }
]

const newStaff = ref({
  email: '',
  password: '',
  full_name: '',
  phone_number: '',
  position: '',
  role: 'admin'
})

const editStaff = ref({
  id: '',
  full_name: '',
  phone_number: '',
  position: '',
  role: ''
})

async function fetchStaff() {
  try {
    loading.value = true
    const response = await get('/tenant/staff', {
      page: pagination.value.page,
      limit: pagination.value.limit
    })
    if (response.success && response.data) {
      staff.value = response.data.staff || []
      pagination.value = response.data.pagination
    }
  } catch (error) {
    console.error('Failed to fetch staff:', error)
  } finally {
    loading.value = false
  }
}

async function createStaff() {
  if (!newStaff.value.email || !newStaff.value.password || !newStaff.value.full_name) return

  try {
    creating.value = true
    const response = await post('/tenant/staff', newStaff.value)
    if (response.success) {
      showCreateModal.value = false
      newStaff.value = { email: '', password: '', full_name: '', phone_number: '', position: '', role: 'admin' }
      fetchStaff()
    }
  } catch (error) {
    console.error('Failed to create staff:', error)
  } finally {
    creating.value = false
  }
}

async function updateStaff() {
  if (!editStaff.value.full_name) return

  try {
    updating.value = true
    const response = await put(`/tenant/staff/${editStaff.value.id}`, {
      full_name: editStaff.value.full_name,
      phone_number: editStaff.value.phone_number,
      position: editStaff.value.position,
      role: editStaff.value.role
    })
    if (response.success) {
      showEditModal.value = false
      fetchStaff()
    }
  } catch (error) {
    console.error('Failed to update staff:', error)
  } finally {
    updating.value = false
  }
}

async function deleteStaff(id) {
  if (!confirm('Are you sure you want to delete this staff member?')) return

  try {
    await del(`/tenant/staff/${id}`)
    fetchStaff()
  } catch (error) {
    console.error('Failed to delete staff:', error)
  }
}

function openEditModal(member) {
  editStaff.value = {
    id: member.id,
    full_name: member.full_name,
    phone_number: member.phone_number || '',
    position: member.position || '',
    role: member.role
  }
  showEditModal.value = true
}

// Password reset
const showResetConfirm = ref(false)
const resetTarget = ref(null)
const resetting = ref(false)
const showResetResult = ref(false)
const resetResult = ref({ email: '', temp_password: '' })
const copied = ref(false)

function openResetConfirm(member) {
  resetTarget.value = member
  showResetConfirm.value = true
}

async function confirmResetPassword() {
  if (!resetTarget.value) return

  try {
    resetting.value = true
    const response = await post(`/tenant/staff/${resetTarget.value.id}/reset-password`)
    if (response.success && response.data) {
      resetResult.value = {
        email: response.data.email,
        temp_password: response.data.temp_password
      }
      copied.value = false
      showResetConfirm.value = false
      showResetResult.value = true
    } else {
      toast.error(response.message || 'Failed to reset password')
    }
  } catch (error) {
    console.error('Failed to reset password:', error)
    toast.error(error.response?.data?.message || 'Failed to reset password')
  } finally {
    resetting.value = false
  }
}

async function copyTempPassword() {
  try {
    await navigator.clipboard.writeText(resetResult.value.temp_password)
    copied.value = true
    toast.success('Temporary password copied to clipboard')
    setTimeout(() => { copied.value = false }, 2000)
  } catch (error) {
    console.error('Failed to copy to clipboard:', error)
    toast.error('Failed to copy to clipboard')
  }
}

function closeResetResult() {
  showResetResult.value = false
  resetResult.value = { email: '', temp_password: '' }
}

function getRoleClass(role) {
  const classes = {
    admin: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
    qc_staff: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
    warehouse_staff: 'bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400'
  }
  return classes[role] || 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
}


onMounted(() => {
  fetchStaff()
})
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Staff Management</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Manage your team members and their roles</p>
      </div>
      <Button @click="showCreateModal = true">Add Staff</Button>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <div v-else>
      <Card v-if="staff.length === 0" class="p-6">
        <div class="text-center py-8">
          <svg class="w-16 h-16 text-gray-300 dark:text-gray-600 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
          </svg>
          <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No staff members yet</h3>
          <p class="text-gray-500 dark:text-gray-400 mb-4">Add your first team member to get started.</p>
          <Button @click="showCreateModal = true">Add Staff</Button>
        </div>
      </Card>

      <div v-else class="grid gap-4">
        <Card v-for="member in staff" :key="member.id" class="p-4">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-4">
              <div class="w-12 h-12 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center">
                <span class="text-lg font-semibold text-gray-600 dark:text-gray-300">
                  {{ member.full_name?.charAt(0)?.toUpperCase() || '?' }}
                </span>
              </div>
              <div>
                <div class="flex items-center gap-2">
                  <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ member.full_name }}</h3>
                  <span v-if="member.is_primary_admin" class="px-2 py-0.5 text-xs bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400 rounded">
                    Primary
                  </span>
                </div>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ member.user?.email }}</p>
                <div class="flex items-center gap-2 mt-1">
                  <span :class="['px-2 py-0.5 text-xs rounded-full', getRoleClass(member.role)]">
                    {{ roleLabels[member.role] || member.role }}
                  </span>
                  <span v-if="member.position" class="text-xs text-gray-400 dark:text-gray-500">
                    • {{ member.position }}
                  </span>
                </div>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <Button variant="outline" size="sm" @click="openEditModal(member)">
                Edit
              </Button>
              <Button variant="outline" size="sm" @click="openResetConfirm(member)">
                Reset Password
              </Button>
              <Button
                v-if="!member.is_primary_admin"
                variant="outline"
                size="sm"
                class="text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
                @click="deleteStaff(member.id)"
              >
                Delete
              </Button>
            </div>
          </div>
        </Card>

        <!-- Pagination -->
        <div v-if="pagination.total_page > 1" class="flex justify-center gap-2 mt-4">
          <Button
            variant="outline"
            size="sm"
            :disabled="pagination.page === 1"
            @click="pagination.page--; fetchStaff()"
          >
            Previous
          </Button>
          <span class="py-2 px-4 text-sm text-gray-600 dark:text-gray-400">
            Page {{ pagination.page }} of {{ pagination.total_page }}
          </span>
          <Button
            variant="outline"
            size="sm"
            :disabled="pagination.page >= pagination.total_page"
            @click="pagination.page++; fetchStaff()"
          >
            Next
          </Button>
        </div>
      </div>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreateModal" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="fixed inset-0 bg-black/50" @click="showCreateModal = false"></div>
      <div class="relative z-10 w-full max-w-md bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Add New Staff</h2>
        <form @submit.prevent="createStaff" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Email *</label>
            <Input v-model="newStaff.email" type="email" required placeholder="staff@example.com" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Password *</label>
            <Input v-model="newStaff.password" type="password" required placeholder="Minimum 8 characters" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Full Name *</label>
            <Input v-model="newStaff.full_name" type="text" required placeholder="Enter full name" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Phone Number</label>
            <PhoneInput v-model="newStaff.phone_number" placeholder="Phone number" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Position</label>
            <Input v-model="newStaff.position" type="text" placeholder="e.g., Quality Inspector" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Role *</label>
            <select
              v-model="newStaff.role"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
            >
              <option v-for="opt in allRoleOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>
          <div class="flex gap-3 pt-4">
            <Button type="button" variant="outline" class="flex-1" @click="showCreateModal = false">
              Cancel
            </Button>
            <Button type="submit" class="flex-1" :disabled="creating">
              {{ creating ? 'Creating...' : 'Add Staff' }}
            </Button>
          </div>
        </form>
      </div>
    </div>

    <!-- Edit Modal -->
    <div v-if="showEditModal" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="fixed inset-0 bg-black/50" @click="showEditModal = false"></div>
      <div class="relative z-10 w-full max-w-md bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Edit Staff</h2>
        <form @submit.prevent="updateStaff" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Full Name *</label>
            <Input v-model="editStaff.full_name" type="text" required placeholder="Enter full name" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Phone Number</label>
            <PhoneInput v-model="editStaff.phone_number" placeholder="Phone number" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Position</label>
            <Input v-model="editStaff.position" type="text" placeholder="e.g., Quality Inspector" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Role *</label>
            <select
              v-model="editStaff.role"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
            >
              <option v-for="opt in allRoleOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>
          <div class="flex gap-3 pt-4">
            <Button type="button" variant="outline" class="flex-1" @click="showEditModal = false">
              Cancel
            </Button>
            <Button type="submit" class="flex-1" :disabled="updating">
              {{ updating ? 'Saving...' : 'Save Changes' }}
            </Button>
          </div>
        </form>
      </div>
    </div>

    <!-- Reset Password Confirmation -->
    <ConfirmDialog
      :open="showResetConfirm"
      title="Reset Password"
      :message="`Reset the password for ${resetTarget?.full_name || 'this staff member'}? Their current password will stop working immediately.`"
      confirm-text="Reset Password"
      :loading="resetting"
      @confirm="confirmResetPassword"
      @cancel="showResetConfirm = false"
    />

    <!-- Reset Password Result Modal -->
    <div v-if="showResetResult" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="fixed inset-0 bg-black/50" @click="closeResetResult"></div>
      <div class="relative z-10 w-full max-w-md bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Temporary Password Created</h2>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Email</label>
            <p class="text-sm text-gray-900 dark:text-white bg-gray-50 dark:bg-gray-700 rounded-lg px-3 py-2 break-all">
              {{ resetResult.email }}
            </p>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Temporary Password</label>
            <div class="flex items-center gap-2">
              <p class="flex-1 font-mono text-sm text-gray-900 dark:text-white bg-gray-50 dark:bg-gray-700 rounded-lg px-3 py-2 break-all">
                {{ resetResult.temp_password }}
              </p>
              <Button variant="outline" size="sm" class="shrink-0" @click="copyTempPassword">
                <Check v-if="copied" class="w-4 h-4 text-green-500" />
                <Copy v-else class="w-4 h-4" />
              </Button>
            </div>
          </div>

          <p class="text-xs text-amber-700 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg px-3 py-2">
            Share this password with the staff member. They must change it at first login. It will not be shown again.
          </p>
        </div>

        <div class="flex justify-end pt-4">
          <Button @click="closeResetResult">Done</Button>
        </div>
      </div>
    </div>
  </div>
</template>
