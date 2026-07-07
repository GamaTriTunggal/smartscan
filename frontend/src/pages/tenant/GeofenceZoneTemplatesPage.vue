<script setup>
import { ref, onMounted } from 'vue'
import { useAPI } from '@/composables/useAPI'
import { useDateTime } from '@/composables/useDateTime'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import GeofenceMapPicker from '@/components/GeofenceMapPicker.vue'
import { MapPin, Plus, Pencil, Trash2, RotateCcw } from 'lucide-vue-next'

const { get, post, put, del } = useAPI()
const { formatDateTime } = useDateTime()

const loading = ref(false)
const templates = ref([])
const page = ref(1)
const limit = ref(20)
const totalPages = ref(0)
const statusFilter = ref('active')

// Modal state
const showModal = ref(false)
const editingTemplate = ref(null)
const saving = ref(false)
const saveError = ref('')

const formData = ref({
  template_name: '',
  latitude: null,
  longitude: null,
  radius_km: 25,
  label: ''
})

const mapData = ref({
  latitude: null,
  longitude: null,
  radius_km: 25,
  label: ''
})

// Delete confirmation
const showDeleteModal = ref(false)
const deletingTemplate = ref(null)
const deleting = ref(false)

async function fetchTemplates() {
  loading.value = true
  try {
    const response = await get('/tenant/geofence/zone-templates', {
      page: page.value,
      limit: limit.value,
      status: statusFilter.value
    })
    if (response.success) {
      templates.value = response.data?.zone_templates || []
      totalPages.value = response.data?.pagination?.total_page || 0
      // Self-heal: if this page emptied out (e.g. last row deleted), snap back
      if (templates.value.length === 0 && page.value > 1) {
        page.value = Math.max(1, totalPages.value)
        return fetchTemplates()
      }
    }
  } catch (error) {
    console.error('Failed to fetch zone templates:', error)
  } finally {
    loading.value = false
  }
}

function onStatusChange() {
  page.value = 1
  fetchTemplates()
}

function openCreateModal() {
  editingTemplate.value = null
  formData.value = { template_name: '', latitude: null, longitude: null, radius_km: 25, label: '' }
  mapData.value = { latitude: null, longitude: null, radius_km: 25, label: '' }
  saveError.value = ''
  showModal.value = true
}

function openEditModal(template) {
  editingTemplate.value = template
  formData.value = {
    template_name: template.template_name,
    latitude: template.latitude,
    longitude: template.longitude,
    radius_km: template.radius_km,
    label: template.label || ''
  }
  mapData.value = {
    latitude: template.latitude,
    longitude: template.longitude,
    radius_km: template.radius_km,
    label: template.label || ''
  }
  saveError.value = ''
  showModal.value = true
}

function onMapUpdate(val) {
  mapData.value = val
  formData.value.latitude = val.latitude
  formData.value.longitude = val.longitude
  formData.value.radius_km = val.radius_km
  formData.value.label = val.label
}

async function saveTemplate() {
  if (!formData.value.template_name?.trim()) {
    saveError.value = 'Template name is required'
    return
  }
  if (!formData.value.latitude || !formData.value.longitude) {
    saveError.value = 'Please select a location on the map'
    return
  }

  saving.value = true
  saveError.value = ''

  try {
    const payload = {
      template_name: formData.value.template_name.trim(),
      latitude: formData.value.latitude,
      longitude: formData.value.longitude,
      radius_km: formData.value.radius_km,
      label: formData.value.label?.trim() || ''
    }

    let response
    if (editingTemplate.value) {
      response = await put(`/tenant/geofence/zone-templates/${editingTemplate.value.id}`, payload)
    } else {
      response = await post('/tenant/geofence/zone-templates', payload)
    }

    if (response.success) {
      showModal.value = false
      fetchTemplates()
    } else {
      saveError.value = response.message || 'Failed to save template'
    }
  } catch (error) {
    saveError.value = error.response?.data?.message || 'Failed to save template'
  } finally {
    saving.value = false
  }
}

function confirmDelete(template) {
  deletingTemplate.value = template
  showDeleteModal.value = true
}

async function deleteTemplate() {
  if (!deletingTemplate.value) return
  deleting.value = true
  try {
    const response = await del(`/tenant/geofence/zone-templates/${deletingTemplate.value.id}`)
    if (response.success) {
      showDeleteModal.value = false
      deletingTemplate.value = null
      fetchTemplates()
    }
  } catch (error) {
    console.error('Failed to delete template:', error)
  } finally {
    deleting.value = false
  }
}

async function restoreTemplate(template) {
  if (!confirm(`Are you sure you want to restore "${template.template_name}"?`)) return
  try {
    const response = await post(`/tenant/geofence/zone-templates/${template.id}/restore`)
    if (response.success) {
      fetchTemplates()
    } else {
      alert(response.message || 'Failed to restore template')
    }
  } catch (error) {
    alert(error.response?.data?.message || 'Failed to restore template')
  }
}

function goToPage(p) {
  page.value = p
  fetchTemplates()
}

