<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import { ASPECT_RATIOS, DEFAULT_ASPECT_RATIO } from '@/constants/previewOptions'
import BackgroundConfigEditor from '@/components/BackgroundConfigEditor.vue'
import { VueDraggable } from 'vue-draggable-plus'
import { Image, Video, Share2, Award, ExternalLink, FileText, Shield, GripVertical, Upload, Trash2 } from 'lucide-vue-next'
import { useAPI } from '@/composables/useAPI'
import { SOCIAL_ICON_PATHS } from '@/lib/socialIcons'

// Section metadata for drag-and-drop ordering
const SECTION_METADATA = {
  images: { label: 'Gallery', icon: Image },
  videos: { label: 'Videos', icon: Video },
  social_accounts: { label: 'Social Media', icon: Share2 },
  certifications: { label: 'Certifications', icon: Award },
  website_link: { label: 'Website Button', icon: ExternalLink },
  description: { label: 'Product Description', icon: FileText },
  warranty_button: { label: 'Warranty Button', icon: Shield }
}

const DEFAULT_SECTION_ORDER = [
  'images', 'description', 'certifications', 'social_accounts',
  'videos', 'website_link', 'warranty_button'
]

const selectedRatio = ref(DEFAULT_ASPECT_RATIO)

// Dynamic year for preview examples
const currentYear = new Date().getFullYear()

onMounted(() => {
  initSectionOrder()
})

const props = defineProps({
  modelValue: { type: Object, required: true },
  backgroundConfig: { type: Object, default: null },
  presets: { type: Array, default: () => [] },
  templateId: { type: String, default: null },
})

const emit = defineEmits(['update:modelValue', 'update:backgroundConfig'])

const { post, del } = useAPI()

// Logo upload URL resolution
const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
const UPLOAD_BASE = API_BASE.replace('/api/v1', '')

function getLogoUrl(url) {
  if (!url) return ''
  if (url.startsWith('http://') || url.startsWith('https://')) return url
  if (url.startsWith('/uploads/')) return UPLOAD_BASE + url
  return url
}

// Logo upload state
const logoUploading = ref(false)

async function handleLogoUpload(event) {
  const file = event.target.files?.[0]
  if (!file) return

  const allowedTypes = ['image/jpeg', 'image/png', 'image/webp']
  if (!allowedTypes.includes(file.type)) {
    alert('Only JPEG, PNG, and WebP images are allowed')
    return
  }
  if (file.size > 2 * 1024 * 1024) {
    alert('Logo must be less than 2MB')
    return
  }

  if (!props.templateId) {
    alert('Please save the template first before uploading a logo')
    return
  }

  logoUploading.value = true
  try {
    const formData = new FormData()
    formData.append('logo', file)

    const response = await post(`/tenant/templates/${props.templateId}/logo`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })

    if (response.success) {
      updateHeader('logo_url', response.data.logo_url)
    } else {
      alert(response.message || 'Failed to upload logo')
    }
  } catch (err) {
    console.error('Failed to upload logo:', err)
    alert('Failed to upload logo')
  } finally {
    logoUploading.value = false
    event.target.value = ''
  }
}

async function handleLogoDelete() {
  if (!props.templateId) return

  logoUploading.value = true
  try {
    const response = await del(`/tenant/templates/${props.templateId}/logo`)
    if (response.success) {
      updateHeader('logo_url', '')
    } else {
      alert(response.message || 'Failed to delete logo')
    }
  } catch (err) {
    console.error('Failed to delete logo:', err)
    alert('Failed to delete logo')
  } finally {
    logoUploading.value = false
  }
}

const config = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// Update button config
const updateWarrantyButton = (key, value) => {
  const newConfig = { ...config.value }
  newConfig.warranty_button = { ...newConfig.warranty_button, [key]: value }
  emit('update:modelValue', newConfig)
}

// Update styling
const updateStyling = (key, value) => {
  const newConfig = { ...config.value }
  newConfig.styling = { ...newConfig.styling, [key]: value }
  emit('update:modelValue', newConfig)
}

// Update header
const updateHeader = (key, value) => {
  const newConfig = { ...config.value }
  newConfig.header = { ...newConfig.header, [key]: value }
  emit('update:modelValue', newConfig)
}

// Update certifications section
const updateCertificationsSection = (key, value) => {
  const newConfig = { ...config.value }
  newConfig.certifications_section = { ...newConfig.certifications_section, [key]: value }
  emit('update:modelValue', newConfig)
}

// Update social media section
const updateSocialMediaSection = (key, value) => {
  const newConfig = { ...config.value }
  newConfig.social_media_section = { ...newConfig.social_media_section, [key]: value }
  emit('update:modelValue', newConfig)
}

// Section order for drag-and-drop reordering
const sectionOrderList = ref([])

