<script setup>
import { ref, onMounted, computed } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useListFilter } from '@/composables/useListFilter'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { getSocialIconPath } from '@/lib/socialIcons'

const { get, post, put, del } = useAPI()

// State
const platforms = ref([])
const loading = ref(false)
const showModal = ref(false)
const editingPlatform = ref(null)
const statusFilter = ref('active')
const { search: searchQuery, watchFilter } = useListFilter(fetchPlatforms)
watchFilter(statusFilter)

const form = ref({
  code: '',
  name: '',
  icon: '',
  base_url: '',
  deep_link_pattern: '',
  placeholder_text: '',
  display_order: 0
})

const errorMessage = ref('')

async function fetchPlatforms() {
  loading.value = true
  try {
    let url = `/tenant/social-media/platforms/all?status=${statusFilter.value}`
    if (searchQuery.value) {
      url += `&search=${searchQuery.value}`
    }
    const response = await get(url)
    if (response.success) {
      platforms.value = response.data?.platforms || []
    }
  } catch (error) {
    console.error('Failed to fetch platforms:', error)
  } finally {
    loading.value = false
  }
}

function openCreateModal() {
  editingPlatform.value = null
  form.value = {
    code: '',
    name: '',
    icon: '',
    base_url: '',
    deep_link_pattern: '',
    placeholder_text: '',
    display_order: 0
  }
  errorMessage.value = ''
  showModal.value = true
}

function openEditModal(platform) {
  editingPlatform.value = platform
  form.value = {
    code: platform.code,
    name: platform.name,
    icon: platform.icon || '',
    base_url: platform.base_url || '',
    deep_link_pattern: platform.deep_link_pattern || '',
    placeholder_text: platform.placeholder_text || '',
    display_order: platform.display_order || 0
  }
  errorMessage.value = ''
  showModal.value = true
}

async function savePlatform() {
  errorMessage.value = ''
  try {
    if (editingPlatform.value) {
      const response = await put(`/tenant/social-media/platforms/${editingPlatform.value.id}`, form.value)
      if (response.success) {
        showModal.value = false
        fetchPlatforms()
      } else {
        errorMessage.value = response.message || 'Failed to save platform'
      }
    } else {
      const response = await post('/tenant/social-media/platforms', form.value)
      if (response.success) {
        showModal.value = false
        fetchPlatforms()
      } else {
        errorMessage.value = response.message || 'Failed to save platform'
      }
    }
  } catch (error) {
    console.error('Failed to save platform:', error)
    errorMessage.value = error.response?.data?.message || 'Failed to save platform'
  }
}

async function deletePlatform(platform) {
  if (!confirm(`Are you sure you want to delete "${platform.name}"?`)) return

  try {
    const response = await del(`/tenant/social-media/platforms/${platform.id}`)
    if (response.success) {
      fetchPlatforms()
    } else {
      alert(response.message || 'Failed to delete platform')
    }
  } catch (error) {
    console.error('Failed to delete platform:', error)
    alert(error.response?.data?.message || 'Failed to delete platform')
  }
}

async function restorePlatform(platform) {
  if (!confirm(`Are you sure you want to restore "${platform.name}"?`)) return

  try {
    const response = await post(`/tenant/social-media/platforms/${platform.id}/restore`)
    if (response.success) {
      fetchPlatforms()
    } else {
      alert(response.message || 'Failed to restore platform')
    }
  } catch (error) {
    console.error('Failed to restore platform:', error)
    alert(error.response?.data?.message || 'Failed to restore platform')
  }
}

function getStatus(item) {
  return item.deleted_at ? 'deleted' : 'active'
}

const isFormValid = computed(() => {
  return form.value.code && form.value.name
})

function getIconPath(iconName) {
  return getSocialIconPath(iconName)
}

