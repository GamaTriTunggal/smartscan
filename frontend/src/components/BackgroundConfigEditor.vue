<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useImageUpload } from '@/composables/useImageUpload'
import { isTourActive, getTourNonce } from '@/composables/useTour'

const props = defineProps({
  modelValue: {
    type: Object,
    required: true
  },
  type: {
    type: String,
    default: 'landing', // Only 'landing' is supported by backend
    validator: (value) => value === 'landing'
  },
  disabled: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue'])

const { get } = useAPI()
const {
  isUploading,
  uploadProgress,
  uploadError,
  uploadBackground,
  validateFile,
  reset: resetUpload,
  MAX_FILE_SIZE,
  MIN_DIMENSION
} = useImageUpload()

// State
const presets = ref([])
const loadingPresets = ref(false)
const fileInput = ref(null)
const selectedFile = ref(null)
const previewUrl = ref(null)
const validationError = ref(null)

// Computed config with defaults
const config = computed(() => ({
  background_type: props.modelValue?.background_type || 'none',
  preset_id: props.modelValue?.preset_id || null,
  custom_background_url: props.modelValue?.custom_background_url || null,
  overlay_color: props.modelValue?.overlay_color || '#000000',
  overlay_opacity: props.modelValue?.overlay_opacity ?? 30,
  card_opacity: props.modelValue?.card_opacity ?? 90,
  card_blur: props.modelValue?.card_blur ?? 0
}))

// Get selected preset
const selectedPreset = computed(() => {
  if (config.value.background_type !== 'preset' || !config.value.preset_id) return null
  return presets.value.find(p => p.id === config.value.preset_id)
})

// Get current background URL for preview
const currentBackgroundUrl = computed(() => {
  if (config.value.background_type === 'preset' && selectedPreset.value) {
    return selectedPreset.value.background_url
  }
  if (config.value.background_type === 'custom') {
    return previewUrl.value || config.value.custom_background_url
  }
  return null
})

// Update helper
const updateConfig = (key, value) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

// Update multiple keys at once
const updateConfigMultiple = (updates) => {
  emit('update:modelValue', { ...props.modelValue, ...updates })
}

// Set background type
const setBackgroundType = (type) => {
  if (type === 'none') {
    updateConfigMultiple({
      background_type: 'none',
      preset_id: null,
      custom_background_url: null
    })
    previewUrl.value = null
  } else if (type === 'preset') {
    updateConfigMultiple({
      background_type: 'preset',
      custom_background_url: null
    })
    previewUrl.value = null
  } else if (type === 'custom') {
    updateConfigMultiple({
      background_type: 'custom',
      preset_id: null
    })
  }
}

// Select preset
const selectPreset = (preset) => {
  updateConfigMultiple({
    background_type: 'preset',
    preset_id: preset.id,
    custom_background_url: null,
    // Apply preset's default values
    overlay_color: preset.overlay_color || '#000000',
    overlay_opacity: preset.overlay_opacity ?? 30,
    card_opacity: preset.card_opacity ?? 90,
    card_blur: preset.card_blur ?? 0
  })
  previewUrl.value = null
}

// Trigger file input
const triggerFileInput = () => {
  fileInput.value?.click()
}

// Handle file selection
const handleFileSelect = async (event) => {
  const file = event.target.files?.[0]
  if (!file) return

  validationError.value = null
  selectedFile.value = file

  // Validate file
  const validation = await validateFile(file)
  if (!validation.valid) {
    validationError.value = validation.error
    selectedFile.value = null
    return
  }

  // Create preview URL
  previewUrl.value = URL.createObjectURL(file)

  // Upload the file
  const result = await uploadBackground(file, props.type)
  if (result.success) {
    updateConfigMultiple({
      background_type: 'custom',
      preset_id: null,
      custom_background_url: result.url
    })
  } else {
    validationError.value = result.error
    previewUrl.value = null
  }

  // Reset file input
  event.target.value = ''
}

// Remove custom background
const removeCustomBackground = () => {
  updateConfigMultiple({
    background_type: 'none',
    custom_background_url: null
  })
  previewUrl.value = null
  selectedFile.value = null
  resetUpload()
}

// Load presets
const loadPresets = async () => {
  loadingPresets.value = true
  try {
    const response = await get('/tenant/theme-presets', { type: props.type })
    if (response.success) {
      presets.value = response.data.theme_presets || []
    }
  } catch (err) {
    console.error('Failed to load theme presets:', err)
  } finally {
    loadingPresets.value = false
  }
}

// ── Tour support: select last preset when tour requests it ──
function handleTourSetValue(e) {
  if (!isTourActive()) return
  if (e.detail._nonce !== getTourNonce()) return
  const { field, value } = e.detail
  if (field === 'bg_preset_last' && value && presets.value.length > 0) {
    selectPreset(presets.value[presets.value.length - 1])
  }
}

// Load presets on mount
onMounted(() => {
  loadPresets()
  window.addEventListener('tour-set-value', handleTourSetValue)
})

onBeforeUnmount(() => {
  window.removeEventListener('tour-set-value', handleTourSetValue)
})

// Reload presets when type changes
watch(() => props.type, () => {
  loadPresets()
})
</script>

<template>
  <div class="background-config-editor space-y-6">
    <!-- Background Type Selection -->
    <div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <svg class="w-5 h-5 text-zinc-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        Background Image
      </h3>

      <!-- Type Selection Tabs -->
      <div class="flex gap-2 mb-4">
        <button
          type="button"
          :disabled="disabled"
          @click="setBackgroundType('none')"
          :class="[
            'px-4 py-2 text-sm font-medium rounded-lg transition-colors',
            config.background_type === 'none'
              ? 'bg-zinc-600 text-white'
              : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
          ]"
        >
          None
        </button>
        <button
          type="button"
          data-tour="bg-preset-tab"
          :disabled="disabled || presets.length === 0"
          @click="setBackgroundType('preset')"
          :class="[
            'px-4 py-2 text-sm font-medium rounded-lg transition-colors',
            config.background_type === 'preset'
              ? 'bg-zinc-600 text-white'
              : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600',
            (disabled || presets.length === 0) && 'opacity-50 cursor-not-allowed'
          ]"
        >
          Preset
        </button>
        <button
          type="button"
          :disabled="disabled"
          @click="setBackgroundType('custom')"
          :class="[
            'px-4 py-2 text-sm font-medium rounded-lg transition-colors',
            config.background_type === 'custom'
              ? 'bg-zinc-600 text-white'
              : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
          ]"
        >
          Custom Upload
        </button>
      </div>

      <!-- Preset Gallery -->
      <div v-if="config.background_type === 'preset'" class="mt-4">
        <div v-if="loadingPresets" class="flex items-center justify-center py-8">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-600"></div>
        </div>
        <div v-else-if="presets.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
          No presets available
        </div>
        <div v-else class="grid grid-cols-4 sm:grid-cols-6 gap-2">
          <button
            v-for="(preset, pIdx) in presets"
            :key="preset.id"
            type="button"
            :disabled="disabled"
            :data-tour="pIdx === presets.length - 1 ? 'bg-preset-last' : undefined"
            @click="selectPreset(preset)"
            :class="[
              'relative aspect-[9/16] rounded-lg overflow-hidden border-2 transition-all',
              config.preset_id === preset.id
                ? 'border-zinc-600 ring-2 ring-zinc-600 ring-offset-2'
                : 'border-gray-200 dark:border-gray-600 hover:border-zinc-400'
            ]"
          >
            <img
              :src="preset.thumbnail_url || preset.background_url"
              :alt="preset.name"
              class="w-full h-full object-cover"
            />
            <div class="absolute inset-0 bg-black/30 flex items-end p-2">
              <span class="text-xs text-white font-medium truncate">{{ preset.name }}</span>
            </div>
            <div v-if="config.preset_id === preset.id" class="absolute top-2 right-2">
              <svg class="w-5 h-5 text-white drop-shadow-lg" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
              </svg>
            </div>
          </button>
        </div>
      </div>

      <!-- Custom Upload -->
      <div v-if="config.background_type === 'custom'" class="mt-4">
        <input
          ref="fileInput"
          type="file"
          accept="image/jpeg,image/png,image/webp"
          class="hidden"
          @change="handleFileSelect"
        />

        <!-- Upload Area -->
        <div v-if="!config.custom_background_url && !previewUrl" class="space-y-3">
          <button
            type="button"
            :disabled="disabled || isUploading"
            @click="triggerFileInput"
            class="w-full py-8 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg hover:border-zinc-500 transition-colors flex flex-col items-center justify-center gap-2"
          >
            <svg class="w-10 h-10 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span class="text-sm text-gray-600 dark:text-gray-400">Click to upload background image</span>
            <span class="text-xs text-gray-500">Min {{ MIN_DIMENSION }}px (smallest side), Max {{ MAX_FILE_SIZE / (1024 * 1024) }}MB</span>
          </button>
        </div>

        <!-- Upload Progress -->
        <div v-if="isUploading" class="space-y-2">
          <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
            <div
              class="bg-zinc-600 h-2 rounded-full transition-all"
              :style="{ width: `${uploadProgress}%` }"
            ></div>
          </div>
          <p class="text-sm text-gray-600 dark:text-gray-400 text-center">Uploading... {{ uploadProgress }}%</p>
        </div>

        <!-- Preview with Remove Button -->
        <div v-if="(config.custom_background_url || previewUrl) && !isUploading" class="relative">
          <img
            :src="previewUrl || config.custom_background_url"
            alt="Background preview"
            class="w-full aspect-video object-cover rounded-lg"
          />
          <button
            type="button"
            :disabled="disabled"
            @click="removeCustomBackground"
            class="absolute top-2 right-2 p-1.5 bg-red-600 text-white rounded-full hover:bg-red-700 transition-colors"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Validation/Upload Error -->
        <p v-if="validationError || uploadError" class="mt-2 text-sm text-red-600 dark:text-red-400">
          {{ validationError || uploadError }}
        </p>
      </div>
    </div>

    <!-- Appearance Settings (only when background is set) -->
    <div
      v-if="config.background_type !== 'none'"
      class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4"
    >
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <svg class="w-5 h-5 text-zinc-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
        </svg>
        Appearance Settings
      </h3>

      <div class="space-y-4">
        <!-- Overlay Color -->
        <div>
          <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Overlay Color</label>
          <div class="flex items-center gap-2">
            <input
              type="color"
              :value="config.overlay_color"
              :disabled="disabled"
              @input="updateConfig('overlay_color', $event.target.value)"
              class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
            />
            <input
              type="text"
              :value="config.overlay_color"
              :disabled="disabled"
              @input="updateConfig('overlay_color', $event.target.value)"
              class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
            />
          </div>
        </div>

        <!-- Overlay Opacity -->
        <div data-tour="appearance-overlay-opacity">
          <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
            Overlay Opacity: {{ config.overlay_opacity }}%
          </label>
          <input
            type="range"
            min="0"
            max="100"
            :value="config.overlay_opacity"
            :disabled="disabled"
            @input="updateConfig('overlay_opacity', parseInt($event.target.value))"
            class="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer"
          />
        </div>

        <!-- Card Opacity -->
        <div data-tour="appearance-card-opacity">
          <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
            Card Opacity: {{ config.card_opacity }}%
          </label>
          <input
            type="range"
            min="50"
            max="100"
            :value="config.card_opacity"
            :disabled="disabled"
            @input="updateConfig('card_opacity', parseInt($event.target.value))"
            class="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer"
          />
        </div>

        <!-- Card Blur -->
        <div data-tour="appearance-card-blur">
          <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
            Card Blur: {{ config.card_blur }}px
          </label>
          <input
            type="range"
            min="0"
            max="20"
            :value="config.card_blur"
            :disabled="disabled"
            @input="updateConfig('card_blur', parseInt($event.target.value))"
            class="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer"
          />
        </div>
      </div>
    </div>

  </div>
</template>
