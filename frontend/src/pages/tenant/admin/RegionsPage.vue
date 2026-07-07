<script setup>
import { ref, onMounted, computed } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import { useAPI } from '@/composables/useAPI'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'

const { get, post, put, del } = useAPI()

// Tabs
const activeTab = ref('countries')

// Countries state
const countries = ref([])
const countriesLoading = ref(false)
const countriesStatusFilter = ref('active')
const countriesSearch = ref('')
const showCountryModal = ref(false)
const editingCountry = ref(null)
const countryForm = ref({ code: '', name: '', phone_code: '' })
const countryError = ref('')

// Provinces state
const provinces = ref([])
const provincesLoading = ref(false)
const provincesStatusFilter = ref('active')
const provincesSearch = ref('')
const provincesCountryFilter = ref('')
const showProvinceModal = ref(false)
const editingProvince = ref(null)
const provinceForm = ref({ country_code: '', name: '', code: '' })
const provinceError = ref('')

// Cities state
const cities = ref([])
const citiesLoading = ref(false)
const citiesStatusFilter = ref('active')
const citiesSearch = ref('')
const citiesProvinceFilter = ref('')
const showCityModal = ref(false)
const editingCity = ref(null)
const cityForm = ref({ province_id: '', name: '', postal_code_prefix: '' })
const cityError = ref('')

// Get active countries for dropdowns
const activeCountries = computed(() => {
  return countries.value.filter(c => !c.deleted_at)
})

// All provinces for city dropdown (fetched separately without filters/limits)
const allProvincesForDropdown = ref([])

// Get active provinces for dropdowns (filtered by country if city form has province)
const activeProvinces = computed(() => {
  return provinces.value.filter(p => !p.deleted_at)
})

// Provinces for city modal dropdown - use allProvincesForDropdown if available
const provincesForCityDropdown = computed(() => {
  return allProvincesForDropdown.value.filter(p => !p.deleted_at)
})

// ==================== COUNTRIES ====================
async function fetchCountries() {
  countriesLoading.value = true
  try {
    let url = `/tenant/location-master/countries?status=${countriesStatusFilter.value}&limit=100`
    if (countriesSearch.value) url += `&search=${countriesSearch.value}`
    const response = await get(url)
    if (response.success) {
      countries.value = response.data?.countries || []
    }
  } catch (error) {
    console.error('Failed to fetch countries:', error)
  } finally {
    countriesLoading.value = false
  }
}

function openCreateCountryModal() {
  editingCountry.value = null
  countryForm.value = { code: '', name: '', phone_code: '' }
  countryError.value = ''
  showCountryModal.value = true
}

function openEditCountryModal(country) {
  editingCountry.value = country
  countryForm.value = {
    code: country.code,
    name: country.name,
    phone_code: country.phone_code || ''
  }
  countryError.value = ''
  showCountryModal.value = true
}

async function saveCountry() {
  countryError.value = ''
  try {
    if (editingCountry.value) {
      const response = await put(`/tenant/location-master/countries/${editingCountry.value.code}`, {
        name: countryForm.value.name,
        phone_code: countryForm.value.phone_code
      })
      if (response.success) {
        showCountryModal.value = false
        fetchCountries()
      } else {
        countryError.value = response.message || 'Failed to save country'
      }
    } else {
      const response = await post('/tenant/location-master/countries', countryForm.value)
      if (response.success) {
        showCountryModal.value = false
        fetchCountries()
      } else {
        countryError.value = response.message || 'Failed to save country'
      }
    }
  } catch (error) {
    countryError.value = error.response?.data?.message || 'Failed to save country'
  }
}

async function deleteCountry(country) {
  if (!confirm(`Are you sure you want to delete "${country.name}"?`)) return
  try {
    const response = await del(`/tenant/location-master/countries/${country.code}`)
    if (response.success) {
      fetchCountries()
    } else {
      alert(response.message || 'Failed to delete country')
    }
  } catch (error) {
    alert(error.response?.data?.message || 'Failed to delete country')
  }
}

