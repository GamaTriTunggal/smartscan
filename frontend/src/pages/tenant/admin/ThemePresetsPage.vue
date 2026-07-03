<script setup>
import { ref, onMounted, computed } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useImageUpload } from '@/composables/useImageUpload'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'

const { get, post, put, del } = useAPI()
const {
  isUploading,
  uploadProgress,
  uploadError,
  uploadPresetBackground,
  validateFile,
  reset: resetUpload,
  MAX_FILE_SIZE,
  MIN_DIMENSION
} = useImageUpload()

// State
const presets = ref([])
const loading = ref(false)
const showModal = ref(false)
const editingPreset = ref(null)
const statusFilter = ref('active')
const typeFilter = ref('')

const presetTypes = [
  { value: 'landing', label: 'Landing Page' }
]
const form = ref({
  name: '',
  description: '',
  preset_type: 'landing',
  background_url: '',
  thumbnail_url: '',
  overlay_color: '#000000',
  overlay_opacity: 30,
  card_opacity: 90,
  card_blur: 0,
  display_order: 0,
  is_active: true
})

const errorMessage = ref('')
const fileInput = ref(null)
const previewUrl = ref(null)

// Filtered presets
const filteredPresets = computed(() => {
  let result = presets.value
  if (typeFilter.value) {
    result = result.filter(p => p.preset_type === typeFilter.value)
  }
  return result
})

async function fetchPresets() {
  loading.value = true
  try {
    let url = `/tenant/theme-presets/manage?status=${statusFilter.value}`
    if (typeFilter.value) {
      url += `&type=${typeFilter.value}`
    }
    const response = await get(url)
    if (response.success) {
      presets.value = response.data.theme_presets || []
    }
  } catch (error) {
    console.error('Failed to fetch theme presets:', error)
  } finally {
    loading.value = false
  }
}

function openCreateModal() {
  editingPreset.value = null
  form.value = {
    name: '',
    description: '',
    preset_type: 'landing',
    background_url: '',
    thumbnail_url: '',
    overlay_color: '#000000',
    overlay_opacity: 30,
    card_opacity: 90,
    card_blur: 0,
    display_order: 0,
    is_active: true
  }
  previewUrl.value = null
  errorMessage.value = ''
  resetUpload()
  showModal.value = true
}

function openEditModal(preset) {
  editingPreset.value = preset
  form.value = {
    name: preset.name,
    description: preset.description || '',
    preset_type: preset.preset_type,
    background_url: preset.background_url,
    thumbnail_url: preset.thumbnail_url || '',
    overlay_color: preset.overlay_color || '#000000',
    overlay_opacity: preset.overlay_opacity ?? 30,
    card_opacity: preset.card_opacity ?? 90,
    card_blur: preset.card_blur ?? 0,
    display_order: preset.display_order || 0,
    is_active: preset.is_active
  }
  previewUrl.value = preset.background_url
  errorMessage.value = ''
  resetUpload()
  showModal.value = true
}

function triggerFileInput() {
  fileInput.value?.click()
}

async function handleFileSelect(event) {
  const file = event.target.files?.[0]
  if (!file) return

  errorMessage.value = ''

  // Validate file
  const validation = await validateFile(file)
  if (!validation.valid) {
    errorMessage.value = validation.error
    return
  }

  // Create preview
  previewUrl.value = URL.createObjectURL(file)

  // Upload the file
  const result = await uploadPresetBackground(file)
  if (result.success) {
    form.value.background_url = result.url
  } else {
    errorMessage.value = result.error
    previewUrl.value = null
  }

  // Reset file input
  event.target.value = ''
}

async function savePreset() {
  errorMessage.value = ''

  if (!form.value.name) {
    errorMessage.value = 'Name is required'
    return
  }

  if (!form.value.background_url) {
    errorMessage.value = 'Background image is required'
    return
  }

  try {
    if (editingPreset.value) {
      const response = await put(`/tenant/theme-presets/${editingPreset.value.id}`, form.value)
      if (response.success) {
        showModal.value = false
        fetchPresets()
      } else {
        errorMessage.value = response.message || 'Failed to save theme preset'
      }
    } else {
      const response = await post('/tenant/theme-presets', form.value)
      if (response.success) {
        showModal.value = false
        fetchPresets()
      } else {
        errorMessage.value = response.message || 'Failed to save theme preset'
      }
    }
  } catch (error) {
    console.error('Failed to save theme preset:', error)
    errorMessage.value = error.response?.data?.message || 'Failed to save theme preset'
  }
}

async function deletePreset(preset) {
  if (!confirm(`Are you sure you want to delete "${preset.name}"?`)) return

  try {
    const response = await del(`/tenant/theme-presets/${preset.id}`)
    if (response.success) {
      fetchPresets()
    } else {
      alert(response.message || 'Failed to delete theme preset')
    }
  } catch (error) {
    console.error('Failed to delete theme preset:', error)
    alert(error.response?.data?.message || 'Failed to delete theme preset')
  }
}

