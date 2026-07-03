<script setup>
import { ref, computed } from 'vue'
import { useEscapeKey } from '@/composables/useEscapeKey'
import Button from '@/components/ui/Button.vue'
import { Download, X, Search } from 'lucide-vue-next'

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: 'Export Data' },
  loading: { type: Boolean, default: false },
})

const emit = defineEmits(['confirm', 'cancel'])

useEscapeKey(() => {
  if (props.open) emit('cancel')
})

const TIMEZONE_GROUPS = [
  {
    label: 'Indonesia',
    zones: [
      { value: 'Asia/Jakarta', label: 'WIB — Jakarta, Surabaya (UTC+7)' },
      { value: 'Asia/Makassar', label: 'WITA — Makassar, Bali, Manado (UTC+8)' },
      { value: 'Asia/Jayapura', label: 'WIT — Jayapura, Papua (UTC+9)' },
    ],
  },
  {
    label: 'Southeast Asia',
    zones: [
      { value: 'Asia/Singapore', label: 'Singapore (UTC+8)' },
      { value: 'Asia/Kuala_Lumpur', label: 'Kuala Lumpur (UTC+8)' },
      { value: 'Asia/Bangkok', label: 'Bangkok, Hanoi (UTC+7)' },
      { value: 'Asia/Ho_Chi_Minh', label: 'Ho Chi Minh City (UTC+7)' },
      { value: 'Asia/Manila', label: 'Manila (UTC+8)' },
      { value: 'Asia/Yangon', label: 'Yangon (UTC+6:30)' },
    ],
  },
  {
    label: 'East Asia',
    zones: [
      { value: 'Asia/Tokyo', label: 'Tokyo (UTC+9)' },
      { value: 'Asia/Seoul', label: 'Seoul (UTC+9)' },
      { value: 'Asia/Shanghai', label: 'Shanghai, Beijing (UTC+8)' },
      { value: 'Asia/Hong_Kong', label: 'Hong Kong (UTC+8)' },
      { value: 'Asia/Taipei', label: 'Taipei (UTC+8)' },
    ],
  },
  {
    label: 'South & Central Asia',
    zones: [
      { value: 'Asia/Kolkata', label: 'India (UTC+5:30)' },
      { value: 'Asia/Dhaka', label: 'Dhaka (UTC+6)' },
      { value: 'Asia/Karachi', label: 'Karachi (UTC+5)' },
      { value: 'Asia/Tashkent', label: 'Tashkent (UTC+5)' },
    ],
  },
  {
    label: 'Middle East',
    zones: [
      { value: 'Asia/Dubai', label: 'Dubai, Abu Dhabi (UTC+4)' },
      { value: 'Asia/Riyadh', label: 'Riyadh, Kuwait (UTC+3)' },
      { value: 'Asia/Tehran', label: 'Tehran (UTC+3:30)' },
      { value: 'Asia/Jerusalem', label: 'Jerusalem (UTC+2/+3)' },
    ],
  },
  {
    label: 'Australia & Pacific',
    zones: [
      { value: 'Australia/Sydney', label: 'Sydney, Melbourne (UTC+10/+11)' },
      { value: 'Australia/Perth', label: 'Perth (UTC+8)' },
      { value: 'Australia/Brisbane', label: 'Brisbane (UTC+10)' },
      { value: 'Australia/Adelaide', label: 'Adelaide (UTC+9:30/+10:30)' },
      { value: 'Pacific/Auckland', label: 'Auckland (UTC+12/+13)' },
    ],
  },
  {
    label: 'Europe',
    zones: [
      { value: 'Europe/London', label: 'London (UTC+0/+1)' },
      { value: 'Europe/Paris', label: 'Paris, Berlin, Rome (UTC+1/+2)' },
      { value: 'Europe/Moscow', label: 'Moscow (UTC+3)' },
      { value: 'Europe/Istanbul', label: 'Istanbul (UTC+3)' },
      { value: 'Europe/Amsterdam', label: 'Amsterdam (UTC+1/+2)' },
    ],
  },
  {
    label: 'Africa',
    zones: [
      { value: 'Africa/Cairo', label: 'Cairo (UTC+2)' },
      { value: 'Africa/Lagos', label: 'Lagos (UTC+1)' },
      { value: 'Africa/Johannesburg', label: 'Johannesburg (UTC+2)' },
      { value: 'Africa/Nairobi', label: 'Nairobi (UTC+3)' },
    ],
  },
  {
    label: 'Americas',
    zones: [
      { value: 'America/New_York', label: 'New York, Miami (UTC-5/-4)' },
      { value: 'America/Chicago', label: 'Chicago, Houston (UTC-6/-5)' },
      { value: 'America/Denver', label: 'Denver (UTC-7/-6)' },
      { value: 'America/Los_Angeles', label: 'Los Angeles, Seattle (UTC-8/-7)' },
      { value: 'America/Sao_Paulo', label: 'Sao Paulo (UTC-3)' },
      { value: 'America/Mexico_City', label: 'Mexico City (UTC-6/-5)' },
      { value: 'America/Toronto', label: 'Toronto (UTC-5/-4)' },
      { value: 'America/Vancouver', label: 'Vancouver (UTC-8/-7)' },
    ],
  },
  {
    label: 'Other',
    zones: [
      { value: 'UTC', label: 'UTC — Coordinated Universal Time' },
    ],
  },
]