async function restoreCountry(country) {
  if (!confirm(`Are you sure you want to restore "${country.name}"?`)) return
  try {
    const response = await post(`/tenant/location-master/countries/${country.code}/restore`)
    if (response.success) {
      fetchCountries()
    } else {
      alert(response.message || 'Failed to restore country')
    }
  } catch (error) {
    alert(error.response?.data?.message || 'Failed to restore country')
  }
}

const isCountryFormValid = computed(() => {
  return countryForm.value.code && countryForm.value.code.length === 2 && countryForm.value.name
})

// ==================== PROVINCES ====================

// Fetch all active provinces for dropdowns (without filters/limits)
async function fetchAllProvincesForDropdown() {
  try {
    const response = await get('/tenant/location-master/provinces?status=active&limit=1000')
    if (response.success) {
      allProvincesForDropdown.value = response.data?.provinces || []
    }
  } catch (error) {
    console.error('Failed to fetch provinces for dropdown:', error)
  }
}

async function fetchProvinces() {
  provincesLoading.value = true
  try {
    let url = `/tenant/location-master/provinces?status=${provincesStatusFilter.value}&limit=100`
    if (provincesSearch.value) url += `&search=${provincesSearch.value}`
    if (provincesCountryFilter.value) url += `&country_code=${provincesCountryFilter.value}`
    const response = await get(url)
    if (response.success) {
      provinces.value = response.data?.provinces || []
    }
  } catch (error) {
    console.error('Failed to fetch provinces:', error)
  } finally {
    provincesLoading.value = false
  }
}

function openCreateProvinceModal() {
  editingProvince.value = null
  provinceForm.value = { country_code: '', name: '', code: '' }
  provinceError.value = ''
  showProvinceModal.value = true
}

function openEditProvinceModal(province) {
  editingProvince.value = province
  provinceForm.value = {
    country_code: province.country_code,
    name: province.name,
    code: province.code || ''
  }
  provinceError.value = ''
  showProvinceModal.value = true
}

async function saveProvince() {
  provinceError.value = ''
  try {
    if (editingProvince.value) {
      const response = await put(`/tenant/location-master/provinces/${editingProvince.value.id}`, provinceForm.value)
      if (response.success) {
        showProvinceModal.value = false
        fetchProvinces()
      } else {
        provinceError.value = response.message || 'Failed to save province'
      }
    } else {
      const response = await post('/tenant/location-master/provinces', provinceForm.value)
      if (response.success) {
        showProvinceModal.value = false
        fetchProvinces()
      } else {
        provinceError.value = response.message || 'Failed to save province'
      }
    }
  } catch (error) {
    provinceError.value = error.response?.data?.message || 'Failed to save province'
  }
}

async function deleteProvince(province) {
  if (!confirm(`Are you sure you want to delete "${province.name}"?`)) return
  try {
    const response = await del(`/tenant/location-master/provinces/${province.id}`)
    if (response.success) {
      fetchProvinces()
    } else {
      alert(response.message || 'Failed to delete province')
    }
  } catch (error) {
    alert(error.response?.data?.message || 'Failed to delete province')
  }
}

async function restoreProvince(province) {
  if (!confirm(`Are you sure you want to restore "${province.name}"?`)) return
  try {
    const response = await post(`/tenant/location-master/provinces/${province.id}/restore`)
    if (response.success) {
      fetchProvinces()
    } else {
      alert(response.message || 'Failed to restore province')
    }
  } catch (error) {
    alert(error.response?.data?.message || 'Failed to restore province')
  }
}

const isProvinceFormValid = computed(() => {
  return provinceForm.value.country_code && provinceForm.value.name
})

