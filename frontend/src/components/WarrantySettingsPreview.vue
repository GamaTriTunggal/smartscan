<script setup>
import { ref, computed } from 'vue'
import { ASPECT_RATIOS, DEFAULT_ASPECT_RATIO } from '@/constants/previewOptions'

const selectedRatio = ref(DEFAULT_ASPECT_RATIO)
const previewMode = ref('form')

const props = defineProps({
  fieldsConfig: { type: Object, default: () => ({ enabled: false, fields: {} }) },
  customFields: { type: Array, default: () => [] },
  templateConfig: { type: Object, default: null },
  productName: { type: String, default: '' },
  warrantyMonths: { type: Number, default: 12 }
})

// Template styling with defaults
const styling = computed(() => {
  const tc = props.templateConfig
  return {
    header_bg_color: tc?.styling?.header_bg_color || '#18181b',
    form_bg_color: tc?.styling?.form_bg_color || '#ffffff',
    text_color: tc?.styling?.text_color || '#1f2937',
    accent_color: tc?.styling?.accent_color || '#18181b'
  }
})

const submitButton = computed(() => {
  const tc = props.templateConfig
  return {
    text: tc?.submit_button?.text || 'Activate Warranty',
    bg_color: tc?.submit_button?.bg_color || '#18181b',
    text_color: tc?.submit_button?.text_color || '#ffffff'
  }
})

const messages = computed(() => {
  const tc = props.templateConfig
  return {
    success_title: tc?.messages?.success_title || 'Warranty Activated!',
    success_message: tc?.messages?.success_message || 'Your product warranty has been successfully activated.',
    already_activated_title: tc?.messages?.already_activated_title || 'Already Activated',
    already_activated_message: tc?.messages?.already_activated_message || 'This product warranty has already been activated.'
  }
})

const fields = computed(() => props.fieldsConfig?.fields || {})

const hasAddressSection = computed(() => {
  const f = fields.value
  return isVisible('country') || isVisible('province') || isVisible('city') || isVisible('address')
})

function isRequired(fieldKey) {
  return fields.value[fieldKey] === 'required'
}

function isVisible(fieldKey) {
  return fields.value[fieldKey] && fields.value[fieldKey] !== 'hidden'
}

// Map custom field types to input display
function getFieldPlaceholder(field) {
  const map = {
    text: 'Enter text...',
    textarea: 'Enter details...',
    number: '0',
    date: '',
    select: '',
    email: 'email@example.com',
    phone: '+62...'
  }
  return map[field.type] || 'Enter value...'
}
</script>

