<script setup>
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import { useDarkMode } from '@/composables/useDarkMode'
import { DEFAULT_APP_NAME } from '@/stores/branding'
import Toast from '@/components/ui/Toast.vue'

const { initDarkMode } = useDarkMode()

onMounted(() => {
  initDarkMode()
  // Set document title from app name (VITE_APP_NAME env variable)
  document.title = `${DEFAULT_APP_NAME} - Anti-Counterfeit QR Code Platform`
})
</script>

<template>
  <RouterView />
  <Toast />
</template>
