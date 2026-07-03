<script setup>
import { ref, computed } from 'vue'
import { ASPECT_RATIOS, DEFAULT_ASPECT_RATIO } from '@/constants/previewOptions'
import { SOCIAL_ICON_PATHS } from '@/lib/socialIcons'
import { Image, Video, Share2, Award, ExternalLink, FileText, Shield } from 'lucide-vue-next'

const SECTION_ICONS = {
  images: Image,
  videos: Video,
  social_accounts: Share2,
  certifications: Award,
  website_link: ExternalLink,
  description: FileText,
  warranty_button: Shield
}

const SECTION_LABELS = {
  images: 'Gallery',
  videos: 'Videos',
  social_accounts: 'Social Media',
  certifications: 'Certifications',
  website_link: 'Website Button',
  description: 'Product Description',
  warranty_button: 'Warranty Button'
}

const DEFAULT_SECTION_ORDER = [
  'images', 'videos', 'social_accounts', 'certifications',
  'website_link', 'description', 'warranty_button'
]

const DEFAULT_FIELD_ORDER = [
  'product_name', 'brand_name', 'product_code', 'show_verification_count',
  'batch_code', 'production_date', 'expiry_date'
]

const selectedRatio = ref(DEFAULT_ASPECT_RATIO)
const currentYear = new Date().getFullYear()
const previewCertExpanded = ref(true)

const props = defineProps({
  config: { type: Object, default: () => ({}) },
  backgroundConfig: { type: Object, default: null },
  productName: { type: String, default: 'Sample Product' },
  productCode: { type: String, default: '' },
  brandName: { type: String, default: '' },
  description: { type: String, default: '' },
  websiteUrl: { type: String, default: '' },
  websiteCaption: { type: String, default: '' },
  images: { type: Array, default: () => [] },
  videos: { type: Array, default: () => [] },
  certifications: { type: Array, default: () => [] },
  socialAccounts: { type: Array, default: () => [] },
  displayConfig: { type: Object, default: () => ({}) },
  warrantyEnabled: { type: Boolean, default: false },
  sectionOrder: { type: Array, default: null },
  loading: { type: Boolean, default: false }
})

// URL resolution
const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
const UPLOAD_BASE = API_BASE.replace('/api/v1', '')

function getImageUrl(url) {
  if (!url) return ''
  if (url.startsWith('http://') || url.startsWith('https://')) return url
  if (url.startsWith('/uploads/')) return UPLOAD_BASE + url
  return url
}

function getSocialIcon(account) {
  if (account.platform_icon && account.platform_icon.includes(' ')) {
    return account.platform_icon
  }
  const code = (account.platform_code || account.platform_icon || account.platform_name || '').toUpperCase()
  return SOCIAL_ICON_PATHS[code] || ''
}

// Main image (is_main=true or first image)
const mainImage = computed(() => {
  if (!props.images.length) return null
  return props.images.find(i => i.is_main) || props.images[0]
})

// Gallery images (non-main, limit to 6)
const galleryImages = computed(() => {
  const mainId = mainImage.value?.id
  return props.images.filter(i => i.id !== mainId).slice(0, 6)
})

// Section order
const orderedSections = computed(() => {
  const order = props.sectionOrder || props.config?.section_order || DEFAULT_SECTION_ORDER
  const known = new Set(DEFAULT_SECTION_ORDER)
  const filtered = order.filter(s => known.has(s))
  const missing = DEFAULT_SECTION_ORDER.filter(s => !filtered.includes(s))
  return [...new Set([...filtered, ...missing])]
})

// Section visibility (based on actual data availability + product settings)
const sectionVisible = computed(() => ({
  images: galleryImages.value.length > 0,
  videos: props.videos.length > 0,
  social_accounts: props.socialAccounts.length > 0,
  certifications: props.certifications.length > 0,
  website_link: !!props.websiteUrl,
  description: !!(props.description && props.description.trim()),
  warranty_button: props.warrantyEnabled
}))

// Display field helpers
const dc = computed(() => props.displayConfig || {})
const fieldOrder = computed(() => dc.value.field_order || DEFAULT_FIELD_ORDER)

const showProductCode = computed(() => dc.value.product_code && props.productCode)
const showVerificationCount = computed(() => dc.value.show_verification_count !== false)
const showBatchCode = computed(() => dc.value.batch_code)
const showProductionDate = computed(() => dc.value.production_date)
const showExpiryDate = computed(() => dc.value.expiry_date)
const fieldOrderWithoutHeader = computed(() => fieldOrder.value.filter(k => k !== 'product_name' && k !== 'brand_name'))
const hasVisibleFields = computed(() => showProductCode.value || showVerificationCount.value || showBatchCode.value || showProductionDate.value || showExpiryDate.value)

