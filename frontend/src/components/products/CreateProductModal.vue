<script setup>
import { ref, computed, watch, toRef, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAPI } from '@/composables/useAPI'
import { useToast } from '@/composables/useToast'
import { useEscapeKey } from '@/composables/useEscapeKey'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { isTourActive, getTourNonce } from '@/composables/useTour.js'

const props = defineProps({
  show: { type: Boolean, required: true },
})

const emit = defineEmits(['close', 'created'])

const { post } = useAPI()
const toast = useToast()
const router = useRouter()

const creating = ref(false)
const errorMessage = ref('')

const newProduct = ref({
  product_name: '',
  product_code: '',
  description: '',
})

const canCreate = computed(() => newProduct.value.product_name.trim() !== '')

const resetForm = () => {
  newProduct.value = { product_name: '', product_code: '', description: '' }
  errorMessage.value = ''
}

// Reset form whenever the modal opens.
watch(() => props.show, (isShowing) => {
  if (isShowing) resetForm()
})

const createProduct = async () => {
  if (!canCreate.value) return
  errorMessage.value = ''

  try {
    creating.value = true
    const response = await post('/tenant/products', {
      product_name: newProduct.value.product_name,
      product_code: newProduct.value.product_code || null,
      description: newProduct.value.description || null,
    })

    if (!response.success) {
      errorMessage.value = response.message || 'Failed to create product'
      return
    }

    const productId = response.data?.id
    emit('created')
    emit('close')
    toast.success('Product created')
    if (productId) {
      router.push(`/tenant/products/${productId}`)
    }
  } catch (error) {
    console.error('Failed to create product:', error)
    errorMessage.value = error.response?.data?.message || 'Failed to create product. Please try again.'
  } finally {
    creating.value = false
  }
}

// Tour auto-fill listener — sets reactive values directly (no DOM manipulation)
function handleTourSetValue(e) {
  if (!isTourActive()) return
  if (e.detail._nonce !== getTourNonce()) return
  const { field, value } = e.detail
  switch (field) {
    case 'product_name':
      newProduct.value.product_name = value
      break
    case 'description':
      newProduct.value.description = value
      break
  }
}
onMounted(() => window.addEventListener('tour-set-value', handleTourSetValue))
onUnmounted(() => window.removeEventListener('tour-set-value', handleTourSetValue))

// Close modal on Escape key (accessibility best practice)
useEscapeKey(() => emit('close'), toRef(props, 'show'))
</script>

<template>
  <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <div class="fixed inset-0 bg-black/50" @click="emit('close')"></div>
    <div class="relative z-10 w-full max-w-md bg-white dark:bg-gray-800 rounded-lg shadow-xl">
      <!-- Header -->
      <div class="p-6 border-b border-gray-200 dark:border-gray-700">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white">Create New Product</h2>
      </div>

      <!-- Error Message -->
      <div v-if="errorMessage" class="mx-6 mt-4 p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 rounded-lg text-sm">
        {{ errorMessage }}
      </div>

      <!-- Body -->
      <div class="p-6 space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Product Name <span class="text-red-500">*</span>
          </label>
          <Input v-model="newProduct.product_name" placeholder="Enter product name" data-tour="product-name-input" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Product Code</label>
          <Input v-model="newProduct.product_code" placeholder="e.g., PROD001" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
          <textarea
            v-model="newProduct.description"
            rows="2"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500"
            placeholder="Product description..."
            data-tour="product-description"
          ></textarea>
        </div>

        <p class="text-xs text-gray-500 dark:text-gray-400">
          Configure the landing page, gallery, warranty and more in the product's settings after creating it.
        </p>
      </div>

      <!-- Footer -->
      <div class="p-6 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-2">
        <Button variant="outline" @click="emit('close')">Cancel</Button>
        <Button @click="createProduct" :disabled="creating || !canCreate" data-tour="create-product-btn">
          {{ creating ? 'Creating...' : 'Create' }}
        </Button>
      </div>
    </div>
  </div>
</template>
