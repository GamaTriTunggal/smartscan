<script setup>
import { ref, toRef, watch, onMounted, onUnmounted } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useDateTime } from '@/composables/useDateTime'
import { useEscapeKey } from '@/composables/useEscapeKey'
import { useBatchGeofence } from '@/composables/useBatchGeofence'
import { useQRGenerationStore } from '@/stores/qrGeneration'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import GeofenceMapPicker from '@/components/GeofenceMapPicker.vue'
import { isTourActive, getTourNonce } from '@/composables/useTour.js'

const props = defineProps({
  productId: { type: [String, Number], default: null },
  productName: { type: String, default: '' },
  // Whether the product has warranty enabled (for the info banner).
  warrantyEnabled: { type: Boolean, default: false },
  open: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'created'])

const { get, post } = useAPI()
const { toUTCString } = useDateTime()
const qrGenerationStore = useQRGenerationStore()

// Dynamic year for placeholders
const currentYear = new Date().getFullYear()

const creatingBatch = ref(false)
const batchError = ref('')

// Geofence state (shared composable)
const {
  geofenceData, zoneTemplates,
  getDefaultGeofenceFields,
  onGeofenceUpdate: _onGeofenceUpdate,
  loadZoneTemplate: _loadZoneTemplate,
  fetchZoneTemplates, resetGeofence, buildGeofencePayload,
} = useBatchGeofence(get)

const newBatch = ref({
  batch_name: '',
  qr_count: 100,
  prefix: '',
  suffix: '',
  production_date: '',
  expiry_date: '',
  ...getDefaultGeofenceFields(),
})

function onGeofenceUpdate(val) {
  _onGeofenceUpdate(val, newBatch.value)
}

function loadZoneTemplate(template) {
  _loadZoneTemplate(template, newBatch.value)
}

const resetForm = () => {
  newBatch.value = {
    batch_name: '',
    qr_count: 100,
    prefix: '',
    suffix: '',
    production_date: '',
    expiry_date: '',
  }
  resetGeofence(newBatch.value)
  batchError.value = ''
}

// Reset form + load zone templates whenever the modal opens.
watch(() => props.open, (isOpen) => {
  if (isOpen) {
    resetForm()
    fetchZoneTemplates()
  }
})

const createBatch = async () => {
  if (!props.productId || !newBatch.value.batch_name) return

  try {
    creatingBatch.value = true
    batchError.value = ''

    const payload = {
      product_id: props.productId,
      ...newBatch.value,
      production_date: toUTCString(newBatch.value.production_date),
      expiry_date: toUTCString(newBatch.value.expiry_date),
      ...buildGeofencePayload(),
    }

    const response = await post('/tenant/qr-batches', payload)
    if (response.success) {
      // Start tracking the new batch for progress polling + toast notifications
      if (response.data && response.data.id) {
        qrGenerationStore.trackNewBatch(response.data)
      }
      emit('created', response.data)
      emit('close')
    } else {
      batchError.value = response.message || 'Failed to create batch'
    }
  } catch (error) {
    console.error('Failed to create batch:', error)
    batchError.value = error.response?.data?.message || 'Failed to create batch. Please try again.'
  } finally {
    creatingBatch.value = false
  }
}

// Tour auto-fill listener — sets reactive values directly (no DOM manipulation)
function handleTourSetValue(e) {
  if (!isTourActive()) return
  if (e.detail._nonce !== getTourNonce()) return
  const { field, value } = e.detail
  switch (field) {
    case 'batch_name':
      newBatch.value.batch_name = value
      break
    case 'production_date':
      newBatch.value.production_date = value
      break
    case 'geofence_enabled':
      newBatch.value.geofence_enabled = value
      break
  }
}
onMounted(() => window.addEventListener('tour-set-value', handleTourSetValue))
onUnmounted(() => window.removeEventListener('tour-set-value', handleTourSetValue))

