<script setup>
import { computed } from 'vue'
import Button from '@/components/ui/Button.vue'
import { allTours } from '@/lib/tours/index.js'
import { useTour } from '@/composables/useTour.js'

const props = defineProps({
  show: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'start-tour'])

const { isTourCompleted } = useTour()

// Build a lookup by ID so we can resolve the display name of a required tour
const tourIndex = computed(() => {
  const idx = {}
  for (const t of allTours) idx[t.id] = t
  return idx
})

const tours = computed(() =>
  allTours
    .filter(t => !t.hidden)
    .map(t => ({
      ...t,
      completed: isTourCompleted(t.id),
      locked: !!t.requires && !isTourCompleted(t.requires),
      // Resolve the human-readable name of the prerequisite tour so the lock
      // message can be generic rather than hardcoded per-tour.
      requiresName: t.requires ? tourIndex.value[t.requires]?.name : null,
    }))
)

function onStartTour(tourId) {
  emit('start-tour', tourId)
}
</script>

<template>
  <!-- Backdrop -->
  <div
    v-if="show"
    class="fixed inset-0 z-[59] bg-black/30"
    @click="emit('close')"
  ></div>

  <!-- Slide-out Panel -->
  <div
    :class="[
      'fixed inset-y-0 right-0 z-[60] w-80 bg-white dark:bg-gray-800 shadow-xl transform transition-transform duration-300 ease-in-out flex flex-col',
      show ? 'translate-x-0' : 'translate-x-full'
    ]"
  >
    <!-- Header -->
    <div class="p-5 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <svg class="w-5 h-5 text-zinc-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
        </svg>
        <h2 class="text-lg font-bold text-gray-900 dark:text-white">Tutorials</h2>
      </div>
      <button
        @click="emit('close')"
        class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>

    <!-- Tour List -->
    <div class="flex-1 overflow-y-auto p-4 space-y-3">
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-2">
        Interactive walkthroughs to help you get started.
      </p>

      <div
        v-for="tour in tours"
        :key="tour.id"
        :class="[
          'border rounded-lg p-4 transition-colors',
          tour.locked
            ? 'opacity-50 border-gray-200 dark:border-gray-700'
            : 'border-gray-200 dark:border-gray-700 hover:border-zinc-300 dark:hover:border-zinc-700'
        ]"
      >
        <h3 class="font-semibold text-gray-900 dark:text-white text-sm leading-tight">
          {{ tour.name }}
        </h3>

        <div class="flex items-center gap-1 mt-1.5 mb-2">
          <span
            v-if="tour.completed"
            class="px-2 py-0.5 text-[10px] font-semibold rounded-full bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
          >
            Completed
          </span>
          <span
            v-if="tour.locked"
            class="px-2 py-0.5 text-[10px] font-semibold rounded-full bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400"
          >
            Locked
          </span>
        </div>

        <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
          {{ tour.description }}
        </p>

        <p v-if="tour.locked" class="text-[11px] text-amber-600 dark:text-amber-400 mb-3">
          Complete "{{ tour.requiresName || tour.requires }}" first
        </p>

        <div class="flex items-center justify-between">
          <span class="text-xs text-gray-400 dark:text-gray-500">
            ~{{ tour.estimatedMinutes }} min
          </span>
          <Button
            size="sm"
            :variant="tour.locked ? 'outline' : (tour.completed ? 'outline' : 'default')"
            :disabled="tour.locked"
            @click="onStartTour(tour.id)"
          >
            {{ tour.locked ? 'Locked' : (tour.completed ? 'Restart' : 'Start Tour') }}
          </Button>
        </div>
      </div>

      <!-- Placeholder for future tours -->
      <div v-if="tours.length <= 1" class="text-center py-4">
        <p class="text-xs text-gray-400 dark:text-gray-500">
          More tutorials coming soon.
        </p>
      </div>
    </div>
  </div>
</template>
