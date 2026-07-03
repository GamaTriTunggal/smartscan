<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useAPI } from '@/composables/useAPI'
import Button from '@/components/ui/Button.vue'
import { Download } from 'lucide-vue-next'

const props = defineProps({
  batchId: { type: [String, Number], required: true },
  qrCount: { type: Number, default: 0 },
  batchCode: { type: String, default: '' },
  // Match the size used by the sibling CSV/Excel buttons.
  size: { type: String, default: 'default' },
})

const { getAuthHeaders } = useAPI()

const PDF_MAX = 10000

const open = ref(false)
const rootEl = ref(null)
const downloading = ref(false)
const error = ref('')

const label = ref('25')
const start = ref(1)
// Default to the whole batch when it fits in a single PDF; otherwise leave the
// upper bound at the cap so the user picks a range explicitly.
const end = ref(props.qrCount > 0 ? Math.min(props.qrCount, PDF_MAX) : PDF_MAX)

const requiresRange = computed(() => props.qrCount > PDF_MAX)

const rangeCount = computed(() => {
  const s = Number(start.value)
  const e = Number(end.value)
  if (!s || !e || e < s) return 0
  return e - s + 1
})

const rangeError = computed(() => {
  const s = Number(start.value)
  const e = Number(end.value)
  if (!s || s < 1) return 'Start must be 1 or greater.'
  if (!e || e < s) return 'End must be greater than or equal to start.'
  if (rangeCount.value > PDF_MAX) return `A single PDF is limited to ${PDF_MAX.toLocaleString()} codes. Narrow the range.`
  return ''
})

const toggle = () => {
  open.value = !open.value
  if (open.value) error.value = ''
}

const onClickOutside = (e) => {
  if (open.value && rootEl.value && !rootEl.value.contains(e.target)) {
    open.value = false
  }
}
onMounted(() => document.addEventListener('click', onClickOutside))
onUnmounted(() => document.removeEventListener('click', onClickOutside))

const download = async () => {
  if (rangeError.value) {
    error.value = rangeError.value
    return
  }
  error.value = ''
  downloading.value = true
  try {
    const params = new URLSearchParams({
      label: label.value,
      start: String(start.value),
      end: String(end.value),
    })
    const endpoint = `/tenant/qr-batches/${props.batchId}/export/pdf?${params.toString()}`
    const response = await fetch(`${import.meta.env.VITE_API_URL}${endpoint}`, {
      method: 'GET',
      headers: getAuthHeaders(),
      credentials: 'include',
    })

    if (!response.ok) {
      let message = 'Failed to generate PDF. Please try again.'
      try {
        const body = await response.json()
        if (body?.message) message = body.message
      } catch { /* non-JSON error body */ }
      error.value = message
      return
    }

    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${props.batchCode || props.batchId}_labels_${label.value}mm.pdf`
    a.click()
    URL.revokeObjectURL(url)
    open.value = false
  } catch (e) {
    console.error('Failed to export PDF:', e)
    error.value = 'Failed to generate PDF. Please try again.'
  } finally {
    downloading.value = false
  }
}
</script>

<template>
  <div ref="rootEl" class="relative inline-block">
    <Button variant="outline" :size="size" @click="toggle">
      <Download class="w-4 h-4 mr-2" />
      Download PDF
    </Button>

    <div
      v-if="open"
      class="absolute right-0 z-30 mt-2 w-72 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-xl p-4 space-y-3"
    >
      <div>
        <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Label size</label>
        <select
          v-model="label"
          class="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
        >
          <option value="25">25 mm</option>
          <option value="38">38 mm</option>
          <option value="50">50 mm</option>
        </select>
      </div>

      <div class="grid grid-cols-2 gap-2">
        <div>
          <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">From (#)</label>
          <input
            v-model.number="start"
            type="number"
            min="1"
            class="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
          />
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">To (#)</label>
          <input
            v-model.number="end"
            type="number"
            min="1"
            class="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
          />
        </div>
      </div>

      <p v-if="rangeCount > 0 && !rangeError" class="text-xs text-gray-500 dark:text-gray-400">
        {{ rangeCount.toLocaleString() }} label{{ rangeCount === 1 ? '' : 's' }} in this PDF.
      </p>

      <p v-if="requiresRange" class="text-xs text-amber-600 dark:text-amber-400">
        This batch has {{ qrCount.toLocaleString() }} codes. A PDF is limited to {{ PDF_MAX.toLocaleString() }} codes at a time —
        pick a range of {{ PDF_MAX.toLocaleString() }} or fewer, or use CSV for print vendors.
      </p>

      <p v-if="error || rangeError" class="text-xs text-red-600 dark:text-red-400">
        {{ error || rangeError }}
      </p>

      <Button class="w-full" :disabled="downloading || !!rangeError" @click="download">
        {{ downloading ? 'Generating...' : 'Download' }}
      </Button>
    </div>
  </div>
</template>
