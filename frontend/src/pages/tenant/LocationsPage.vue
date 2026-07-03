<script setup>
import { ref, onMounted } from 'vue'
import { useAPI } from '@/composables/useAPI'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import PhoneInput from '@/components/PhoneInput.vue'

const { get, post, put, del } = useAPI()

// State
const locations = ref([])
const loading = ref(false)
const showModal = ref(false)
const editingLocation = ref(null)
const typeFilter = ref('')
const statusFilter = ref('active')

const typeOptions = [
  { value: '', label: 'All Types' },
  { value: 'warehouse', label: 'Warehouse' },
  { value: 'qc_area', label: 'QC Area' },
  { value: 'production', label: 'Production' },
  { value: 'office', label: 'Office' }
]

const statusOptions = [
  { value: 'active', label: 'Active' },
  { value: 'all', label: 'All' },
  { value: 'deleted', label: 'Deleted' }
]

const typeLabels = {
  warehouse: 'Warehouse',
  qc_area: 'QC Area',
  production: 'Production',
  office: 'Office'
}

const form = ref({
  location_name: '',
  location_type: 'warehouse',
  address: '',
  city: '',
  province: '',
  postal_code: '',
  phone_number: '',
  latitude: null,
  longitude: null,
  allowed_radius: null,
  status: 'active'
})

async function fetchLocations() {
  loading.value = true
  try {
    const params = { status: statusFilter.value }
    if (typeFilter.value) params.type = typeFilter.value

    const response = await get('/tenant/locations', params)
    if (response.success) {
      locations.value = response.data || []
    }
  } catch (error) {
    console.error('Failed to fetch locations:', error)
  } finally {
    loading.value = false
  }
}

function openCreateModal() {
  editingLocation.value = null
  form.value = {
    location_name: '',
    location_type: 'warehouse',
    address: '',
    city: '',
    province: '',
    postal_code: '',
    phone_number: '',
    latitude: null,
    longitude: null,
    allowed_radius: null,
    status: 'active'
  }
  showModal.value = true
}

function openEditModal(location) {
  editingLocation.value = location
  // Parse geolocation if exists
  let lat = null, lng = null
  if (location.geolocation) {
    try {
      const geo = typeof location.geolocation === 'string'
        ? JSON.parse(location.geolocation)
        : location.geolocation
      lat = geo.lat
      lng = geo.lng
    } catch {
      // ignore parse errors
    }
  }

  form.value = {
    location_name: location.location_name,
    location_type: location.location_type,
    address: location.address || '',
    city: location.city || '',
    province: location.province || '',
    postal_code: location.postal_code || '',
    phone_number: location.phone_number || '',
    latitude: lat,
    longitude: lng,
    allowed_radius: location.allowed_radius,
    status: location.status || 'active'
  }
  showModal.value = true
}

async function saveLocation() {
  try {
    const data = { ...form.value }
    // Convert empty strings to null for numeric fields
    if (data.latitude === '') data.latitude = null
    if (data.longitude === '') data.longitude = null
    if (data.allowed_radius === '' || data.allowed_radius === null) {
      data.allowed_radius = null
    } else {
      data.allowed_radius = parseInt(data.allowed_radius)
    }
    if (data.latitude !== null) data.latitude = parseFloat(data.latitude)
    if (data.longitude !== null) data.longitude = parseFloat(data.longitude)

    if (editingLocation.value) {
      const response = await put(`/tenant/locations/${editingLocation.value.id}`, data)
      if (response.success) {
        showModal.value = false
        fetchLocations()
      }
    } else {
      const response = await post('/tenant/locations', data)
      if (response.success) {
        showModal.value = false
        fetchLocations()
      }
    }
  } catch (error) {
    console.error('Failed to save location:', error)
    alert(error.response?.data?.message || 'Failed to save location')
  }
}

async function deleteLocation(location) {
  if (!confirm(`Are you sure you want to delete "${location.location_name}"?`)) {
    return
  }
  try {
    const response = await del(`/tenant/locations/${location.id}`)
    if (response.success) {
      fetchLocations()
    }
  } catch (error) {
    console.error('Failed to delete location:', error)
    alert(error.response?.data?.message || 'Failed to delete location')
  }
}

async function restoreLocation(location) {
  try {
    const response = await post(`/tenant/locations/${location.id}/restore`)
    if (response.success) {
      fetchLocations()
    }
  } catch (error) {
    console.error('Failed to restore location:', error)
    alert(error.response?.data?.message || 'Failed to restore location')
  }
}

function handleTypeChange() {
  fetchLocations()
}

function handleStatusChange() {
  fetchLocations()
}

function getTypeClass(type, isDeleted = false) {
  if (isDeleted) {
    return 'bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
  }
  const classes = {
    warehouse: 'bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400',
    qc_area: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
    production: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
    office: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
  }
  return classes[type] || 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
}

function formatGeolocation(location) {
  if (!location.geolocation) return '-'
  try {
    const geo = typeof location.geolocation === 'string'
      ? JSON.parse(location.geolocation)
      : location.geolocation
    return `${geo.lat.toFixed(4)}, ${geo.lng.toFixed(4)}`
  } catch {
    return '-'
  }
}

