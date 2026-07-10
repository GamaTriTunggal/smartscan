<script setup>
import { ref, computed } from 'vue'
import { useAPI } from '@/composables/useAPI'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  disabled: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue'])

const { api } = useAPI()

// State
const fileInput = ref(null)
const isUploading = ref(false)
const uploadProgress = ref(0)
const uploadError = ref(null)
const previewUrl = ref(null)

// Logo constraints (smaller than background images)
const MAX_FILE_SIZE = 1 * 1024 * 1024 // 1MB for logos
const ALLOWED_TYPES = ['image/jpeg', 'image/png', 'image/webp']

// Trigger file input
const triggerFileInput = () => {
  fileInput.value?.click()
}

// Validate file
const validateFile = (file) => {
  if (!ALLOWED_TYPES.includes(file.type)) {
    return { valid: false, error: 'Invalid file type. Allowed: JPEG, PNG, WebP' }
  }
  if (file.size > MAX_FILE_SIZE) {
    return { valid: false, error: `File size exceeds ${MAX_FILE_SIZE / (1024 * 1024)}MB limit` }
  }
  return { valid: true }
}

// Handle file selection
const handleFileSelect = async (event) => {
  const file = event.target.files?.[0]
  if (!file) return

  uploadError.value = null

  // Validate file
  const validation = validateFile(file)
  if (!validation.valid) {
    uploadError.value = validation.error
    return
  }

  // Create preview URL
  previewUrl.value = URL.createObjectURL(file)
  isUploading.value = true
  uploadProgress.value = 0

  try {
    const formData = new FormData()
    formData.append('image', file)
    formData.append('type', 'landing') // Use landing upload type (same storage)

    const response = await api.post('/tenant/uploads/background', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (progressEvent) => {
        // total is undefined when the browser can't determine upload size
        if (progressEvent.total) {
          uploadProgress.value = Math.round((progressEvent.loaded * 100) / progressEvent.total)
        }
      }
    })

    if (response.data.success) {
      emit('update:modelValue', response.data.data.url)
    } else {
      throw new Error(response.data.message || 'Upload failed')
    }
  } catch (err) {
    uploadError.value = err.response?.data?.message || err.message || 'Upload failed'
    previewUrl.value = null
  } finally {
    isUploading.value = false
    event.target.value = ''
  }
}

// Remove logo
const removeLogo = () => {
  emit('update:modelValue', '')
  previewUrl.value = null
  uploadError.value = null
}

// Display URL (previewUrl during upload, or saved URL)
const displayUrl = computed(() => previewUrl.value || props.modelValue)
</script>

<template>
  <div class="logo-uploader">
    <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-2 flex items-center gap-2">
      <svg class="w-5 h-5 text-zinc-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
      Company Logo
    </h3>

    <input
      ref="fileInput"
      type="file"
      accept="image/jpeg,image/png,image/webp"
      class="hidden"
      @change="handleFileSelect"
    />

    <!-- No logo yet - Upload area -->
    <div v-if="!displayUrl && !isUploading">
      <button
        type="button"
        :disabled="disabled"
        @click="triggerFileInput"
        class="w-full py-4 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg hover:border-zinc-500 transition-colors flex flex-col items-center justify-center gap-1"
      >
        <svg class="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        <span class="text-sm text-gray-600 dark:text-gray-400">Click to upload company logo</span>
        <span class="text-xs text-gray-500">PNG with transparent background recommended. Max 1MB</span>
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
    <div v-if="displayUrl && !isUploading" class="relative inline-block">
      <div class="p-2 bg-gray-100 dark:bg-gray-800 rounded-lg inline-flex items-center gap-3">
        <img
          :src="displayUrl"
          alt="Logo preview"
          class="h-12 max-w-[200px] object-contain"
        />
        <button
          type="button"
          :disabled="disabled"
          @click="removeLogo"
          class="p-1.5 bg-red-600 text-white rounded-full hover:bg-red-700 transition-colors"
          title="Remove logo"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
      <p class="text-xs text-gray-500 mt-1">Logo will appear at the top of your landing page</p>
    </div>

    <!-- Upload Error -->
    <p v-if="uploadError" class="mt-2 text-sm text-red-600 dark:text-red-400">
      {{ uploadError }}
    </p>
  </div>
</template>
