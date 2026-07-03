<script setup>
import { computed } from 'vue'
import { Loader2, CheckCircle2, XCircle, Clock } from 'lucide-vue-next'

const props = defineProps({
  status: {
    type: String,
    required: true,
    // pending_queue, queued, processing, completed, failed
  },
  generatedCount: {
    type: Number,
    default: 0,
  },
  totalQrCount: {
    type: Number,
    required: true,
  },
  progressPercent: {
    type: Number,
    default: 0,
  },
  etaSeconds: {
    type: Number,
    default: null,
  },
  errorMessage: {
    type: String,
    default: '',
  },
})

const statusLabel = computed(() => {
  switch (props.status) {
    case 'pending_queue':
      return 'Waiting for queue'
    case 'queued':
      return 'Queued'
    case 'processing':
      return 'Generating'
    case 'completed':
      return 'Completed'
    case 'failed':
      return 'Failed'
    default:
      return props.status
  }
})

const statusColor = computed(() => {
  switch (props.status) {
    case 'pending_queue':
    case 'queued':
      return 'text-yellow-600 dark:text-yellow-400'
    case 'processing':
      return 'text-zinc-600 dark:text-zinc-400'
    case 'completed':
      return 'text-green-600 dark:text-green-400'
    case 'failed':
      return 'text-red-600 dark:text-red-400'
    default:
      return 'text-gray-600 dark:text-gray-400'
  }
})

const progressBarColor = computed(() => {
  switch (props.status) {
    case 'completed':
      return 'bg-green-500 dark:bg-green-400'
    case 'failed':
      return 'bg-red-500 dark:bg-red-400'
    default:
      return 'bg-zinc-500 dark:bg-zinc-400'
  }
})

const displayProgress = computed(() => {
  // Cap at 100, round to int, ensure non-negative
  const pct = Math.round(Math.max(0, Math.min(100, props.progressPercent || 0)))
  return pct
})

const formattedGenerated = computed(() => {
  return (props.generatedCount || 0).toLocaleString()
})

const formattedTotal = computed(() => {
  return (props.totalQrCount || 0).toLocaleString()
})

const etaDisplay = computed(() => {
  if (props.etaSeconds == null || props.etaSeconds < 0) return null
  const seconds = props.etaSeconds
  if (seconds < 60) return `${seconds}s remaining`
  if (seconds < 3600) {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}m ${secs}s remaining`
  }
  const hours = Math.floor(seconds / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  return `${hours}h ${mins}m remaining`
})

const StatusIcon = computed(() => {
  switch (props.status) {
    case 'completed':
      return CheckCircle2
    case 'failed':
      return XCircle
    case 'pending_queue':
    case 'queued':
      return Clock
    default:
      return Loader2
  }
})

const isSpinning = computed(() => props.status === 'processing')
</script>

<template>
  <div class="space-y-2">
    <!-- Status line -->
    <div class="flex items-center justify-between text-sm">
      <div class="flex items-center gap-2" :class="statusColor">
        <component
          :is="StatusIcon"
          class="w-4 h-4"
          :class="{ 'animate-spin': isSpinning }"
        />
        <span class="font-medium">{{ statusLabel }}</span>
      </div>
      <div class="text-gray-600 dark:text-gray-400">
        <span class="font-mono">{{ formattedGenerated }}</span>
        <span class="text-gray-400 mx-1">/</span>
        <span class="font-mono">{{ formattedTotal }}</span>
        <span class="ml-2">({{ displayProgress }}%)</span>
      </div>
    </div>

    <!-- Progress bar -->
    <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2 overflow-hidden">
      <div
        class="h-full transition-all duration-300 ease-out"
        :class="progressBarColor"
        :style="{ width: `${displayProgress}%` }"
      ></div>
    </div>

    <!-- ETA -->
    <div v-if="etaDisplay && status === 'processing'" class="text-xs text-gray-500 dark:text-gray-400">
      {{ etaDisplay }}
    </div>

    <!-- Error message -->
    <div
      v-if="status === 'failed' && errorMessage"
      class="mt-2 p-2 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-xs text-red-700 dark:text-red-300"
    >
      {{ errorMessage }}
    </div>
  </div>
</template>