onMounted(() => {
  fetchPlatforms()
})
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Social Media Platforms</h1>
      <div class="flex items-center gap-4">
        <Input
          v-model="searchQuery"
          placeholder="Search..."
          class="w-48"
        />
        <select
          v-model="statusFilter"
          class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] transition-all duration-200"
        >
          <option value="active">Active</option>
          <option value="all">All</option>
          <option value="deleted">Deleted</option>
        </select>
        <Button @click="openCreateModal">Add Platform</Button>
      </div>
    </div>

    <div v-if="loading" class="text-center py-12 text-gray-500 dark:text-gray-400">Loading...</div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      <Card
        v-for="platform in platforms"
        :key="platform.id"
        :class="[
          'p-4 transition-all',
          getStatus(platform) === 'deleted'
            ? 'opacity-60 bg-gray-50 dark:bg-gray-800/50 border-dashed'
            : ''
        ]"
      >
        <div class="flex items-start gap-3 mb-3">
          <!-- Icon -->
          <div class="w-10 h-10 rounded-lg bg-gray-100 dark:bg-gray-700 flex items-center justify-center flex-shrink-0">
            <svg
              v-if="getIconPath(platform.icon)"
              class="w-5 h-5 text-gray-600 dark:text-gray-300"
              fill="currentColor"
              viewBox="0 0 24 24"
            >
              <path :d="getIconPath(platform.icon)" />
            </svg>
            <span v-else class="text-lg font-semibold text-gray-600 dark:text-gray-300">
              {{ platform.name?.charAt(0) || '?' }}
            </span>
          </div>

          <div class="flex-1 min-w-0">
            <h3
              :class="[
                'font-semibold',
                getStatus(platform) === 'deleted'
                  ? 'text-gray-500 dark:text-gray-400'
                  : 'text-gray-900 dark:text-white'
              ]"
            >
              {{ platform.name }}
            </h3>
            <p class="text-xs text-gray-500 dark:text-gray-400 font-mono">{{ platform.code }}</p>
          </div>

          <span
            :class="[
              'px-2 py-0.5 text-xs font-medium rounded-full flex-shrink-0',
              getStatus(platform) === 'active'
                ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
            ]"
          >
            {{ platform.is_active ? 'Active' : 'Inactive' }}
          </span>
        </div>

        <div class="text-xs text-gray-600 dark:text-gray-300 space-y-1 mb-3">
          <div v-if="platform.base_url" class="truncate" :title="platform.base_url">
            <span class="text-gray-500">URL:</span> {{ platform.base_url }}
          </div>
          <div v-if="platform.placeholder_text" class="truncate" :title="platform.placeholder_text">
            <span class="text-gray-500">User ID:</span> {{ platform.placeholder_text }}
          </div>
        </div>

        <div class="flex gap-2 pt-2 border-t border-gray-200 dark:border-gray-700">
          <Button variant="outline" size="sm" @click="openEditModal(platform)">Edit</Button>
          <Button
            v-if="!platform.deleted_at"
            variant="outline"
            size="sm"
            class="text-red-600 border-red-200 hover:bg-red-50 dark:text-red-400 dark:border-red-800 dark:hover:bg-red-900/20"
            @click="deletePlatform(platform)"
          >
            Delete
          </Button>
          <Button
            v-else
            variant="outline"
            size="sm"
            class="text-green-600 border-green-200 hover:bg-green-50 dark:text-green-400 dark:border-green-800 dark:hover:bg-green-900/20"
            @click="restorePlatform(platform)"
          >
            Restore
          </Button>
        </div>
      </Card>
    </div>

    <div v-if="!loading && platforms.length === 0" class="text-center py-12 text-gray-500 dark:text-gray-400">
      No social media platforms found.
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md max-h-[90vh] overflow-y-auto">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          {{ editingPlatform ? 'Edit Platform' : 'Create Platform' }}
        </h2>

        <div v-if="errorMessage" class="mb-4 p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-md text-sm">
          {{ errorMessage }}
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Code *</label>
            <Input v-model="form.code" placeholder="e.g. instagram, tiktok" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name *</label>
            <Input v-model="form.name" placeholder="e.g. Instagram, TikTok" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Icon</label>
            <Input v-model="form.icon" placeholder="e.g. instagram, facebook, tiktok" />
            <p class="text-xs text-gray-500 mt-1">Available: instagram, facebook, tiktok, whatsapp, youtube, twitter, linkedin</p>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Base URL</label>
            <Input v-model="form.base_url" placeholder="https://www.instagram.com/" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Deep Link Pattern</label>
            <Input v-model="form.deep_link_pattern" placeholder="instagram://user?username={handle}" />
            <p class="text-xs text-gray-500 mt-1">Use {handle} as placeholder</p>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">User ID</label>
            <Input v-model="form.placeholder_text" placeholder="@username" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Display Order</label>
            <Input v-model.number="form.display_order" type="number" min="0" />
          </div>
        </div>

        <div class="flex justify-end gap-2 mt-6">
          <Button variant="outline" @click="showModal = false">Cancel</Button>
          <Button @click="savePlatform" :disabled="!isFormValid">Save</Button>
        </div>
      </div>
    </div>
  </div>
</template>
