<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAPI } from '@/composables/useAPI'
import { useTour, isTourActive, getTourNonce } from '@/composables/useTour'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import ValidationTemplateEditor from '@/components/ValidationTemplateEditor.vue'
import WarrantyTemplateEditor from '@/components/WarrantyTemplateEditor.vue'
import BackgroundConfigEditor from '@/components/BackgroundConfigEditor.vue'

const router = useRouter()
const route = useRoute()
const { get, post, put } = useAPI()

const loading = ref(true)
const saving = ref(false)
const themePresets = ref([])

const templateId = computed(() => {
  const id = route.params.id
  return id === 'new' ? null : id
})

const isNew = computed(() => templateId.value === null)

const templateData = ref({
  name: '',
  type: 'validation', // 'validation' | 'warranty'
  is_active: true,
  html_content: '',
  css_content: '',
  custom_fields: null,
})

// Background config for templates (Intermediate+ tier only)
const defaultBackgroundConfig = {
  background_type: 'none',
  preset_id: null,
  custom_background_url: null,
  overlay_color: '#000000',
  overlay_opacity: 30,
  card_opacity: 90,
  card_blur: 0
}
const backgroundConfig = ref({ ...defaultBackgroundConfig })

// Default configs for new templates
const defaultValidationConfig = {
  header: {
    logo_enabled: true,
    logo_url: '',
    logo_max_height: 60,
    bg_color: '#3f3f46',
    badge_text: 'Authentic Product',
    badge_bg_color: '#22c55e',
    badge_text_color: '#ffffff',
  },
  warranty_button: {
    enabled: true,
    text: 'Activate Warranty',
    bg_color: '#8b5cf6',
    text_color: '#ffffff',
  },
  certifications_section: {
    header_text: 'Certifications',
    icon_color: '#10b981',
    bg_color: '#f0fdf4',
    default_expanded: false,
  },
  social_media_section: {
    header_text: 'Follow Us',
    icon_color: '#ec4899',
    bg_color: '#fdf2f8',
    default_expanded: false,
  },
  styling: {
    card_bg_color: '#f3f4f6',
    field_bg_color: '#ffffff',
    text_color: '#1f2937',
    main_image_size: 96,
  },
}

const defaultWarrantyConfig = {
  submit_button: {
    text: 'Activate Warranty',
    bg_color: '#8b5cf6',
    text_color: '#ffffff',
  },
  styling: {
    header_bg_color: '#8b5cf6',
    form_bg_color: '#f3f4f6',
    text_color: '#1f2937',
    accent_color: '#8b5cf6',
  },
  messages: {
    success_title: 'Warranty Activated!',
    success_message: 'Your product warranty has been successfully activated. You will receive a confirmation email shortly.',
    already_activated_title: 'Already Activated',
    already_activated_message: 'This product warranty has already been activated. Please contact support if you need assistance.',
  },
}


const typeLabels = {
  validation: 'Validation / Landing Page',
  warranty: 'Warranty Page',
}

const fetchThemePresets = async () => {
  try {
    const response = await get('/tenant/theme-presets', { type: 'landing' })
    if (response.success) {
      themePresets.value = response.data.theme_presets || []
    }
  } catch (err) {
    console.error('Failed to load theme presets:', err)
  }
}

const fetchTemplate = async () => {
  // Fetch theme presets for Live Preview
  await fetchThemePresets()

  if (!templateId.value) {
    // New template - check query param for type
    const type = route.query.type
    if (type && ['validation', 'warranty'].includes(type)) {
      templateData.value.type = type
    }

    // Initialize config based on type
    if (templateData.value.type === 'validation') {
      validationConfig.value = { ...defaultValidationConfig }
    } else if (templateData.value.type === 'warranty') {
      warrantyConfig.value = { ...defaultWarrantyConfig }
    }
    // Reset background config for new templates
    backgroundConfig.value = { ...defaultBackgroundConfig }
    loading.value = false
    return
  }

  try {
    loading.value = true
    const response = await get(`/tenant/templates/${templateId.value}`)
    if (response.success && response.data) {
      templateData.value = {
        name: response.data.template_name,
        type: response.data.template_type,
        is_active: response.data.is_active,
        html_content: response.data.html_content,
        css_content: response.data.css_content,
        custom_fields: response.data.custom_fields,
      }

      // Load config based on type
      if (response.data.template_type === 'validation' && response.data.custom_fields) {
        validationConfig.value = { ...defaultValidationConfig, ...response.data.custom_fields }
      } else if (response.data.template_type === 'warranty' && response.data.custom_fields) {
        warrantyConfig.value = { ...defaultWarrantyConfig, ...response.data.custom_fields }
      }

      // Load background config (Intermediate+ tier)
      if (response.data.background_config) {
        backgroundConfig.value = { ...defaultBackgroundConfig, ...response.data.background_config }
      } else {
        backgroundConfig.value = { ...defaultBackgroundConfig }
      }
    }
  } catch (error) {
    console.error('Failed to fetch template:', error)
    router.push('/tenant/templates')
  } finally {
    loading.value = false
  }
}

