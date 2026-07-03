<script setup>
import { ref, computed, onMounted } from 'vue'
import { useBrandingStore, DEFAULT_APP_NAME } from '@/stores/branding'
import { useToast } from '@/composables/useToast'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'

const brandingStore = useBrandingStore()
const toast = useToast()

const form = ref({
  app_name: '',
  logo_url: '',
  header_gradient_start: '#18181b',
  header_gradient_end: '#FFAB2E',
  header_text_color: '#ffffff',
  button_bg_color: '#F5A623',
  button_text_color: '#ffffff'
})

const saving = ref(false)

const previewGradient = computed(() =>
  `linear-gradient(135deg, ${form.value.header_gradient_start} 0%, ${form.value.header_gradient_end} 100%)`
)

function loadForm() {
  form.value = {
    app_name: brandingStore.branding.app_name || DEFAULT_APP_NAME,
    logo_url: brandingStore.branding.logo_url || '',
    header_gradient_start: brandingStore.branding.header_gradient_start || '#18181b',
    header_gradient_end: brandingStore.branding.header_gradient_end || '#FFAB2E',
    header_text_color: brandingStore.branding.header_text_color || '#ffffff',
    button_bg_color: brandingStore.branding.button_bg_color || '#F5A623',
    button_text_color: brandingStore.branding.button_text_color || '#ffffff'
  }
}

async function saveBranding() {
  if (!form.value.app_name.trim()) {
    toast.error('App name is required')
    return
  }

  saving.value = true

  try {
    const success = await brandingStore.updateBranding(form.value)
    if (success) {
      toast.success('Branding saved successfully')
    } else {
      toast.error(brandingStore.error || 'Failed to save branding')
    }
  } catch (error) {
    toast.error(error.message || 'Failed to save branding')
  } finally {
    saving.value = false
  }
}

function resetToDefaults() {
  form.value = {
    app_name: DEFAULT_APP_NAME,
    logo_url: '',
    header_gradient_start: '#18181b',
    header_gradient_end: '#FFAB2E',
    header_text_color: '#ffffff',
    button_bg_color: '#F5A623',
    button_text_color: '#ffffff'
  }
}

onMounted(async () => {
  await brandingStore.fetchBranding()
  loadForm()
})
</script>

<template>
  <div>
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">Appearance</h1>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Settings Form -->
      <Card class="p-6">
        <h2 class="text-lg font-semibold mb-4 text-gray-900 dark:text-white">Configuration</h2>
        <div class="space-y-4">
          <!-- App Name -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">App Name</label>
            <Input v-model="form.app_name" :placeholder="DEFAULT_APP_NAME" />
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Displayed in dashboard header and email templates</p>
          </div>

          <!-- Logo URL -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Logo URL (Optional)</label>
            <Input v-model="form.logo_url" placeholder="https://example.com/logo.png" />
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Leave empty to show app name only</p>
          </div>

          <!-- Header Colors -->
          <div class="pt-4 border-t border-gray-200 dark:border-gray-700">
            <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">Header Gradient</h3>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">Start Color</label>
                <div class="flex items-center gap-2">
                  <input
                    type="color"
                    v-model="form.header_gradient_start"
                    class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                  />
                  <Input v-model="form.header_gradient_start" class="flex-1 font-mono text-sm" />
                </div>
              </div>
              <div>
                <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">End Color</label>
                <div class="flex items-center gap-2">
                  <input
                    type="color"
                    v-model="form.header_gradient_end"
                    class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                  />
                  <Input v-model="form.header_gradient_end" class="flex-1 font-mono text-sm" />
                </div>
              </div>
            </div>
          </div>

          <!-- Header Text Color -->
          <div>
            <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">Header Text Color</label>
            <div class="flex items-center gap-2">
              <input
                type="color"
                v-model="form.header_text_color"
                class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
              />
              <Input v-model="form.header_text_color" class="flex-1 font-mono text-sm" />
            </div>
          </div>

          <!-- Button Colors -->
          <div class="pt-4 border-t border-gray-200 dark:border-gray-700">
            <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">Button Styling</h3>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">Background</label>
                <div class="flex items-center gap-2">
                  <input
                    type="color"
                    v-model="form.button_bg_color"
                    class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                  />
                  <Input v-model="form.button_bg_color" class="flex-1 font-mono text-sm" />
                </div>
              </div>
              <div>
                <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">Text Color</label>
                <div class="flex items-center gap-2">
                  <input
                    type="color"
                    v-model="form.button_text_color"
                    class="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-600"
                  />
                  <Input v-model="form.button_text_color" class="flex-1 font-mono text-sm" />
                </div>
              </div>
            </div>
          </div>

          <!-- Actions -->
          <div class="pt-4 flex gap-3">
            <Button @click="saveBranding" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save Changes' }}
            </Button>
            <Button variant="outline" @click="resetToDefaults">
              Reset to Defaults
            </Button>
          </div>
        </div>
      </Card>

      <!-- Live Preview -->
      <div>
        <Card class="p-6 sticky top-4">
          <h2 class="text-lg font-semibold mb-4 text-gray-900 dark:text-white">Email Preview</h2>
          <div class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden shadow-sm">
            <!-- Email Header -->
            <div
              class="p-6 text-center"
              :style="{
                background: previewGradient,
                color: form.header_text_color
              }"
            >
              <img
                v-if="form.logo_url"
                :src="form.logo_url"
                :alt="form.app_name"
                class="h-10 mx-auto mb-2"
                @error="$event.target.style.display = 'none'"
              />
              <h1 class="text-2xl font-bold">{{ form.app_name || 'App Name' }}</h1>
            </div>

            <!-- Email Content -->
            <div class="p-6 bg-white dark:bg-gray-800">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">Sample Email Content</h2>
              <p class="text-gray-600 dark:text-gray-400 mb-4">
                This is a preview of how your email templates will look with the current branding settings.
              </p>

              <div
                class="p-3 mb-4 rounded-l"
                :style="{
                  background: '#f0f4ff',
                  borderLeft: `4px solid ${form.button_bg_color}`
                }"
              >
                <strong>Info Box:</strong> Important information appears here.
              </div>

              <div class="text-center">
                <a
                  href="#"
                  class="inline-block px-6 py-3 rounded-md font-semibold"
                  :style="{
                    background: form.button_bg_color,
                    color: form.button_text_color
                  }"
                >
                  Action Button
                </a>
              </div>
            </div>

            <!-- Email Footer -->
            <div class="p-4 text-center bg-gray-50 dark:bg-gray-900 text-xs text-gray-500 dark:text-gray-400">
              <p>&copy; {{ new Date().getFullYear() }} {{ form.app_name || 'App Name' }}. All rights reserved.</p>
              <p>This is an automated message.</p>
            </div>
          </div>
        </Card>
      </div>
    </div>
  </div>
</template>
