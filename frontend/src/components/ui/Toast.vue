<script setup>
import { useToast } from '@/composables/useToast'
import { CheckCircle, XCircle, Info, AlertTriangle, X } from 'lucide-vue-next'

const { toasts, dismiss } = useToast()

const icons = {
  success: CheckCircle,
  error: XCircle,
  info: Info,
  warning: AlertTriangle
}

const styles = {
  success: 'bg-green-50 dark:bg-green-900/30 border-green-200 dark:border-green-800 text-green-800 dark:text-green-200',
  error: 'bg-red-50 dark:bg-red-900/30 border-red-200 dark:border-red-800 text-red-800 dark:text-red-200',
  info: 'bg-zinc-50 dark:bg-zinc-900/30 border-zinc-200 dark:border-zinc-800 text-zinc-800 dark:text-zinc-200',
  warning: 'bg-yellow-50 dark:bg-yellow-900/30 border-yellow-200 dark:border-yellow-800 text-yellow-800 dark:text-yellow-200'
}

const iconStyles = {
  success: 'text-green-500 dark:text-green-400',
  error: 'text-red-500 dark:text-red-400',
  info: 'text-zinc-500 dark:text-zinc-400',
  warning: 'text-yellow-500 dark:text-yellow-400'
}
</script>

<template>
  <div class="fixed top-4 right-4 z-[9999] flex flex-col gap-2 max-w-sm w-full pointer-events-none">
    <TransitionGroup name="toast">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        :class="[
          'pointer-events-auto flex items-start gap-3 p-4 rounded-lg border shadow-lg',
          styles[toast.type] || styles.info
        ]"
      >
        <component
          :is="icons[toast.type] || icons.info"
          :class="['w-5 h-5 flex-shrink-0 mt-0.5', iconStyles[toast.type] || iconStyles.info]"
        />
        <p class="flex-1 text-sm font-medium">{{ toast.message }}</p>
        <button
          @click="dismiss(toast.id)"
          class="flex-shrink-0 p-1 rounded hover:bg-black/10 dark:hover:bg-white/10 transition-colors"
        >
          <X class="w-4 h-4" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-enter-active {
  transition: all 0.3s ease-out;
}

.toast-leave-active {
  transition: all 0.2s ease-in;
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

.toast-move {
  transition: transform 0.3s ease;
}
</style>