// ==================== CITIES ====================
async function fetchCities() {
  citiesLoading.value = true
  try {
    let url = `/tenant/location-master/cities?status=${citiesStatusFilter.value}&limit=100`
    if (citiesSearch.value) url += `&search=${encodeURIComponent(citiesSearch.value)}`
    if (citiesProvinceFilter.value) url += `&province_id=${citiesProvinceFilter.value}`
    // The backend caps limit at 100, so walk every page — the seed data alone
    // has 800+ cities and this table has no pager of its own.
    const all = []
    let pageNum = 1
    let totalPage = 1
    do {
      const response = await get(`${url}&page=${pageNum}`)
      if (!response.success) break
      all.push(...(response.data?.cities || []))
      totalPage = response.data?.pagination?.total_page || 1
      pageNum++
    } while (pageNum <= totalPage)
    cities.value = all
  } catch (error) {
    console.error('Failed to fetch cities:', error)
  } finally {
    citiesLoading.value = false
  }
}

function openCreateCityModal() {
  editingCity.value = null
  cityForm.value = { province_id: '', name: '', postal_code_prefix: '' }
  cityError.value = ''
  showCityModal.value = true
}

function openEditCityModal(city) {
  editingCity.value = city
  cityForm.value = {
    province_id: String(city.province_id), // Convert to string for select consistency
    name: city.name,
    postal_code_prefix: city.postal_code_prefix || ''
  }
  cityError.value = ''
  showCityModal.value = true
}

async function saveCity() {
  cityError.value = ''
  try {
    const payload = {
      province_id: parseInt(cityForm.value.province_id),
      name: cityForm.value.name,
      postal_code_prefix: cityForm.value.postal_code_prefix
    }
    if (editingCity.value) {
      const response = await put(`/tenant/location-master/cities/${editingCity.value.id}`, payload)
      if (response.success) {
        showCityModal.value = false
        fetchCities()
      } else {
        cityError.value = response.message || 'Failed to save city'
      }
    } else {
      const response = await post('/tenant/location-master/cities', payload)
      if (response.success) {
        showCityModal.value = false
        fetchCities()
      } else {
        cityError.value = response.message || 'Failed to save city'
      }
    }
  } catch (error) {
    cityError.value = error.response?.data?.message || 'Failed to save city'
  }
}

async function deleteCity(city) {
  if (!confirm(`Are you sure you want to delete "${city.name}"?`)) return
  try {
    const response = await del(`/tenant/location-master/cities/${city.id}`)
    if (response.success) {
      fetchCities()
    } else {
      alert(response.message || 'Failed to delete city')
    }
  } catch (error) {
    alert(error.response?.data?.message || 'Failed to delete city')
  }
}

async function restoreCity(city) {
  if (!confirm(`Are you sure you want to restore "${city.name}"?`)) return
  try {
    const response = await post(`/tenant/location-master/cities/${city.id}/restore`)
    if (response.success) {
      fetchCities()
    } else {
      alert(response.message || 'Failed to restore city')
    }
  } catch (error) {
    alert(error.response?.data?.message || 'Failed to restore city')
  }
}

const isCityFormValid = computed(() => {
  return cityForm.value.province_id && cityForm.value.name
})

// ==================== TAB CHANGE ====================
function switchTab(tab) {
  activeTab.value = tab
  if (tab === 'countries') fetchCountries()
  else if (tab === 'provinces') fetchProvinces()
  else if (tab === 'cities') fetchCities()
}

const debouncedFetchCountries = useDebounceFn(fetchCountries, 300)
const debouncedFetchProvinces = useDebounceFn(fetchProvinces, 300)
const debouncedFetchCities = useDebounceFn(fetchCities, 300)

onMounted(() => {
  fetchCountries()
  // Also fetch provinces for city dropdown
  fetchProvinces()
  // Fetch all provinces for city modal dropdown (without filters/limits)
  fetchAllProvincesForDropdown()
})
</script>

