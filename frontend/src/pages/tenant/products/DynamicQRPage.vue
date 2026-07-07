<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAPI } from '@/composables/useAPI'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import CreateProductModal from '@/components/products/CreateProductModal.vue'
import CreateBatchModal from '@/components/products/CreateBatchModal.vue'
import { QrCode, Plus, Package, Settings, Search, History } from 'lucide-vue-next'
import { useTour, isTourActive, getTourNonce } from '@/composables/useTour.js'

const router = useRouter()
const { get } = useAPI()

// Create modal state
const showCreateModal = ref(false)

const loading = ref(true)
const products = ref([])
const searchQuery = ref('')

// Filtered products based on search
const filteredProducts = computed(() => {
  if (!searchQuery.value) return products.value
  const q = searchQuery.value.toLowerCase()
  return products.value.filter(p =>
    p.product_name.toLowerCase().includes(q) ||
    p.product_code?.toLowerCase().includes(q)
  )
})

// Create Batch Modal
const showBatchModal = ref(false)
const selectedProductForBatch = ref(null)

const fetchProducts = async () => {
  try {
    loading.value = true
    // The backend caps limit at 100, so walk every page — search is
    // client-side and must cover the whole catalog.
    const all = []
    let pageNum = 1
    let totalPage = 1
    do {
      const response = await get('/tenant/products', {
        page: pageNum,
        limit: 100,
      })
      if (!response.success || !response.data) break
      all.push(...(response.data.products || []))
      totalPage = response.data.pagination?.total_page || 1
      pageNum++
    } while (pageNum <= totalPage)
    products.value = all
  } catch (error) {
    console.error('Failed to fetch products:', error)
  } finally {
    loading.value = false
  }
}

// Batch creation
const openBatchModal = (product) => {
  selectedProductForBatch.value = product
  showBatchModal.value = true
}

const onBatchCreated = () => {
  showBatchModal.value = false
  // Redirect to batch history page for the product just batched.
  if (selectedProductForBatch.value) {
    router.push(`/tenant/products/${selectedProductForBatch.value.id}/batches`)
  }
}

const viewProductDetail = (productId) => {
  router.push(`/tenant/products/${productId}`)
}

const viewBatchHistory = (productId) => {
  router.push(`/tenant/products/${productId}/batches`)
}

const goToCreateProduct = () => {
  showCreateModal.value = true
}

const onProductCreated = () => {
  fetchProducts()
}

const tour = useTour()

const closeAllModals = () => {
  showCreateModal.value = false
  showBatchModal.value = false
}

// Tour auto-fill listener — sets reactive values directly (no DOM manipulation)
// Batch-related fields are handled inside CreateBatchModal.
function handleTourSetValue(e) {
  if (!isTourActive()) return
  if (e.detail._nonce !== getTourNonce()) return
  const { field, value } = e.detail
  switch (field) {
    case 'search_query':
      searchQuery.value = value
      break
  }
}

onMounted(async () => {
  await fetchProducts()
  tour.resumeIfActive()
  window.addEventListener('tour-cancelled', closeAllModals)
  window.addEventListener('tour-set-value', handleTourSetValue)
})

onUnmounted(() => {
  window.removeEventListener('tour-cancelled', closeAllModals)
  window.removeEventListener('tour-set-value', handleTourSetValue)
})
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-4">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Dynamic QR Products</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
          Manage products with unique QR codes per unit. Create batches and request printing.
        </p>
      </div>
      <Button @click="goToCreateProduct" data-tour="add-product-btn">
        <Plus class="w-4 h-4 mr-2" />
        Add New Product
      </Button>
    </div>

    <!-- Search Bar -->
    <div class="mb-4 max-w-xs">
      <div class="relative">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <Input
          v-model="searchQuery"
          placeholder="Search products..."
          class="pl-9"
          data-tour="search-product"
        />
      </div>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <div v-else>
      <!-- Empty State -->
      <Card v-if="products.length === 0" class="p-6">
        <div class="text-center py-8">
          <QrCode class="w-16 h-16 text-gray-300 dark:text-gray-600 mx-auto mb-4" />
          <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No dynamic QR products yet</h3>
          <p class="text-gray-500 dark:text-gray-400 mb-4">
            Create a dynamic QR product to generate unique QR codes for each unit.
          </p>
          <Button @click="goToCreateProduct">
            <Plus class="w-4 h-4 mr-2" />
            Create Dynamic Product
          </Button>
        </div>
      </Card>

      <!-- Products List -->
      <div v-else class="space-y-4">
        <Card v-for="(product, pIdx) in filteredProducts" :key="product.id" class="p-4">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <Package class="w-5 h-5 text-zinc-600" />
              <div>
                <h3 class="font-semibold text-gray-900 dark:text-white">{{ product.product_name }}</h3>
                <p v-if="product.product_code" class="text-sm text-gray-500 dark:text-gray-400">
                  {{ product.product_code }}
                </p>
              </div>
              <span class="px-2 py-1 text-xs rounded-full bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400">
                Dynamic QR
              </span>
              <span v-if="product.warranty_enabled" class="px-2 py-1 text-xs rounded-full bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400">
                Warranty
              </span>
            </div>
            <div class="flex items-center gap-2">
              <Button variant="outline" size="sm" @click="viewProductDetail(product.id)" :data-tour="pIdx === 0 ? 'product-settings-btn' : undefined">
                <Settings class="w-4 h-4 mr-1" />
                Settings
              </Button>
              <Button variant="outline" size="sm" @click="viewBatchHistory(product.id)">
                <History class="w-4 h-4 mr-1" />
                Batch History
              </Button>
              <Button variant="outline" size="sm" @click="openBatchModal(product)" :data-tour="pIdx === 0 ? 'new-batch-btn' : undefined">
                <Plus class="w-4 h-4 mr-1" />
                New Batch
              </Button>
            </div>
          </div>
        </Card>
      </div>
    </div>

    <!-- Create Batch Modal -->
    <CreateBatchModal
      :open="showBatchModal"
      :product-id="selectedProductForBatch?.id"
      :product-name="selectedProductForBatch?.product_name"
      :warranty-enabled="!!selectedProductForBatch?.warranty_enabled"
      @close="showBatchModal = false"
      @created="onBatchCreated"
    />

    <!-- Create Product Modal -->
    <CreateProductModal
      :show="showCreateModal"
      @close="showCreateModal = false"
      @created="onProductCreated"
    />
  </div>
</template>