// Initialize section order from config
const initSectionOrder = () => {
  const savedOrder = config.value.section_order

  // Handle null, undefined, or empty array - use defaults
  if (!savedOrder || !Array.isArray(savedOrder) || savedOrder.length === 0) {
    sectionOrderList.value = DEFAULT_SECTION_ORDER.map(id => ({ id, ...SECTION_METADATA[id] }))
    return
  }

  // Filter valid sections
  const validOrder = savedOrder.filter(id => SECTION_METADATA[id])

  // Warn about invalid sections that were removed
  const invalidSections = savedOrder.filter(id => !SECTION_METADATA[id])
  if (invalidSections.length > 0) {
    console.warn('[ValidationTemplateEditor] Invalid section IDs removed:', invalidSections)
  }

  // Add any missing sections (for backward compatibility when new sections are added)
  const missingNew = DEFAULT_SECTION_ORDER.filter(id => !validOrder.includes(id))
  const fullOrder = [...validOrder, ...missingNew]
  sectionOrderList.value = fullOrder.map(id => ({ id, ...SECTION_METADATA[id] }))
}

// Emit section order changes when drag ends (with validation and debouncing)
const emitSectionOrder = () => {
  // Extract IDs and filter out any invalid/null values
  const newOrder = sectionOrderList.value
    .map(item => item?.id)
    .filter(id => id && SECTION_METADATA[id])

  // Remove duplicates (should not happen, but defensive)
  const uniqueOrder = [...new Set(newOrder)]

  // Log warning if duplicates were found (helps debug drag-drop issues)
  if (uniqueOrder.length !== newOrder.length) {
    console.warn('[ValidationTemplateEditor] Duplicate section IDs detected and removed:',
      newOrder.filter((id, idx) => newOrder.indexOf(id) !== idx))
  }

  // Don't save empty order
  if (uniqueOrder.length === 0) {
    console.warn('[ValidationTemplateEditor] Empty section order, not saving')
    return
  }

  const newConfig = { ...config.value, section_order: uniqueOrder }
  emit('update:modelValue', newConfig)
}

// Debounce to prevent race conditions during rapid drags
const updateSectionOrder = useDebounceFn(emitSectionOrder, 300)

// Helper function for array comparison (more efficient than JSON.stringify)
const arraysEqual = (a, b) => {
  if (!a || !b) return false
  if (a.length !== b.length) return false
  return a.every((val, idx) => val === b[idx])
}

// Watch for external config changes (e.g., loading saved template)
watch(() => props.modelValue?.section_order, (newOrder) => {
  if (!newOrder) return
  const currentOrder = sectionOrderList.value.map(s => s.id)
  if (!arraysEqual(newOrder, currentOrder)) {
    initSectionOrder()
  }
}, { deep: true })

// Preview states for collapsible sections
const previewCertExpanded = ref(true)

// Sample certifications for preview
const sampleCertifications = [
  { name: 'BPOM', logo_url: '/logos/certs/bpom.png', country: 'Indonesia', registration_number: 'ABC1234567890' },
  { name: 'Halal MUI', logo_url: '/logos/certs/halal-mui.png', country: 'Indonesia', registration_number: 'ABC1234567890' },
  { name: 'SNI', logo_url: '/logos/certs/sni.png', country: 'Indonesia', registration_number: 'ABC1234567890' },
  { name: 'ISO 9001', logo_url: '/logos/certs/iso-9001.png', country: 'International', registration_number: 'ABC1234567890' },
]

// Sample social media for preview (SVG paths matching ValidatePage iconMap)
const sampleSocialMedia = [
  { platform: 'Instagram', svgPath: SOCIAL_ICON_PATHS.INSTAGRAM },
  { platform: 'Facebook', svgPath: SOCIAL_ICON_PATHS.FACEBOOK },
  { platform: 'WhatsApp', svgPath: SOCIAL_ICON_PATHS.WHATSAPP },
]

// Background config update handler
const handleBackgroundConfigUpdate = (newConfig) => {
  emit('update:backgroundConfig', newConfig)
}

// Background config computed properties
const hasBackground = computed(() => {
  if (!props.backgroundConfig) return false
  return props.backgroundConfig.background_type !== 'none'
})

const backgroundUrl = computed(() => {
  if (!props.backgroundConfig) return null
  const bg = props.backgroundConfig

  if (bg.background_type === 'preset' && bg.preset_id && props.presets) {
    const preset = props.presets.find(p => p.id === bg.preset_id)
    return preset?.background_url || bg.background_url || null
  }
  if (bg.background_type === 'custom') {
    return bg.custom_background_url
  }
  return bg.background_url || null
})

const backgroundStyle = computed(() => {
  if (!hasBackground.value || !backgroundUrl.value) return {}
  return {
    backgroundImage: `url(${backgroundUrl.value})`,
    backgroundSize: 'cover',
    backgroundPosition: 'center',
    backgroundRepeat: 'no-repeat'
  }
})

const overlayStyle = computed(() => {
  if (!hasBackground.value || !props.backgroundConfig) return {}
  return {
    backgroundColor: props.backgroundConfig.overlay_color || '#000000',
    opacity: (props.backgroundConfig.overlay_opacity ?? 30) / 100
  }
})

// Glass card style for the main container (wraps header + content)
const glassCardStyle = computed(() => {
  if (!hasBackground.value || !props.backgroundConfig) return {}
  const cardOpacity = (props.backgroundConfig.card_opacity ?? 90) / 100
  const cardBlur = props.backgroundConfig.card_blur ?? 0
  return {
    backgroundColor: `rgba(255, 255, 255, ${cardOpacity})`,
    backdropFilter: `blur(${cardBlur}px)`,
    WebkitBackdropFilter: `blur(${cardBlur}px)`,
    border: '1px solid rgba(255, 255, 255, 0.3)'
  }
})

