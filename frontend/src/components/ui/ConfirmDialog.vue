<script setup>
import Button from '@/components/ui/Button.vue'
import { AlertTriangle, Info } from 'lucide-vue-next'

defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: 'Confirm Action' },
  message: { type: String, default: 'Are you sure you want to proceed?' },
  confirmText: { type: String, default: 'Confirm' },
  cancelText: { type: String, default: 'Cancel' },
  variant: { type: String, default: 'default' },
  loading: { type: Boolean, default: false },
})

const emit = defineEmits(['confirm', 'cancel'])
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      :aria-label="title"
    >
      <!-- Backdrop -->
      <div class="fixed inset-0 bg-black/50 transition-opacity" @click="emit('cancel')" />

      <!-- Dialog -->
      <div class="relative z-10 w-full max-w-md mx-4 bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6">
        <div class="flex items-start gap-3 mb-4">
          <AlertTriangle
            v-if="variant === 'destructive'"
            class="w-5 h-5 mt-0.5 shrink-0 text-red-500"
          />
          <Info
            v-else
            class="w-5 h-5 mt-0.5 shrink-0 text-zinc-500"
          />
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ title }}</h2>
            <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">{{ message }}</p>
          </div>
        </div>
        <div class="flex gap-3 justify-end">
          <Button variant="outline" :disabled="loading" @click="emit('cancel')">
            {{ cancelText }}
          </Button>
          <Button
            :variant="variant === 'destructive' ? 'destructive' : 'default'"
            :disabled="loading"
            :loading="loading"
            @click="emit('confirm')"
          >
            {{ loading ? 'Processing...' : confirmText }}
          </Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