<template>
  <div class="bg-gray-100 dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
    <!-- Preview Header -->
    <div class="bg-gray-200 dark:bg-gray-800 px-4 py-2 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <div class="flex gap-1.5">
          <div class="w-3 h-3 rounded-full bg-red-500"></div>
          <div class="w-3 h-3 rounded-full bg-yellow-500"></div>
          <div class="w-3 h-3 rounded-full bg-green-500"></div>
        </div>
        <span class="text-xs text-gray-500 dark:text-gray-400 ml-2">Warranty Preview</span>
      </div>
      <div class="flex items-center gap-2">
        <select
          v-model="selectedRatio"
          class="text-xs px-2 py-1 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-300 focus:ring-1 focus:ring-[#27272a]"
        >
          <option v-for="ratio in ASPECT_RATIOS" :key="ratio.value" :value="ratio.value">
            {{ ratio.label }}
          </option>
        </select>
        <div class="flex gap-1">
          <button
            @click="previewMode = 'form'"
            data-tour="warranty-preview-form"
            class="px-2 py-1 text-xs rounded transition-colors"
            :class="previewMode === 'form'
              ? 'bg-zinc-600 text-white'
              : 'bg-gray-300 dark:bg-gray-600 text-gray-700 dark:text-gray-300'"
          >
            Form
          </button>
          <button
            @click="previewMode = 'success'"
            data-tour="warranty-preview-success"
            class="px-2 py-1 text-xs rounded transition-colors"
            :class="previewMode === 'success'
              ? 'bg-green-600 text-white'
              : 'bg-gray-300 dark:bg-gray-600 text-gray-700 dark:text-gray-300'"
          >
            Success
          </button>
          <button
            @click="previewMode = 'error'"
            data-tour="warranty-preview-error"
            class="px-2 py-1 text-xs rounded transition-colors"
            :class="previewMode === 'error'
              ? 'bg-red-600 text-white'
              : 'bg-gray-300 dark:bg-gray-600 text-gray-700 dark:text-gray-300'"
          >
            Error
          </button>
        </div>
      </div>
    </div>

    <!-- Phone Frame -->
    <div class="p-4">
      <div
        class="mx-auto max-w-[320px] rounded-[32px] border-[8px] border-gray-800 dark:border-gray-600 overflow-hidden shadow-xl"
        :style="{ backgroundColor: styling.form_bg_color }"
      >
        <div class="overflow-y-auto" :style="{ aspectRatio: selectedRatio }">
          <!-- Header -->
          <div
            class="px-4 py-6 text-center"
            :style="{ backgroundColor: styling.header_bg_color }"
          >
            <svg class="w-12 h-12 mx-auto mb-2 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
            <h1 class="text-lg font-bold text-white">Warranty Activation</h1>
            <p class="text-sm text-white/80 mt-1">{{ productName || 'Product Name' }}</p>
            <p v-if="warrantyMonths" class="text-xs text-white/60 mt-1">{{ warrantyMonths }} months warranty</p>
          </div>

          <!-- Disabled State -->
          <div v-if="!fieldsConfig.enabled" class="px-4 py-12 text-center">
            <svg class="w-10 h-10 mx-auto mb-3 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
            </svg>
            <p class="text-sm text-gray-400">Warranty disabled</p>
          </div>

          <!-- Form Preview -->
          <div v-else-if="previewMode === 'form'" class="px-4 py-4 space-y-3">
            <!-- Fixed Required Fields -->
            <div class="space-y-1">
              <label class="block text-xs font-medium" :style="{ color: styling.text_color }">
                Full Name <span class="text-red-500">*</span>
              </label>
              <input type="text" placeholder="Enter full name" class="w-full px-3 py-2 text-sm border rounded-lg bg-white" :style="{ borderColor: styling.accent_color + '40' }" disabled />
            </div>

            <div class="space-y-1">
              <label class="block text-xs font-medium" :style="{ color: styling.text_color }">
                Email <span class="text-red-500">*</span>
              </label>
              <input type="email" placeholder="Enter email" class="w-full px-3 py-2 text-sm border rounded-lg bg-white" :style="{ borderColor: styling.accent_color + '40' }" disabled />
            </div>

            <div class="space-y-1">
              <label class="block text-xs font-medium" :style="{ color: styling.text_color }">
                Phone <span class="text-red-500">*</span>
              </label>
              <input type="text" placeholder="Enter phone number" class="w-full px-3 py-2 text-sm border rounded-lg bg-white" :style="{ borderColor: styling.accent_color + '40' }" disabled />
            </div>

            <div class="space-y-1">
              <label class="block text-xs font-medium" :style="{ color: styling.text_color }">
                Purchase Date <span class="text-red-500">*</span>
              </label>
              <input type="date" class="w-full px-3 py-2 text-sm border rounded-lg bg-white" :style="{ borderColor: styling.accent_color + '40' }" disabled />
            </div>

            <!-- Store Name (configurable) -->
            <div v-if="isVisible('store_name')" class="space-y-1">
              <label class="block text-xs font-medium" :style="{ color: styling.text_color }">
                Store Name <span v-if="isRequired('store_name')" class="text-red-500">*</span>
              </label>
              <input type="text" placeholder="Enter store name" class="w-full px-3 py-2 text-sm border rounded-lg bg-white" :style="{ borderColor: styling.accent_color + '40' }" disabled />
            </div>

            <!-- Address Section -->
            <template v-if="hasAddressSection">
              <div class="pt-3 border-t border-gray-200">
                <p class="text-xs font-semibold mb-1" :style="{ color: styling.text_color }">Your Address</p>
                <p class="text-[10px] text-gray-400 mb-2">Required for warranty service</p>
              </div>

              <div v-if="isVisible('country')" class="space-y-1">
                <label class="block text-xs font-medium" :style="{ color: styling.text_color }">
                  Country <span v-if="isRequired('country')" class="text-red-500">*</span>
                </label>
                <select class="w-full px-3 py-2 text-sm border rounded-lg bg-white text-gray-400" :style="{ borderColor: styling.accent_color + '40' }" disabled>
                  <option>Select country</option>
                </select>
              </div>

              <div v-if="isVisible('province')" class="space-y-1">
                <label class="block text-xs font-medium" :style="{ color: styling.text_color }">
                  Province <span v-if="isRequired('province')" class="text-red-500">*</span>
                </label>
                <select class="w-full px-3 py-2 text-sm border rounded-lg bg-white text-gray-400" :style="{ borderColor: styling.accent_color + '40' }" disabled>
                  <option>Select province</option>
                </select>
              </div>

              <div v-if="isVisible('city')" class="space-y-1">
                <label class="block text-xs font-medium" :style="{ color: styling.text_color }">
                  City <span v-if="isRequired('city')" class="text-red-500">*</span>
                </label>
                <select class="w-full px-3 py-2 text-sm border rounded-lg bg-white text-gray-400" :style="{ borderColor: styling.accent_color + '40' }" disabled>
                  <option>Select city</option>
                </select>
              </div>

              <div v-if="isVisible('address')" class="space-y-1">
                <label class="block text-xs font-medium" :style="{ color: styling.text_color }">
                  Full Address <span v-if="isRequired('address')" class="text-red-500">*</span>
                </label>
                <textarea placeholder="Street, building, house number..." rows="2" class="w-full px-3 py-2 text-sm border rounded-lg bg-white resize-none" :style="{ borderColor: styling.accent_color + '40' }" disabled></textarea>
              </div>
            </template>

            <!-- Custom Fields -->
            <template v-if="customFields.length > 0">
              <div class="pt-3 border-t border-gray-200">
                <p class="text-xs font-semibold mb-1" :style="{ color: styling.text_color }">Additional Information</p>
              </div>

              <div v-for="field in customFields" :key="field.id" class="space-y-1">
                <label class="block text-xs font-medium" :style="{ color: styling.text_color }">
                  {{ field.label || 'Untitled Field' }}
                  <span v-if="field.required" class="text-red-500">*</span>
                </label>

                <textarea v-if="field.type === 'textarea'" :placeholder="getFieldPlaceholder(field)" rows="2" class="w-full px-3 py-2 text-sm border rounded-lg bg-white resize-none" :style="{ borderColor: styling.accent_color + '40' }" disabled></textarea>

                <select v-else-if="field.type === 'select'" class="w-full px-3 py-2 text-sm border rounded-lg bg-white text-gray-400" :style="{ borderColor: styling.accent_color + '40' }" disabled>
                  <option>Select...</option>
                  <option v-for="(opt, i) in (field.options || [])" :key="i">{{ opt }}</option>
                </select>

                <input v-else :type="field.type === 'number' ? 'number' : field.type === 'date' ? 'date' : 'text'" :placeholder="getFieldPlaceholder(field)" class="w-full px-3 py-2 text-sm border rounded-lg bg-white" :style="{ borderColor: styling.accent_color + '40' }" disabled />
              </div>
            </template>

            <!-- Submit Button -->
            <button
              class="w-full py-3 rounded-lg font-medium text-sm mt-4"
              :style="{ backgroundColor: submitButton.bg_color, color: submitButton.text_color }"
            >
              {{ submitButton.text }}
            </button>
          </div>

          <!-- Success Preview -->
          <div v-else-if="previewMode === 'success'" class="px-4 py-8 text-center">
            <div
              class="w-16 h-16 mx-auto rounded-full flex items-center justify-center mb-4"
              :style="{ backgroundColor: styling.accent_color + '20' }"
            >
              <svg class="w-8 h-8" :style="{ color: styling.accent_color }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <h2 class="text-xl font-bold mb-2" :style="{ color: styling.text_color }">
              {{ messages.success_title }}
            </h2>
            <p class="text-sm opacity-80 mb-6" :style="{ color: styling.text_color }">
              {{ messages.success_message }}
            </p>
          </div>

          <!-- Error Preview -->
          <div v-else class="px-4 py-8 text-center">
            <div class="w-16 h-16 mx-auto rounded-full flex items-center justify-center mb-4 bg-red-100">
              <svg class="w-8 h-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <h2 class="text-xl font-bold mb-2" :style="{ color: styling.text_color }">
              {{ messages.already_activated_title }}
            </h2>
            <p class="text-sm opacity-80" :style="{ color: styling.text_color }">
              {{ messages.already_activated_message }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