const saveTemplate = async () => {
  if (!templateData.value.name.trim()) {
    alert('Please enter a template name')
    return
  }

  try {
    saving.value = true

    let payload = {
      template_name: templateData.value.name,
      template_type: templateData.value.type,
      is_active: templateData.value.is_active,
      custom_fields: null,
      html_content: '',
      css_content: '',
      background_config: null,
    }

    // Build payload based on template type
    if (templateData.value.type === 'validation') {
      payload.custom_fields = validationConfig.value
      payload.html_content = '<!-- Validation template - rendered from custom_fields -->'
    } else if (templateData.value.type === 'warranty') {
      payload.custom_fields = warrantyConfig.value
      payload.html_content = '<!-- Warranty template - rendered from custom_fields -->'
    }

    // Include background config for validation/warranty templates
    if (templateData.value.type === 'validation' || templateData.value.type === 'warranty') {
      payload.background_config = backgroundConfig.value
    }

    let response
    if (isNew.value) {
      response = await post('/tenant/templates', payload)
    } else {
      response = await put(`/tenant/templates/${templateId.value}`, payload)
    }

    if (response.success) {
      router.push('/tenant/templates')
    }
  } catch (error) {
    console.error('Failed to save template:', error)
  } finally {
    saving.value = false
  }
}

const cancel = () => {
  router.push('/tenant/templates')
}

// ── Tour support ──
const { resumeIfActive, cancelTour } = useTour()

function handleTourSetValue(e) {
  if (!isTourActive()) return
  if (e.detail._nonce !== getTourNonce()) return
  const { field, value } = e.detail
  switch (field) {
    case 'template_name':
      templateData.value.name = value
      break
    case 'header_bg_color':
      validationConfig.value = {
        ...validationConfig.value,
        header: { ...validationConfig.value.header, bg_color: value },
      }
      break
    case 'badge_text':
      validationConfig.value = {
        ...validationConfig.value,
        header: { ...validationConfig.value.header, badge_text: value },
      }
      break
    case 'bg_type':
      // Handled by BackgroundConfigEditor via data-tour
      backgroundConfig.value = { ...backgroundConfig.value, background_type: value }
      break
    case 'bg_preset_last':
      // Handled by BackgroundConfigEditor via data-tour
      break
    case 'overlay_opacity':
      backgroundConfig.value = { ...backgroundConfig.value, overlay_opacity: value }
      break
    case 'card_opacity':
      backgroundConfig.value = { ...backgroundConfig.value, card_opacity: value }
      break
    case 'card_blur':
      backgroundConfig.value = { ...backgroundConfig.value, card_blur: value }
      break
  }
}

function handleTourCancelled() {
  // No specific cleanup needed
}

onMounted(() => {
  fetchTemplate()
  window.addEventListener('tour-set-value', handleTourSetValue)
  window.addEventListener('tour-cancelled', handleTourCancelled)
  resumeIfActive()
})

onBeforeUnmount(() => {
  window.removeEventListener('tour-set-value', handleTourSetValue)
  window.removeEventListener('tour-cancelled', handleTourCancelled)
})
</script>

<template>
  <div>
    <div class="sticky top-0 z-20 flex justify-between items-center py-4 -mx-6 px-6 mb-6 bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ isNew ? 'Create Template' : 'Edit Template' }}
        </h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
          {{ typeLabels[templateData.type] || 'Page Template' }}
        </p>
      </div>
      <div class="flex gap-3">
        <Button variant="outline" @click="cancel">Cancel</Button>
        <Button data-tour="create-template-btn" @click="saveTemplate" :disabled="saving">
          {{ saving ? 'Saving...' : isNew ? 'Create Template' : 'Save Changes' }}
        </Button>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <div v-else class="space-y-6">
      <!-- Template Settings -->
      <Card class="p-6">
        <div class="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Template Name *
            </label>
            <Input
              v-model="templateData.name"
              type="text"
              placeholder="e.g., Default Validation Page"
              required
              data-tour="template-name-input"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Status
            </label>
            <label class="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                v-model="templateData.is_active"
                class="sr-only peer"
              />
              <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-zinc-300 dark:peer-focus:ring-zinc-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-zinc-600"></div>
              <span class="ms-3 text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ templateData.is_active ? 'Active' : 'Inactive' }}
              </span>
            </label>
          </div>
        </div>
      </Card>

      <!-- Validation Template Editor -->
      <ValidationTemplateEditor
        v-if="templateData.type === 'validation'"
        v-model="validationConfig"
        v-model:background-config="backgroundConfig"
        :presets="themePresets"
        :template-id="templateId"
      />

      <!-- Warranty Template Editor -->
      <WarrantyTemplateEditor
        v-if="templateData.type === 'warranty'"
        v-model="warrantyConfig"
      />

      <!-- Background Image Configuration for Warranty -->
      <Card v-if="templateData.type === 'warranty'" class="p-6">
        <div class="mb-4">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            Background Image
          </h2>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
            Customize the background image for this template's warranty page.
          </p>
        </div>
        <BackgroundConfigEditor v-model="backgroundConfig" type="landing" />
      </Card>

    </div>
  </div>
</template>