// Header style - semi-transparent tint of the header color
const headerGlassStyle = computed(() => {
  const baseColor = config.value.header?.bg_color || '#3f3f46'
  if (!hasBackground.value) {
    return { backgroundColor: baseColor }
  }
  // Convert hex to rgba with 80% opacity for glass effect
  const hex = baseColor.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  return {
    backgroundColor: `rgba(${r}, ${g}, ${b}, 0.8)`
  }
})

// Field card style - subtle glass effect for inner cards
const fieldCardStyle = computed(() => {
  if (!hasBackground.value) {
    return {
      backgroundColor: config.value.styling?.field_bg_color || '#ffffff',
      boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)'
    }
  }
  // Glass mode - subtle white overlay
  return {
    backgroundColor: 'rgba(255, 255, 255, 0.15)',
    border: '1px solid rgba(255, 255, 255, 0.2)',
    boxShadow: 'none'
  }
})

// Text color for glass mode (ensure readability)
const textColorStyle = computed(() => {
  if (!hasBackground.value) {
    return { color: config.value.styling?.text_color || '#1f2937' }
  }
  // In glass mode, use darker text for contrast
  return { color: 'rgba(0, 0, 0, 0.85)' }
})

const labelColorStyle = computed(() => {
  if (!hasBackground.value) {
    return { color: '#6b7280' } // gray-500
  }
  return { color: 'rgba(0, 0, 0, 0.6)' }
})
</script>