// Background computed properties
const hasBackground = computed(() => {
  if (!props.backgroundConfig) return false
  return props.backgroundConfig.background_type !== 'none'
})

const backgroundUrl = computed(() => {
  if (!props.backgroundConfig) return null
  const bg = props.backgroundConfig
  if (bg.background_type === 'custom') return bg.custom_background_url
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

const headerGlassStyle = computed(() => {
  const baseColor = props.config?.header?.bg_color || '#3f3f46'
  if (!hasBackground.value) return { backgroundColor: baseColor }
  const hex = baseColor.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  return { backgroundColor: `rgba(${r}, ${g}, ${b}, 0.8)` }
})

const fieldCardStyle = computed(() => {
  if (!hasBackground.value) {
    return {
      backgroundColor: props.config?.styling?.field_bg_color || '#ffffff',
      boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)'
    }
  }
  return {
    backgroundColor: 'rgba(255, 255, 255, 0.15)',
    border: '1px solid rgba(255, 255, 255, 0.2)',
    boxShadow: 'none'
  }
})

const textColorStyle = computed(() => {
  if (!hasBackground.value) {
    return { color: props.config?.styling?.text_color || '#1f2937' }
  }
  return { color: 'rgba(0, 0, 0, 0.85)' }
})

const labelColorStyle = computed(() => {
  if (!hasBackground.value) return { color: '#6b7280' }
  return { color: 'rgba(0, 0, 0, 0.6)' }
})

const subtleColorStyle = computed(() => {
  if (hasBackground.value) return { color: 'rgba(0,0,0,0.5)' }
  return { color: '#9ca3af' }
})

// Certifications section style
const certSectionStyle = computed(() => {
  if (hasBackground.value) {
    return { backgroundColor: 'rgba(255, 255, 255, 0.1)', border: '1px solid rgba(255, 255, 255, 0.2)' }
  }
  return {
    backgroundColor: props.config?.certifications_section?.bg_color || '#f0fdf4',
    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)'
  }
})

const certIconColor = computed(() => props.config?.certifications_section?.icon_color || '#10b981')

// Sticky social enabled check
const isSocialSticky = computed(() => props.config?.social_media_section?.sticky_enabled !== false)
</script>