<template>
  <div>
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">Regions</h1>

    <!-- Tabs -->
    <div class="border-b border-gray-200 dark:border-gray-700 mb-6">
      <nav class="-mb-px flex space-x-8">
        <button
          @click="switchTab('countries')"
          :class="[
            'py-4 px-1 border-b-2 font-medium text-sm',
            activeTab === 'countries'
              ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
          ]"
        >
          Countries
        </button>
        <button
          @click="switchTab('provinces')"
          :class="[
            'py-4 px-1 border-b-2 font-medium text-sm',
            activeTab === 'provinces'
              ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
          ]"
        >
          Provinces
        </button>
        <button
          @click="switchTab('cities')"
          :class="[
            'py-4 px-1 border-b-2 font-medium text-sm',
            activeTab === 'cities'
              ? 'border-zinc-500 text-zinc-600 dark:text-zinc-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
          ]"
        >
          Cities
        </button>
      </nav>
    </div>

    <!-- Countries Tab -->
    <div v-if="activeTab === 'countries'">
      <div class="flex justify-between items-center mb-4">
        <div class="flex items-center gap-4">
          <Input
            v-model="countriesSearch"
            placeholder="Search countries..."
            class="w-48"
            @input="debouncedFetchCountries"
          />
          <select
            v-model="countriesStatusFilter"
            @change="fetchCountries"
            class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
          >
            <option value="active">Active</option>
            <option value="all">All</option>
            <option value="deleted">Deleted</option>
          </select>
        </div>
        <Button @click="openCreateCountryModal">Add Country</Button>
      </div>

      <div v-if="countriesLoading" class="text-center py-8 text-gray-500">Loading...</div>
      <div v-else-if="countries.length === 0" class="text-center py-8 text-gray-500">No countries found.</div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-800">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Code</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Name</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Phone Code</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Status</th>
              <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
            <tr
              v-for="country in countries"
              :key="country.code"
              :class="country.deleted_at ? 'opacity-60 bg-gray-50 dark:bg-gray-800/50' : ''"
            >
              <td class="px-4 py-3 text-sm font-mono text-gray-900 dark:text-white">{{ country.code }}</td>
              <td class="px-4 py-3 text-sm text-gray-900 dark:text-white">{{ country.name }}</td>
              <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{{ country.phone_code || '-' }}</td>
              <td class="px-4 py-3">
                <span
                  :class="[
                    'px-2 py-1 text-xs font-medium rounded-full',
                    country.deleted_at
                      ? 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
                      : 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                  ]"
                >
                  {{ country.deleted_at ? 'Deleted' : 'Active' }}
                </span>
              </td>
              <td class="px-4 py-3 text-right">
                <Button variant="outline" size="sm" class="mr-2" @click="openEditCountryModal(country)">Edit</Button>
                <Button
                  v-if="!country.deleted_at"
                  variant="outline"
                  size="sm"
                  class="text-red-600 border-red-200 hover:bg-red-50 dark:text-red-400"
                  @click="deleteCountry(country)"
                >
                  Delete
                </Button>
                <Button
                  v-else
                  variant="outline"
                  size="sm"
                  class="text-green-600 border-green-200 hover:bg-green-50 dark:text-green-400"
                  @click="restoreCountry(country)"
                >
                  Restore
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Provinces Tab -->
    <div v-if="activeTab === 'provinces'">
      <div class="flex justify-between items-center mb-4">
        <div class="flex items-center gap-4">
          <Input
            v-model="provincesSearch"
            placeholder="Search provinces..."
            class="w-48"
            @input="debouncedFetchProvinces"
          />
          <select
            v-model="provincesCountryFilter"
            @change="fetchProvinces"
            class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
          >
            <option value="">All Countries</option>
            <option v-for="c in activeCountries" :key="c.code" :value="c.code">{{ c.name }}</option>
          </select>
          <select
            v-model="provincesStatusFilter"
            @change="fetchProvinces"
            class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
          >
            <option value="active">Active</option>
            <option value="all">All</option>
            <option value="deleted">Deleted</option>
          </select>
        </div>
        <Button @click="openCreateProvinceModal">Add Province</Button>
      </div>

      <div v-if="provincesLoading" class="text-center py-8 text-gray-500">Loading...</div>
      <div v-else-if="provinces.length === 0" class="text-center py-8 text-gray-500">No provinces found.</div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-800">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">ID</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Country</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Name</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Code</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Status</th>
              <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
            <tr
              v-for="province in provinces"
              :key="province.id"
              :class="province.deleted_at ? 'opacity-60 bg-gray-50 dark:bg-gray-800/50' : ''"
            >
              <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{{ province.id }}</td>
              <td class="px-4 py-3 text-sm text-gray-900 dark:text-white">{{ province.country?.name || province.country_code }}</td>
              <td class="px-4 py-3 text-sm text-gray-900 dark:text-white">{{ province.name }}</td>
              <td class="px-4 py-3 text-sm font-mono text-gray-500 dark:text-gray-400">{{ province.code || '-' }}</td>
              <td class="px-4 py-3">
                <span
                  :class="[
                    'px-2 py-1 text-xs font-medium rounded-full',
                    province.deleted_at
                      ? 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
                      : 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                  ]"
                >
                  {{ province.deleted_at ? 'Deleted' : 'Active' }}
                </span>
              </td>
              <td class="px-4 py-3 text-right">
                <Button variant="outline" size="sm" class="mr-2" @click="openEditProvinceModal(province)">Edit</Button>
                <Button
                  v-if="!province.deleted_at"
                  variant="outline"
                  size="sm"
                  class="text-red-600 border-red-200 hover:bg-red-50 dark:text-red-400"
                  @click="deleteProvince(province)"
                >
                  Delete
                </Button>
                <Button
                  v-else
                  variant="outline"
                  size="sm"
                  class="text-green-600 border-green-200 hover:bg-green-50 dark:text-green-400"
                  @click="restoreProvince(province)"
                >
                  Restore
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Cities Tab -->
    <div v-if="activeTab === 'cities'">
      <div class="flex justify-between items-center mb-4">
        <div class="flex items-center gap-4">
          <Input
            v-model="citiesSearch"
            placeholder="Search cities..."
            class="w-48"
            @input="debouncedFetchCities"
          />
          <select
            v-model="citiesProvinceFilter"
            @change="fetchCities"
            class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
          >
            <option value="">All Provinces</option>
            <option v-for="p in provincesForCityDropdown" :key="p.id" :value="p.id">{{ p.name }} ({{ p.country_code }})</option>
          </select>
          <select
            v-model="citiesStatusFilter"
            @change="fetchCities"
            class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
          >
            <option value="active">Active</option>
            <option value="all">All</option>
            <option value="deleted">Deleted</option>
          </select>
        </div>
        <Button @click="openCreateCityModal">Add City</Button>
      </div>

      <div v-if="citiesLoading" class="text-center py-8 text-gray-500">Loading...</div>
      <div v-else-if="cities.length === 0" class="text-center py-8 text-gray-500">No cities found.</div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-800">
            <tr>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">ID</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Province</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Country</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Name</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Postal Prefix</th>
              <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Status</th>
              <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
            <tr
              v-for="city in cities"
              :key="city.id"
              :class="city.deleted_at ? 'opacity-60 bg-gray-50 dark:bg-gray-800/50' : ''"
            >
              <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{{ city.id }}</td>
              <td class="px-4 py-3 text-sm text-gray-900 dark:text-white">{{ city.province?.name || city.province_id }}</td>
              <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{{ city.country?.name || city.country_code }}</td>
              <td class="px-4 py-3 text-sm text-gray-900 dark:text-white">{{ city.name }}</td>
              <td class="px-4 py-3 text-sm font-mono text-gray-500 dark:text-gray-400">{{ city.postal_code_prefix || '-' }}</td>
              <td class="px-4 py-3">
                <span
                  :class="[
                    'px-2 py-1 text-xs font-medium rounded-full',
                    city.deleted_at
                      ? 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
                      : 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                  ]"
                >
                  {{ city.deleted_at ? 'Deleted' : 'Active' }}
                </span>
              </td>
              <td class="px-4 py-3 text-right">
                <Button variant="outline" size="sm" class="mr-2" @click="openEditCityModal(city)">Edit</Button>
                <Button
                  v-if="!city.deleted_at"
                  variant="outline"
                  size="sm"
                  class="text-red-600 border-red-200 hover:bg-red-50 dark:text-red-400"
                  @click="deleteCity(city)"
                >
                  Delete
                </Button>
                <Button
                  v-else
                  variant="outline"
                  size="sm"
                  class="text-green-600 border-green-200 hover:bg-green-50 dark:text-green-400"
                  @click="restoreCity(city)"
                >
                  Restore
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Country Modal -->
    <div v-if="showCountryModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          {{ editingCountry ? 'Edit Country' : 'Add Country' }}
        </h2>

        <div v-if="countryError" class="mb-4 p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-md text-sm">
          {{ countryError }}
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Code (2 letters) *</label>
            <Input
              v-model="countryForm.code"
              placeholder="e.g. ID, US, SG"
              maxlength="2"
              :disabled="!!editingCountry"
              class="uppercase"
            />
            <p class="text-xs text-gray-500 mt-1">ISO 3166-1 alpha-2 code</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name *</label>
            <Input v-model="countryForm.name" placeholder="e.g. Indonesia, United States" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Phone Code</label>
            <Input v-model="countryForm.phone_code" placeholder="e.g. +62, +1" />
          </div>
        </div>

        <div class="flex justify-end gap-2 mt-6">
          <Button variant="outline" @click="showCountryModal = false">Cancel</Button>
          <Button @click="saveCountry" :disabled="!isCountryFormValid">Save</Button>
        </div>
      </div>
    </div>

    <!-- Province Modal -->
    <div v-if="showProvinceModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          {{ editingProvince ? 'Edit Province' : 'Add Province' }}
        </h2>

        <div v-if="provinceError" class="mb-4 p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-md text-sm">
          {{ provinceError }}
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Country *</label>
            <select
              v-model="provinceForm.country_code"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
            >
              <option value="">Select Country</option>
              <option v-for="c in activeCountries" :key="c.code" :value="c.code">{{ c.name }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name *</label>
            <Input v-model="provinceForm.name" placeholder="e.g. DKI Jakarta, California" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Code (optional)</label>
            <Input v-model="provinceForm.code" placeholder="e.g. JKT, CA" />
          </div>
        </div>

        <div class="flex justify-end gap-2 mt-6">
          <Button variant="outline" @click="showProvinceModal = false">Cancel</Button>
          <Button @click="saveProvince" :disabled="!isProvinceFormValid">Save</Button>
        </div>
      </div>
    </div>

    <!-- City Modal -->
    <div v-if="showCityModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          {{ editingCity ? 'Edit City' : 'Add City' }}
        </h2>

        <div v-if="cityError" class="mb-4 p-3 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-md text-sm">
          {{ cityError }}
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Province *</label>
            <select
              v-model="cityForm.province_id"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
            >
              <option value="">Select Province</option>
              <option v-for="p in provincesForCityDropdown" :key="p.id" :value="String(p.id)">{{ p.name }} ({{ p.country_code }})</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name *</label>
            <Input v-model="cityForm.name" placeholder="e.g. Jakarta Selatan, Los Angeles" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Postal Code Prefix (optional)</label>
            <Input v-model="cityForm.postal_code_prefix" placeholder="e.g. 12, 90" />
          </div>
        </div>

        <div class="flex justify-end gap-2 mt-6">
          <Button variant="outline" @click="showCityModal = false">Cancel</Button>
          <Button @click="saveCity" :disabled="!isCityFormValid">Save</Button>
        </div>
      </div>
    </div>
  </div>
</template>
