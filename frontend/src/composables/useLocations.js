import { ref } from 'vue'
import axios from 'axios'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'

// Cache for location data to avoid repeated API calls
const countriesCache = ref(null)
const provincesCache = ref({}) // key: country_code
const citiesCache = ref({}) // key: province_id

export function useLocations() {
  const countries = ref([])
  const provinces = ref([])
  const cities = ref([])

  const selectedCountry = ref(null)
  const selectedProvince = ref(null)
  const selectedCity = ref(null)

  const loadingCountries = ref(false)
  const loadingProvinces = ref(false)
  const loadingCities = ref(false)

  // Fetch countries (cached)
  const fetchCountries = async () => {
    if (countriesCache.value) {
      countries.value = countriesCache.value
      return countries.value
    }

    loadingCountries.value = true
    try {
      const response = await axios.get(`${API_URL}/locations/countries`, { withCredentials: true })
      if (response.data.success) {
        countriesCache.value = response.data.data || []
        countries.value = countriesCache.value
      }
    } catch (error) {
      console.error('Failed to fetch countries:', error)
      countries.value = []
    } finally {
      loadingCountries.value = false
    }
    return countries.value
  }

  // Fetch provinces by country code (cached)
  const fetchProvinces = async (countryCode) => {
    if (!countryCode) {
      provinces.value = []
      return []
    }

    if (provincesCache.value[countryCode]) {
      provinces.value = provincesCache.value[countryCode]
      return provinces.value
    }

    loadingProvinces.value = true
    try {
      const response = await axios.get(`${API_URL}/locations/provinces/${countryCode}`, { withCredentials: true })
      if (response.data.success) {
        provincesCache.value[countryCode] = response.data.data || []
        provinces.value = provincesCache.value[countryCode]
      }
    } catch (error) {
      console.error('Failed to fetch provinces:', error)
      provinces.value = []
    } finally {
      loadingProvinces.value = false
    }
    return provinces.value
  }

  // Fetch cities by province ID (cached)
  const fetchCities = async (provinceId) => {
    if (!provinceId) {
      cities.value = []
      return []
    }

    const cacheKey = String(provinceId)
    if (citiesCache.value[cacheKey]) {
      cities.value = citiesCache.value[cacheKey]
      return cities.value
    }

    loadingCities.value = true
    try {
      const response = await axios.get(`${API_URL}/locations/cities/${provinceId}`, { withCredentials: true })
      if (response.data.success) {
        citiesCache.value[cacheKey] = response.data.data || []
        cities.value = citiesCache.value[cacheKey]
      }
    } catch (error) {
      console.error('Failed to fetch cities:', error)
      cities.value = []
    } finally {
      loadingCities.value = false
    }
    return cities.value
  }

  // Handle country change - reset province and city
  const onCountryChange = async (countryCode) => {
    selectedCountry.value = countryCode
    selectedProvince.value = null
    selectedCity.value = null
    cities.value = []

    if (countryCode) {
      await fetchProvinces(countryCode)
    } else {
      provinces.value = []
    }
  }

  // Handle province change - reset city
  const onProvinceChange = async (provinceId) => {
    selectedProvince.value = provinceId
    selectedCity.value = null

    if (provinceId) {
      await fetchCities(provinceId)
    } else {
      cities.value = []
    }
  }

  // Handle city change
  const onCityChange = (cityId) => {
    selectedCity.value = cityId
  }

  // Reset all selections
  const resetSelections = () => {
    selectedCountry.value = null
    selectedProvince.value = null
    selectedCity.value = null
    provinces.value = []
    cities.value = []
  }

  // Note: Watchers removed to avoid double API calls
  // Use onCountryChange and onProvinceChange handlers instead

  return {
    // Data
    countries,
    provinces,
    cities,

    // Selected values
    selectedCountry,
    selectedProvince,
    selectedCity,

    // Loading states
    loadingCountries,
    loadingProvinces,
    loadingCities,

    // Methods
    fetchCountries,
    fetchProvinces,
    fetchCities,
    onCountryChange,
    onProvinceChange,
    onCityChange,
    resetSelections
  }
}
