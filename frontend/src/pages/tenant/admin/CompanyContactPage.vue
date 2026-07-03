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
  phone: '',
  whatsapp: '',
  email: '',
  website: '',
  address: '',
})

const fetchContact = async () => {
  try {
    loading.value = true
    const response = await get('/tenant/company-contact')
    if (response.success && response.data?.contact) {
      const c = response.data.contact
      form.value = {
        phone: c.phone || '',
        whatsapp: c.whatsapp || '',
        email: c.email || '',
        website: c.website || '',
        address: c.address || '',
      }
    }
  } catch (error) {
    console.error('Failed to load company contact:', error)
    toast.error('Failed to load company contact')
  } finally {
    loading.value = false
  }
}

const save = async () => {
  try {
    saving.value = true
    const response = await put('/tenant/company-contact', {
      phone: form.value.phone,
      whatsapp: form.value.whatsapp,
      email: form.value.email,
      website: form.value.website,
      address: form.value.address,
    })
    if (response.success) {
      toast.success('Company contact saved')
    } else {
      toast.error(response.message || 'Failed to save company contact')
    }
  } catch (error) {
    console.error('Failed to save company contact:', error)
    toast.error(error.response?.data?.message || 'Failed to save company contact')
  } finally {
    saving.value = false
  }
}

onMounted(fetchContact)
</script>

<template>
  <div>
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Company Contact</h1>
      <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
        Shown to consumers on every public page. Leave a field empty to hide it.
      </p>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <Card v-else class="p-6 max-w-xl">
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Phone</label>
          <Input v-model="form.phone" placeholder="e.g., +62 21 1234 5678" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">WhatsApp</label>
          <Input v-model="form.whatsapp" placeholder="e.g., +62 812 3456 7890" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Email</label>
          <Input v-model="form.email" type="email" placeholder="e.g., support@example.com" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Website</label>
          <Input v-model="form.website" placeholder="e.g., www.example.com" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Address</label>
          <textarea
            v-model="form.address"
            rows="3"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
            placeholder="Street, city, postal code, country"
          ></textarea>
        </div>

        <div class="flex justify-end pt-2">
          <Button :disabled="saving" @click="save">
            {{ saving ? 'Saving...' : 'Save' }}
          </Button>
        </div>
      </div>
    </Card>
  </div>
</template>