async function restorePreset(preset) {
  if (!confirm(`Are you sure you want to restore "${preset.name}"?`)) return

  try {
    const response = await post(`/tenant/theme-presets/${preset.id}/restore`)
    if (response.success) {
      fetchPresets()
    } else {
      alert(response.message || 'Failed to restore theme preset')
    }
  } catch (error) {
    console.error('Failed to restore theme preset:', error)
    alert(error.response?.data?.message || 'Failed to restore theme preset')
  }
}

function getTypeLabel(type) {
  const t = presetTypes.find(p => p.value === type)
  return t ? t.label : type
}

onMounted(() => {
  fetchPresets()
})
</script>

<template>
  <div>
    <div class="mb-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Theme Presets</h1>
        <p class="text-gray-600 dark:text-gray-400 text-sm mt-1">
          Manage background theme presets for landing pages
        </p>
      </div>
      <Button @click="openCreateModal">
        Add Theme Preset
      </Button>
    </div>

    <!-- Filters -->
    <Card class="p-4 mb-6">
      <div class="flex flex-wrap gap-4">
        <div>
          <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Status</label>
          <select
            v-model="statusFilter"
            @change="fetchPresets"
            class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm"
          >
            <option value="active">Active</option>
            <option value="deleted">Deleted</option>
            <option value="all">All</option>
          </select>
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Type</label>
          <select
            v-model="typeFilter"
            @change="fetchPresets"
            class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm"
          >
            <option value="">All Types</option>
            <option v-for="type in presetTypes" :key="type.value" :value="type.value">
              {{ type.label }}
            </option>
          </select>
        </div>
      </div>
    </Card>

    <!-- Loading State -->
    <div v-if="loading" class="text-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-600 mx-auto"></div>
      <p class="text-gray-600 dark:text-gray-400 mt-4">Loading theme presets...</p>
    </div>

    <!-- Empty State -->
    <Card v-else-if="filteredPresets.length === 0" class="p-8 text-center">
      <div class="w-16 h-16 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
        <svg class="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
      </div>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">No Theme Presets</h3>
      <p class="text-gray-600 dark:text-gray-400 mb-4">Get started by creating your first theme preset.</p>
      <Button @click="openCreateModal">Add Theme Preset</Button>
    </Card>

    <!-- Presets Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <Card
        v-for="preset in filteredPresets"
        :key="preset.id"
        :class="[
          'overflow-hidden',
          preset.deleted_at && 'opacity-60 bg-gray-50 dark:bg-gray-800/50 border-dashed'
        ]"
      >
        <!-- Preview Image -->
        <div class="aspect-video relative bg-gray-100 dark:bg-gray-700">
          <img
            :src="preset.thumbnail_url || preset.background_url"
            :alt="preset.name"
            class="w-full h-full object-cover"
          />
          <!-- Overlay Preview -->
          <div
            class="absolute inset-0"
            :style="{
              backgroundColor: preset.overlay_color || '#000000',
              opacity: (preset.overlay_opacity ?? 30) / 100
            }"
          ></div>
          <!-- Type Badge -->
          <div class="absolute top-2 left-2">
            <span
              :class="[
                'text-xs px-2 py-1 rounded-full font-medium',
                preset.preset_type === 'landing'
                  ? 'bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400'
                  : 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400'
              ]"
            >
              {{ getTypeLabel(preset.preset_type) }}
            </span>
          </div>
          <!-- Status Badge -->
          <div v-if="preset.deleted_at" class="absolute top-2 right-2">
            <span class="text-xs px-2 py-1 rounded-full bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400">
              Deleted
            </span>
          </div>
          <div v-else-if="!preset.is_active" class="absolute top-2 right-2">
            <span class="text-xs px-2 py-1 rounded-full bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400">
              Inactive
            </span>
          </div>
        </div>

        <!-- Content -->
        <div class="p-4">
          <h3 class="font-semibold text-gray-900 dark:text-white mb-1">{{ preset.name }}</h3>
          <p v-if="preset.description" class="text-sm text-gray-600 dark:text-gray-400 line-clamp-2 mb-3">
            {{ preset.description }}
          </p>

          <!-- Settings Preview -->
          <div class="flex flex-wrap gap-2 text-xs text-gray-500 dark:text-gray-400 mb-4">
            <span>Overlay: {{ preset.overlay_opacity }}%</span>
            <span>Card: {{ preset.card_opacity }}%</span>
            <span>Blur: {{ preset.card_blur }}px</span>
          </div>

          <!-- Actions -->
          <div class="flex gap-2">
            <Button
              v-if="!preset.deleted_at"
              size="sm"
              variant="outline"
              @click="openEditModal(preset)"
            >
              Edit
            </Button>
            <Button
              v-if="!preset.deleted_at"
              size="sm"
              variant="outline"
              class="text-red-600 border-red-200 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-900/20"
              @click="deletePreset(preset)"
            >
              Delete
            </Button>
            <Button
              v-if="preset.deleted_at"
              size="sm"
              variant="outline"
              class="text-green-600 border-green-200 hover:bg-green-50 dark:border-green-800 dark:hover:bg-green-900/20"
              @click="restorePreset(preset)"
            >
              Restore
            </Button>
          </div>
        </div>
      </Card>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <Card class="w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <div class="p-6">
          <div class="flex items-center justify-between mb-6">
            <h2 class="text-xl font-bold text-gray-900 dark:text-white">
              {{ editingPreset ? 'Edit Theme Preset' : 'Add Theme Preset' }}
            </h2>
            <button
              @click="showModal = false"
              class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
            >
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div class="space-y-4">
            <!-- Name -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name *</label>
              <Input v-model="form.name" placeholder="e.g., Modern Dark" />
            </div>

            <!-- Description -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
              <textarea
                v-model="form.description"
                rows="2"
                placeholder="Brief description of this theme preset"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              ></textarea>
            </div>

            <!-- Type -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Type *</label>
              <select
                v-model="form.preset_type"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              >
                <option v-for="type in presetTypes" :key="type.value" :value="type.value">
                  {{ type.label }}
                </option>
              </select>
            </div>

            <!-- Background Image Upload -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Background Image *</label>
              <input
                ref="fileInput"
                type="file"
                accept="image/jpeg,image/png,image/webp"
                class="hidden"
                @change="handleFileSelect"
              />

              <div v-if="!form.background_url && !previewUrl" class="space-y-2">
                <button
                  type="button"
                  :disabled="isUploading"
                  @click="triggerFileInput"
                  class="w-full py-8 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg hover:border-zinc-500 transition-colors"
                >
                  <svg class="w-10 h-10 mx-auto text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                  <span class="text-sm text-gray-600 dark:text-gray-400">Click to upload background image</span>
                  <span class="block text-xs text-gray-500 mt-1">Min {{ MIN_DIMENSION }}px (smallest side), Max {{ MAX_FILE_SIZE / (1024 * 1024) }}MB</span>
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

              <!-- Preview -->
              <div v-if="(form.background_url || previewUrl) && !isUploading" class="relative">
                <img
                  :src="previewUrl || form.background_url"
                  alt="Background preview"
                  class="w-full aspect-video object-cover rounded-lg"
                />
                <button
                  type="button"
                  @click="form.background_url = ''; previewUrl = null"
                  class="absolute top-2 right-2 p-1.5 bg-red-600 text-white rounded-full hover:bg-red-700 transition-colors"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>

            <!-- Appearance Settings -->
            <div class="border-t border-gray-200 dark:border-gray-600 pt-4">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-4">Appearance Settings</h3>

              <div class="grid grid-cols-2 gap-4">
                <!-- Overlay Color -->
                <div>
                  <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Overlay Color</label>
                  <div class="flex items-center gap-2">
                    <input
                      type="color"
                      v-model="form.overlay_color"
                      class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                    />
                    <input
                      type="text"
                      v-model="form.overlay_color"
                      class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                    />
                  </div>
                </div>

                <!-- Display Order -->
                <div>
                  <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Display Order</label>
                  <input
                    type="number"
                    v-model.number="form.display_order"
                    min="0"
                    class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  />
                </div>

                <!-- Overlay Opacity -->
                <div class="col-span-2">
                  <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Overlay Opacity: {{ form.overlay_opacity }}%
                  </label>
                  <input
                    type="range"
                    min="0"
                    max="100"
                    v-model.number="form.overlay_opacity"
                    class="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer"
                  />
                </div>

                <!-- Card Opacity -->
                <div class="col-span-2">
                  <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Card Opacity: {{ form.card_opacity }}%
                  </label>
                  <input
                    type="range"
                    min="50"
                    max="100"
                    v-model.number="form.card_opacity"
                    class="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer"
                  />
                </div>

                <!-- Card Blur -->
                <div class="col-span-2">
                  <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Card Blur: {{ form.card_blur }}px
                  </label>
                  <input
                    type="range"
                    min="0"
                    max="20"
                    v-model.number="form.card_blur"
                    class="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer"
                  />
                </div>
              </div>
            </div>

            <!-- Active Status -->
            <div class="flex items-center gap-2">
              <input
                type="checkbox"
                id="is_active"
                v-model="form.is_active"
                class="w-4 h-4 text-zinc-600 rounded focus:ring-[#27272a]"
              />
              <label for="is_active" class="text-sm text-gray-700 dark:text-gray-300">Active</label>
            </div>

            <!-- Error Message -->
            <div v-if="errorMessage || uploadError" class="text-sm text-red-600 dark:text-red-400">
              {{ errorMessage || uploadError }}
            </div>

            <!-- Actions -->
            <div class="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-600">
              <Button variant="outline" @click="showModal = false">Cancel</Button>
              <Button @click="savePreset" :disabled="isUploading">
                {{ editingPreset ? 'Save Changes' : 'Create Preset' }}
              </Button>
            </div>
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>
