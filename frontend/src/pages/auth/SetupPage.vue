<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useAPI } from '@/composables/useAPI'
import { useDarkMode } from '@/composables/useDarkMode'
import { setSetupComplete } from '@/router'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Alert from '@/components/ui/Alert.vue'

const router = useRouter()
const authStore = useAuthStore()
const { post } = useAPI()
useDarkMode() // Initialize dark mode

const companyName = ref('')
const adminName = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const error = ref('')
const alreadySetUp = ref(false)

async function handleSetup() {
  error.value = ''
  alreadySetUp.value = false

  if (!companyName.value || !adminName.value || !email.value || !password.value) {
    error.value = 'Please fill in all fields'
    return
  }
  if (password.value.length < 8) {
    error.value = 'Password must be at least 8 characters'
    return
  }
  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }

  loading.value = true
  try {
    const response = await post('/setup', {
      company_name: companyName.value,
      admin_name: adminName.value,
      email: email.value,
      password: password.value,
    })
    if (response.success && response.data) {
      // Same login-state handling as LoginPage after a successful login
      authStore.setAuthenticated(true)
      authStore.setUser(response.data.user)
      if (response.data.expires_in) {
        authStore.setTokenExpiry(response.data.expires_in)
      }
      setSetupComplete()
      router.push('/tenant/dashboard')
    } else {
      error.value = response.message || 'Setup failed. Please try again.'
    }
  } catch (e) {
    if (e.response?.status === 409) {
      alreadySetUp.value = true
      error.value = 'This application has already been set up.'
    } else {
      error.value = e.response?.data?.message || 'Setup failed. Please try again.'
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
        <h1 class="text-3xl font-extrabold text-gray-900 dark:text-white tracking-tight">
          Welcome to smartscan
        </h1>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
          Create your company and administrator account to get started.
        </p>
      </div>

      <Alert v-if="error" variant="destructive" class="mb-6">
        {{ error }}
        <router-link v-if="alreadySetUp" to="/login" class="block mt-1 font-medium underline">
          Go to sign in
        </router-link>
      </Alert>

      <form @submit.prevent="handleSetup" class="space-y-5">
        <div>
          <Label for="company-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Company Name</Label>
          <Input
            id="company-name"
            v-model="companyName"
            type="text"
            placeholder="Acme Inc."
            :disabled="loading"
          />
        </div>

        <div>
          <Label for="admin-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Your Name</Label>
          <Input
            id="admin-name"
            v-model="adminName"
            type="text"
            placeholder="Jane Doe"
            :disabled="loading"
          />
        </div>

        <div>
          <Label for="email" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Email</Label>
          <Input
            id="email"
            v-model="email"
            type="email"
            placeholder="name@company.com"
            :disabled="loading"
          />
        </div>

        <div>
          <Label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Password</Label>
          <Input
            id="password"
            v-model="password"
            type="password"
            placeholder="••••••••"
            :disabled="loading"
          />
          <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">Minimum 8 characters.</p>
        </div>

        <div>
          <Label for="confirm-password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Confirm Password</Label>
          <Input
            id="confirm-password"
            v-model="confirmPassword"
            type="password"
            placeholder="••••••••"
            :disabled="loading"
          />
        </div>

        <Button type="submit" class="w-full" :loading="loading">
          <span v-if="!loading">Create Account</span>
          <span v-else>Setting up...</span>
        </Button>
      </form>
    </Card>
  </div>
</template>