const searchQuery = ref('')
const selectedTimezone = ref('Asia/Jakarta')

const filteredGroups = computed(() => {
  const q = searchQuery.value.toLowerCase().trim()
  if (!q) return TIMEZONE_GROUPS

  return TIMEZONE_GROUPS
    .map((group) => ({
      ...group,
      zones: group.zones.filter(
        (tz) =>
          tz.label.toLowerCase().includes(q) ||
          tz.value.toLowerCase().includes(q) ||
          group.label.toLowerCase().includes(q)
      ),
    }))
    .filter((group) => group.zones.length > 0)
})

const selectedLabel = computed(() => {
  for (const group of TIMEZONE_GROUPS) {
    const found = group.zones.find((tz) => tz.value === selectedTimezone.value)
    if (found) return found.label
  }
  return selectedTimezone.value
})

function handleConfirm() {
  emit('confirm', selectedTimezone.value)
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-[9999] flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      :aria-label="title"
    >
      <!-- Backdrop -->
      <div class="fixed inset-0 bg-black/50 transition-opacity" @click="emit('cancel')" />

      <!-- Dialog -->
      <div class="relative z-10 w-full max-w-md mx-4 bg-white dark:bg-gray-800 rounded-lg shadow-xl flex flex-col max-h-[80vh]">
        <!-- Header -->
        <div class="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700 shrink-0">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ title }}</h2>
          <button
            class="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-500 dark:text-gray-400"
            @click="emit('cancel')"
          >
            <X class="w-5 h-5" />
          </button>
        </div>

        <!-- Body -->
        <div class="p-4 space-y-3 overflow-hidden flex flex-col min-h-0">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 shrink-0">
            Select timezone for date/time columns
          </label>

          <!-- Search input -->
          <div class="relative shrink-0">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Search city or timezone..."
              class="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white pl-9 pr-3 py-2 text-sm focus:ring-2 focus:ring-zinc-500 focus:border-zinc-500 outline-none"
            />
          </div>

          <!-- Timezone list -->
          <div class="overflow-y-auto border border-gray-200 dark:border-gray-600 rounded-md min-h-0 max-h-64">
            <template v-for="group in filteredGroups" :key="group.label">
              <div class="sticky top-0 px-3 py-1.5 text-xs font-semibold text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-700/50 uppercase tracking-wider">
                {{ group.label }}
              </div>
              <label
                v-for="tz in group.zones"
                :key="tz.value"
                class="flex items-center gap-3 px-3 py-2 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors"
                :class="selectedTimezone === tz.value ? 'bg-zinc-50 dark:bg-zinc-900/20' : ''"
              >
                <input
                  type="radio"
                  name="timezone"
                  :value="tz.value"
                  v-model="selectedTimezone"
                  class="text-zinc-600 focus:ring-zinc-500"
                />
                <span class="text-sm text-gray-900 dark:text-white">{{ tz.label }}</span>
              </label>
            </template>
            <div v-if="filteredGroups.length === 0" class="p-4 text-sm text-gray-500 dark:text-gray-400 text-center">
              No timezone found
            </div>
          </div>

          <p class="text-xs text-gray-500 dark:text-gray-400 shrink-0">
            Selected: <span class="font-medium text-gray-700 dark:text-gray-300">{{ selectedLabel }}</span>
          </p>
        </div>

        <!-- Footer -->
        <div class="flex gap-3 justify-end p-4 border-t border-gray-200 dark:border-gray-700 shrink-0">
          <Button variant="outline" :disabled="loading" @click="emit('cancel')">
            Cancel
          </Button>
          <Button :disabled="loading" :loading="loading" @click="handleConfirm">
            <Download class="w-4 h-4 mr-2" />
            {{ loading ? 'Exporting...' : 'Download' }}
          </Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
