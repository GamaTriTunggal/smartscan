<script setup>
import { ref, onMounted } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useToast } from '@/composables/useToast'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'

const { get, put } = useAPI()
const toast = useToast()

const loading = ref(true)
const saving = ref(false)

const form = ref({
  enabled: false,
  url: '',
  secret: '',
  events: [],
})
const hasSecret = ref(false)
const availableEvents = ref([])

async function fetchConfig() {
  try {
    loading.value = true
    const response = await get('/tenant/integrations/webhook')
    if (response.success && response.data) {
      const { config, has_secret, events } = response.data
      form.value = {
        enabled: config?.enabled || false,
        url: config?.url || '',
        secret: '',
        events: config?.events || [],
      }
      hasSecret.value = has_secret || false
      availableEvents.value = events || []
    }
  } catch (error) {
    console.error('Failed to fetch webhook config:', error)
    toast.error('Failed to load webhook settings')
  } finally {
    loading.value = false
  }
}

function toggleEvent(event) {
  const index = form.value.events.indexOf(event)
  if (index === -1) {
    form.value.events.push(event)
  } else {
    form.value.events.splice(index, 1)
  }
}

async function saveConfig() {
  if (form.value.enabled && !form.value.url) {
    toast.error('Webhook URL is required when the webhook is enabled')
    return
  }

  try {
    saving.value = true
    const response = await put('/tenant/integrations/webhook', {
      url: form.value.url,
      enabled: form.value.enabled,
      events: form.value.events,
      // Empty secret input means "keep existing secret" (backend expects null)
      secret: form.value.secret || null,
    })
    if (response.success) {
      toast.success('Webhook settings saved')
      if (form.value.secret) {
        hasSecret.value = true
      }
      form.value.secret = ''
    } else {
      toast.error(response.message || 'Failed to save webhook settings')
    }
  } catch (error) {
    console.error('Failed to save webhook config:', error)
    toast.error(error.response?.data?.message || 'Failed to save webhook settings')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  fetchConfig()
})
</script>

<template>
  <div>
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Integrations</h1>
      <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
        Send event notifications from your tenant to external systems via webhooks
      </p>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <div v-else class="max-w-2xl space-y-6">
      <Card class="p-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Outbound Webhook</h2>

        <form @submit.prevent="saveConfig" class="space-y-5">
          <!-- Enabled toggle -->
          <div class="flex items-center justify-between">
            <div>
              <label for="webhook-enabled" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Enabled</label>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Deliver events to the URL below</p>
            </div>
            <button
              id="webhook-enabled"
              type="button"
              role="switch"
              :aria-checked="form.enabled"
              :class="[
                'relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-[#27272a] focus:ring-offset-2 dark:focus:ring-offset-gray-800',
                form.enabled ? 'bg-zinc-500' : 'bg-gray-300 dark:bg-gray-600'
              ]"
              @click="form.enabled = !form.enabled"
            >
              <span
                :class="[
                  'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform duration-200',
                  form.enabled ? 'translate-x-6' : 'translate-x-1'
                ]"
              ></span>
            </button>
          </div>

          <!-- URL -->
          <div>
            <label for="webhook-url" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Webhook URL</label>
            <Input
              id="webhook-url"
              v-model="form.url"
              type="url"
              placeholder="https://example.com/webhooks/smartscan"
              :disabled="saving"
            />
          </div>

          <!-- Secret -->
          <div>
            <label for="webhook-secret" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Signing Secret</label>
            <Input
              id="webhook-secret"
              v-model="form.secret"
              type="password"
              :placeholder="hasSecret ? '•••••• (unchanged)' : 'Enter a signing secret'"
              :disabled="saving"
              autocomplete="new-password"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ hasSecret ? 'A secret is configured. Leave empty to keep it, or enter a new value to replace it.' : 'Used to sign each delivery so your endpoint can verify authenticity.' }}
            </p>
          </div>

          <!-- Events -->
          <div>
            <span class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Events</span>
            <p class="text-xs text-gray-500 dark:text-gray-400 mb-2">
              Select the events to deliver. Leave all unchecked to receive every event.
            </p>
            <div class="space-y-2 rounded-lg border border-gray-200 dark:border-gray-700 p-3 max-h-64 overflow-y-auto">
              <label
                v-for="event in availableEvents"
                :key="event"
                class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 cursor-pointer"
              >
                <input
                  type="checkbox"
                  class="h-4 w-4 rounded border-gray-300 dark:border-gray-600 text-zinc-600 focus:ring-[#27272a] dark:bg-gray-700"
                  :checked="form.events.includes(event)"
                  :disabled="saving"
                  @change="toggleEvent(event)"
                />
                <span class="font-mono text-xs">{{ event }}</span>
              </label>
              <p v-if="availableEvents.length === 0" class="text-xs text-gray-400 dark:text-gray-500">
                No event types available.
              </p>
            </div>
          </div>

          <div class="pt-2">
            <Button type="submit" :disabled="saving" :loading="saving">
              {{ saving ? 'Saving...' : 'Save' }}
            </Button>
          </div>
        </form>
      </Card>

      <!-- How it works -->
      <Card class="p-6 bg-zinc-50/50 dark:bg-zinc-900/10 border-zinc-100 dark:border-zinc-900/30">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-2">How webhook delivery works</h3>
        <ul class="text-xs text-gray-600 dark:text-gray-400 space-y-1.5 list-disc pl-4">
          <li>Events are delivered as outbound <span class="font-mono">POST</span> requests with a JSON body.</li>
          <li>
            Each request is signed with HMAC-SHA256 using your signing secret. The signature is sent in the
            <span class="font-mono">X-Smartscan-Signature</span> header as <span class="font-mono">sha256=&lt;hex&gt;</span>.
            Verify it before trusting the payload.
          </li>
          <li>Requests time out after 5 seconds. Delivery is best-effort — failed deliveries are not retried.</li>
        </ul>
      </Card>
    </div>
  </div>
</template>
