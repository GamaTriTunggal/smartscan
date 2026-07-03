<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useEscapeKey } from '@/composables/useEscapeKey'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { Upload, Trash2, Star, GripVertical, X, Image as ImageIcon } from 'lucide-vue-next'

// Get backend base URL for uploads (without /api/v1)
const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
const UPLOAD_BASE = API_BASE.replace('/api/v1', '')

// Convert relative upload URLs to absolute backend URLs
function getImageUrl(url) {
  if (!url) return ''
  // Already absolute URL (e.g., placehold.co)
  if (url.startsWith('http://') || url.startsWith('https://')) {
    return url
  }
  // Relative upload URL - prepend backend base
  if (url.startsWith('/uploads/')) {
    return UPLOAD_BASE + url
  }
  return url
}

const props = defineProps({
  productId: {
    type: String,
    required: true
  }
})

const { get, post, put, del } = useAPI()

const images = ref([])
const loading = ref(false)
const uploading = ref(false)
const error = ref('')

// Lightbox state
const lightboxOpen = ref(false)
const lightboxIndex = ref(0)

// Edit caption modal
const editingImage = ref(null)
const editCaption = ref('')

const MAX_IMAGES = 15

const canUpload = computed(() => images.value.length < MAX_IMAGES)

async function fetchImages() {
  loading.value = true
  error.value = ''
  try {
    const response = await get(`/tenant/products/${props.productId}/images`)
    if (response.success) {
      images.value = response.data?.images || []
    } else {
      error.value = response.message || 'Failed to load images'
    }
  } catch (err) {
    console.error('Failed to fetch images:', err)
    error.value = 'Failed to load images'
  } finally {
    loading.value = false
  }
}

async function handleUpload(event) {
  const file = event.target.files?.[0]
  if (!file) return

  // Validate file type
  const allowedTypes = ['image/jpeg', 'image/png', 'image/webp']
  if (!allowedTypes.includes(file.type)) {
    alert('Only JPEG, PNG, and WebP images are allowed')
    return
  }

  // Validate file size (5MB)
  if (file.size > 5 * 1024 * 1024) {
    alert('Image must be less than 5MB')
    return
  }

  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('image', file)
    // Set as main if it's the first image
    if (images.value.length === 0) {
      formData.append('is_main', 'true')
    }

    const response = await post(`/tenant/products/${props.productId}/images`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })

    if (response.success) {
      await fetchImages()
    } else {
      alert(response.message || 'Failed to upload image')
    }
  } catch (err) {
    console.error('Failed to upload image:', err)
    alert('Failed to upload image')
  } finally {
    uploading.value = false
    // Reset input
    event.target.value = ''
  }
}

async function setAsMain(imageId) {
  try {
    const response = await put(`/tenant/products/${props.productId}/images/${imageId}/main`)
    if (response.success) {
      await fetchImages()
    } else {
      alert(response.message || 'Failed to set main image')
    }
  } catch (err) {
    console.error('Failed to set main image:', err)
  }
}

async function deleteImage(imageId) {
  if (!confirm('Delete this image?')) return

  try {
    const response = await del(`/tenant/products/${props.productId}/images/${imageId}`)
    if (response.success) {
      await fetchImages()
    } else {
      alert(response.message || 'Failed to delete image')
    }
  } catch (err) {
    console.error('Failed to delete image:', err)
  }
}

function openEditCaption(image) {
  editingImage.value = image
  editCaption.value = image.caption || ''
}

async function saveCaption() {
  if (!editingImage.value) return

  try {
    const response = await put(`/tenant/products/${props.productId}/images/${editingImage.value.id}`, {
      caption: editCaption.value
    })
    if (response.success) {
      await fetchImages()
      editingImage.value = null
    } else {
      alert(response.message || 'Failed to update caption')
    }
  } catch (err) {
    console.error('Failed to update caption:', err)
  }
}

function openLightbox(index) {
  lightboxIndex.value = index
  lightboxOpen.value = true
}

function closeLightbox() {
  lightboxOpen.value = false
}

function nextImage() {
  if (lightboxIndex.value < images.value.length - 1) {
    lightboxIndex.value++
  }
}

function prevImage() {
  if (lightboxIndex.value > 0) {
    lightboxIndex.value--
  }
}

// Close lightbox on Escape key
useEscapeKey(closeLightbox, lightboxOpen)

// Handle arrow key navigation in lightbox
function handleArrowKeys(e) {
  if (!lightboxOpen.value) return
  if (e.key === 'ArrowRight') nextImage()
  if (e.key === 'ArrowLeft') prevImage()
}

onMounted(() => {
  fetchImages()
  window.addEventListener('keydown', handleArrowKeys)
})
onUnmounted(() => {
  window.removeEventListener('keydown', handleArrowKeys)
})

// Re-fetch when product changes
watch(() => props.productId, () => {
  fetchImages()
})
</script>

