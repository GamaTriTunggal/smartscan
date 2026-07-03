<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useDarkMode } from '@/composables/useDarkMode'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Alert from '@/components/ui/Alert.vue'
import ThemeSwitcher from '@/components/ui/ThemeSwitcher.vue'
import { ShieldCheck, Zap } from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
useDarkMode() // Initialize dark mode

// Clear any stale auth state when landing on login page due to inactivity timeout
onMounted(() => {
  if (route.query.reason === 'inactivity') {
    authStore.logout()
    // Clear query param so refresh shows clean login page
    router.replace('/login')
  }
})

const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const showLogo = ref(true)

// Check if redirected due to inactivity
const inactivityMessage = computed(() => {
  return route.query.reason === 'inactivity' ? 'Your session has expired due to inactivity. Please sign in again.' : ''
})

const appName = import.meta.env.VITE_APP_NAME || 'smartscan'
const appTagline = (import.meta.env.VITE_APP_TAGLINE || 'Guard Your Brand\\nLike Nature Intended').replace(/\\n/g, '\n')

async function handleLogin() {
  if (!email.value || !password.value) {
    error.value = 'Please enter email and password'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const success = await authStore.login(email.value, password.value)
    if (success) {
      // Forced password change goes to the /change-password interstitial
      if (authStore.mustChangePassword) {
        router.push('/change-password')
      } else {
        // Use centralized dashboardPath (single source of truth for the
        // post-auth landing page, with a /login fallback for unknown users).
        router.push(authStore.dashboardPath)
      }
    } else {
      error.value = 'Invalid email or password'
    }
  } catch (e) {
    error.value = 'Login failed. Please try again.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex bg-white dark:bg-gray-900 transition-colors duration-300">
    <!-- Left Side - Visual & Branding (Hidden on mobile) -->
    <div class="hidden lg:flex w-1/2 relative bg-gray-900 text-white overflow-hidden">
      <!-- Animated Background -->
      <div class="absolute inset-0 bg-gradient-to-br from-zinc-900 via-zinc-800 to-[#27272a] opacity-90 transition-all duration-1000 ease-in-out"></div>

      <!-- Decorative Elements -->
      <div class="absolute -top-24 -left-24 w-96 h-96 bg-white opacity-10 rounded-full blur-3xl"></div>
      <div class="absolute top-1/2 left-1/4 w-64 h-64 bg-zinc-400 opacity-20 rounded-full blur-2xl animate-pulse"></div>
      <div class="absolute bottom-0 right-0 w-[500px] h-[500px] bg-zinc-500 opacity-10 rounded-full blur-3xl"></div>

      <!-- Content -->
      <div class="relative z-10 w-full flex flex-col justify-center px-16 xl:px-24">
        <div class="mb-12">
          <div class="mb-8">
             <img src="/logo.svg" alt="smartscan" class="w-64 h-auto" />
          </div>
          <h1 class="text-5xl font-bold tracking-tight mb-6 leading-tight whitespace-pre-line">
            {{ appTagline }}
          </h1>
          <p class="text-lg text-zinc-50 max-w-md leading-relaxed">
            Manage your product authentication and warranty services with the next-generation smart labeling platform.
          </p>
        </div>

        <div class="grid grid-cols-1 gap-6">
          <div class="flex items-center space-x-4 text-zinc-50 bg-white/5 p-4 rounded-xl backdrop-blur-sm border border-white/10">
            <Zap class="w-6 h-6 text-yellow-400" />
            <span class="font-medium">Real-time Analytics</span>
          </div>
          <div class="flex items-center space-x-4 text-zinc-50 bg-white/5 p-4 rounded-xl backdrop-blur-sm border border-white/10">
            <ShieldCheck class="w-6 h-6 text-green-400" />
            <span class="font-medium">Anti-counterfeit Protection</span>
          </div>
        </div>

        <div class="mt-16 text-sm text-zinc-100 opacity-60">
          {{ appName }} — open-source product authentication
        </div>
      </div>
    </div>

    <!-- Right Side - Login Form -->
    <div class="w-full lg:w-1/2 flex flex-col items-center justify-center p-8 sm:p-12 relative overflow-hidden">
        <!-- Theme Switcher Top Right -->
        <div class="absolute top-6 right-6 z-20">
          <ThemeSwitcher />
        </div>

        <!-- Decorative background blobb layout mobile -->
        <div class="lg:hidden absolute top-0 left-0 w-full h-2 bg-gradient-to-r from-zinc-600 via-zinc-500 to-[#27272a]"></div>

        <div class="w-full max-w-md space-y-8 relative z-10">
          <div class="text-center lg:text-left">
            <div class="flex justify-center lg:justify-start mb-6 items-center gap-3">
                 <img v-if="showLogo" src="/logo.svg" alt="App Logo" class="h-10 w-auto" @error="showLogo = false" />
                 <span class="text-2xl font-bold text-gray-900 dark:text-white">{{ appName }}</span>
            </div>
            <h2 class="text-3xl font-extrabold text-gray-900 dark:text-white tracking-tight">
              Welcome back
            </h2>
            <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
              Please enter your details to sign in
            </p>
          </div>

          <Alert v-if="inactivityMessage" variant="default" class="animate-in fade-in slide-in-from-top-4 duration-300 bg-zinc-50 dark:bg-zinc-900/20 border-zinc-200 dark:border-zinc-800 text-zinc-800 dark:text-zinc-200">
            {{ inactivityMessage }}
          </Alert>

          <Alert v-if="error" variant="destructive" class="animate-in fade-in slide-in-from-top-4 duration-300">
            {{ error }}
          </Alert>

          <form @submit.prevent="handleLogin" class="mt-8 space-y-6">
            <div class="space-y-5">
              <div class="group">
                <Label for="email" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Email address</Label>
                <div class="relative">
                    <Input
                      id="email"
                      v-model="email"
                      type="email"
                      placeholder="name@company.com"
                      :disabled="loading"
                      class="pl-4 py-3 bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700 focus:ring-2 focus:ring-[#27272a] focus:border-transparent transition-all duration-200"
                    />
                </div>
              </div>

              <div class="group">
                <Label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Password</Label>
                <Input
                  id="password"
                  v-model="password"
                  type="password"
                  placeholder="••••••••"
                  :disabled="loading"
                   class="pl-4 py-3 bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700 focus:ring-2 focus:ring-[#27272a] focus:border-transparent transition-all duration-200"
                />
                <p class="mt-2 text-xs text-gray-400 dark:text-gray-500">
                  Forgot your password? Contact your administrator.
                </p>
              </div>
            </div>

            <Button type="submit" class="w-full py-3 bg-gradient-to-r from-zinc-600 to-[#27272a] hover:from-zinc-700 hover:to-[#2bcac9] text-white font-bold rounded-lg shadow-md hover:shadow-lg transform transition-all duration-200 hover:-translate-y-0.5" :loading="loading">
              <span v-if="!loading">Sign in to Dashboard</span>
              <span v-else>Signing in...</span>
            </Button>
          </form>

          <p class="mt-6 text-xs text-center text-gray-500 dark:text-gray-500">
            By clicking sign in, you agree to our
            <a href="#" class="font-medium text-zinc-600 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300 underline transition-colors">Terms of Service</a> and
            <a href="#" class="font-medium text-zinc-600 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300 underline transition-colors">Privacy Policy</a>.
          </p>

        </div>
    </div>
  </div>
</template>

<style scoped>
/* Any custom styles not covered by Tailwind can go here */
</style>
