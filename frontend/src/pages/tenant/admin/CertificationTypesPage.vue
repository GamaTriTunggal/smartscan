<script setup>
import { ref, onMounted, computed } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useListFilter } from '@/composables/useListFilter'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'

const { get, post, put, del } = useAPI()

// State
const certTypes = ref([])
const countries = ref([])
const loading = ref(false)
const showModal = ref(false)
const editingType = ref(null)
const statusFilter = ref('active')
const countryFilter = ref('')
const { search: searchQuery, pagination, watchFilter, prevPage, nextPage } = useListFilter(fetchCertTypes, { limit: 50 })
watchFilter(statusFilter, countryFilter)

const form = ref({
  code: '',
  name: '',
  description: '',
  country_code: null,
  logo_url: '',
  website_url: '',
  display_order: 0
})

const errorMessage = ref('')

async function fetchCertTypes() {
  loading.value = true
  try {
    let url = `/tenant/certifications/types/all?status=${statusFilter.value}&page=${pagination.value.page}&limit=${pagination.value.limit}`
    if (countryFilter.value) {
      url += `&country_code=${countryFilter.value}`
    }
    if (searchQuery.value) {
      url += `&search=${encodeURIComponent(searchQuery.value)}`
    }
    const response = await get(url)
    if (response.success) {
      certTypes.value = response.data?.certification_types || []
      pagination.value.total = response.data?.pagination?.total || 0
      pagination.value.total_page = response.data?.pagination?.total_page || 0
      // Self-heal: if this page emptied out (e.g. last row deleted), snap back
      if (certTypes.value.length === 0 && pagination.value.page > 1) {
        pagination.value.page = Math.max(1, pagination.value.total_page)
        return fetchCertTypes()
      }
    }
  } catch (error) {
    console.error('Failed to fetch certification types:', error)
  } finally {
    loading.value = false
  }
}

async function fetchCountries() {
  try {
    const response = await get('/locations/countries')
    if (response.success) {
      // Filter to only SEA countries
      const seaCountryCodes = ['ID', 'MY', 'PH', 'SG', 'TH', 'VN']
      countries.value = (response.data || []).filter(c => seaCountryCodes.includes(c.code))
    }
  } catch (error) {
    console.error('Failed to fetch countries:', error)
  }
}

function openCreateModal() {
  editingType.value = null
  form.value = {
    code: '',
    name: '',
    description: '',
    country_code: null,
    logo_url: '',
    website_url: '',
    display_order: 0
  }
  errorMessage.value = ''
  showModal.value = true
}

function openEditModal(type) {
  editingType.value = type
  form.value = {
    code: type.code,
    name: type.name,
    description: type.description || '',
    country_code: type.country_code,
    logo_url: type.logo_url || '',
    website_url: type.website_url || '',
    display_order: type.display_order || 0
  }
  errorMessage.value = ''
  showModal.value = true
}

async function saveType() {
  errorMessage.value = ''
  try {
    const payload = { ...form.value }
    // Convert empty string to null for country_code
    if (payload.country_code === '') {
      payload.country_code = null
    }

    if (editingType.value) {
      const response = await put(`/tenant/certifications/types/${editingType.value.id}`, payload)
      if (response.success) {
        showModal.value = false
        fetchCertTypes()
      } else {
        errorMessage.value = response.message || 'Failed to save certification type'
      }
    } else {
      const response = await post('/tenant/certifications/types', payload)
      if (response.success) {
        showModal.value = false
        fetchCertTypes()
      } else {
        errorMessage.value = response.message || 'Failed to save certification type'
      }
    }
  } catch (error) {
    console.error('Failed to save certification type:', error)
    errorMessage.value = error.response?.data?.message || 'Failed to save certification type'
  }
}

async function deleteType(type) {
  if (!confirm(`Are you sure you want to delete "${type.name}"?`)) return

  try {
    const response = await del(`/tenant/certifications/types/${type.id}`)
    if (response.success) {
      fetchCertTypes()
    } else {
      alert(response.message || 'Failed to delete certification type')
    }
  } catch (error) {
    console.error('Failed to delete certification type:', error)
    alert(error.response?.data?.message || 'Failed to delete certification type')
  }
}

async function restoreType(type) {
  if (!confirm(`Are you sure you want to restore "${type.name}"?`)) return

  try {
    const response = await post(`/tenant/certifications/types/${type.id}/restore`)
    if (response.success) {
      fetchCertTypes()
    } else {
      alert(response.message || 'Failed to restore certification type')
    }
  } catch (error) {
    console.error('Failed to restore certification type:', error)
    alert(error.response?.data?.message || 'Failed to restore certification type')
  }
}

function getStatus(item) {
  return item.deleted_at ? 'deleted' : 'active'
}

function getCountryName(code) {
  if (!code) return 'International'
  const country = countries.value.find(c => c.code === code)
  return country ? country.name : code
}

const isFormValid = computed(() => {
  return form.value.code && form.value.name
})

// Group by country for display
const groupedCertTypes = computed(() => {
  const groups = {}

  // International first
  groups['International'] = certTypes.value.filter(t => !t.country_code)

  // Then by country
  countries.value.forEach(country => {
    const items = certTypes.value.filter(t => t.country_code === country.code)
    if (items.length > 0) {
      groups[country.name] = items
    }
  })

  return groups
})

