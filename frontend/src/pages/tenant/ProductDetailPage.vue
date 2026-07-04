<script setup>
import { ref, onMounted, onBeforeUnmount, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { VueDraggable } from 'vue-draggable-plus'
import { useAPI } from '@/composables/useAPI'
import { useEscapeKey } from '@/composables/useEscapeKey'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { ArrowLeft, GripVertical } from 'lucide-vue-next'
import ProductImageGallery from '@/components/ProductImageGallery.vue'
import VideoEmbedEditor from '@/components/VideoEmbedEditor.vue'
import ProductSocialAccountEditor from '@/components/ProductSocialAccountEditor.vue'
import ProductTemplatePreview from '@/components/ProductTemplatePreview.vue'
import WarrantySettingsPreview from '@/components/WarrantySettingsPreview.vue'
import { Image, Video, Share2, Award, ExternalLink, FileText, Shield, ChevronDown, RotateCcw } from 'lucide-vue-next'
import { useTour, isTourActive, getTourNonce } from '@/composables/useTour.js'
// NOTE: Background config has been moved to Templates. Edit in TemplateEditorPage instead.

const route = useRoute()
const router = useRouter()
const { get, post, put, del } = useAPI()

// Reactive route param - re-fetches when route changes without full remount
const productId = computed(() => route.params.id)
const loading = ref(true)
const saving = ref(false)
const allTabs = ['info', 'certifications', 'gallery', 'videos', 'links', 'template', 'warranty']
const validTabs = computed(() => allTabs)
const initialTab = allTabs.includes(route.query.tab) ? route.query.tab : 'info'
const activeTab = ref(initialTab)

// Sync tab selection with URL query param
function setActiveTab(tab) {
  activeTab.value = tab
  if (route.query.tab !== tab) {
    router.replace({ query: { ...route.query, tab } })
  }
}

// React to external URL changes (e.g. browser back/forward, cross-product navigation)
watch(() => route.query.tab, (newTab) => {
  if (newTab && validTabs.value.includes(newTab)) {
    activeTab.value = newTab
  }
})

// Product data
const product = ref(null)
const form = ref({
  product_name: '',
  product_code: '',
  description: '',
  status: 'active',
  counterfeit_scan_max: null
})

// Counterfeit settings (tenant global)
const globalEndUserScanMax = ref(3)

// Display config for validation page
const displayConfig = ref({
  product_name: true,
  product_code: false,
  batch_code: false,
  production_date: false,
  expiry_date: false,
  brand_name: true,
  show_verification_count: true,
  field_order: null
})

// Field metadata for drag-and-drop ordering (all fields including required ones)
const DISPLAY_FIELDS = [
  { key: 'product_name', label: 'Product Name', description: 'Always displayed', required: true },
  { key: 'brand_name', label: 'Company Name', description: 'Always displayed', required: true },
  { key: 'product_code', label: 'Product Code', description: 'Show the product code/SKU' },
  { key: 'show_verification_count', label: 'Verification Count', description: 'Show how many times this product has been scanned' },
  { key: 'batch_code', label: 'Batch Code', description: 'Show production batch identifier' },
  { key: 'production_date', label: 'Production Date', description: 'Show when the product was manufactured' },
  { key: 'expiry_date', label: 'Expiry Date', description: 'Show product expiration date' },
]
const DEFAULT_FIELD_ORDER = DISPLAY_FIELDS.map(f => f.key)
const REQUIRED_KEYS = DISPLAY_FIELDS.filter(f => f.required).map(f => f.key)

const orderedFields = computed({
  get() {
    const order = displayConfig.value.field_order || DEFAULT_FIELD_ORDER
    // Ensure required keys are present (backward compat for old 6-key field_order), deduplicate
    const missing = REQUIRED_KEYS.filter(k => !order.includes(k))
    const fullOrder = [...new Set([...missing, ...order])]
    return fullOrder
      .filter(key => DISPLAY_FIELDS.some(f => f.key === key))
      .map(key => DISPLAY_FIELDS.find(f => f.key === key))
  },
  set(newOrder) {
    displayConfig.value.field_order = newOrder.map(f => f.key)
  }
})

// Website and videos config
const websiteUrl = ref('')
const websiteCaption = ref('')
const videos = ref([])
const savingVideos = ref(false)

// NOTE: Landing appearance config has been moved to Templates (background_config in page_templates)

// Warranty fields config - new structure with enabled toggle and field states
const warrantyFieldsConfig = ref({
  enabled: true,
  fields: {
    store_name: 'optional',    // 'hidden' | 'optional' | 'required'
    country: 'required',
    province: 'required',
    city: 'required',
    address: 'required'
  }
})
// Warranty duration config
const warrantyMonths = ref(12)
const maxWarrantyRegistrationDays = ref(null)

// Custom fields for warranty form
const customFields = ref([])
// { id: 'serial_number', label: 'Serial Number', type: 'text', required: true, options: [] }

// Available field types
const fieldTypes = [
  { value: 'text', label: 'Text' },
  { value: 'textarea', label: 'Text Area' },
  { value: 'number', label: 'Number' },
  { value: 'date', label: 'Date' },
  { value: 'select', label: 'Dropdown' },
  { value: 'email', label: 'Email' },
  { value: 'phone', label: 'Phone' }
]

// Add new custom field
function addCustomField() {
  customFields.value.push({
    id: 'field_' + Date.now(),
    label: '',
    type: 'text',
    required: false,
    options: []
  })
}

// Remove custom field
function removeCustomField(index) {
  customFields.value.splice(index, 1)
}

// Add option to select field
function addFieldOption(field) {
  if (!field.options) field.options = []
  field.options.push('')
}

// Remove option from select field
function removeFieldOption(field, optIndex) {
  field.options.splice(optIndex, 1)
}

// Warranty config validation
const isWarrantyConfigValid = computed(() => {
  // Warranty months validation (1-120, empty = backend default)
  if (warrantyMonths.value !== null && warrantyMonths.value !== '') {
    if (warrantyMonths.value < 1 || warrantyMonths.value > 120) return false
  }
  // Max registration days validation (0-365, empty/0 = unlimited)
  if (maxWarrantyRegistrationDays.value !== null && maxWarrantyRegistrationDays.value !== '') {
    if (maxWarrantyRegistrationDays.value < 0 || maxWarrantyRegistrationDays.value > 365) return false
  }
  return true
})

const savingConfig = ref(false)

// Template selection
const validationTemplates = ref([])
const warrantyTemplates = ref([])
const selectedValidationTemplateId = ref('')
const selectedWarrantyTemplateId = ref('')

// Template preview & overrides
const selectedTemplateConfig = ref(null)
const loadingTemplateConfig = ref(false)
const templateOverrides = ref({})
const warrantyTemplateOverrides = ref({})
const previewImages = ref([])

// Section ordering for template tab
const SECTION_META = {
  images: { label: 'Gallery', icon: Image },
  videos: { label: 'Videos', icon: Video },
  social_accounts: { label: 'Social Media', icon: Share2 },
  certifications: { label: 'Certifications', icon: Award },
  website_link: { label: 'Website Button', icon: ExternalLink },
  description: { label: 'Product Description', icon: FileText },
  warranty_button: { label: 'Warranty Button', icon: Shield }
}
const TEMPLATE_SECTION_ORDER = [
  'images', 'videos', 'social_accounts', 'certifications',
  'website_link', 'description', 'warranty_button'
]
const sectionOrderList = ref([])

// Merged config for preview (template base + product overrides)
const mergedPreviewConfig = computed(() => {
  const base = selectedTemplateConfig.value?.custom_fields || {}
  const ov = templateOverrides.value || {}
  return {
    header: { ...(base.header || {}), ...(ov.header || {}) },
    styling: { ...(base.styling || {}), ...(ov.styling || {}) },
    certifications_section: { ...(base.certifications_section || {}), ...(ov.certifications_section || {}) },
    social_media_section: { ...(base.social_media_section || {}), ...(ov.social_media_section || {}) },
    warranty_button: { ...(base.warranty_button || {}), ...(ov.warranty_button || {}) },
    section_order: sectionOrderList.value.map(s => s.id)
  }
})

const previewBackgroundConfig = computed(() => selectedTemplateConfig.value?.background_config || null)

// Warranty template config for preview styling
const selectedWarrantyTemplateConfig = computed(() => {
  if (!selectedWarrantyTemplateId.value) return null
  const tmpl = warrantyTemplates.value.find(t => t.id === selectedWarrantyTemplateId.value)
  if (!tmpl) return null
  try {
    return typeof tmpl.custom_fields === 'string' ? JSON.parse(tmpl.custom_fields) : tmpl.custom_fields
  } catch {
    return null
  }
})

// Merged warranty config for preview (template base + product overrides)
const mergedWarrantyPreviewConfig = computed(() => {
  const base = selectedWarrantyTemplateConfig.value || {}
  const ov = warrantyTemplateOverrides.value || {}
  return {
    styling: { ...(base.styling || {}), ...(ov.styling || {}) },
    submit_button: { ...(base.submit_button || {}), ...(ov.submit_button || {}) },
    messages: { ...(base.messages || {}), ...(ov.messages || {}) },
  }
})

// Section data availability check
function sectionHasData(sectionId) {
  switch (sectionId) {
    case 'images': return previewImages.value.length > 1
    case 'videos': return videos.value.length > 0
    case 'social_accounts': return socialLinks.value.length > 0
    case 'certifications': return certifications.value.length > 0
    case 'website_link': return !!websiteUrl.value
    case 'description': return !!(product.value?.description && product.value.description.trim())
    case 'warranty_button': return !!product.value?.warranty_enabled
    default: return false
  }
}

function initSectionOrder() {
  const overrideOrder = templateOverrides.value?.section_order
  const templateOrder = selectedTemplateConfig.value?.custom_fields?.section_order
  const order = overrideOrder || templateOrder || TEMPLATE_SECTION_ORDER
  const known = new Set(TEMPLATE_SECTION_ORDER)
  const filtered = order.filter(s => known.has(s))
  const missing = TEMPLATE_SECTION_ORDER.filter(s => !filtered.includes(s))
  const finalOrder = [...new Set([...filtered, ...missing])]
  sectionOrderList.value = finalOrder.map(id => ({
    id,
    label: SECTION_META[id]?.label || id,
    icon: SECTION_META[id]?.icon || null
  }))
}

function onSectionDragEnd() {
  templateOverrides.value = {
    ...templateOverrides.value,
    section_order: sectionOrderList.value.map(s => s.id)
  }
}

// Use auth store for tenant brand name
import { useAuthStore } from '@/stores/auth'
const authStore = useAuthStore()

// Certifications
const certifications = ref([])
const previewCertifications = computed(() =>
  certifications.value.map(cert => ({
    name: cert.certification_type?.name,
    logo_url: cert.certification_type?.logo_url,
    country: cert.certification_type?.country?.name || 'International',
    registration_number: cert.registration_number,
    website_url: cert.certification_type?.website_url,
  }))
)
const availableCertTypes = ref([])
const loadingCerts = ref(false)
const showAddCertModal = ref(false)
const newCert = ref({ certification_type_id: '', registration_number: '' })

// Social Links
const socialLinks = ref([])
const availablePlatforms = ref([])
const loadingSocial = ref(false)
const showAddSocialModal = ref(false)
const newSocial = ref({ platform_id: '', handle_or_url: '' })


// Escape key to close modals
useEscapeKey(() => { showAddCertModal.value = false }, showAddCertModal)
useEscapeKey(() => { showAddSocialModal.value = false }, showAddSocialModal)

// Navigate back to products list
const goBack = () => {
  router.push('/tenant/products/dynamic')
}

async function fetchProduct() {
  loading.value = true
  try {
    const response = await get(`/tenant/products/${productId.value}`)
    if (response.success) {
      product.value = response.data
      form.value = {
        product_name: response.data.product_name,
        product_code: response.data.product_code || '',
        description: response.data.description || '',
        status: response.data.status,
        counterfeit_scan_max: response.data.counterfeit_scan_max || null
      }

      // Load display config
      if (response.data.display_config) {
        const dc = typeof response.data.display_config === 'string'
          ? JSON.parse(response.data.display_config)
          : response.data.display_config
        displayConfig.value = {
          product_name: dc.product_name ?? true,
          product_code: dc.product_code ?? false,
          batch_code: dc.batch_code ?? false,
          production_date: dc.production_date ?? false,
          expiry_date: dc.expiry_date ?? false,
          brand_name: dc.brand_name ?? true,
          show_verification_count: dc.show_verification_count ?? true,
          field_order: dc.field_order || DEFAULT_FIELD_ORDER
        }
      }

      // Load website and videos config
      websiteUrl.value = response.data.website_url || ''
      websiteCaption.value = response.data.website_caption || ''
      if (response.data.videos) {
        videos.value = typeof response.data.videos === 'string'
          ? JSON.parse(response.data.videos)
          : response.data.videos || []
      } else {
        videos.value = []
      }

      // Load warranty fields config - new structure
      if (response.data.warranty_fields_config) {
        const wfc = typeof response.data.warranty_fields_config === 'string'
          ? JSON.parse(response.data.warranty_fields_config)
          : response.data.warranty_fields_config
        warrantyFieldsConfig.value = {
          enabled: wfc.enabled ?? true,
          fields: {
            store_name: wfc.fields?.store_name ?? 'optional',
            country: wfc.fields?.country ?? 'required',
            province: wfc.fields?.province ?? 'required',
            city: wfc.fields?.city ?? 'required',
            address: wfc.fields?.address ?? 'required'
          }
        }
        // Load custom fields
        customFields.value = wfc.custom_fields || []
      }

      // Load warranty duration config
      warrantyMonths.value = response.data.warranty_months || 12
      maxWarrantyRegistrationDays.value = response.data.max_warranty_registration_days || null

      // Load template selections
      selectedValidationTemplateId.value = response.data.default_validation_template_id || ''
      selectedWarrantyTemplateId.value = response.data.default_warranty_template_id || ''

      // Load template overrides
      if (response.data.template_overrides) {
        templateOverrides.value = typeof response.data.template_overrides === 'string'
          ? JSON.parse(response.data.template_overrides)
          : response.data.template_overrides
      } else {
        templateOverrides.value = {}
      }

      // Load warranty template overrides
      if (response.data.warranty_template_overrides) {
        warrantyTemplateOverrides.value = typeof response.data.warranty_template_overrides === 'string'
          ? JSON.parse(response.data.warranty_template_overrides)
          : response.data.warranty_template_overrides
      } else {
        warrantyTemplateOverrides.value = {}
      }
    }
  } catch (error) {
    console.error('Failed to fetch product:', error)
  } finally {
    loading.value = false
  }
}

async function saveProduct() {
  saving.value = true
  try {
    // Normalize counterfeit_scan_max: empty/0 → 0 signals backend to reset to NULL (global)
    const payload = { ...form.value }
    if (!payload.counterfeit_scan_max || payload.counterfeit_scan_max < 1) {
      payload.counterfeit_scan_max = 0
    }
    const response = await put(`/tenant/products/${productId.value}`, payload)
    if (response.success) {
      product.value = { ...product.value, ...form.value }
      alert('Product saved successfully')
    } else {
      alert(response.message || 'Failed to save product')
    }
  } catch (error) {
    console.error('Failed to save product:', error)
    alert('Failed to save product')
  } finally {
    saving.value = false
  }
}

async function saveDisplayConfig() {
  savingConfig.value = true
  try {
    // Build overrides payload (only send non-empty)
    const hasOverrides = templateOverrides.value && Object.keys(templateOverrides.value).length > 0
    const response = await put(`/tenant/products/${productId.value}`, {
      display_config: displayConfig.value,
      default_validation_template_id: selectedValidationTemplateId.value || null,
      template_overrides: hasOverrides ? templateOverrides.value : null
    })
    if (response.success) {
      product.value = {
        ...product.value,
        display_config: displayConfig.value,
        default_validation_template_id: selectedValidationTemplateId.value || null,
        template_overrides: hasOverrides ? templateOverrides.value : null
      }
      alert('Landing page settings saved successfully')
    } else {
      alert(response.message || 'Failed to save landing page settings')
    }
  } catch (error) {
    console.error('Failed to save landing page settings:', error)
    alert('Failed to save landing page settings')
  } finally {
    savingConfig.value = false
  }
}

// NOTE: saveLandingAppearance removed - background config is now in Templates

async function saveVideosConfig() {
  savingVideos.value = true
  try {
    const response = await put(`/tenant/products/${productId.value}`, {
      videos: videos.value
    })
    if (response.success) {
      product.value = {
        ...product.value,
        videos: videos.value
      }
      alert('Videos saved successfully')
    } else {
      alert(response.message || 'Failed to save videos')
    }
  } catch (error) {
    console.error('Failed to save videos:', error)
    alert('Failed to save videos')
  } finally {
    savingVideos.value = false
  }
}

async function saveWebsiteConfig() {
  savingVideos.value = true
  try {
    const response = await put(`/tenant/products/${productId.value}`, {
      website_url: websiteUrl.value || null,
      website_caption: websiteCaption.value || null
    })
    if (response.success) {
      product.value = {
        ...product.value,
        website_url: websiteUrl.value,
        website_caption: websiteCaption.value
      }
      alert('Website link saved successfully')
    } else {
      alert(response.message || 'Failed to save website link')
    }
  } catch (error) {
    console.error('Failed to save website link:', error)
    alert('Failed to save website link')
  } finally {
    savingVideos.value = false
  }
}

async function saveWarrantyConfig() {
  // Validate before submit
  if (!isWarrantyConfigValid.value) {
    alert('Warranty period must be between 1 and 120 months')
    return
  }

  savingConfig.value = true
  try {
    // Filter out empty custom fields and clean options
    const cleanedCustomFields = customFields.value
      .filter(f => f.label && f.label.trim())
      .map(f => ({
        id: f.id || ('field_' + Date.now() + Math.random().toString(36).substr(2, 9)),
        label: f.label.trim(),
        type: f.type,
        required: f.required || false,
        options: f.type === 'select' ? (f.options || []).filter(o => o && o.trim()) : undefined
      }))

    const payload = {
      warranty_enabled: warrantyFieldsConfig.value.enabled,  // Send warranty_enabled to backend
      warranty_fields_config: {
        ...warrantyFieldsConfig.value,
        custom_fields: cleanedCustomFields
      },
      warranty_months: warrantyMonths.value || null, // Let backend use default if empty
      max_warranty_registration_days: maxWarrantyRegistrationDays.value || null
    }
    // Only save warranty template and overrides if warranty is enabled
    if (warrantyFieldsConfig.value.enabled) {
      payload.default_warranty_template_id = selectedWarrantyTemplateId.value || null
      const hasWarrantyOverrides = warrantyTemplateOverrides.value && Object.keys(warrantyTemplateOverrides.value).length > 0
      payload.warranty_template_overrides = hasWarrantyOverrides ? warrantyTemplateOverrides.value : null
    }
    const response = await put(`/tenant/products/${productId.value}`, payload)
    if (response.success) {
      product.value = {
        ...product.value,
        warranty_enabled: warrantyFieldsConfig.value.enabled,
        warranty_fields_config: warrantyFieldsConfig.value,
        warranty_months: warrantyMonths.value,
        max_warranty_registration_days: maxWarrantyRegistrationDays.value,
        default_warranty_template_id: warrantyFieldsConfig.value.enabled ? (selectedWarrantyTemplateId.value || null) : product.value.default_warranty_template_id
      }
      alert('Warranty settings saved successfully')
    } else {
      alert(response.message || 'Failed to save warranty settings')
    }
  } catch (error) {
    console.error('Failed to save warranty settings:', error)
    alert('Failed to save warranty settings')
  } finally {
    savingConfig.value = false
  }
}

// Templates
async function fetchTemplates() {
  try {
    const response = await get('/tenant/templates', { status: 'active' })
    if (response.success && response.data?.templates) {
      validationTemplates.value = response.data.templates.filter(t => t.template_type === 'validation')
      warrantyTemplates.value = response.data.templates.filter(t => t.template_type === 'warranty')
    }
  } catch (error) {
    console.error('Failed to fetch templates:', error)
  }
}

async function fetchTemplateConfig(templateId) {
  loadingTemplateConfig.value = true
  try {
    const response = await get(`/tenant/templates/${templateId}`)
    if (response.success && response.data) {
      const t = response.data
      const customFields = typeof t.custom_fields === 'string' ? JSON.parse(t.custom_fields) : t.custom_fields || {}
      const bgConfig = typeof t.background_config === 'string' ? JSON.parse(t.background_config) : t.background_config || null
      selectedTemplateConfig.value = { custom_fields: customFields, background_config: bgConfig }
      initSectionOrder()
    }
  } catch (error) {
    console.error('Failed to fetch template config:', error)
  } finally {
    loadingTemplateConfig.value = false
  }
}

async function fetchDefaultTemplateConfig() {
  loadingTemplateConfig.value = true
  try {
    // Find default template: first validation template in the list
    const valTemplates = validationTemplates.value
    if (valTemplates.length > 0) {
      await fetchTemplateConfig(valTemplates[0].id)
      return
    }
    selectedTemplateConfig.value = { custom_fields: {}, background_config: null }
    initSectionOrder()
  } catch (error) {
    console.error('Failed to fetch default template config:', error)
  } finally {
    loadingTemplateConfig.value = false
  }
}

function onTemplateChange() {
  if (selectedValidationTemplateId.value) {
    fetchTemplateConfig(selectedValidationTemplateId.value)
  } else {
    fetchDefaultTemplateConfig()
  }
}

async function fetchPreviewImages() {
  try {
    const response = await get(`/tenant/products/${productId.value}/images`)
    if (response.success && response.data?.images) {
      previewImages.value = response.data.images
    }
  } catch (error) {
    console.error('Failed to fetch preview images:', error)
  }
}

function onOverridesUpdate(newOverrides) {
  templateOverrides.value = newOverrides
}

// Advanced customization fields (inline expandable)
const OVERRIDE_FIELDS = [
  { group: 'Header', section: 'header', key: 'logo_enabled', label: 'Show Company Logo', type: 'toggle', fallback: false },
  { group: 'Header', section: 'header', key: 'bg_color', label: 'Header Background', type: 'color', fallback: '#3f3f46' },
  { group: 'Header', section: 'header', key: 'badge_text', label: 'Badge Text', type: 'text', fallback: 'Authentic Product' },
  { group: 'Header', section: 'header', key: 'badge_bg_color', label: 'Badge Background', type: 'color', fallback: '#22c55e' },
  { group: 'Header', section: 'header', key: 'badge_text_color', label: 'Badge Text Color', type: 'color', fallback: '#ffffff' },
  { group: 'Page Styling', section: 'styling', key: 'card_bg_color', label: 'Page Background', type: 'color', fallback: '#f3f4f6' },
  { group: 'Page Styling', section: 'styling', key: 'field_bg_color', label: 'Field Background', type: 'color', fallback: '#ffffff' },
  { group: 'Page Styling', section: 'styling', key: 'text_color', label: 'Text Color', type: 'color', fallback: '#1f2937' },
  { group: 'Page Styling', section: 'styling', key: 'main_image_size', label: 'Product Image Size', type: 'range', fallback: 96, min: 48, max: 128 },
  { group: 'Certifications', section: 'certifications_section', key: 'header_text', label: 'Title', type: 'text', fallback: 'Certifications' },
  { group: 'Certifications', section: 'certifications_section', key: 'icon_color', label: 'Icon Color', type: 'color', fallback: '#10b981' },
  { group: 'Certifications', section: 'certifications_section', key: 'bg_color', label: 'Background', type: 'color', fallback: '#f0fdf4' },
  { group: 'Certifications', section: 'certifications_section', key: 'default_expanded', label: 'Expanded by Default', type: 'toggle', fallback: false },
  { group: 'Social Media', section: 'social_media_section', key: 'header_text', label: 'Title', type: 'text', fallback: 'Follow Us' },
  { group: 'Social Media', section: 'social_media_section', key: 'icon_color', label: 'Icon Color', type: 'color', fallback: '#ec4899' },
  { group: 'Social Media', section: 'social_media_section', key: 'bg_color', label: 'Background', type: 'color', fallback: '#fdf2f8' },
  { group: 'Social Media', section: 'social_media_section', key: 'sticky_enabled', label: 'Sticky Bar', type: 'toggle', fallback: true },
  { group: 'Social Media', section: 'social_media_section', key: 'default_expanded', label: 'Expanded by Default', type: 'toggle', fallback: false },
  { group: 'Warranty Button', section: 'warranty_button', key: 'text', label: 'Button Text', type: 'text', fallback: 'Activate Warranty' },
  { group: 'Warranty Button', section: 'warranty_button', key: 'bg_color', label: 'Background', type: 'color', fallback: '#8b5cf6' },
  { group: 'Warranty Button', section: 'warranty_button', key: 'text_color', label: 'Text Color', type: 'color', fallback: '#ffffff' },
]

const groupedOverrideFields = computed(() => {
  const groups = {}
  for (const field of OVERRIDE_FIELDS) {
    const g = field.group
    if (!groups[g]) groups[g] = []
    groups[g].push(field)
  }
  return groups
})

const expandedGroups = ref({})

function toggleGroup(groupName) {
  expandedGroups.value = { ...expandedGroups.value, [groupName]: !expandedGroups.value[groupName] }
}

const baseConfig = computed(() => selectedTemplateConfig.value?.custom_fields || {})

function getEffective(section, key, fallback) {
  if (templateOverrides.value?.[section]?.[key] !== undefined) return templateOverrides.value[section][key]
  if (baseConfig.value?.[section]?.[key] !== undefined) return baseConfig.value[section][key]
  return fallback
}

function isFieldOverridden(section, key) {
  return templateOverrides.value?.[section]?.[key] !== undefined
}

function setOverride(section, key, value) {
  const current = { ...templateOverrides.value }
  if (!current[section]) current[section] = {}
  current[section] = { ...current[section], [key]: value }
  templateOverrides.value = current
}

function resetOverrideField(section, key) {
  const current = { ...templateOverrides.value }
  if (!current[section]) return
  const copy = { ...current[section] }
  delete copy[key]
  if (Object.keys(copy).length === 0) {
    delete current[section]
  } else {
    current[section] = copy
  }
  templateOverrides.value = current
}

function resetAllOverrides() {
  templateOverrides.value = {}
}

const overrideCount = computed(() => {
  let count = 0
  for (const section of Object.values(templateOverrides.value || {})) {
    if (section && typeof section === 'object' && !Array.isArray(section)) {
      count += Object.keys(section).length
    }
  }
  return count
})

// Warranty template override fields (lightweight subset - most common customizations)
const WARRANTY_OVERRIDE_FIELDS = [
  { group: 'Page Styling', section: 'styling', key: 'header_bg_color', label: 'Header Background', type: 'color', fallback: '#18181b' },
  { group: 'Page Styling', section: 'styling', key: 'form_bg_color', label: 'Form Background', type: 'color', fallback: '#ffffff' },
  { group: 'Page Styling', section: 'styling', key: 'text_color', label: 'Text Color', type: 'color', fallback: '#1f2937' },
  { group: 'Page Styling', section: 'styling', key: 'accent_color', label: 'Accent Color', type: 'color', fallback: '#18181b' },
  { group: 'Activate Button', section: 'submit_button', key: 'text', label: 'Button Text', type: 'text', fallback: 'Activate Warranty' },
  { group: 'Activate Button', section: 'submit_button', key: 'bg_color', label: 'Background', type: 'color', fallback: '#18181b' },
  { group: 'Activate Button', section: 'submit_button', key: 'text_color', label: 'Text Color', type: 'color', fallback: '#ffffff' },
]

const warrantyGroupedOverrideFields = computed(() => {
  const groups = {}
  for (const field of WARRANTY_OVERRIDE_FIELDS) {
    const g = field.group
    if (!groups[g]) groups[g] = []
    groups[g].push(field)
  }
  return groups
})

const warrantyExpandedGroups = ref({})

function toggleWarrantyGroup(groupName) {
  warrantyExpandedGroups.value = { ...warrantyExpandedGroups.value, [groupName]: !warrantyExpandedGroups.value[groupName] }
}

const warrantyBaseConfig = computed(() => selectedWarrantyTemplateConfig.value || {})

function getWarrantyEffective(section, key, fallback) {
  if (warrantyTemplateOverrides.value?.[section]?.[key] !== undefined) return warrantyTemplateOverrides.value[section][key]
  if (warrantyBaseConfig.value?.[section]?.[key] !== undefined) return warrantyBaseConfig.value[section][key]
  return fallback
}

function isWarrantyFieldOverridden(section, key) {
  return warrantyTemplateOverrides.value?.[section]?.[key] !== undefined
}

function setWarrantyOverride(section, key, value) {
  const current = { ...warrantyTemplateOverrides.value }
  if (!current[section]) current[section] = {}
  current[section] = { ...current[section], [key]: value }
  warrantyTemplateOverrides.value = current
}

function resetWarrantyOverrideField(section, key) {
  const current = { ...warrantyTemplateOverrides.value }
  if (!current[section]) return
  const copy = { ...current[section] }
  delete copy[key]
  if (Object.keys(copy).length === 0) {
    delete current[section]
  } else {
    current[section] = copy
  }
  warrantyTemplateOverrides.value = current
}

function resetAllWarrantyOverrides() {
  warrantyTemplateOverrides.value = {}
}

const warrantyOverrideCount = computed(() => {
  let count = 0
  for (const section of Object.values(warrantyTemplateOverrides.value || {})) {
    if (section && typeof section === 'object' && !Array.isArray(section)) {
      count += Object.keys(section).length
    }
  }
  return count
})

// Certifications
async function fetchCertifications() {
  loadingCerts.value = true
  try {
    const response = await get(`/tenant/certifications/products/${productId.value}`)
    if (response.success) {
      certifications.value = response.data || []
    }
  } catch (error) {
    console.error('Failed to fetch certifications:', error)
  } finally {
    loadingCerts.value = false
  }
}

async function fetchAvailableCertTypes() {
  try {
    const response = await get('/tenant/certifications/types')
    if (response.success) {
      availableCertTypes.value = response.data || []
    }
  } catch (error) {
    console.error('Failed to fetch cert types:', error)
  }
}

async function addCertification() {
  if (!newCert.value.certification_type_id || !newCert.value.registration_number) return

  try {
    const response = await post(`/tenant/certifications/products/${productId.value}`, newCert.value)
    if (response.success) {
      showAddCertModal.value = false
      newCert.value = { certification_type_id: '', registration_number: '' }
      fetchCertifications()
    } else {
      alert(response.message || 'Failed to add certification')
    }
  } catch (error) {
    console.error('Failed to add certification:', error)
    alert('Failed to add certification')
  }
}

async function removeCertification(certId) {
  if (!confirm('Remove this certification?')) return

  try {
    const response = await del(`/tenant/certifications/products/${productId.value}/${certId}`)
    if (response.success) {
      fetchCertifications()
    }
  } catch (error) {
    console.error('Failed to remove certification:', error)
  }
}

async function reorderCertifications(certIds) {
  const originalOrder = [...certifications.value]
  try {
    await put(`/tenant/certifications/products/${productId.value}/reorder`, {
      cert_ids: certIds
    })
  } catch (error) {
    console.error('Failed to reorder certifications:', error)
    certifications.value = originalOrder
  }
}

function onCertDragEnd() {
  reorderCertifications(certifications.value.map(c => c.id))
}

// Social Links
async function fetchSocialLinks() {
  loadingSocial.value = true
  try {
    const response = await get(`/tenant/social-media/products/${productId.value}`)
    if (response.success) {
      socialLinks.value = response.data || []
    }
  } catch (error) {
    console.error('Failed to fetch social links:', error)
  } finally {
    loadingSocial.value = false
  }
}

async function fetchAvailablePlatforms() {
  try {
    const response = await get('/tenant/social-media/platforms')
    if (response.success) {
      availablePlatforms.value = response.data || []
    }
  } catch (error) {
    console.error('Failed to fetch platforms:', error)
  }
}

async function addSocialLink() {
  if (!newSocial.value.platform_id || !newSocial.value.handle_or_url) return

  try {
    const response = await post(`/tenant/social-media/products/${productId.value}`, newSocial.value)
    if (response.success) {
      showAddSocialModal.value = false
      newSocial.value = { platform_id: '', handle_or_url: '' }
      fetchSocialLinks()
    } else {
      alert(response.message || 'Failed to add social link')
    }
  } catch (error) {
    console.error('Failed to add social link:', error)
    alert('Failed to add social link')
  }
}

async function removeSocialLink(linkId) {
  if (!confirm('Remove this social link?')) return

  try {
    const response = await del(`/tenant/social-media/products/${productId.value}/${linkId}`)
    if (response.success) {
      fetchSocialLinks()
    }
  } catch (error) {
    console.error('Failed to remove social link:', error)
  }
}


function getSelectedPlaceholder() {
  const platform = availablePlatforms.value.find(p => p.id === newSocial.value.platform_id)
  return platform?.placeholder_text || 'Enter handle or URL'
}

// Filter out already added items
const unusedCertTypes = computed(() => {
  const usedIds = certifications.value.map(c => c.certification_type_id)
  return availableCertTypes.value.filter(t => !usedIds.includes(t.id))
})

const unusedPlatforms = computed(() => {
  const usedIds = socialLinks.value.map(l => l.platform_id)
  return availablePlatforms.value.filter(p => !usedIds.includes(p.id))
})

// Watch for route param changes (handles navigation without full remount)
watch(productId, (newId, oldId) => {
  if (newId && newId !== oldId) {
    // Re-fetch product-specific data
    fetchProduct()
    fetchCertifications()
    fetchSocialLinks()
  }
})

const fetchCounterfeitSettings = async () => {
  try {
    const response = await get('/tenant/counterfeit/settings')
    if (response.success && response.data) {
      globalEndUserScanMax.value = response.data.end_user_scan_max ?? 3
    }
  } catch (error) {
    console.error('Failed to fetch counterfeit settings:', error)
  }
}

// Tour support
const tour = useTour()

function closeAllModals() {
  showAddCertModal.value = false
  showAddSocialModal.value = false
}

function handleTourSetValue(e) {
  if (!isTourActive()) return
  if (e.detail._nonce !== getTourNonce()) return
  const { field, value } = e.detail
  switch (field) {
    case 'product_code':
      form.value.product_code = value
      break
    case 'tab_switch':
      setActiveTab(value)
      break
    case 'detail_cert_type_id': {
      // Find cert type by partial name match (e.g. 'haccp')
      const match = unusedCertTypes.value.find(t =>
        t.name.toLowerCase().includes(value.toLowerCase())
      )
      if (match) newCert.value.certification_type_id = match.id
      break
    }
    case 'detail_cert_reg_number':
      newCert.value.registration_number = value
      break
    case 'validation_template': {
      const tmpl = validationTemplates.value.find(t =>
        t.template_name.toLowerCase().includes(value.toLowerCase())
      )
      if (tmpl) {
        selectedValidationTemplateId.value = tmpl.id
        onTemplateChange()
      }
      break
    }
    case 'display_product_code':
      displayConfig.value.product_code = value
      break
    case 'display_field_order':
      displayConfig.value.field_order = value
      break
    case 'section_order': {
      const known = new Set(TEMPLATE_SECTION_ORDER)
      const filtered = value.filter(s => known.has(s))
      sectionOrderList.value = filtered.map(id => ({
        id,
        label: SECTION_META[id]?.label || id,
        icon: SECTION_META[id]?.icon || null
      }))
      templateOverrides.value = {
        ...templateOverrides.value,
        section_order: filtered
      }
      break
    }
    case 'warranty_toggle':
      warrantyFieldsConfig.value.enabled = value
      break
    case 'warranty_period':
      warrantyMonths.value = value
      break
    case 'warranty_reg_days':
      maxWarrantyRegistrationDays.value = value
      break
    case 'warranty_template': {
      const wt = warrantyTemplates.value.find(t =>
        t.template_name.toLowerCase().includes(value.toLowerCase())
      )
      if (wt) selectedWarrantyTemplateId.value = wt.id
      break
    }
  }
}

// Interval/timeout references for cleanup
let templateCheckInterval = null
let templateCheckTimeout = null

onBeforeUnmount(() => {
  if (templateCheckInterval) clearInterval(templateCheckInterval)
  if (templateCheckTimeout) clearTimeout(templateCheckTimeout)
  window.removeEventListener('tour-cancelled', closeAllModals)
  window.removeEventListener('tour-set-value', handleTourSetValue)
})

onMounted(async () => {
  fetchProduct().then(() => {
    // After product loads, fetch template config for preview
    if (selectedValidationTemplateId.value) {
      fetchTemplateConfig(selectedValidationTemplateId.value)
    } else {
      // Wait for templates to load then fetch default
      templateCheckInterval = setInterval(() => {
        if (validationTemplates.value.length > 0) {
          clearInterval(templateCheckInterval)
          templateCheckInterval = null
          fetchDefaultTemplateConfig()
        }
      }, 100)
      // Timeout after 5s
      templateCheckTimeout = setTimeout(() => {
        if (templateCheckInterval) clearInterval(templateCheckInterval)
        templateCheckInterval = null
      }, 5000)
    }
  })
  fetchTemplates()
  fetchCertifications()
  fetchSocialLinks()
  fetchPreviewImages()
  fetchAvailableCertTypes()
  fetchAvailablePlatforms()
  fetchCounterfeitSettings()
  // Tour support
  window.addEventListener('tour-cancelled', closeAllModals)
  window.addEventListener('tour-set-value', handleTourSetValue)
  tour.resumeIfActive()
})

// Redirect ?tab=display to ?tab=template
watch(() => route.query.tab, (newTab) => {
  if (newTab === 'display') {
    router.replace({ query: { ...route.query, tab: 'template' } })
  }
}, { immediate: true })
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <div class="flex items-center gap-4">
        <button
          @click="goBack"
          class="p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors"
        >
          <ArrowLeft class="w-5 h-5" />
        </button>
        <div>
          <div class="flex items-center gap-3">
            <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ product?.product_name || 'Loading...' }}</h1>
            <span
              v-if="product"
              class="px-2 py-1 text-xs rounded-full bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400"
            >
              Dynamic QR
            </span>
          </div>
          <p v-if="product?.product_code" class="text-sm text-gray-500 dark:text-gray-400">Code: {{ product.product_code }}</p>
        </div>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <div v-else>
      <!-- Tabs -->
      <div class="flex flex-wrap border-b border-gray-200 dark:border-gray-700 mb-6">
        <button
          @click="setActiveTab('info')"
          :class="[
            'px-4 py-2 font-medium text-sm border-b-2 transition-colors',
            activeTab === 'info'
              ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'
          ]"
        >
          Basic Info
        </button>
        <button
          @click="setActiveTab('certifications')"
          data-tour="tab-certifications"
          :class="[
            'px-4 py-2 font-medium text-sm border-b-2 transition-colors',
            activeTab === 'certifications'
              ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'
          ]"
        >
          Certifications
          <span v-if="certifications.length" class="ml-1 px-1.5 py-0.5 text-xs rounded-full bg-gray-100 dark:bg-gray-700">
            {{ certifications.length }}
          </span>
        </button>
        <button
          @click="setActiveTab('gallery')"
          data-tour="tab-gallery"
          :class="[
            'px-4 py-2 font-medium text-sm border-b-2 transition-colors',
            activeTab === 'gallery'
              ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'
          ]"
        >
          Gallery
        </button>
        <button
          @click="setActiveTab('videos')"
          data-tour="tab-videos"
          :class="[
            'px-4 py-2 font-medium text-sm border-b-2 transition-colors',
            activeTab === 'videos'
              ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'
          ]"
        >
          Videos
        </button>
        <button
          @click="setActiveTab('links')"
          data-tour="tab-social"
          :class="[
            'px-4 py-2 font-medium text-sm border-b-2 transition-colors',
            activeTab === 'links'
              ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'
          ]"
        >
          Social Media
        </button>
        <button
          @click="setActiveTab('template')"
          data-tour="tab-template"
          :class="[
            'px-4 py-2 font-medium text-sm border-b-2 transition-colors',
            activeTab === 'template'
              ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'
          ]"
        >
          Landing Page Template
        </button>
        <button
          @click="setActiveTab('warranty')"
          data-tour="tab-warranty"
          :class="[
            'px-4 py-2 font-medium text-sm border-b-2 transition-colors',
            activeTab === 'warranty'
              ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'
          ]"
        >
          Warranty Settings
        </button>
      </div>

      <!-- Product Info Tab -->
      <div v-if="activeTab === 'info'">
        <Card class="p-6">
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Product Name *</label>
              <Input v-model="form.product_name" required />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Product Code</label>
              <Input v-model="form.product_code" data-tour="product-code-input" />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
              <textarea
                v-model="form.description"
                rows="3"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
              ></textarea>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Status</label>
              <select
                v-model="form.status"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
              >
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
              </select>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">End User Scan Limit</label>
              <Input
                v-model.number="form.counterfeit_scan_max"
                type="number"
                min="1"
                placeholder="Use global setting"
              />
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                <template v-if="globalEndUserScanMax > 0">
                  Current global setting: {{ globalEndUserScanMax }} scans. Leave empty to use global setting.
                </template>
                <template v-else>
                  Global counterfeit detection is currently disabled. Set a value here to enable it for this product.
                </template>
              </p>
            </div>
            <div class="pt-4">
              <Button @click="saveProduct" :disabled="saving" data-tour="save-basic-info">
                {{ saving ? 'Saving...' : 'Save Changes' }}
              </Button>
            </div>
          </div>
        </Card>
      </div>

      <!-- Landing Page Template Tab (merged with Display) -->
      <div v-if="activeTab === 'template'">
        <div class="grid lg:grid-cols-[1fr,380px] gap-6">

          <!-- LEFT: Settings Panel -->
          <div class="space-y-6">
            <!-- Template Selector -->
            <Card class="p-6">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-1">Landing Page Template</h2>
              <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                Select and customize the template for the product validation page.
              </p>
              <select
                v-model="selectedValidationTemplateId"
                @change="onTemplateChange"
                data-tour="template-select"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
              >
                <option value="">Use tenant default</option>
                <option v-for="template in validationTemplates" :key="template.id" :value="template.id">
                  {{ template.template_name }}
                </option>
              </select>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
                <router-link v-if="selectedValidationTemplateId" :to="`/tenant/templates/${selectedValidationTemplateId}`" class="text-zinc-600 dark:text-zinc-400 hover:underline">
                  Edit Selected Template
                </router-link>
                <span v-if="selectedValidationTemplateId" class="mx-1">&middot;</span>
                <router-link to="/tenant/templates?type=validation" class="text-zinc-600 dark:text-zinc-400 hover:underline">
                  Manage Templates
                </router-link>
              </p>
            </Card>

            <!-- Display Fields (moved from old Display tab) -->
            <Card class="p-6" data-tour="display-fields">
              <div class="mb-4">
                <h3 class="font-semibold text-gray-900 dark:text-white">Display Fields</h3>
                <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  Select which fields to show. Drag to reorder.
                </p>
              </div>
              <VueDraggable
                v-model="orderedFields"
                :animation="150"
                handle=".field-drag-handle"
                ghost-class="opacity-50"
                class="space-y-2"
              >
                <label v-for="field in orderedFields" :key="field.key"
                  class="flex items-center gap-3 p-2.5 bg-gray-50 dark:bg-gray-700 rounded-lg transition-colors"
                  :class="field.required ? 'opacity-60' : 'cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600'"
                >
                  <div class="field-drag-handle cursor-grab active:cursor-grabbing text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
                    <GripVertical class="w-4 h-4" />
                  </div>
                  <input type="checkbox"
                    :checked="field.required || displayConfig[field.key]"
                    :disabled="field.required"
                    @change="!field.required && (displayConfig[field.key] = $event.target.checked)"
                    class="w-4 h-4 text-zinc-600 rounded" />
                  <div class="min-w-0">
                    <span class="font-medium text-sm text-gray-900 dark:text-white">{{ field.label }}</span>
                    <span v-if="field.required" class="ml-2 text-xs px-1.5 py-0.5 rounded bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400">Required</span>
                  </div>
                </label>
              </VueDraggable>
            </Card>

            <!-- Section Order -->
            <Card class="p-6" data-tour="section-order">
              <div class="mb-4">
                <h3 class="font-semibold text-gray-900 dark:text-white">Section Order</h3>
                <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                  Drag to reorder sections on the landing page.
                </p>
              </div>
              <VueDraggable
                v-model="sectionOrderList"
                :animation="150"
                handle=".section-drag-handle"
                ghost-class="opacity-50"
                @end="onSectionDragEnd"
                class="space-y-2"
              >
                <div v-for="section in sectionOrderList" :key="section.id"
                  class="flex items-center gap-3 p-2.5 bg-gray-50 dark:bg-gray-700 rounded-lg"
                  :class="{ 'opacity-50': !sectionHasData(section.id) }"
                >
                  <div class="section-drag-handle cursor-grab active:cursor-grabbing text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
                    <GripVertical class="w-4 h-4" />
                  </div>
                  <component v-if="section.icon" :is="section.icon" class="w-4 h-4 text-gray-500 dark:text-gray-400" />
                  <span class="font-medium text-sm text-gray-900 dark:text-white">{{ section.label }}</span>
                  <span v-if="!sectionHasData(section.id)" class="text-xs text-gray-400 dark:text-gray-500 ml-auto">(no data)</span>
                </div>
              </VueDraggable>
            </Card>

            <!-- Advanced Customization (Expandable Sections) -->
            <Card class="overflow-hidden" data-tour="advanced-customization">
              <div class="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">Advanced Customization</h3>
                <button
                  v-if="overrideCount > 0"
                  @click="resetAllOverrides"
                  class="flex items-center gap-1 text-xs text-red-500 hover:text-red-600 dark:text-red-400 dark:hover:text-red-300 transition-colors"
                >
                  <RotateCcw class="w-3 h-3" />
                  Reset All ({{ overrideCount }})
                </button>
              </div>

              <div v-for="(fields, groupName) in groupedOverrideFields" :key="groupName" class="border-b border-gray-100 dark:border-gray-700 last:border-b-0">
                <button
                  @click="toggleGroup(groupName)"
                  class="w-full flex items-center justify-between px-4 py-3 text-left hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
                >
                  <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ groupName }}</span>
                  <ChevronDown class="w-4 h-4 text-gray-400 transition-transform" :class="{ 'rotate-180': expandedGroups[groupName] }" />
                </button>

                <div v-if="expandedGroups[groupName]" class="px-4 pb-4 space-y-3">
                  <div v-for="field in fields" :key="`${field.section}-${field.key}`" class="space-y-1.5">
                    <div class="flex items-center justify-between">
                      <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ field.label }}</label>
                      <button
                        v-if="isFieldOverridden(field.section, field.key)"
                        @click="resetOverrideField(field.section, field.key)"
                        class="text-xs text-gray-400 hover:text-red-500 dark:hover:text-red-400 transition-colors flex items-center gap-1"
                      >
                        <RotateCcw class="w-3 h-3" />
                        Reset
                      </button>
                    </div>

                    <!-- Text -->
                    <Input
                      v-if="field.type === 'text'"
                      :modelValue="getEffective(field.section, field.key, field.fallback)"
                      @update:modelValue="setOverride(field.section, field.key, $event)"
                      class="w-full text-sm"
                      :class="{ 'ring-2 ring-zinc-300 dark:ring-zinc-700': isFieldOverridden(field.section, field.key) }"
                    />

                    <!-- Color -->
                    <div v-else-if="field.type === 'color'" class="flex items-center gap-2">
                      <input
                        type="color"
                        :value="getEffective(field.section, field.key, field.fallback)"
                        @input="setOverride(field.section, field.key, $event.target.value)"
                        class="w-8 h-8 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                      />
                      <Input
                        :modelValue="getEffective(field.section, field.key, field.fallback)"
                        @update:modelValue="setOverride(field.section, field.key, $event)"
                        class="flex-1 font-mono text-xs"
                        :class="{ 'ring-2 ring-zinc-300 dark:ring-zinc-700': isFieldOverridden(field.section, field.key) }"
                      />
                    </div>

                    <!-- Range -->
                    <div v-else-if="field.type === 'range'" class="flex items-center gap-3">
                      <input
                        type="range"
                        :min="field.min"
                        :max="field.max"
                        :value="getEffective(field.section, field.key, field.fallback)"
                        @input="setOverride(field.section, field.key, Number($event.target.value))"
                        class="flex-1 accent-zinc-500"
                      />
                      <span class="text-xs font-mono text-gray-500 dark:text-gray-400 w-10 text-right"
                        :class="{ 'text-zinc-600 dark:text-zinc-400 font-semibold': isFieldOverridden(field.section, field.key) }"
                      >
                        {{ getEffective(field.section, field.key, field.fallback) }}px
                      </span>
                    </div>

                    <!-- Toggle -->
                    <div v-else-if="field.type === 'toggle'" class="flex items-center gap-2">
                      <button
                        @click="setOverride(field.section, field.key, !getEffective(field.section, field.key, field.fallback))"
                        class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none"
                        :class="getEffective(field.section, field.key, field.fallback) ? 'bg-zinc-500' : 'bg-gray-300 dark:bg-gray-600'"
                      >
                        <span
                          class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
                          :class="getEffective(field.section, field.key, field.fallback) ? 'translate-x-5' : 'translate-x-0'"
                        />
                      </button>
                      <span class="text-xs text-gray-500 dark:text-gray-400"
                        :class="{ 'text-zinc-600 dark:text-zinc-400': isFieldOverridden(field.section, field.key) }"
                      >
                        {{ getEffective(field.section, field.key, field.fallback) ? 'On' : 'Off' }}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </Card>

            <!-- Save Button -->
            <Button @click="saveDisplayConfig" :disabled="savingConfig" class="w-full" data-tour="save-template">
              {{ savingConfig ? 'Saving...' : 'Save Landing Page Settings' }}
            </Button>
          </div>

          <!-- RIGHT: Sticky Live Preview -->
          <div class="lg:sticky lg:top-4 h-fit">
            <ProductTemplatePreview
              :config="mergedPreviewConfig"
              :background-config="previewBackgroundConfig"
              :product-name="product?.product_name"
              :product-code="product?.product_code"
              :brand-name="authStore.user?.tenant_name || ''"
              :description="product?.description || ''"
              :website-url="websiteUrl"
              :website-caption="websiteCaption"
              :images="previewImages"
              :videos="videos"
              :certifications="previewCertifications"
              :social-accounts="socialLinks"
              :display-config="displayConfig"
              :warranty-enabled="product?.warranty_enabled || false"
              :section-order="sectionOrderList.map(s => s.id)"
              :loading="loadingTemplateConfig"
            />
          </div>
        </div>

      </div>

      <!-- Certifications Tab -->
      <div v-if="activeTab === 'certifications'">
        <Card class="p-6">
          <div class="flex justify-between items-center mb-4">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Product Certifications</h2>
            <div class="flex gap-2">
              <Button size="sm" @click="showAddCertModal = true" :disabled="unusedCertTypes.length === 0" data-tour="add-cert-btn-detail">
                Add Certification
              </Button>
            </div>
          </div>

          <div v-if="loadingCerts" class="text-center py-8 text-gray-500">Loading...</div>

          <div v-else-if="certifications.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
            No certifications added yet. Add certifications to display them on the product page.
          </div>

          <VueDraggable
            v-else
            v-model="certifications"
            :animation="150"
            handle=".drag-handle"
            ghost-class="opacity-50"
            class="space-y-3"
            @end="onCertDragEnd"
          >
            <div
              v-for="cert in certifications"
              :key="cert.id"
              class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg"
            >
              <div class="flex items-center gap-3">
                <div class="drag-handle cursor-grab active:cursor-grabbing p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
                  <GripVertical class="w-4 h-4" />
                </div>
                <div v-if="cert.certification_type?.logo_url" class="w-10 h-10 flex-shrink-0 bg-white dark:bg-gray-600 rounded p-1">
                  <img
                    :src="cert.certification_type.logo_url"
                    :alt="cert.certification_type?.name"
                    class="w-full h-full object-contain"
                  />
                </div>
                <div>
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-gray-900 dark:text-white">{{ cert.certification_type?.name }}</span>
                    <span class="text-xs px-2 py-0.5 rounded bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400">
                      {{ cert.certification_type?.country?.name || 'International' }}
                    </span>
                  </div>
                  <p class="text-sm text-gray-600 dark:text-gray-300">Reg. No: {{ cert.registration_number }}</p>
                </div>
              </div>
              <Button
                variant="outline"
                size="sm"
                class="text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
                @click="removeCertification(cert.id)"
              >
                Remove
              </Button>
            </div>
          </VueDraggable>
        </Card>
      </div>

      <!-- Gallery Tab -->
      <div v-if="activeTab === 'gallery'">
        <Card class="p-6">
          <ProductImageGallery :product-id="productId" />
        </Card>
      </div>

      <!-- Videos Tab -->
      <div v-if="activeTab === 'videos'">
        <Card class="p-6">
          <div class="space-y-6">
            <!-- Video Embeds Section -->
            <VideoEmbedEditor v-model="videos" />

            <!-- Save Button -->
            <div class="pt-4 border-t border-gray-200 dark:border-gray-600">
              <Button @click="saveVideosConfig" :disabled="savingVideos">
                {{ savingVideos ? 'Saving...' : 'Save Videos' }}
              </Button>
            </div>
          </div>
        </Card>
      </div>

      <!-- Social Media Tab (Website + Social Accounts) -->
      <div v-if="activeTab === 'links'">
        <Card class="p-6">
          <div class="space-y-6">
            <!-- Website Link Section -->
            <div class="pb-6 border-b border-gray-200 dark:border-gray-600">
              <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-1">Website Link</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
                Add a link to your website or online store. This will appear as a button on the landing page.
              </p>
              <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Website URL
                  </label>
                  <Input
                    v-model="websiteUrl"
                    type="url"
                    placeholder="https://example.com"
                  />
                </div>
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Button Text
                  </label>
                  <Input
                    v-model="websiteCaption"
                    placeholder="Visit Our Store"
                    maxlength="100"
                  />
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Default: "Visit Website" if left empty
                  </p>
                </div>
              </div>
              <!-- Save Website Button -->
              <div class="mt-4">
                <Button @click="saveWebsiteConfig" :disabled="savingVideos">
                  {{ savingVideos ? 'Saving...' : 'Save Website Link' }}
                </Button>
              </div>
            </div>

            <!-- Social Media Section -->
            <div>
              <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-1">Social Media</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
                Link your social media accounts to display on the landing page.
              </p>
              <ProductSocialAccountEditor :product-id="productId" />
            </div>
          </div>
        </Card>
      </div>

      <!-- Warranty Settings Tab -->
      <div v-if="activeTab === 'warranty'">
        <div class="grid lg:grid-cols-[1fr,380px] gap-6">
        <Card class="p-6">
          <!-- Enable/Disable Toggle -->
          <div class="flex items-center justify-between mb-6 pb-4 border-b border-gray-200 dark:border-gray-600">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Warranty Registration</h2>
              <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                Enable warranty registration for this product
              </p>
            </div>
            <button
              @click="warrantyFieldsConfig.enabled = !warrantyFieldsConfig.enabled"
              data-tour="warranty-toggle"
              :class="[
                'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                warrantyFieldsConfig.enabled ? 'bg-zinc-600' : 'bg-gray-300 dark:bg-gray-600'
              ]"
            >
              <span
                :class="[
                  'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                  warrantyFieldsConfig.enabled ? 'translate-x-6' : 'translate-x-1'
                ]"
              />
            </button>
          </div>

          <div v-if="warrantyFieldsConfig.enabled">
            <!-- Warranty Duration Settings -->
            <div class="mb-6 pb-6 border-b border-gray-200 dark:border-gray-600">
              <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">Warranty Duration</h3>
              <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Warranty Period (months)
                  </label>
                  <Input
                    v-model.number="warrantyMonths"
                    type="number"
                    min="1"
                    max="120"
                    placeholder="12"
                    data-tour="warranty-period"
                  />
                  <p v-if="warrantyMonths !== null && warrantyMonths !== '' && (warrantyMonths < 1 || warrantyMonths > 120)"
                     class="text-xs text-red-600 dark:text-red-400 mt-1">
                    Warranty period must be between 1 and 120 months
                  </p>
                  <p v-else class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Default: 12 months. How long the warranty lasts after purchase.
                  </p>
                </div>
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Max Registration Days
                  </label>
                  <Input
                    v-model.number="maxWarrantyRegistrationDays"
                    type="number"
                    min="0"
                    max="365"
                    placeholder="Leave empty for unlimited"
                    data-tour="warranty-reg-days"
                  />
                  <p v-if="maxWarrantyRegistrationDays !== null && maxWarrantyRegistrationDays !== '' && (maxWarrantyRegistrationDays < 0 || maxWarrantyRegistrationDays > 365)"
                     class="text-xs text-red-600 dark:text-red-400 mt-1">
                    Max registration days must be between 0 and 365
                  </p>
                  <p v-else class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Days after purchase within which warranty can be registered. Leave empty for no limit.
                  </p>
                </div>
              </div>
            </div>

            <!-- Warranty Template Selection -->
            <div class="mb-6 pb-6 border-b border-gray-200 dark:border-gray-600">
              <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">Warranty Page Template</h3>
              <select
                v-model="selectedWarrantyTemplateId"
                data-tour="warranty-template-select"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
              >
                <option value="">Use tenant default</option>
                <option v-for="template in warrantyTemplates" :key="template.id" :value="template.id">
                  {{ template.template_name }}
                </option>
              </select>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
                The page shown when customers register their warranty.
                <router-link v-if="selectedWarrantyTemplateId" :to="`/tenant/templates/${selectedWarrantyTemplateId}`" class="text-zinc-600 dark:text-zinc-400 hover:underline ml-1">
                  Edit Selected Template
                </router-link>
                <span v-if="selectedWarrantyTemplateId" class="mx-1">&middot;</span>
                <router-link to="/tenant/templates?type=warranty" class="text-zinc-600 dark:text-zinc-400 hover:underline" :class="{ 'ml-1': !selectedWarrantyTemplateId }">
                  Manage Templates
                </router-link>
              </p>
            </div>

            <!-- Fixed Required Fields Section -->
            <div class="mb-6">
              <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">Required Fields (always collected)</h3>
              <div class="grid grid-cols-2 gap-2">
                <div class="flex items-center gap-2 p-2 bg-gray-50 dark:bg-gray-700 rounded opacity-60">
                  <svg class="w-4 h-4 text-zinc-600" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                  </svg>
                  <span class="text-sm text-gray-900 dark:text-white">Full Name</span>
                  <span class="ml-auto text-xs px-1.5 py-0.5 rounded bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400">Required</span>
                </div>
                <div class="flex items-center gap-2 p-2 bg-gray-50 dark:bg-gray-700 rounded opacity-60">
                  <svg class="w-4 h-4 text-zinc-600" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                  </svg>
                  <span class="text-sm text-gray-900 dark:text-white">Email</span>
                  <span class="ml-auto text-xs px-1.5 py-0.5 rounded bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400">Required</span>
                </div>
                <div class="flex items-center gap-2 p-2 bg-gray-50 dark:bg-gray-700 rounded opacity-60">
                  <svg class="w-4 h-4 text-zinc-600" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                  </svg>
                  <span class="text-sm text-gray-900 dark:text-white">Phone</span>
                  <span class="ml-auto text-xs px-1.5 py-0.5 rounded bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400">Required</span>
                </div>
                <div class="flex items-center gap-2 p-2 bg-gray-50 dark:bg-gray-700 rounded opacity-60">
                  <svg class="w-4 h-4 text-zinc-600" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                  </svg>
                  <span class="text-sm text-gray-900 dark:text-white">Purchase Date</span>
                  <span class="ml-auto text-xs px-1.5 py-0.5 rounded bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400">Required</span>
                </div>
              </div>
            </div>

            <!-- Customizable Fields Section -->
            <div>
              <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">Customizable Fields</h3>
              <p class="text-xs text-gray-500 dark:text-gray-400 mb-4">
                Configure which additional fields to collect and whether they are optional or required.
              </p>
              <div class="space-y-3">
                <!-- Store Name -->
                <div class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                  <span class="font-medium text-gray-900 dark:text-white">Store Name</span>
                  <select
                    v-model="warrantyFieldsConfig.fields.store_name"
                    class="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
                  >
                    <option value="hidden">Hidden</option>
                    <option value="optional">Visible (Optional)</option>
                    <option value="required">Visible (Required)</option>
                  </select>
                </div>

                <!-- Country -->
                <div class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                  <span class="font-medium text-gray-900 dark:text-white">Country</span>
                  <select
                    v-model="warrantyFieldsConfig.fields.country"
                    class="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
                  >
                    <option value="hidden">Hidden</option>
                    <option value="optional">Visible (Optional)</option>
                    <option value="required">Visible (Required)</option>
                  </select>
                </div>

                <!-- Province -->
                <div class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                  <div>
                    <span class="font-medium text-gray-900 dark:text-white">Province</span>
                    <p class="text-xs text-gray-500 dark:text-gray-400">Requires Country to be visible</p>
                  </div>
                  <select
                    v-model="warrantyFieldsConfig.fields.province"
                    :disabled="warrantyFieldsConfig.fields.country === 'hidden'"
                    class="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <option value="hidden">Hidden</option>
                    <option value="optional">Visible (Optional)</option>
                    <option value="required">Visible (Required)</option>
                  </select>
                </div>

                <!-- City -->
                <div class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                  <div>
                    <span class="font-medium text-gray-900 dark:text-white">City</span>
                    <p class="text-xs text-gray-500 dark:text-gray-400">Requires Province to be visible</p>
                  </div>
                  <select
                    v-model="warrantyFieldsConfig.fields.city"
                    :disabled="warrantyFieldsConfig.fields.province === 'hidden'"
                    class="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <option value="hidden">Hidden</option>
                    <option value="optional">Visible (Optional)</option>
                    <option value="required">Visible (Required)</option>
                  </select>
                </div>

                <!-- Full Address -->
                <div class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                  <span class="font-medium text-gray-900 dark:text-white">Full Address</span>
                  <select
                    v-model="warrantyFieldsConfig.fields.address"
                    class="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
                  >
                    <option value="hidden">Hidden</option>
                    <option value="optional">Visible (Optional)</option>
                    <option value="required">Visible (Required)</option>
                  </select>
                </div>
              </div>
            </div>

            <!-- Custom Fields Section -->
            <div class="mt-6 pt-6 border-t border-gray-200 dark:border-gray-600">
              <div class="flex items-center justify-between mb-4">
                <div>
                  <h3 class="text-md font-semibold text-gray-900 dark:text-white">Custom Fields</h3>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Add custom fields to collect additional data during warranty registration
                  </p>
                </div>
                <button
                  @click="addCustomField"
                  class="flex items-center gap-1 px-3 py-1.5 text-sm bg-zinc-600 text-white rounded-lg hover:bg-zinc-700 transition-colors"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                  </svg>
                  Add Field
                </button>
              </div>

              <!-- Empty state -->
              <div v-if="customFields.length === 0" class="text-center py-6 bg-gray-50 dark:bg-gray-700 rounded-lg">
                <svg class="w-8 h-8 mx-auto text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                <p class="text-sm text-gray-500 dark:text-gray-400">No custom fields configured</p>
                <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">Click "Add Field" to create custom fields</p>
              </div>

              <!-- Custom fields list -->
              <div v-else class="space-y-4">
                <div
                  v-for="(field, index) in customFields"
                  :key="field.id"
                  class="p-4 bg-gray-50 dark:bg-gray-700 rounded-lg border border-gray-200 dark:border-gray-600"
                >
                  <div class="flex items-start justify-between mb-3">
                    <span class="text-xs font-medium text-gray-500 dark:text-gray-400">Field {{ index + 1 }}</span>
                    <button
                      @click="removeCustomField(index)"
                      class="text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                      title="Remove field"
                    >
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>

                  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <!-- Field Label -->
                    <div>
                      <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Label *</label>
                      <input
                        v-model="field.label"
                        type="text"
                        placeholder="e.g., Serial Number"
                        class="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-zinc-500"
                      >
                    </div>

                    <!-- Field Type -->
                    <div>
                      <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Type</label>
                      <select
                        v-model="field.type"
                        class="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-zinc-500"
                      >
                        <option v-for="ft in fieldTypes" :key="ft.value" :value="ft.value">{{ ft.label }}</option>
                      </select>
                    </div>
                  </div>

                  <!-- Required toggle -->
                  <div class="flex items-center gap-2 mt-3">
                    <input
                      type="checkbox"
                      :id="'required_' + field.id"
                      v-model="field.required"
                      class="w-4 h-4 text-zinc-600 border-gray-300 rounded focus:ring-zinc-500"
                    >
                    <label :for="'required_' + field.id" class="text-sm text-gray-700 dark:text-gray-300">
                      Required field
                    </label>
                  </div>

                  <!-- Options for select type -->
                  <div v-if="field.type === 'select'" class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-600">
                    <div class="flex items-center justify-between mb-2">
                      <label class="text-xs font-medium text-gray-600 dark:text-gray-400">Dropdown Options</label>
                      <button
                        @click="addFieldOption(field)"
                        class="text-xs text-zinc-600 hover:text-zinc-800 dark:text-zinc-400"
                      >
                        + Add Option
                      </button>
                    </div>
                    <div v-if="!field.options || field.options.length === 0" class="text-xs text-gray-400 py-2">
                      No options added. Click "Add Option" to add dropdown choices.
                    </div>
                    <div v-else class="space-y-2">
                      <div v-for="(opt, optIndex) in field.options" :key="optIndex" class="flex items-center gap-2">
                        <input
                          v-model="field.options[optIndex]"
                          type="text"
                          placeholder="Option value"
                          class="flex-1 px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                        >
                        <button
                          @click="removeFieldOption(field, optIndex)"
                          class="text-red-500 hover:text-red-700"
                        >
                          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Advanced Customization (Warranty Template Overrides) -->
            <Card class="overflow-hidden mt-6">
              <div class="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">Advanced Customization</h3>
                <button
                  v-if="warrantyOverrideCount > 0"
                  @click="resetAllWarrantyOverrides"
                  class="flex items-center gap-1 text-xs text-red-500 hover:text-red-600 dark:text-red-400 dark:hover:text-red-300 transition-colors"
                >
                  <RotateCcw class="w-3 h-3" />
                  Reset All ({{ warrantyOverrideCount }})
                </button>
              </div>

              <div v-for="(fields, groupName) in warrantyGroupedOverrideFields" :key="groupName" class="border-b border-gray-100 dark:border-gray-700 last:border-b-0">
                <button
                  @click="toggleWarrantyGroup(groupName)"
                  class="w-full flex items-center justify-between px-4 py-3 text-left hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
                >
                  <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ groupName }}</span>
                  <ChevronDown class="w-4 h-4 text-gray-400 transition-transform" :class="{ 'rotate-180': warrantyExpandedGroups[groupName] }" />
                </button>

                <div v-if="warrantyExpandedGroups[groupName]" class="px-4 pb-4 space-y-3">
                  <div v-for="field in fields" :key="`w-${field.section}-${field.key}`" class="space-y-1.5">
                    <div class="flex items-center justify-between">
                      <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ field.label }}</label>
                      <button
                        v-if="isWarrantyFieldOverridden(field.section, field.key)"
                        @click="resetWarrantyOverrideField(field.section, field.key)"
                        class="text-xs text-gray-400 hover:text-red-500 dark:hover:text-red-400 transition-colors flex items-center gap-1"
                      >
                        <RotateCcw class="w-3 h-3" />
                        Reset
                      </button>
                    </div>

                    <!-- Text -->
                    <Input
                      v-if="field.type === 'text'"
                      :modelValue="getWarrantyEffective(field.section, field.key, field.fallback)"
                      @update:modelValue="setWarrantyOverride(field.section, field.key, $event)"
                      class="w-full text-sm"
                      :class="{ 'ring-2 ring-zinc-300 dark:ring-zinc-700': isWarrantyFieldOverridden(field.section, field.key) }"
                    />

                    <!-- Color -->
                    <div v-else-if="field.type === 'color'" class="flex items-center gap-2">
                      <input
                        type="color"
                        :value="getWarrantyEffective(field.section, field.key, field.fallback)"
                        @input="setWarrantyOverride(field.section, field.key, $event.target.value)"
                        class="w-8 h-8 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                      />
                      <Input
                        :modelValue="getWarrantyEffective(field.section, field.key, field.fallback)"
                        @update:modelValue="setWarrantyOverride(field.section, field.key, $event)"
                        class="flex-1 font-mono text-xs"
                        :class="{ 'ring-2 ring-zinc-300 dark:ring-zinc-700': isWarrantyFieldOverridden(field.section, field.key) }"
                      />
                    </div>
                  </div>
                </div>
              </div>
            </Card>

            <div class="pt-4 mt-6 border-t border-gray-200 dark:border-gray-600">
              <Button @click="saveWarrantyConfig" :disabled="savingConfig || !isWarrantyConfigValid">
                {{ savingConfig ? 'Saving...' : 'Save Warranty Settings' }}
              </Button>
            </div>
          </div>

          <!-- Disabled state message -->
          <div v-else class="text-center py-8 text-gray-500 dark:text-gray-400">
            <svg class="w-12 h-12 mx-auto mb-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
            <p class="font-medium">Warranty registration is disabled</p>
            <p class="text-sm mt-2">New QR batches for this product will not show warranty registration option.</p>
            <p class="text-xs mt-4 text-gray-400">Note: Existing batches with warranty enabled will continue to work.</p>
          </div>
        </Card>

        <!-- RIGHT: Sticky Live Preview -->
        <div class="lg:sticky lg:top-4 h-fit">
          <WarrantySettingsPreview
            :fields-config="warrantyFieldsConfig"
            :custom-fields="customFields"
            :template-config="mergedWarrantyPreviewConfig"
            :product-name="product?.product_name"
            :warranty-months="warrantyMonths || 12"
          />
        </div>
        </div>
      </div>
    </div>

    <!-- Add Certification Modal -->
    <div v-if="showAddCertModal" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="fixed inset-0 bg-black/50" @click="showAddCertModal = false"></div>
      <div class="relative z-10 w-full max-w-md bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Add Certification</h2>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Certification Type *</label>
            <select
              v-model="newCert.certification_type_id"
              data-tour="cert-type-select-detail"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
            >
              <option value="">Select certification...</option>
              <option v-for="type in unusedCertTypes" :key="type.id" :value="type.id">
                {{ type.name }} ({{ type.country?.name || 'International' }})
              </option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Registration Number *</label>
            <Input v-model="newCert.registration_number" placeholder="Enter registration number" data-tour="cert-reg-detail" />
          </div>
        </div>
        <div class="flex gap-3 pt-4">
          <Button variant="outline" class="flex-1" @click="showAddCertModal = false">Cancel</Button>
          <Button class="flex-1" @click="addCertification" :disabled="!newCert.certification_type_id || !newCert.registration_number" data-tour="cert-submit-detail">
            Add
          </Button>
        </div>
      </div>
    </div>

    <!-- Add Social Link Modal -->
    <div v-if="showAddSocialModal" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="fixed inset-0 bg-black/50" @click="showAddSocialModal = false"></div>
      <div class="relative z-10 w-full max-w-md bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Add Social Link</h2>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Platform *</label>
            <select
              v-model="newSocial.platform_id"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
            >
              <option value="">Select platform...</option>
              <option v-for="platform in unusedPlatforms" :key="platform.id" :value="platform.id">
                {{ platform.name }}
              </option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Handle or URL *</label>
            <Input v-model="newSocial.handle_or_url" :placeholder="getSelectedPlaceholder()" />
          </div>
        </div>
        <div class="flex gap-3 pt-4">
          <Button variant="outline" class="flex-1" @click="showAddSocialModal = false">Cancel</Button>
          <Button class="flex-1" @click="addSocialLink" :disabled="!newSocial.platform_id || !newSocial.handle_or_url">
            Add
          </Button>
        </div>
      </div>
    </div>

  </div>
</template>
