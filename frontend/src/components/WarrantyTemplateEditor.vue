<script setup>
import { ref, computed } from 'vue'
import { ASPECT_RATIOS, DEFAULT_ASPECT_RATIO } from '@/constants/previewOptions'

const selectedRatio = ref(DEFAULT_ASPECT_RATIO)

const props = defineProps({
  modelValue: { type: Object, required: true },
})

const emit = defineEmits(['update:modelValue'])

const config = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// Preview mode
const previewMode = ref('form')

// Update submit button
const updateSubmitButton = (key, value) => {
  const newConfig = { ...config.value }
  newConfig.submit_button = { ...newConfig.submit_button, [key]: value }
  emit('update:modelValue', newConfig)
}

// Update styling
const updateStyling = (key, value) => {
  const newConfig = { ...config.value }
  newConfig.styling = { ...newConfig.styling, [key]: value }
  emit('update:modelValue', newConfig)
}

// Update messages
const updateMessage = (key, value) => {
  const newConfig = { ...config.value }
  newConfig.messages = { ...newConfig.messages, [key]: value }
  emit('update:modelValue', newConfig)
}
</script>

<template>
  <div class="warranty-template-editor">
    <div class="grid lg:grid-cols-2 gap-6">
      <!-- Configuration Panel -->
      <div class="space-y-6">
        <!-- Submit Button Section -->
        <div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
            <svg class="w-5 h-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            Activate Button
          </h3>

          <div class="space-y-4">
            <div>
              <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Button Text</label>
              <input
                type="text"
                :value="config.submit_button.text"
                @input="updateSubmitButton('text', $event.target.value)"
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
                    :value="config.submit_button.bg_color"
                    @input="updateSubmitButton('bg_color', $event.target.value)"
                    class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                  />
                  <input
                    type="text"
                    :value="config.submit_button.bg_color"
                    @input="updateSubmitButton('bg_color', $event.target.value)"
                    class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                  />
                </div>
              </div>
              <div>
                <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Text Color</label>
                <div class="flex items-center gap-2">
                  <input
                    type="color"
                    :value="config.submit_button.text_color"
                    @input="updateSubmitButton('text_color', $event.target.value)"
                    class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                  />
                  <input
                    type="text"
                    :value="config.submit_button.text_color"
                    @input="updateSubmitButton('text_color', $event.target.value)"
                    class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Styling Section -->
        <div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
            <svg class="w-5 h-5 text-zinc-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
            </svg>
            Page Styling
          </h3>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Header Background</label>
              <div class="flex items-center gap-2">
                <input
                  type="color"
                  :value="config.styling.header_bg_color"
                  @input="updateStyling('header_bg_color', $event.target.value)"
                  class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                />
                <input
                  type="text"
                  :value="config.styling.header_bg_color"
                  @input="updateStyling('header_bg_color', $event.target.value)"
                  class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                />
              </div>
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Form Background</label>
              <div class="flex items-center gap-2">
                <input
                  type="color"
                  :value="config.styling.form_bg_color"
                  @input="updateStyling('form_bg_color', $event.target.value)"
                  class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                />
                <input
                  type="text"
                  :value="config.styling.form_bg_color"
                  @input="updateStyling('form_bg_color', $event.target.value)"
                  class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                />
              </div>
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Text Color</label>
              <div class="flex items-center gap-2">
                <input
                  type="color"
                  :value="config.styling.text_color"
                  @input="updateStyling('text_color', $event.target.value)"
                  class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                />
                <input
                  type="text"
                  :value="config.styling.text_color"
                  @input="updateStyling('text_color', $event.target.value)"
                  class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                />
              </div>
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Accent Color</label>
              <div class="flex items-center gap-2">
                <input
                  type="color"
                  :value="config.styling.accent_color"
                  @input="updateStyling('accent_color', $event.target.value)"
                  class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                />
                <input
                  type="text"
                  :value="config.styling.accent_color"
                  @input="updateStyling('accent_color', $event.target.value)"
                  class="flex-1 px-2 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- Messages Section -->
        <div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
            <svg class="w-5 h-5 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
            </svg>
            Response Messages
          </h3>

          <div class="space-y-4">
            <div class="p-3 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800">
              <p class="text-xs font-medium text-green-800 dark:text-green-400 mb-2">Success Message</p>
              <input
                type="text"
                :value="config.messages.success_title"
                @input="updateMessage('success_title', $event.target.value)"
                placeholder="Warranty Activated!"
                class="w-full px-2 py-1.5 text-sm border border-green-300 dark:border-green-700 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white mb-2"
              />
              <textarea
                :value="config.messages.success_message"
                @input="updateMessage('success_message', $event.target.value)"
                placeholder="Your product warranty has been successfully activated."
                rows="2"
                class="w-full px-2 py-1.5 text-sm border border-green-300 dark:border-green-700 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white resize-none"
              ></textarea>
            </div>

            <div class="p-3 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
              <p class="text-xs font-medium text-red-800 dark:text-red-400 mb-2">Already Activated Message</p>
              <input
                type="text"
                :value="config.messages.already_activated_title"
                @input="updateMessage('already_activated_title', $event.target.value)"
                placeholder="Already Activated"
                class="w-full px-2 py-1.5 text-sm border border-red-300 dark:border-red-700 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white mb-2"
              />
              <textarea
                :value="config.messages.already_activated_message"
                @input="updateMessage('already_activated_message', $event.target.value)"
                placeholder="This product warranty has already been activated."
                rows="2"
                class="w-full px-2 py-1.5 text-sm border border-red-300 dark:border-red-700 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white resize-none"
              ></textarea>
            </div>
          </div>
        </div>
      </div>

      <!-- Live Preview Panel -->
      <div class="lg:sticky lg:top-4 h-fit">
        <div class="bg-gray-100 dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
          <!-- Preview Header -->
          <div class="bg-gray-200 dark:bg-gray-800 px-4 py-2 flex items-center justify-between">
            <div class="flex items-center gap-2">
              <div class="flex gap-1.5">
                <div class="w-3 h-3 rounded-full bg-red-500"></div>
                <div class="w-3 h-3 rounded-full bg-yellow-500"></div>
                <div class="w-3 h-3 rounded-full bg-green-500"></div>
              </div>
              <span class="text-xs text-gray-500 dark:text-gray-400 ml-2">Live Preview</span>
            </div>
            <div class="flex items-center gap-2">
              <!-- Aspect Ratio Selector -->
              <select
                v-model="selectedRatio"
                class="text-xs px-2 py-1 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-300 focus:ring-1 focus:ring-[#27272a]"
              >
                <option v-for="ratio in ASPECT_RATIOS" :key="ratio.value" :value="ratio.value">
                  {{ ratio.label }}
                </option>
              </select>
              <!-- Preview mode toggle -->
              <div class="flex gap-1">
                <button
                  @click="previewMode = 'form'"
                  class="px-2 py-1 text-xs rounded transition-colors"
                  :class="previewMode === 'form'
                    ? 'bg-zinc-600 text-white'
                    : 'bg-gray-300 dark:bg-gray-600 text-gray-700 dark:text-gray-300'"
                >
                  Form
                </button>
                <button
                  @click="previewMode = 'success'"
                  class="px-2 py-1 text-xs rounded transition-colors"
                  :class="previewMode === 'success'
                    ? 'bg-green-600 text-white'
                    : 'bg-gray-300 dark:bg-gray-600 text-gray-700 dark:text-gray-300'"
                >
                  Success
                </button>
                <button
                  @click="previewMode = 'error'"
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
              :style="{ backgroundColor: config.styling.form_bg_color }"
            >
              <!-- Phone Screen -->
              <div class="overflow-y-auto" :style="{ aspectRatio: selectedRatio }">
                <!-- Header -->
                <div
                  class="px-4 py-6 text-center"
                  :style="{ backgroundColor: config.styling.header_bg_color }"
                >
                  <svg class="w-12 h-12 mx-auto mb-2 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                  </svg>
                  <h1 class="text-lg font-bold text-white">Warranty Activation</h1>
                  <p class="text-sm text-white/80 mt-1">Register your product warranty</p>
                </div>

                <!-- Form Preview (Sample - actual fields controlled at Product level) -->
                <div v-if="previewMode === 'form'" class="px-4 py-4 space-y-3">
                  <div class="space-y-1">
                    <label class="block text-xs font-medium" :style="{ color: config.styling.text_color }">
                      Full Name <span class="text-red-500">*</span>
                    </label>
                    <input
                      type="text"
                      placeholder="Enter full name"
                      class="w-full px-3 py-2 text-sm border rounded-lg bg-white"
                      :style="{ borderColor: config.styling.accent_color + '40' }"
                      disabled
                    />
                  </div>

                  <div class="space-y-1">
                    <label class="block text-xs font-medium" :style="{ color: config.styling.text_color }">
                      Email <span class="text-red-500">*</span>
                    </label>
                    <input
                      type="email"
                      placeholder="Enter email"
                      class="w-full px-3 py-2 text-sm border rounded-lg bg-white"
                      :style="{ borderColor: config.styling.accent_color + '40' }"
                      disabled
                    />
                  </div>

                  <div class="space-y-1">
                    <label class="block text-xs font-medium" :style="{ color: config.styling.text_color }">
                      Phone <span class="text-red-500">*</span>
                    </label>
                    <input
                      type="text"
                      placeholder="Enter phone number"
                      class="w-full px-3 py-2 text-sm border rounded-lg bg-white"
                      :style="{ borderColor: config.styling.accent_color + '40' }"
                      disabled
                    />
                  </div>

                  <div class="space-y-1">
                    <label class="block text-xs font-medium" :style="{ color: config.styling.text_color }">
                      Purchase Date <span class="text-red-500">*</span>
                    </label>
                    <input
                      type="date"
                      class="w-full px-3 py-2 text-sm border rounded-lg bg-white"
                      :style="{ borderColor: config.styling.accent_color + '40' }"
                      disabled
                    />
                  </div>

                  <div class="space-y-1">
                    <label class="block text-xs font-medium" :style="{ color: config.styling.text_color }">
                      Store Name
                    </label>
                    <input
                      type="text"
                      placeholder="Enter store name"
                      class="w-full px-3 py-2 text-sm border rounded-lg bg-white"
                      :style="{ borderColor: config.styling.accent_color + '40' }"
                      disabled
                    />
                  </div>

                  <!-- Customer Address Section -->
                  <div class="pt-3 border-t border-gray-200">
                    <p class="text-xs font-semibold mb-1" :style="{ color: config.styling.text_color }">Your Address</p>
                    <p class="text-[10px] text-gray-400 mb-2">Required for warranty service</p>
                  </div>

                  <div class="space-y-1">
                    <label class="block text-xs font-medium" :style="{ color: config.styling.text_color }">
                      Country <span class="text-red-500">*</span>
                    </label>
                    <select
                      class="w-full px-3 py-2 text-sm border rounded-lg bg-white text-gray-400"
                      :style="{ borderColor: config.styling.accent_color + '40' }"
                      disabled
                    >
                      <option>Select country</option>
                    </select>
                  </div>

                  <div class="space-y-1">
                    <label class="block text-xs font-medium" :style="{ color: config.styling.text_color }">
                      Province <span class="text-red-500">*</span>
                    </label>
                    <select
                      class="w-full px-3 py-2 text-sm border rounded-lg bg-white text-gray-400"
                      :style="{ borderColor: config.styling.accent_color + '40' }"
                      disabled
                    >
                      <option>Select province</option>
                    </select>
                  </div>

                  <div class="space-y-1">
                    <label class="block text-xs font-medium" :style="{ color: config.styling.text_color }">
                      City <span class="text-red-500">*</span>
                    </label>
                    <select
                      class="w-full px-3 py-2 text-sm border rounded-lg bg-white text-gray-400"
                      :style="{ borderColor: config.styling.accent_color + '40' }"
                      disabled
                    >
                      <option>Select city</option>
                    </select>
                  </div>

                  <div class="space-y-1">
                    <label class="block text-xs font-medium" :style="{ color: config.styling.text_color }">
                      Full Address <span class="text-red-500">*</span>
                    </label>
                    <textarea
                      placeholder="Street, building, house number..."
                      rows="2"
                      class="w-full px-3 py-2 text-sm border rounded-lg bg-white resize-none"
                      :style="{ borderColor: config.styling.accent_color + '40' }"
                      disabled
                    ></textarea>
                  </div>

                  <p class="text-[10px] text-gray-400 italic text-center">
                    * Actual form fields are configured at Product level
                  </p>

                  <!-- Submit Button -->
                  <button
                    class="w-full py-3 rounded-lg font-medium text-sm mt-4"
                    :style="{
                      backgroundColor: config.submit_button.bg_color,
                      color: config.submit_button.text_color
                    }"
                  >
                    {{ config.submit_button.text || 'Activate Warranty' }}
                  </button>
                </div>

                <!-- Success Preview -->
                <div v-else-if="previewMode === 'success'" class="px-4 py-8 text-center">
                  <div
                    class="w-16 h-16 mx-auto rounded-full flex items-center justify-center mb-4"
                    :style="{ backgroundColor: config.styling.accent_color + '20' }"
                  >
                    <svg class="w-8 h-8" :style="{ color: config.styling.accent_color }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                    </svg>
                  </div>
                  <h2
                    class="text-xl font-bold mb-2"
                    :style="{ color: config.styling.text_color }"
                  >
                    {{ config.messages.success_title || 'Warranty Activated!' }}
                  </h2>
                  <p
                    class="text-sm opacity-80 mb-6"
                    :style="{ color: config.styling.text_color }"
                  >
                    {{ config.messages.success_message || 'Your product warranty has been successfully activated.' }}
                  </p>
                </div>

                <!-- Error Preview -->
                <div v-else class="px-4 py-8 text-center">
                  <div class="w-16 h-16 mx-auto rounded-full flex items-center justify-center mb-4 bg-red-100">
                    <svg class="w-8 h-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                  </div>
                  <h2
                    class="text-xl font-bold mb-2"
                    :style="{ color: config.styling.text_color }"
                  >
                    {{ config.messages.already_activated_title || 'Already Activated' }}
                  </h2>
                  <p
                    class="text-sm opacity-80"
                    :style="{ color: config.styling.text_color }"
                  >
                    {{ config.messages.already_activated_message || 'This product warranty has already been activated.' }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>