// Close modal on Escape key
useEscapeKey(() => emit('close'), toRef(props, 'open'))
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <div class="fixed inset-0 bg-black/50" @click="emit('close')"></div>
    <div class="relative z-10 w-full max-w-lg max-h-[90vh] bg-white dark:bg-gray-800 rounded-lg shadow-xl flex flex-col">
      <div class="p-6 border-b border-gray-200 dark:border-gray-700">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white">Create New Batch</h2>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
          For: {{ productName }}
        </p>
      </div>
      <div class="p-6 space-y-4 overflow-y-auto flex-1">
        <!-- Error Alert -->
        <div v-if="batchError" class="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <p class="text-sm text-red-700 dark:text-red-300">{{ batchError }}</p>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Batch Name *</label>
          <Input v-model="newBatch.batch_name" :placeholder="`e.g., January ${currentYear} Production`" data-tour="batch-name-input" />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Number of QR Codes *</label>
          <Input v-model.number="newBatch.qr_count" type="number" min="1" max="5000000" />
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">Max 5,000,000 per batch. Large batches run in the background — you can navigate away while they generate.</p>
        </div>

        

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Prefix</label>
            <Input v-model="newBatch.prefix" placeholder="e.g., PROD-" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Suffix</label>
            <Input v-model="newBatch.suffix" :placeholder="`e.g., -${currentYear}`" />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Production Date</label>
            <Input v-model="newBatch.production_date" type="date" data-tour="batch-production-date" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Expiry Date</label>
            <Input v-model="newBatch.expiry_date" type="date" />
          </div>
        </div>

        <!-- Warranty status from product (info display) -->
        <div v-if="warrantyEnabled" class="p-3 bg-zinc-50 dark:bg-zinc-900/20 border border-zinc-200 dark:border-zinc-800 rounded-lg">
          <div class="flex items-center gap-2">
            <svg class="w-4 h-4 text-zinc-600" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
            </svg>
            <span class="text-sm text-zinc-700 dark:text-zinc-300">Warranty Registration enabled (from product settings)</span>
          </div>
        </div>
        <div v-else class="p-3 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
          <p class="text-sm text-gray-500 dark:text-gray-400">
            Warranty is disabled for this product. Enable it in product settings.
          </p>
        </div>

        <!-- Geofence Distribution Zone -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-4 mt-4">
          <div class="flex items-center justify-between mb-3">
            <label class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              <svg class="w-4 h-4 text-zinc-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              Distribution Zone (Geofence)
            </label>
            <label class="relative inline-flex items-center cursor-pointer" data-tour="geofence-toggle">
              <input
                v-model="newBatch.geofence_enabled"
                type="checkbox"
                class="sr-only peer"
              />
              <div class="w-9 h-5 bg-gray-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-zinc-300 dark:peer-focus:ring-zinc-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all dark:after:border-gray-500 peer-checked:bg-zinc-600"></div>
            </label>
          </div>

          <div v-if="newBatch.geofence_enabled" class="space-y-3">
            <div class="bg-zinc-50 dark:bg-zinc-900/20 border border-zinc-200 dark:border-zinc-800 rounded-lg p-3">
              <p class="text-xs text-zinc-800 dark:text-zinc-300">
                Set a distribution zone for this batch. Scans outside this zone will be recorded as geofence violations for grey market detection.
              </p>
            </div>

            <!-- Zone Template Selector -->
            <div v-if="zoneTemplates.length > 0" class="flex items-center gap-2" data-tour="geofence-zone-template">
              <label class="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">Load template:</label>
              <select
                class="flex-1 px-2 py-1 text-sm border rounded-md bg-white dark:bg-gray-900 dark:border-gray-700"
                @change="(e) => { if (e.target.value) loadZoneTemplate(zoneTemplates.find(t => t.id === e.target.value)); e.target.value = '' }"
              >
                <option value="">Select a saved zone...</option>
                <option v-for="t in zoneTemplates" :key="t.id" :value="t.id">
                  {{ t.template_name }} ({{ t.radius_km }}km)
                </option>
              </select>
            </div>

            <GeofenceMapPicker
              :model-value="geofenceData"
              @update:model-value="onGeofenceUpdate"
            />
          </div>
        </div>
      </div>
      <div class="p-6 border-t border-gray-200 dark:border-gray-700 flex gap-3">
        <Button variant="outline" class="flex-1" @click="emit('close')">Cancel</Button>
        <Button class="flex-1" :disabled="creatingBatch || !newBatch.batch_name" @click="createBatch" data-tour="create-batch-btn">
          {{ creatingBatch ? 'Creating...' : 'Create Batch' }}
        </Button>
      </div>
    </div>
  </div>
</template>