onMounted(() => {
  fetchLocations()
})
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Locations</h1>
      <Button @click="openCreateModal">Add Location</Button>
    </div>

    <!-- Filters -->
    <Card class="p-4 mb-6">
      <div class="flex flex-wrap gap-4 items-end">
        <div class="w-48">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Type</label>
          <select
            v-model="typeFilter"
            @change="handleTypeChange"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
          >
            <option v-for="opt in typeOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>
        <div class="w-32">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Status</label>
          <select
            v-model="statusFilter"
            @change="handleStatusChange"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
          >
            <option v-for="opt in statusOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>
      </div>
    </Card>

    <!-- Locations Table -->
    <Card class="overflow-hidden">
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-800">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Name
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Type
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Address
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Geolocation
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Radius (m)
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
            <tr v-if="loading">
              <td colspan="6" class="px-6 py-12 text-center text-gray-500 dark:text-gray-400">
                Loading...
              </td>
            </tr>
            <tr v-else-if="locations.length === 0">
              <td colspan="6" class="px-6 py-12 text-center text-gray-500 dark:text-gray-400">
                No locations found
              </td>
            </tr>
            <tr
              v-else
              v-for="location in locations"
              :key="location.id"
              :class="[
                location.deleted_at
                  ? 'opacity-60 bg-gray-50 dark:bg-gray-800/50'
                  : 'hover:bg-gray-50 dark:hover:bg-gray-800'
              ]"
            >
              <td class="px-6 py-4 whitespace-nowrap">
                <div :class="[
                  'font-medium',
                  location.deleted_at ? 'text-gray-500 dark:text-gray-400' : 'text-gray-900 dark:text-white'
                ]">
                  {{ location.location_name }}
                </div>
                <div v-if="location.phone_number" :class="[
                  'text-sm',
                  location.deleted_at ? 'text-gray-400 dark:text-gray-500' : 'text-gray-500 dark:text-gray-400'
                ]">
                  {{ location.phone_number }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span
                  :class="[
                    'px-2 py-1 text-xs font-medium rounded-full',
                    getTypeClass(location.location_type, !!location.deleted_at)
                  ]"
                >
                  {{ typeLabels[location.location_type] || location.location_type }}
                </span>
              </td>
              <td :class="[
                'px-6 py-4 text-sm max-w-xs truncate',
                location.deleted_at ? 'text-gray-400 dark:text-gray-500' : 'text-gray-500 dark:text-gray-400'
              ]">
                <div>{{ location.address || '-' }}</div>
                <div v-if="location.city || location.province" class="text-xs">
                  {{ [location.city, location.province].filter(Boolean).join(', ') }}
                </div>
              </td>
              <td :class="[
                'px-6 py-4 whitespace-nowrap text-sm',
                location.deleted_at ? 'text-gray-400 dark:text-gray-500' : 'text-gray-500 dark:text-gray-400'
              ]">
                {{ formatGeolocation(location) }}
              </td>
              <td :class="[
                'px-6 py-4 whitespace-nowrap text-sm',
                location.deleted_at ? 'text-gray-400 dark:text-gray-500' : 'text-gray-500 dark:text-gray-400'
              ]">
                {{ location.allowed_radius || '-' }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm">
                <div class="flex gap-2">
                  <template v-if="!location.deleted_at">
                    <Button variant="outline" size="sm" @click="openEditModal(location)">
                      Edit
                    </Button>
                    <button
                      @click="deleteLocation(location)"
                      class="text-xs text-red-600 dark:text-red-400 hover:underline"
                    >
                      Delete
                    </button>
                  </template>
                  <button
                    v-else
                    @click="restoreLocation(location)"
                    class="text-xs text-zinc-600 dark:text-zinc-400 hover:underline"
                  >
                    Restore
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Summary -->
      <div class="px-6 py-4 border-t border-gray-200 dark:border-gray-700">
        <div class="text-sm text-gray-500 dark:text-gray-400">
          Total: {{ locations.length }} locations
        </div>
      </div>
    </Card>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          {{ editingLocation ? 'Edit Location' : 'Add Location' }}
        </h2>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Location Name *</label>
            <Input v-model="form.location_name" placeholder="e.g., Gudang Pusat" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Type *</label>
            <select
              v-model="form.location_type"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
            >
              <option value="warehouse">Warehouse</option>
              <option value="qc_area">QC Area</option>
              <option value="production">Production</option>
              <option value="office">Office</option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Address</label>
            <Input v-model="form.address" placeholder="Full address" />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">City</label>
              <Input v-model="form.city" placeholder="City" />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Province</label>
              <Input v-model="form.province" placeholder="Province" />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Postal Code</label>
              <Input v-model="form.postal_code" placeholder="12345" />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Phone Number</label>
              <PhoneInput v-model="form.phone_number" placeholder="Location phone" />
            </div>
          </div>

          <div class="border-t border-gray-200 dark:border-gray-700 pt-4 mt-4">
            <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">Geolocation Settings</h3>
            <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
              Set coordinates and allowed radius for scan location validation
            </p>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Latitude</label>
                <Input v-model="form.latitude" type="number" step="any" placeholder="-6.2088" />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Longitude</label>
                <Input v-model="form.longitude" type="number" step="any" placeholder="106.8456" />
              </div>
            </div>
            <div class="mt-4">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Allowed Radius (meters)</label>
              <Input v-model="form.allowed_radius" type="number" placeholder="500 (leave empty for no limit)" />
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                Scans outside this radius will trigger a warning
              </p>
            </div>
          </div>

          <div v-if="editingLocation">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Status</label>
            <select
              v-model="form.status"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-[#27272a]"
            >
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </select>
          </div>
        </div>

        <div class="flex justify-end gap-2 mt-6">
          <Button variant="outline" @click="showModal = false">Cancel</Button>
          <Button @click="saveLocation">Save</Button>
        </div>
      </div>
    </div>
  </div>
</template>