<template>
  <div class="validation-template-editor">
    <div class="grid lg:grid-cols-2 gap-6">
      <!-- Configuration Panel -->
      <div class="space-y-6">
        <!-- Header Section -->
        <div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
            <svg class="w-5 h-5 text-zinc-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6z" />
            </svg>
            Header Configuration
          </h3>

          <div class="space-y-4">
            <!-- Background Color -->
            <div data-tour="header-bg-color">
              <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Background Color</label>
              <div class="flex items-center gap-2">
                <input
                  type="color"
                  :value="config.header?.bg_color || '#3f3f46'"
                  @input="updateHeader('bg_color', $event.target.value)"
                  class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                />
                <input
                  type="text"
                  :value="config.header?.bg_color || '#3f3f46'"
                  @input="updateHeader('bg_color', $event.target.value)"
                  class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                />
              </div>
            </div>

            <!-- Logo Toggle -->
            <div data-tour="company-logo-section" class="flex items-center justify-between py-2 border-t border-gray-200 dark:border-gray-700">
              <div>
                <span class="block text-sm font-medium text-gray-900 dark:text-white">Company Logo</span>
                <span class="block text-xs text-gray-500 dark:text-gray-400">Display logo at top of header</span>
              </div>
              <label class="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  :checked="config.header?.logo_enabled"
                  @change="updateHeader('logo_enabled', !config.header?.logo_enabled)"
                  class="sr-only peer"
                />
                <div class="w-9 h-5 bg-gray-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-zinc-300 dark:peer-focus:ring-zinc-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all dark:border-gray-500 peer-checked:bg-zinc-600"></div>
              </label>
            </div>

            <!-- Logo Options (when enabled) -->
            <div v-if="config.header?.logo_enabled" class="pl-4 border-l-2 border-zinc-200 dark:border-zinc-800 space-y-3">
              <!-- Logo Upload / Preview -->
              <div>
                <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Logo Image</label>

                <!-- Preview with actions (when logo exists) -->
                <div v-if="config.header?.logo_url" class="flex items-start gap-3">
                  <img
                    :src="getLogoUrl(config.header.logo_url)"
                    alt="Logo preview"
                    class="max-h-16 max-w-32 object-contain rounded border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-700 p-1"
                    @error="$event.target.style.display = 'none'"
                  />
                  <div class="flex flex-col gap-1">
                    <label
                      class="inline-flex items-center gap-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
                      :class="{ 'opacity-50 cursor-wait': logoUploading }"
                    >
                      <Upload class="w-3 h-3" />
                      Replace
                      <input type="file" accept="image/jpeg,image/png,image/webp" class="hidden" :disabled="logoUploading" @change="handleLogoUpload" />
                    </label>
                    <button
                      @click="handleLogoDelete"
                      :disabled="logoUploading"
                      class="inline-flex items-center gap-1 px-2 py-1 text-xs text-red-600 dark:text-red-400 border border-red-300 dark:border-red-700 rounded hover:bg-red-50 dark:hover:bg-red-900/20"
                    >
                      <Trash2 class="w-3 h-3" />
                      Remove
                    </button>
                  </div>
                </div>

                <!-- Upload button (when no logo) -->
                <label
                  v-else
                  class="flex items-center justify-center gap-2 w-full px-3 py-4 border-2 border-dashed rounded-lg cursor-pointer transition-colors"
                  :class="logoUploading
                    ? 'border-zinc-500 bg-zinc-50 dark:bg-zinc-900/30 cursor-wait'
                    : 'border-gray-300 dark:border-gray-600 hover:border-zinc-500 hover:bg-zinc-50/50 dark:hover:bg-zinc-900/20'"
                >
                  <template v-if="logoUploading">
                    <div class="animate-spin rounded-full h-5 w-5 border-2 border-zinc-200 dark:border-zinc-800 border-t-zinc-600 dark:border-t-zinc-400"></div>
                    <span class="text-sm text-zinc-600 dark:text-zinc-400">Uploading...</span>
                  </template>
                  <template v-else>
                    <Upload class="w-5 h-5 text-gray-400" />
                    <span class="text-sm text-gray-500 dark:text-gray-400">Click to upload logo</span>
                    <span class="text-xs text-gray-400">(max 2MB)</span>
                  </template>
                  <input type="file" accept="image/jpeg,image/png,image/webp" class="hidden" :disabled="logoUploading" @change="handleLogoUpload" />
                </label>

                <!-- Unsaved template notice -->
                <p v-if="!templateId" class="text-xs text-amber-600 dark:text-amber-400 mt-1">
                  Save the template first to upload a logo
                </p>
              </div>

              <!-- Max Height -->
              <div>
                <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Max Height (px)</label>
                <input
                  type="number"
                  :value="config.header?.logo_max_height || 60"
                  @input="updateHeader('logo_max_height', parseInt($event.target.value) || 60)"
                  min="30"
                  max="120"
                  class="w-24 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
              </div>
            </div>

            <!-- Badge Configuration -->
            <div class="pt-2 border-t border-gray-200 dark:border-gray-700 space-y-3">
              <p class="text-xs font-medium text-gray-700 dark:text-gray-300">Authenticity Badge</p>

              <div data-tour="badge-text-input">
                <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Badge Text</label>
                <input
                  type="text"
                  :value="config.header?.badge_text || 'Authentic Product'"
                  @input="updateHeader('badge_text', $event.target.value)"
                  class="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  placeholder="Authentic Product"
                />
              </div>

              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Badge Background</label>
                  <div class="flex items-center gap-2">
                    <input
                      type="color"
                      :value="config.header?.badge_bg_color || '#22c55e'"
                      @input="updateHeader('badge_bg_color', $event.target.value)"
                      class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                    />
                    <input
                      type="text"
                      :value="config.header?.badge_bg_color || '#22c55e'"
                      @input="updateHeader('badge_bg_color', $event.target.value)"
                      class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                    />
                  </div>
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Badge Text Color</label>
                  <div class="flex items-center gap-2">
                    <input
                      type="color"
                      :value="config.header?.badge_text_color || '#ffffff'"
                      @input="updateHeader('badge_text_color', $event.target.value)"
                      class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                    />
                    <input
                      type="text"
                      :value="config.header?.badge_text_color || '#ffffff'"
                      @input="updateHeader('badge_text_color', $event.target.value)"
                      class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Background Image Configuration -->
        <div v-if="backgroundConfig">
          <BackgroundConfigEditor
            :model-value="backgroundConfig"
            @update:model-value="handleBackgroundConfigUpdate"
            type="landing"
          />
        </div>

        <!-- Card Styling Section -->
        <div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
            <svg class="w-5 h-5 text-zinc-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
            </svg>
            Card Styling
          </h3>
          <p class="text-xs text-gray-500 dark:text-gray-400 mb-4">
            Customize the colors for the product info cards
          </p>
            <div class="grid grid-cols-3 gap-3">
              <div>
                <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Page BG</label>
                <div class="flex items-center gap-1">
                  <input
                    type="color"
                    :value="config.styling?.card_bg_color || '#f3f4f6'"
                    @input="updateStyling('card_bg_color', $event.target.value)"
                    class="w-8 h-8 rounded cursor-pointer border border-gray-300 dark:border-gray-600 flex-shrink-0"
                  />
                  <input
                    type="text"
                    :value="config.styling?.card_bg_color || '#f3f4f6'"
                    @input="updateStyling('card_bg_color', $event.target.value)"
                    class="w-16 px-1 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                  />
                </div>
              </div>
              <div>
                <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Field BG</label>
                <div class="flex items-center gap-1">
                  <input
                    type="color"
                    :value="config.styling?.field_bg_color || '#ffffff'"
                    @input="updateStyling('field_bg_color', $event.target.value)"
                    class="w-8 h-8 rounded cursor-pointer border border-gray-300 dark:border-gray-600 flex-shrink-0"
                  />
                  <input
                    type="text"
                    :value="config.styling?.field_bg_color || '#ffffff'"
                    @input="updateStyling('field_bg_color', $event.target.value)"
                    class="w-16 px-1 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                  />
                </div>
              </div>
              <div>
                <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Text Color</label>
                <div class="flex items-center gap-1">
                  <input
                    type="color"
                    :value="config.styling?.text_color || '#1f2937'"
                    @input="updateStyling('text_color', $event.target.value)"
                    class="w-8 h-8 rounded cursor-pointer border border-gray-300 dark:border-gray-600 flex-shrink-0"
                  />
                  <input
                    type="text"
                    :value="config.styling?.text_color || '#1f2937'"
                    @input="updateStyling('text_color', $event.target.value)"
                    class="w-16 px-1 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                  />
                </div>
              </div>

              <!-- Main Image Size -->
              <div>
                <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Main Image Size (px)</label>
                <input
                  type="number"
                  :value="config.styling?.main_image_size || 96"
                  @input="updateStyling('main_image_size', Math.min(128, Math.max(48, parseInt($event.target.value) || 96)))"
                  min="48"
                  max="128"
                  class="w-24 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
                <p class="text-xs text-gray-400 mt-1">48 - 128px (square)</p>
              </div>
            </div>
        </div>

        <!-- Certifications Section Styling -->
        <div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-2 flex items-center gap-2">
            <svg class="w-5 h-5 text-emerald-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
            </svg>
            Certifications Section
          </h3>
          <p class="text-xs text-gray-500 dark:text-gray-400 mb-4">
            Styling for certifications accordion. Section appears automatically if product has certifications.
          </p>

          <div class="space-y-4">
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Icon Color</label>
                <div class="flex items-center gap-2">
                  <input
                    type="color"
                    :value="config.certifications_section?.icon_color || '#10b981'"
                    @input="updateCertificationsSection('icon_color', $event.target.value)"
                    class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                  />
                  <input
                    type="text"
                    :value="config.certifications_section?.icon_color || '#10b981'"
                    @input="updateCertificationsSection('icon_color', $event.target.value)"
                    class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                  />
                </div>
              </div>
              <div>
                <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Background</label>
                <div class="flex items-center gap-2">
                  <input
                    type="color"
                    :value="config.certifications_section?.bg_color || '#f0fdf4'"
                    @input="updateCertificationsSection('bg_color', $event.target.value)"
                    class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                  />
                  <input
                    type="text"
                    :value="config.certifications_section?.bg_color || '#f0fdf4'"
                    @input="updateCertificationsSection('bg_color', $event.target.value)"
                    class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                  />
                </div>
              </div>
            </div>

            <div class="flex items-center justify-between py-2 border-t border-gray-200 dark:border-gray-700">
              <div>
                <span class="block text-sm font-medium text-gray-900 dark:text-white">Default Expanded</span>
                <span class="block text-xs text-gray-500 dark:text-gray-400">Open by default on page load</span>
              </div>
              <label class="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  :checked="config.certifications_section?.default_expanded"
                  @change="updateCertificationsSection('default_expanded', !config.certifications_section?.default_expanded)"
                  class="sr-only peer"
                />
                <div class="w-9 h-5 bg-gray-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-zinc-300 dark:peer-focus:ring-zinc-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all dark:border-gray-500 peer-checked:bg-emerald-600"></div>
              </label>
            </div>
          </div>
        </div>

        <!-- Warranty Button Section -->
        <div data-tour="warranty-button-section" class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
            <svg class="w-5 h-5 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
            Warranty Activation Button
          </h3>
          <p class="text-xs text-gray-500 dark:text-gray-400 mb-4">
            Button appears only if warranty is enabled when creating QR batch
          </p>

          <div class="space-y-4">
            <div>
              <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Button Text</label>
              <input
                type="text"
                :value="config.warranty_button.text"
                @input="updateWarrantyButton('text', $event.target.value)"
                class="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                placeholder="Activate Warranty"
              />
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Background Color</label>
                <div class="flex items-center gap-2">
                  <input
                    type="color"
                    :value="config.warranty_button.bg_color"
                    @input="updateWarrantyButton('bg_color', $event.target.value)"
                    class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                  />
                  <input
                    type="text"
                    :value="config.warranty_button.bg_color"
                    @input="updateWarrantyButton('bg_color', $event.target.value)"
                    class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                  />
                </div>
              </div>
              <div>
                <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Text Color</label>
                <div class="flex items-center gap-2">
                  <input
                    type="color"
                    :value="config.warranty_button.text_color"
                    @input="updateWarrantyButton('text_color', $event.target.value)"
                    class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                  />
                  <input
                    type="text"
                    :value="config.warranty_button.text_color"
                    @input="updateWarrantyButton('text_color', $event.target.value)"
                    class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Social Media Section -->
        <div data-tour="social-media-section" class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-2 flex items-center gap-2">
            <svg class="w-5 h-5 text-pink-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
            </svg>
            Social Media Section
          </h3>
          <p class="text-xs text-gray-500 dark:text-gray-400 mb-4">
            Display social media icons. Appears automatically if tenant has social media accounts.
          </p>

          <div class="flex items-center justify-between py-2">
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">Sticky Bottom Bar</span>
              <span class="block text-xs text-gray-500 dark:text-gray-400">Pin social icons to the bottom of the screen</span>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                :checked="config.social_media_section?.sticky_enabled !== false"
                @change="updateSocialMediaSection('sticky_enabled', !(config.social_media_section?.sticky_enabled !== false))"
                class="sr-only peer"
              />
              <div class="w-9 h-5 bg-gray-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-zinc-300 dark:peer-focus:ring-zinc-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all dark:border-gray-500 peer-checked:bg-zinc-600"></div>
            </label>
          </div>
        </div>

        <!-- Section Order Configuration -->
        <div data-tour="section-order-section" class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-2 flex items-center gap-2">
            <svg class="w-5 h-5 text-zinc-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
            </svg>
            Section Order
          </h3>
          <p class="text-xs text-gray-500 dark:text-gray-400 mb-4">
            Drag and drop to reorder sections on the landing page. Header, product info, and footer are fixed.
          </p>

          <VueDraggable
            v-model="sectionOrderList"
            :animation="150"
            handle=".drag-handle"
            ghost-class="opacity-50"
            class="space-y-2"
            @end="updateSectionOrder"
          >
            <div
              v-for="section in sectionOrderList"
              :key="section.id"
              v-show="!(section.id === 'social_accounts' && config.social_media_section?.sticky_enabled !== false)"
              class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-700 rounded-lg border border-gray-200 dark:border-gray-600 hover:border-zinc-300 dark:hover:border-zinc-600 transition-colors"
            >
              <div class="drag-handle cursor-grab active:cursor-grabbing text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
                <GripVertical class="w-4 h-4" />
              </div>
              <component :is="section.icon" class="w-4 h-4 text-gray-500 dark:text-gray-400" />
              <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ section.label }}</span>
            </div>
          </VueDraggable>
        </div>

      </div>

      <!-- Live Preview Panel -->
      <div class="lg:sticky lg:top-4 h-fit">
        <div class="bg-gray-100 dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
          <div class="bg-gray-200 dark:bg-gray-800 px-4 py-2 flex items-center justify-between">
            <div class="flex items-center gap-2">
              <div class="flex gap-1.5">
                <div class="w-3 h-3 rounded-full bg-red-500"></div>
                <div class="w-3 h-3 rounded-full bg-yellow-500"></div>
                <div class="w-3 h-3 rounded-full bg-green-500"></div>
              </div>
              <span class="text-xs text-gray-500 dark:text-gray-400 ml-2">Live Preview</span>
            </div>
            <!-- Aspect Ratio Selector -->
            <select
              v-model="selectedRatio"
              class="text-xs px-2 py-1 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-300 focus:ring-1 focus:ring-[#27272a]"
            >
              <option v-for="ratio in ASPECT_RATIOS" :key="ratio.value" :value="ratio.value">
                {{ ratio.label }}
              </option>
            </select>
          </div>

          <!-- Phone Frame -->
          <div class="p-4">
            <div
              class="mx-auto max-w-[320px] rounded-[32px] border-[8px] border-gray-800 dark:border-gray-600 overflow-hidden shadow-xl relative"
              :style="{ backgroundColor: hasBackground ? 'transparent' : (config.styling?.card_bg_color || '#f3f4f6') }"
            >
              <!-- Background Image Layer (when background is configured) -->
              <div
                v-if="hasBackground && backgroundUrl"
                class="absolute inset-0"
                :style="backgroundStyle"
              ></div>
              <!-- Background Overlay Layer -->
              <div
                v-if="hasBackground && backgroundUrl"
                class="absolute inset-0"
                :style="overlayStyle"
              ></div>

              <!-- Phone Screen -->
              <div class="overflow-y-auto relative z-10" :style="{ aspectRatio: selectedRatio }">
                <!-- Glass Card Container (wraps everything when background is enabled) -->
                <div
                  :class="hasBackground ? 'm-3 rounded-2xl overflow-hidden' : ''"
                  :style="glassCardStyle"
                >
                  <!-- Header -->
                  <div
                    class="px-4 py-4 text-center"
                    :style="headerGlassStyle"
                  >
                    <!-- Logo (only when uploaded) -->
                    <div v-if="config.header?.logo_enabled && config.header?.logo_url" class="mb-3">
                      <img
                        :src="getLogoUrl(config.header.logo_url)"
                        alt="Company Logo"
                        class="mx-auto object-contain"
                        :style="{ maxHeight: (config.header?.logo_max_height || 60) + 'px' }"
                        @error="$event.target.style.display = 'none'"
                      />
                    </div>
                    <!-- Authentic Badge (smaller) -->
                    <div
                      class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium"
                      :style="{
                        backgroundColor: config.header?.badge_bg_color || '#22c55e',
                        color: config.header?.badge_text_color || '#ffffff'
                      }"
                    >
                      <svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                      </svg>
                      {{ config.header?.badge_text || 'Authentic Product' }}
                    </div>
                  </div>

                  <!-- Content -->
                  <div class="px-4 py-4 space-y-3">
                    <!-- Product Header Preview (image thumbnail + name + company) -->
                    <div class="rounded-lg p-3 mb-1" :style="fieldCardStyle">
                      <div class="flex items-center gap-3">
                        <div class="flex-shrink-0 rounded-lg bg-gray-200 dark:bg-gray-600 flex items-center justify-center" :style="{ width: (config.styling?.main_image_size || 96) * 0.5 + 'px', height: (config.styling?.main_image_size || 96) * 0.5 + 'px' }">
                          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                          </svg>
                        </div>
                        <div class="min-w-0 flex-1">
                          <p class="font-semibold text-sm leading-tight" :style="textColorStyle">Sample Product</p>
                          <p class="text-xs mt-0.5" :style="labelColorStyle">Contoh Company</p>
                        </div>
                      </div>
                    </div>

                    <!-- Consolidated info fields card (matches real landing page) -->
                    <div class="rounded-lg p-4 space-y-3" :style="fieldCardStyle">
                      <div class="flex justify-between text-sm">
                        <span :style="labelColorStyle">Product Code</span>
                        <span class="font-medium" :style="textColorStyle">SKU-12345</span>
                      </div>
                      <div class="flex justify-between text-sm">
                        <span :style="labelColorStyle">Verified</span>
                        <span class="font-medium" :style="textColorStyle">3 times</span>
                      </div>
                      <div class="flex justify-between text-sm">
                        <span :style="labelColorStyle">Batch</span>
                        <span class="font-medium" :style="textColorStyle">BATCH-{{ currentYear }}-001</span>
                      </div>
                      <div class="flex justify-between text-sm">
                        <span :style="labelColorStyle">Production Date</span>
                        <span class="font-medium" :style="textColorStyle">10 Jan {{ currentYear }}</span>
                      </div>
                      <div class="flex justify-between text-sm">
                        <span :style="labelColorStyle">Expiry Date</span>
                        <span class="font-medium" :style="textColorStyle">10 Jan {{ currentYear + 2 }}</span>
                      </div>
                    </div>

                    <p class="text-[10px] italic text-center" :style="{ color: hasBackground ? 'rgba(0,0,0,0.5)' : '#9ca3af' }">
                      * Actual fields displayed are configured at Product level
                    </p>

                    <!-- Dynamic Sections Based on Section Order -->
                    <template v-for="section in sectionOrderList" :key="section.id">
                      <!-- Product Gallery Preview -->
                      <div v-if="section.id === 'images'" class="mt-3 rounded-lg p-3" :style="fieldCardStyle">
                        <div class="flex items-center gap-2 mb-2">
                          <component :is="section.icon" class="w-4 h-4" :style="{ color: hasBackground ? 'rgba(0,0,0,0.6)' : '#6b7280' }" />
                          <span class="text-xs font-medium" :style="textColorStyle">{{ section.label }}</span>
                        </div>
                        <div class="grid grid-cols-3 gap-1.5">
                          <div class="aspect-square rounded bg-gray-200 dark:bg-gray-600 flex items-center justify-center">
                            <svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                            </svg>
                          </div>
                          <div class="aspect-square rounded bg-gray-200 dark:bg-gray-600 flex items-center justify-center">
                            <svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                            </svg>
                          </div>
                          <div class="aspect-square rounded bg-gray-200 dark:bg-gray-600 flex items-center justify-center text-xs text-gray-400">+3</div>
                        </div>
                      </div>

                      <!-- Videos Preview -->
                      <div v-if="section.id === 'videos'" class="mt-3 rounded-lg p-3" :style="fieldCardStyle">
                        <div class="flex items-center gap-2 mb-2">
                          <component :is="section.icon" class="w-4 h-4" :style="{ color: hasBackground ? 'rgba(0,0,0,0.6)' : '#6b7280' }" />
                          <span class="text-xs font-medium" :style="textColorStyle">{{ section.label }}</span>
                        </div>
                        <div class="aspect-video rounded bg-gray-200 dark:bg-gray-600 flex items-center justify-center">
                          <div class="w-10 h-10 rounded-full bg-gray-300 dark:bg-gray-500 flex items-center justify-center">
                            <svg class="w-5 h-5 text-gray-500 dark:text-gray-400 ml-0.5" fill="currentColor" viewBox="0 0 24 24">
                              <path d="M8 5v14l11-7z" />
                            </svg>
                          </div>
                        </div>
                      </div>

                      <!-- Social Media Section (inline, only when NOT sticky) -->
                      <div v-if="section.id === 'social_accounts' && config.social_media_section?.sticky_enabled === false" class="mt-4">
                        <div class="flex items-center justify-center gap-3 flex-wrap">
                          <a
                            v-for="social in sampleSocialMedia"
                            :key="social.platform"
                            href="#"
                            @click.prevent
                            class="w-10 h-10 rounded-full bg-gray-100 flex items-center justify-center hover:bg-gray-200 transition-colors"
                            :title="social.platform"
                          >
                            <svg class="w-5 h-5 text-gray-700" viewBox="0 0 24 24" fill="currentColor">
                              <path :d="social.svgPath" />
                            </svg>
                          </a>
                        </div>
                      </div>

                      <!-- Certifications Section (Collapsible) -->
                      <div
                        v-if="section.id === 'certifications'"
                        class="mt-3 rounded-lg overflow-hidden"
                        :style="hasBackground ? { backgroundColor: 'rgba(255, 255, 255, 0.1)', border: '1px solid rgba(255, 255, 255, 0.2)' } : { backgroundColor: config.certifications_section?.bg_color || '#f0fdf4', boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)' }"
                      >
                        <button
                          @click="previewCertExpanded = !previewCertExpanded"
                          class="w-full py-3 px-4 flex items-center justify-between text-left"
                        >
                          <div class="flex items-center gap-2 min-w-0">
                            <!-- Logo previews when at least one cert has a logo -->
                            <template v-if="sampleCertifications.some(c => c.logo_url)">
                              <div class="flex items-center gap-1.5 min-w-0">
                                <template v-for="(cert, idx) in sampleCertifications.slice(0, 7)" :key="idx">
                                  <img
                                    v-if="cert.logo_url"
                                    :src="cert.logo_url"
                                    :alt="cert.name"
                                    class="w-10 h-10 rounded-full object-contain bg-white border border-white/50 flex-shrink-0"
                                  />
                                  <span
                                    v-else
                                    class="w-10 h-10 rounded-full flex items-center justify-center bg-white/60 flex-shrink-0"
                                  >
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" :style="{ color: config.certifications_section?.icon_color || '#10b981' }">
                                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                                    </svg>
                                  </span>
                                </template>
                                <span
                                  v-if="sampleCertifications.length > 7"
                                  class="text-sm font-semibold flex-shrink-0"
                                  :style="textColorStyle"
                                >
                                  +{{ sampleCertifications.length - 7 }}
                                </span>
                              </div>
                            </template>
                            <!-- Fallback: show label if no cert has a logo -->
                            <template v-else>
                              <component :is="section.icon" class="w-4 h-4" :style="{ color: config.certifications_section?.icon_color || '#10b981' }" />
                              <span class="text-xs font-medium" :style="textColorStyle">
                                {{ config.certifications_section?.header_text || 'Certifications' }}
                              </span>
                              <span
                                class="text-xs px-2 py-0.5 rounded-full"
                                :style="{ backgroundColor: hasBackground ? 'rgba(255,255,255,0.2)' : 'rgba(255,255,255,0.6)', ...textColorStyle }"
                              >
                                {{ sampleCertifications.length }}
                              </span>
                            </template>
                          </div>
                          <svg
                            class="w-4 h-4 transition-transform flex-shrink-0"
                            :class="{ 'rotate-180': previewCertExpanded }"
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                            :style="textColorStyle"
                          >
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                          </svg>
                        </button>
                        <div v-if="previewCertExpanded" class="px-3 pb-3 space-y-1.5">
                          <div
                            v-for="cert in sampleCertifications"
                            :key="cert.name"
                            class="flex items-center gap-3 p-2 rounded-lg"
                            :style="{ backgroundColor: hasBackground ? 'rgba(255,255,255,0.1)' : 'rgba(255,255,255,0.15)' }"
                          >
                            <div v-if="cert.logo_url" class="w-10 h-10 flex-shrink-0">
                              <img
                                :src="cert.logo_url"
                                :alt="cert.name"
                                class="w-full h-full object-contain"
                              />
                            </div>
                            <div
                              v-else
                              class="w-10 h-10 flex-shrink-0 rounded-full flex items-center justify-center"
                              style="background: rgba(255,255,255,0.2)"
                            >
                              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" :style="{ color: config.certifications_section?.icon_color || '#10b981' }">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                              </svg>
                            </div>
                            <div class="flex-1 min-w-0">
                              <p class="text-xs font-medium truncate" :style="textColorStyle">{{ cert.name }}</p>
                              <p class="text-xs" :style="{ ...textColorStyle, opacity: 0.7 }">
                                {{ cert.country }} <span v-if="cert.registration_number">- {{ cert.registration_number }}</span>
                              </p>
                            </div>
                          </div>
                        </div>
                      </div>

                      <!-- Website Button -->
                      <div v-if="section.id === 'website_link'" class="mt-3">
                        <button
                          class="w-full py-2.5 rounded-lg font-medium text-xs transition-colors flex items-center justify-center gap-2"
                          :style="{ backgroundColor: hasBackground ? 'rgba(59, 130, 246, 0.9)' : '#3f3f46', color: '#ffffff' }"
                        >
                          <component :is="section.icon" class="w-4 h-4" />
                          Visit Our Website
                        </button>
                      </div>

                      <!-- Description Preview -->
                      <div v-if="section.id === 'description'" class="mt-3 rounded-lg p-3" :style="fieldCardStyle">
                        <div class="flex items-center gap-2 mb-2">
                          <component :is="section.icon" class="w-4 h-4" :style="{ color: hasBackground ? 'rgba(0,0,0,0.6)' : '#6b7280' }" />
                          <span class="text-xs font-medium" :style="textColorStyle">{{ section.label }}</span>
                        </div>
                        <p class="text-xs leading-relaxed" :style="{ color: hasBackground ? 'rgba(0,0,0,0.7)' : '#6b7280' }">
                          This is a sample product description that will be shown to customers when they scan the QR code...
                        </p>
                      </div>

                      <!-- Warranty Button -->
                      <div v-if="section.id === 'warranty_button'" class="mt-3">
                        <button
                          class="w-full py-3 rounded-lg font-medium text-sm transition-colors flex items-center justify-center gap-2"
                          :style="{
                            backgroundColor: config.warranty_button.bg_color,
                            color: config.warranty_button.text_color
                          }"
                        >
                          <component :is="section.icon" class="w-4 h-4" />
                          {{ config.warranty_button.text || 'Activate Warranty' }}
                        </button>
                      </div>
                    </template>
                  </div>
                </div>
              </div>

              <!-- Sticky Social Media Bar (bottom of phone frame) -->
              <div
                v-if="config.social_media_section?.sticky_enabled !== false"
                class="sticky bottom-0 left-0 right-0 z-10 bg-white/95 backdrop-blur-sm border-t border-gray-200 shadow-lg"
              >
                <div class="px-4 py-3">
                  <div class="flex items-center justify-center gap-3">
                    <a
                      v-for="social in sampleSocialMedia"
                      :key="social.platform"
                      href="#"
                      @click.prevent
                      class="w-10 h-10 rounded-full bg-gray-100 flex items-center justify-center hover:bg-gray-200 transition-colors"
                      :title="social.platform"
                    >
                      <svg class="w-5 h-5 text-gray-700" viewBox="0 0 24 24" fill="currentColor">
                        <path :d="social.svgPath" />
                      </svg>
                    </a>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>