onMounted(() => {
  fetchCountries()
  fetchCertTypes()
})
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Certification Types</h1>
      <div class="flex items-center gap-4">
        <Input
          v-model="searchQuery"
          placeholder="Search..."
          class="w-48"
        />
        <select
          v-model="countryFilter"
          class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] transition-all duration-200"
        >
          <option value="">All Countries</option>
          <option value="international">International</option>
          <option v-for="country in countries" :key="country.code" :value="country.code">
            {{ country.name }}
          </option>
        </select>
        <select
          v-model="statusFilter"
          class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] transition-all duration-200"
        >
          <option value="active">Active</option>
          <option value="all">All</option>
          <option value="deleted">Deleted</option>
        </select>
        <Button @click="openCreateModal">Add Certification</Button>
      </div>
    </div>

    <div v-if="loading" class="text-center py-12 text-gray-500 dark:text-gray-400">Loading...</div>

    <div v-else>
      <!-- Grouped by Country -->
      <div v-for="(items, group) in groupedCertTypes" :key="group" class="mb-8">
        <h2 v-if="items.length > 0" class="text-lg font-semibold text-gray-800 dark:text-gray-200 mb-4 flex items-center gap-2">
          <span class="w-2 h-2 rounded-full" :class="group === 'International' ? 'bg-purple-500' : 'bg-zinc-500'"></span>
          {{ group }}
          <span class="text-sm font-normal text-gray-500">({{ items.length }})</span>
        </h2>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          <Card
            v-for="cert in items"
            :key="cert.id"
            :class="[
              'p-4 transition-all',
              getStatus(cert) === 'deleted'
                ? 'opacity-60 bg-gray-50 dark:bg-gray-800/50 border-dashed'
                : ''
            ]"
          >
            <div class="flex justify-between items-start mb-2">
              <div class="flex-1">
                <h3
                  :class="[
                    'text-sm font-semibold',
                    getStatus(cert) === 'deleted'
                      ? 'text-gray-500 dark:text-gray-400'
                      : 'text-gray-900 dark:text-white'
                  ]"
                >
                  {{ cert.name }}
                </h3>
                <p class="text-xs text-gray-500 dark:text-gray-400 font-mono">{{ cert.code }}</p>
              </div>
              <span
                :class="[
                  'px-2 py-0.5 text-xs font-medium rounded-full',
                  getStatus(cert) === 'active'
                    ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                    : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
                ]"
              >
                {{ cert.is_active ? 'Active' : 'Inactive' }}
              </span>
            </div>

            <p v-if="cert.description" class="text-xs text-gray-600 dark:text-gray-300 mb-2 line-clamp-2">
              {{ cert.description }}
            </p>

            <div v-if="cert.website_url" class="mb-2">
              <a :href="cert.website_url" target="_blank" class="text-xs text-zinc-600 dark:text-zinc-400 hover:underline">
                Visit Website
              </a>
            </div>

            <div class="flex gap-2 pt-2 border-t border-gray-200 dark:border-gray-700">
              <Button variant="outline" size="sm" @click="openEditModal(cert)">Edit</Button>
              <Button
                v-if="!cert.deleted_at"
                variant="outline"
                size="sm"
                class="text-red-600 border-red-200 hover:bg-red-50 dark:text-red-400 dark:border-red-800 dark:hover:bg-red-900/20"
                @click="deleteType(cert)"
              >
                Delete
              </Button>
              <Button
                v-else
                variant="outline"
                size="sm"
                class="text-green-600 border-green-200 hover:bg-green-50 dark:text-green-400 dark:border-green-800 dark:hover:bg-green-900/20"
                @click="restoreType(cert)"
              >
                Restore
              </Button>
            </div>
          </Card>
        </div>
      </div>
    </div>

    <div v-if="!loading && certTypes.length === 0" class="text-center py-12 text-gray-500 dark:text-gray-400">
      No certification types found.
    </div>

    <!-- Pagination -->
    <div v-if="pagination.total_page > 1" class="flex justify-center gap-2 mt-6">
      <Button
        variant="outline"
        size="sm"
        :disabled="pagination.page === 1"
        @click="prevPage"
      >
        Previous
      </Button>
      <span class="flex items-center text-sm text-gray-600 dark:text-gray-400">
        Page {{ pagination.page }} of {{ pagination.total_page }}
      </span>
      <Button
        variant="outline"
        size="sm"
        :disabled="pagination.page >= pagination.total_page"
        @click="nextPage"
      >
        Next
      </Button>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md max-h-[90vh] overflow-y-auto">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          {{ editingType ? 'Edit Certification Type' : 'Create Certification Type' }}
        </h2>

        <div v-if="errorMessage" class="mb-4 p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-md text-sm">
          {{ errorMessage }}
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Code *</label>
            <Input v-model="form.code" placeholder="e.g. BPOM, HALAL_MUI" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name *</label>
            <Input v-model="form.name" placeholder="e.g. BPOM (Food & Drug)" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Country</label>
            <select
              v-model="form.country_code"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] transition-all duration-200"
            >
              <option :value="null">International (No Country)</option>
              <option v-for="country in countries" :key="country.code" :value="country.code">
                {{ country.name }}
              </option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
            <textarea
              v-model="form.description"
              rows="2"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a] transition-all duration-200"
              placeholder="Brief description..."
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Logo URL</label>
            <Input v-model="form.logo_url" placeholder="https://..." />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Website URL</label>
            <Input v-model="form.website_url" placeholder="https://..." />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Display Order</label>
            <Input v-model.number="form.display_order" type="number" min="0" />
          </div>
        </div>

        <div class="flex justify-end gap-2 mt-6">
          <Button variant="outline" @click="showModal = false">Cancel</Button>
          <Button @click="saveType" :disabled="!isFormValid">Save</Button>
        </div>
      </div>
    </div>
  </div>
</template>