<template>
  <div class="bg-gray-100 dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
    <!-- Header Bar -->
    <div class="bg-gray-200 dark:bg-gray-800 px-4 py-2 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <div class="flex gap-1.5">
          <div class="w-3 h-3 rounded-full bg-red-500"></div>
          <div class="w-3 h-3 rounded-full bg-yellow-500"></div>
          <div class="w-3 h-3 rounded-full bg-green-500"></div>
        </div>
        <span class="text-xs text-gray-500 dark:text-gray-400 ml-2">Live Preview</span>
      </div>
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
      <!-- Loading state -->
      <div v-if="loading" class="mx-auto max-w-[320px] flex items-center justify-center py-20">
        <div class="text-center">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500 mx-auto mb-3"></div>
          <p class="text-xs text-gray-500 dark:text-gray-400">Loading template...</p>
        </div>
      </div>

      <div
        v-else
        class="mx-auto max-w-[320px] rounded-[32px] border-[8px] border-gray-800 dark:border-gray-600 overflow-hidden shadow-xl relative"
        :style="{ backgroundColor: hasBackground ? 'transparent' : (config?.styling?.card_bg_color || '#f3f4f6') }"
      >
        <!-- Background Layers -->
        <div v-if="hasBackground && backgroundUrl" class="absolute inset-0" :style="backgroundStyle"></div>
        <div v-if="hasBackground && backgroundUrl" class="absolute inset-0" :style="overlayStyle"></div>

        <!-- Phone Screen -->
        <div class="overflow-y-auto relative z-10" :style="{ aspectRatio: selectedRatio }">
          <div :class="hasBackground ? 'm-3 rounded-2xl overflow-hidden' : ''" :style="glassCardStyle">

            <!-- Header -->
            <div class="px-4 py-4 text-center" :style="headerGlassStyle">
              <div v-if="config?.header?.logo_enabled && config?.header?.logo_url" class="mb-3">
                <img
                  :src="getImageUrl(config.header.logo_url)"
                  alt="Company Logo"
                  class="mx-auto object-contain"
                  :style="{ maxHeight: (config.header?.logo_max_height || 60) + 'px' }"
                  @error="$event.target.style.display = 'none'"
                />
              </div>
              <div
                class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium"
                :style="{
                  backgroundColor: config?.header?.badge_bg_color || '#22c55e',
                  color: config?.header?.badge_text_color || '#ffffff'
                }"
              >
                <svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
                  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                </svg>
                {{ config?.header?.badge_text || 'Authentic Product' }}
              </div>
            </div>

            <!-- Content -->
            <div class="px-4 py-4 space-y-3">
              <!-- Dynamic Field Order -->
              <template v-for="fieldKey in fieldOrder" :key="fieldKey">

                <!-- Product Header (image + name + company) — brand_name is part of this card -->
                <div v-if="fieldKey === 'product_name'" data-field="product_name" class="rounded-lg p-3 mb-1" :style="fieldCardStyle">
                  <div class="flex items-center gap-3">
                    <div
                      class="flex-shrink-0 rounded-lg overflow-hidden flex items-center justify-center"
                      :class="{ 'bg-gray-200 dark:bg-gray-600': !mainImage }"
                      :style="{ width: (config?.styling?.main_image_size || 96) * 0.5 + 'px', height: (config?.styling?.main_image_size || 96) * 0.5 + 'px' }"
                    >
                      <img
                        v-if="mainImage"
                        :src="getImageUrl(mainImage.image_url)"
                        :alt="productName"
                        class="w-full h-full object-cover"
                        @error="$event.target.style.display = 'none'"
                      />
                      <svg v-else class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                      </svg>
                    </div>
                    <div class="min-w-0 flex-1">
                      <p class="font-semibold text-sm leading-tight" :style="textColorStyle">{{ productName || 'Sample Product' }}</p>
                      <p class="text-xs mt-0.5" :style="labelColorStyle">{{ brandName || 'Company Name' }}</p>
                    </div>
                  </div>
                </div>

              </template>

              <!-- Consolidated info fields card (matches real landing page) -->
              <div v-if="hasVisibleFields" class="rounded-lg p-4 space-y-3" :style="fieldCardStyle">
                <template v-for="fieldKey in fieldOrderWithoutHeader" :key="fieldKey">
                  <div v-if="fieldKey === 'product_code' && showProductCode" data-field="product_code" class="flex justify-between text-sm">
                    <span :style="labelColorStyle">Product Code</span>
                    <span class="font-medium" :style="textColorStyle">{{ productCode }}</span>
                  </div>
                  <div v-else-if="fieldKey === 'show_verification_count' && showVerificationCount" data-field="show_verification_count" class="flex justify-between text-sm">
                    <span :style="labelColorStyle">Verified</span>
                    <span class="font-medium" :style="textColorStyle">3 times</span>
                  </div>
                  <div v-else-if="fieldKey === 'batch_code' && showBatchCode" data-field="batch_code" class="flex justify-between text-sm">
                    <span :style="labelColorStyle">Batch</span>
                    <span class="font-medium" :style="textColorStyle">BATCH-{{ currentYear }}-001</span>
                  </div>
                  <div v-else-if="fieldKey === 'production_date' && showProductionDate" data-field="production_date" class="flex justify-between text-sm">
                    <span :style="labelColorStyle">Production Date</span>
                    <span class="font-medium" :style="textColorStyle">10 Jan {{ currentYear }}</span>
                  </div>
                  <div v-else-if="fieldKey === 'expiry_date' && showExpiryDate" data-field="expiry_date" class="flex justify-between text-sm">
                    <span :style="labelColorStyle">Expiry Date</span>
                    <span class="font-medium" :style="textColorStyle">10 Jan {{ currentYear + 2 }}</span>
                  </div>
                </template>
              </div>

              <!-- Dynamic Sections -->
              <template v-for="sectionId in orderedSections" :key="sectionId">

                <!-- Gallery -->
                <div v-if="sectionId === 'images' && sectionVisible.images" class="mt-3 rounded-lg p-3" :style="fieldCardStyle">
                  <div class="flex items-center gap-2 mb-2">
                    <component :is="SECTION_ICONS.images" class="w-4 h-4" :style="{ color: hasBackground ? 'rgba(0,0,0,0.6)' : '#6b7280' }" />
                    <span class="text-xs font-medium" :style="textColorStyle">Gallery</span>
                  </div>
                  <div class="grid grid-cols-3 gap-1.5">
                    <div
                      v-for="(img, idx) in galleryImages.slice(0, 3)"
                      :key="idx"
                      class="aspect-square rounded overflow-hidden bg-gray-200 dark:bg-gray-600"
                    >
                      <img
                        :src="getImageUrl(img.image_url)"
                        :alt="img.caption || ''"
                        class="w-full h-full object-cover"
                        @error="$event.target.style.display = 'none'"
                      />
                    </div>
                    <div
                      v-if="galleryImages.length > 3"
                      class="aspect-square rounded bg-gray-200 dark:bg-gray-600 flex items-center justify-center text-xs text-gray-400"
                    >
                      +{{ galleryImages.length - 3 }}
                    </div>
                  </div>
                </div>

                <!-- Videos -->
                <div v-if="sectionId === 'videos' && sectionVisible.videos" class="mt-3 rounded-lg p-3" :style="fieldCardStyle">
                  <div class="flex items-center gap-2 mb-2">
                    <component :is="SECTION_ICONS.videos" class="w-4 h-4" :style="{ color: hasBackground ? 'rgba(0,0,0,0.6)' : '#6b7280' }" />
                    <span class="text-xs font-medium" :style="textColorStyle">Videos</span>
                  </div>
                  <div class="aspect-video rounded bg-gray-200 dark:bg-gray-600 flex items-center justify-center">
                    <div class="w-10 h-10 rounded-full bg-gray-300 dark:bg-gray-500 flex items-center justify-center">
                      <svg class="w-5 h-5 text-gray-500 dark:text-gray-400 ml-0.5" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M8 5v14l11-7z" />
                      </svg>
                    </div>
                  </div>
                </div>

                <!-- Social Media (inline, when NOT sticky) -->
                <div v-if="sectionId === 'social_accounts' && sectionVisible.social_accounts && !isSocialSticky" class="mt-4">
                  <div class="flex items-center justify-center gap-3 flex-wrap">
                    <a
                      v-for="(account, idx) in socialAccounts.slice(0, 5)"
                      :key="idx"
                      href="#"
                      @click.prevent
                      class="w-10 h-10 rounded-full bg-gray-100 flex items-center justify-center"
                      :title="account.platform_name || account.platform_code"
                    >
                      <svg v-if="getSocialIcon(account)" class="w-5 h-5 text-gray-700" viewBox="0 0 24 24" fill="currentColor">
                        <path :d="getSocialIcon(account)" />
                      </svg>
                      <span v-else class="text-xs text-gray-500">{{ (account.platform_name || '?').charAt(0) }}</span>
                    </a>
                  </div>
                </div>

                <!-- Certifications (Collapsible) -->
                <div
                  v-if="sectionId === 'certifications' && sectionVisible.certifications"
                  class="mt-3 rounded-lg overflow-hidden"
                  :style="certSectionStyle"
                >
                  <button
                    @click="previewCertExpanded = !previewCertExpanded"
                    class="w-full py-3 px-4 flex items-center justify-between text-left"
                  >
                    <div class="flex items-center gap-1.5 min-w-0">
                      <template v-if="certifications.some(c => c.logo_url)">
                        <template v-for="(cert, idx) in certifications.slice(0, 7)" :key="idx">
                          <img
                            v-if="cert.logo_url"
                            :src="getImageUrl(cert.logo_url)"
                            :alt="cert.name || cert.certification_name"
                            class="w-10 h-10 rounded-full object-contain bg-white border border-white/50 flex-shrink-0"
                            @error="$event.target.style.display = 'none'"
                          />
                          <span v-else class="w-10 h-10 rounded-full flex items-center justify-center bg-white/60 flex-shrink-0">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" :style="{ color: certIconColor }">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                          </span>
                        </template>
                        <span v-if="certifications.length > 7" class="text-sm font-semibold flex-shrink-0" :style="textColorStyle">
                          +{{ certifications.length - 7 }}
                        </span>
                      </template>
                      <template v-else>
                        <component :is="SECTION_ICONS.certifications" class="w-4 h-4" :style="{ color: certIconColor }" />
                        <span class="text-xs font-medium" :style="textColorStyle">
                          {{ config?.certifications_section?.header_text || 'Certifications' }}
                        </span>
                        <span
                          class="text-xs px-2 py-0.5 rounded-full"
                          :style="{ backgroundColor: hasBackground ? 'rgba(255,255,255,0.2)' : 'rgba(255,255,255,0.6)', ...textColorStyle }"
                        >
                          {{ certifications.length }}
                        </span>
                      </template>
                    </div>
                    <svg
                      class="w-4 h-4 transition-transform flex-shrink-0"
                      :class="{ 'rotate-180': previewCertExpanded }"
                      fill="none" stroke="currentColor" viewBox="0 0 24 24" :style="textColorStyle"
                    >
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                    </svg>
                  </button>
                  <div v-if="previewCertExpanded" class="px-3 pb-3 space-y-1.5">
                    <div
                      v-for="cert in certifications"
                      :key="cert.id || cert.name"
                      class="flex items-center gap-3 p-2 rounded-lg"
                      :style="{ backgroundColor: hasBackground ? 'rgba(255,255,255,0.1)' : 'rgba(255,255,255,0.15)' }"
                    >
                      <div v-if="cert.logo_url" class="w-10 h-10 flex-shrink-0">
                        <img
                          :src="getImageUrl(cert.logo_url)"
                          :alt="cert.name || cert.certification_name"
                          class="w-full h-full object-contain"
                          @error="$event.target.style.display = 'none'"
                        />
                      </div>
                      <div
                        v-else
                        class="w-10 h-10 flex-shrink-0 rounded-full flex items-center justify-center"
                        style="background: rgba(255,255,255,0.2)"
                      >
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" :style="{ color: certIconColor }">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                      </div>
                      <div class="flex-1 min-w-0">
                        <p class="text-sm font-medium truncate" :style="textColorStyle">{{ cert.name || cert.certification_name }}</p>
                        <p class="text-xs" :style="{ ...textColorStyle, opacity: 0.7 }">
                          {{ cert.country }} <span v-if="cert.registration_number">- {{ cert.registration_number }}</span>
                        </p>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- Website Button -->
                <div v-if="sectionId === 'website_link' && sectionVisible.website_link" class="mt-3">
                  <button
                    class="w-full py-2.5 rounded-lg font-medium text-xs transition-colors flex items-center justify-center gap-2"
                    :style="{ backgroundColor: hasBackground ? 'rgba(59, 130, 246, 0.9)' : '#3f3f46', color: '#ffffff' }"
                  >
                    <component :is="SECTION_ICONS.website_link" class="w-4 h-4" />
                    {{ websiteCaption || 'Visit Website' }}
                  </button>
                </div>

                <!-- Description -->
                <div v-if="sectionId === 'description' && sectionVisible.description" class="mt-3 rounded-lg p-3" :style="fieldCardStyle">
                  <div class="flex items-center gap-2 mb-2">
                    <component :is="SECTION_ICONS.description" class="w-4 h-4" :style="{ color: hasBackground ? 'rgba(0,0,0,0.6)' : '#6b7280' }" />
                    <span class="text-xs font-medium" :style="textColorStyle">Description</span>
                  </div>
                  <p class="text-xs leading-relaxed line-clamp-4" :style="{ color: hasBackground ? 'rgba(0,0,0,0.7)' : '#6b7280' }">
                    {{ description }}
                  </p>
                </div>

                <!-- Warranty Button -->
                <div v-if="sectionId === 'warranty_button' && sectionVisible.warranty_button" class="mt-3">
                  <button
                    class="w-full py-3 rounded-lg font-medium text-sm transition-colors flex items-center justify-center gap-2"
                    :style="{
                      backgroundColor: config?.warranty_button?.bg_color || '#9333ea',
                      color: config?.warranty_button?.text_color || '#ffffff'
                    }"
                  >
                    <component :is="SECTION_ICONS.warranty_button" class="w-4 h-4" />
                    {{ config?.warranty_button?.text || 'Activate Warranty' }}
                  </button>
                </div>

              </template>
            </div>
          </div>
        </div>

        <!-- Sticky Social Media Bar -->
        <div
          v-if="isSocialSticky && sectionVisible.social_accounts"
          class="sticky bottom-0 left-0 right-0 z-10 bg-white/95 backdrop-blur-sm border-t border-gray-200 shadow-lg"
        >
          <div class="px-4 py-3">
            <div class="flex items-center justify-center gap-3">
              <a
                v-for="(account, idx) in socialAccounts.slice(0, 5)"
                :key="idx"
                href="#"
                @click.prevent
                class="w-10 h-10 rounded-full bg-gray-100 flex items-center justify-center"
                :title="account.platform_name || account.platform_code"
              >
                <svg v-if="getSocialIcon(account)" class="w-5 h-5 text-gray-700" viewBox="0 0 24 24" fill="currentColor">
                  <path :d="getSocialIcon(account)" />
                </svg>
                <span v-else class="text-xs text-gray-500">{{ (account.platform_name || '?').charAt(0) }}</span>
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