onMounted(() => {
  fetchTemplates()
})
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Zone Templates</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Reusable distribution zone presets for batch creation</p>
      </div>
      <div class="flex items-center gap-3">
        <select
          v-model="statusFilter"
          @change="onStatusChange"
          class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm"
        >
          <option value="active">Active</option>
          <option value="all">All</option>
          <option value="deleted">Deleted</option>
        </select>
        <Button @click="openCreateModal">
          <Plus class="w-4 h-4 mr-2" />
          New Template
        </Button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <!-- Templates Grid -->
    <div v-else-if="templates.length > 0" class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <Card
        v-for="t in templates"
        :key="t.id"
        class="p-4"
        :class="t.deleted_at ? 'opacity-60 bg-gray-50 dark:bg-gray-800/50 border-dashed' : ''"
      >
        <div class="flex items-start justify-between mb-3">
          <div class="flex items-center gap-2">
            <MapPin class="w-5 h-5 text-zinc-500" />
            <h3 class="font-semibold text-gray-900 dark:text-white">{{ t.template_name }}</h3>
          </div>
          <div class="flex items-center gap-1">
            <!-- Status badge -->
            <span v-if="t.deleted_at" class="px-2 py-0.5 text-xs font-medium rounded-full bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 mr-1">
              Deleted
            </span>

            <!-- Edit (only for active) -->
            <button
              v-if="!t.deleted_at"
              @click="openEditModal(t)"
              class="p-1.5 rounded hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            >
              <Pencil class="w-4 h-4" />
            </button>

            <!-- Delete (active) or Restore (deleted) -->
            <button
              v-if="!t.deleted_at"
              @click="confirmDelete(t)"
              class="p-1.5 rounded hover:bg-red-50 dark:hover:bg-red-900/20 text-gray-400 hover:text-red-600 dark:hover:text-red-400"
            >
              <Trash2 class="w-4 h-4" />
            </button>
            <button
              v-else
              @click="restoreTemplate(t)"
              class="p-1.5 rounded hover:bg-green-50 dark:hover:bg-green-900/20 text-green-600 dark:text-green-400 hover:text-green-700 dark:hover:text-green-300"
              title="Restore template"
            >
              <RotateCcw class="w-4 h-4" />
            </button>
          </div>
        </div>

        <div class="space-y-2 text-sm">
          <div v-if="t.label" class="text-gray-700 dark:text-gray-300">
            {{ t.label }}
          </div>
          <div class="flex justify-between text-gray-500 dark:text-gray-400">
            <span>Radius</span>
            <span class="font-medium text-gray-900 dark:text-white">{{ t.radius_km }} km</span>
          </div>
          <div class="flex justify-between text-gray-500 dark:text-gray-400">
            <span>Coordinates</span>
            <span class="font-medium text-gray-900 dark:text-white">{{ t.latitude?.toFixed(4) }}, {{ t.longitude?.toFixed(4) }}</span>
          </div>
          <div class="flex justify-between text-gray-500 dark:text-gray-400">
            <span>Used in batches</span>
            <span class="font-medium text-gray-900 dark:text-white">{{ t.usage_count || 0 }}</span>
          </div>
        </div>
      </Card>
    </div>

    <!-- Empty State -->
    <div v-else class="text-center py-12">
      <MapPin class="w-16 h-16 text-gray-300 dark:text-gray-600 mx-auto mb-4" />
      <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">
        {{ statusFilter === 'deleted' ? 'No deleted zone templates' : 'No zone templates yet' }}
      </h3>
      <p class="text-gray-500 dark:text-gray-400 mb-4">
        {{ statusFilter === 'deleted' ? 'There are no deleted templates to show.' : 'Create reusable zone presets to quickly configure geofencing on new batches.' }}
      </p>
      <Button v-if="statusFilter !== 'deleted'" @click="openCreateModal">
        <Plus class="w-4 h-4 mr-2" />
        Create First Template
      </Button>
    </div>

    <!-- Pagination -->
    <div v-if="totalPages > 1" class="flex justify-center gap-2 mt-6">
      <Button variant="outline" size="sm" :disabled="page === 1" @click="goToPage(page - 1)">Previous</Button>
      <span class="flex items-center text-sm text-gray-600 dark:text-gray-400">Page {{ page }} of {{ totalPages }}</span>
      <Button variant="outline" size="sm" :disabled="page >= totalPages" @click="goToPage(page + 1)">Next</Button>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <div class="p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ editingTemplate ? 'Edit Zone Template' : 'Create Zone Template' }}
          </h2>
        </div>

        <div class="p-6 space-y-4">
          <!-- Template Name -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Template Name</label>
            <Input
              v-model="formData.template_name"
              placeholder="e.g. Semarang Distribution Zone"
              maxlength="255"
            />
          </div>

          <!-- Map Picker -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Location & Radius</label>
            <GeofenceMapPicker
              :modelValue="mapData"
              @update:modelValue="onMapUpdate"
              height="300px"
            />
          </div>

          <!-- Error -->
          <div v-if="saveError" class="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-700 dark:text-red-400">
            {{ saveError }}
          </div>
        </div>

        <div class="p-6 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
          <Button variant="outline" @click="showModal = false" :disabled="saving">Cancel</Button>
          <Button @click="saveTemplate" :disabled="saving">
            {{ saving ? 'Saving...' : (editingTemplate ? 'Update' : 'Create') }}
          </Button>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md">
        <div class="p-6">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">Delete Zone Template</h2>
          <p class="text-gray-600 dark:text-gray-400">
            Are you sure you want to delete "{{ deletingTemplate?.template_name }}"? This won't affect existing batches using this zone.
          </p>
        </div>
        <div class="p-6 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
          <Button variant="outline" @click="showDeleteModal = false; deletingTemplate = null" :disabled="deleting">Cancel</Button>
          <Button variant="destructive" @click="deleteTemplate" :disabled="deleting">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>
