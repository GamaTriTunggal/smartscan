<script setup>
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  type: { type: String, default: 'text' },
  placeholder: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  error: { type: Boolean, default: false },
  class: { type: String, default: '' },
  min: { type: [String, Number], default: undefined },
  max: { type: [String, Number], default: undefined },
})

const emit = defineEmits(['update:modelValue'])

const inputClasses = computed(() => {
  return cn(
    'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
    props.error && 'border-destructive focus-visible:ring-destructive',
    props.class
  )
})

function handleInput(event) {
  emit('update:modelValue', event.target.value)
}
</script>

<template>
  <input
    :class="inputClasses"
    :type="type"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    :min="min"
    :max="max"
    @input="handleInput"
  />
</template>
