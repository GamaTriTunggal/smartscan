<script setup>
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps({
  variant: { type: String, default: 'default' },
  class: { type: String, default: '' },
})

const alertClasses = computed(() => {
  const base = 'relative w-full rounded-lg border p-4'

  const variants = {
    default: 'bg-background text-foreground',
    destructive: 'border-destructive/50 text-destructive dark:border-destructive bg-destructive/10',
    success: 'border-green-500/50 text-green-700 dark:text-green-400 bg-green-500/10',
  }

  return cn(base, variants[props.variant], props.class)
})
</script>

<template>
  <div :class="alertClasses" role="alert">
    <slot />
  </div>
</template>
