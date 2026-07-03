<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useDarkMode } from '@/composables/useDarkMode'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Alert from '@/components/ui/Alert.vue'

const router = useRouter()
const authStore = useAuthStore()
useDarkMode() // Initialize dark mode

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const error = ref('')

async function handleChangePassword() {
  error.value = ''

  if (!currentPassword.value || !newPassword.value || !confirmPassword.value) {
    error.value = 'Please fill in all fields'
    return
  }
  if (newPassword.value.length < 8) {
    error.value = 'Password must be at least 8 characters'
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }

  loading.value = true
  try {
    // authStore.changePassword also clears must_change_password on success
    const result = await authStore.changePassword(currentPassword.value, newPassword.value)
    if (result.success) {
      router.push(authStore.dashboardPath)
    } else {
      error.value = result.error || 'Failed to change password'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 p-4 transition-colors duration-300">
    <Card class="w-full max-w-md p-8 bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
      <div class="text-center mb-8">
        <h1 class="text-2xl font-extrabold text-gray-900 dark:text-white tracking-tight">
          Set a new password
        </h1>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
          You need to change your password before continuing.
        </p>
      </div>

      <Alert v-if="error" variant="destructive" class="mb-6">
        {{ error }}
      </Alert>

      <form @submit.prevent="handleChangePassword" class="space-y-5">
        <div>
          <Label for="current-password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Current Password</Label>
          <Input
            id="current-password"
            v-model="currentPassword"
            type="password"
            placeholder="••••••••"
            :disabled="loading"
          />
        </div>

        <div>
          <Label for="new-password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">New Password</Label>
          <Input
            id="new-password"
            v-model="newPassword"
            type="password"
            placeholder="••••••••"
            :disabled="loading"
          />
          <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">Minimum 8 characters.</p>
        </div>

        <div>
          <Label for="confirm-new-password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Confirm New Password</Label>
          <Input
            id="confirm-new-password"
            v-model="confirmPassword"
            type="password"
            placeholder="••••••••"
            :disabled="loading"
          />
        </div>

        <Button type="submit" class="w-full" :loading="loading">
          <span v-if="!loading">Change Password</span>
          <span v-else>Saving...</span>
        </Button>
      </form>
    </Card>
  </div>
</template>
