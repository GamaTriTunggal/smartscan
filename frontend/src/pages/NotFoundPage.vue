<script setup>
import { useRouter } from 'vue-router'
import Button from '@/components/ui/Button.vue'

const router = useRouter()
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50">
    <div class="text-center">
      <h1 class="text-9xl font-bold text-gray-200">404</h1>
      <p class="text-2xl font-semibold text-gray-900 mb-4">Page Not Found</p>
      <p class="text-gray-600 mb-8">The page you're looking for doesn't exist or has been moved.</p>
      <Button @click="router.push('/')">Go Home</Button>
    </div>
  </div>
</template>