<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100">Product Gallery</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          Upload up to {{ MAX_IMAGES }} images. The main image will be shown first.
        </p>
      </div>
      <span class="text-sm text-gray-500">
        {{ images.length }} / {{ MAX_IMAGES }} images
      </span>
    </div>

    <!-- Error -->
    <div v-if="error" class="p-3 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300 rounded-lg text-sm">
      {{ error }}
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-8">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-600"></div>
    </div>

    <!-- Images Grid -->
    <div v-else class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-4">
      <!-- Existing Images -->
      <div
        v-for="(image, index) in images"
        :key="image.id"
        class="relative group aspect-square rounded-lg overflow-hidden border-2 transition-all"
        :class="image.is_main ? 'border-zinc-500 ring-2 ring-zinc-500/20' : 'border-gray-200 dark:border-gray-700'"
      >
        <img
          :src="getImageUrl(image.image_url)"
          :alt="image.caption || 'Product image'"
          class="w-full h-full object-cover cursor-pointer"
          @click="openLightbox(index)"
        />

        <!-- Main badge -->
        <div v-if="image.is_main" class="absolute top-2 left-2 bg-zinc-500 text-white text-xs px-2 py-0.5 rounded-full">
          Main
        </div>

        <!-- Hover overlay -->
        <div class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
          <button
            v-if="!image.is_main"
            @click="setAsMain(image.id)"
            class="p-2 bg-white rounded-full text-gray-700 hover:bg-zinc-50 hover:text-zinc-600"
            title="Set as main"
          >
            <Star class="w-4 h-4" />
          </button>
          <button
            @click="openEditCaption(image)"
            class="p-2 bg-white rounded-full text-gray-700 hover:bg-zinc-50 hover:text-zinc-600"
            title="Edit caption"
          >
            <ImageIcon class="w-4 h-4" />
          </button>
          <button
            @click="deleteImage(image.id)"
            class="p-2 bg-white rounded-full text-gray-700 hover:bg-red-50 hover:text-red-600"
            title="Delete"
          >
            <Trash2 class="w-4 h-4" />
          </button>
        </div>

        <!-- Caption -->
        <div v-if="image.caption" class="absolute bottom-0 left-0 right-0 bg-black/60 text-white text-xs px-2 py-1 truncate">
          {{ image.caption }}
        </div>
      </div>

      <!-- Upload button -->
      <label
        v-if="canUpload"
        class="aspect-square rounded-lg border-2 border-dashed flex flex-col items-center justify-center transition-colors"
        :class="uploading
          ? 'border-zinc-500 bg-zinc-50 dark:bg-zinc-900/30 cursor-wait'
          : 'border-gray-300 dark:border-gray-600 cursor-pointer hover:border-zinc-500 hover:bg-zinc-50/50 dark:hover:bg-zinc-900/20'"
      >
        <!-- Uploading state with spinner -->
        <template v-if="uploading">
          <div class="relative">
            <div class="animate-spin rounded-full h-10 w-10 border-3 border-zinc-200 dark:border-zinc-800 border-t-zinc-600 dark:border-t-zinc-400"></div>
          </div>
          <span class="text-sm text-zinc-600 dark:text-zinc-400 mt-3 font-medium">Uploading...</span>
          <span class="text-xs text-gray-500 dark:text-gray-400 mt-1">Please wait</span>
        </template>
        <!-- Normal state -->
        <template v-else>
          <Upload class="w-8 h-8 text-gray-400 mb-2" />
          <span class="text-sm text-gray-500">Add Image</span>
        </template>
        <input
          type="file"
          accept="image/jpeg,image/png,image/webp"
          class="hidden"
          :disabled="uploading"
          @change="handleUpload"
        />
      </label>
    </div>

    <!-- Empty state -->
    <div v-if="!loading && images.length === 0" class="text-center py-8 text-gray-500">
      <ImageIcon class="w-12 h-12 mx-auto mb-2 text-gray-300" />
      <p>No images uploaded yet</p>
    </div>

    <!-- Lightbox -->
    <Teleport to="body">
      <div
        v-if="lightboxOpen"
        class="fixed inset-0 z-50 bg-black/90 flex items-center justify-center"
        @click.self="closeLightbox"
      >
        <!-- Close button -->
        <button
          @click="closeLightbox"
          class="absolute top-4 right-4 p-2 text-white hover:text-gray-300"
        >
          <X class="w-8 h-8" />
        </button>

        <!-- Navigation -->
        <button
          v-if="lightboxIndex > 0"
          @click="prevImage"
          class="absolute left-4 p-2 text-white hover:text-gray-300"
        >
          <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <button
          v-if="lightboxIndex < images.length - 1"
          @click="nextImage"
          class="absolute right-4 p-2 text-white hover:text-gray-300"
        >
          <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </button>

        <!-- Image -->
        <div class="max-w-4xl max-h-[90vh] px-4">
          <img
            :src="getImageUrl(images[lightboxIndex]?.image_url)"
            :alt="images[lightboxIndex]?.caption || 'Product image'"
            class="max-w-full max-h-[85vh] object-contain"
          />
          <p v-if="images[lightboxIndex]?.caption" class="text-white text-center mt-4">
            {{ images[lightboxIndex].caption }}
          </p>
        </div>

        <!-- Counter -->
        <div class="absolute bottom-4 left-1/2 -translate-x-1/2 text-white text-sm">
          {{ lightboxIndex + 1 }} / {{ images.length }}
        </div>
      </div>
    </Teleport>

    <!-- Edit Caption Modal -->
    <Teleport to="body">
      <div
        v-if="editingImage"
        class="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4"
        @click.self="editingImage = null"
      >
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md p-6">
          <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Edit Caption</h3>
          <Input
            v-model="editCaption"
            placeholder="Enter image caption..."
            maxlength="255"
          />
          <div class="flex justify-end gap-3 mt-4">
            <Button variant="ghost" @click="editingImage = null">Cancel</Button>
            <Button @click="saveCaption">Save</Button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
