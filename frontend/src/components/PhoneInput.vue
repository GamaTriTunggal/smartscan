<script setup>
import { ref, computed, watch, onMounted } from 'vue'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  placeholder: {
    type: String,
    default: 'Phone number'
  },
  required: {
    type: Boolean,
    default: false
  },
  disabled: {
    type: Boolean,
    default: false
  },
  defaultCountry: {
    type: String,
    default: 'ID'
  }
})

const emit = defineEmits(['update:modelValue'])

// Common country codes for Southeast Asia and international
// Hardcoded for phone input component - this is intentional and acceptable because:
// 1. ITU dial codes are standardized and rarely change (last major change was decades ago)
// 2. Server-side validation uses libphonenumber for comprehensive E.164 validation
// 3. Fetching from API would add latency for every phone input mount
// 4. This list covers our primary market (SEA) plus major international countries
const countryCodes = [
  { code: 'ID', dialCode: '+62', name: 'Indonesia', flag: '🇮🇩' },
  { code: 'PH', dialCode: '+63', name: 'Philippines', flag: '🇵🇭' },
  { code: 'VN', dialCode: '+84', name: 'Vietnam', flag: '🇻🇳' },
  { code: 'MY', dialCode: '+60', name: 'Malaysia', flag: '🇲🇾' },
  { code: 'SG', dialCode: '+65', name: 'Singapore', flag: '🇸🇬' },
  { code: 'TH', dialCode: '+66', name: 'Thailand', flag: '🇹🇭' },
  { code: 'JP', dialCode: '+81', name: 'Japan', flag: '🇯🇵' },
  { code: 'KR', dialCode: '+82', name: 'South Korea', flag: '🇰🇷' },
  { code: 'CN', dialCode: '+86', name: 'China', flag: '🇨🇳' },
  { code: 'IN', dialCode: '+91', name: 'India', flag: '🇮🇳' },
  { code: 'AU', dialCode: '+61', name: 'Australia', flag: '🇦🇺' },
  { code: 'US', dialCode: '+1', name: 'United States', flag: '🇺🇸' },
  { code: 'GB', dialCode: '+44', name: 'United Kingdom', flag: '🇬🇧' }
]

const selectedCountry = ref(null)
const localNumber = ref('')
const showDropdown = ref(false)

// Find country by dial code from E.164 number
const findCountryByDialCode = (e164Number) => {
  if (!e164Number || !e164Number.startsWith('+')) return null
  // Sort by dial code length descending to match longer codes first
  const sorted = [...countryCodes].sort((a, b) => b.dialCode.length - a.dialCode.length)
  return sorted.find(c => e164Number.startsWith(c.dialCode))
}

// Parse E.164 number into country and local number
const parseE164 = (e164Number) => {
  if (!e164Number) return { country: null, local: '' }

  const country = findCountryByDialCode(e164Number)
  if (country) {
    const local = e164Number.slice(country.dialCode.length)
    return { country, local }
  }
  return { country: null, local: e164Number.replace(/^\+/, '') }
}

// Format to E.164
const formatE164 = computed(() => {
  if (!selectedCountry.value || !localNumber.value) return ''
  // Remove leading zeros and non-digits
  const cleanNumber = localNumber.value.replace(/^0+/, '').replace(/\D/g, '')
  if (!cleanNumber) return ''
  return `${selectedCountry.value.dialCode}${cleanNumber}`
})

// Initialize from modelValue
const initFromModelValue = () => {
  if (props.modelValue) {
    const { country, local } = parseE164(props.modelValue)
    if (country) {
      selectedCountry.value = country
      localNumber.value = local
    } else {
      // Default to Indonesia if not parseable
      selectedCountry.value = countryCodes.find(c => c.code === props.defaultCountry) || countryCodes[0]
      localNumber.value = props.modelValue.replace(/^\+?\d{1,3}/, '').replace(/^0+/, '')
    }
  } else {
    selectedCountry.value = countryCodes.find(c => c.code === props.defaultCountry) || countryCodes[0]
    localNumber.value = ''
  }
}

// Watch for changes and emit
watch([selectedCountry, localNumber], () => {
  emit('update:modelValue', formatE164.value)
})

// Watch for external modelValue changes
watch(() => props.modelValue, (newVal, oldVal) => {
  if (newVal !== formatE164.value) {
    initFromModelValue()
  }
})

onMounted(() => {
  initFromModelValue()
})

const selectCountry = (country) => {
  selectedCountry.value = country
  showDropdown.value = false
}

// Handle number input - only allow digits
const handleNumberInput = (e) => {
  const value = e.target.value
  // Remove non-digits and leading zeros
  localNumber.value = value.replace(/\D/g, '').replace(/^0+/, '')
}

// Close dropdown when clicking outside
const closeDropdown = () => {
  showDropdown.value = false
}
</script>

<template>
  <div class="phone-input-container relative">
    <div class="flex min-w-0">
      <!-- Country Code Selector -->
      <div class="relative">
        <button
          type="button"
          @click="showDropdown = !showDropdown"
          :disabled="disabled"
          class="flex items-center gap-1 px-3 py-2 border border-r-0 border-gray-300 dark:border-gray-600 rounded-l-md bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-white hover:bg-gray-100 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-[#27272a] disabled:opacity-50 disabled:cursor-not-allowed min-w-[90px]"
        >
          <span v-if="selectedCountry">{{ selectedCountry.flag }}</span>
          <span class="text-sm font-medium">{{ selectedCountry?.dialCode || '+62' }}</span>
          <svg class="w-4 h-4 ml-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        <!-- Dropdown -->
        <div
          v-if="showDropdown"
          class="absolute z-50 mt-1 w-64 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md shadow-lg max-h-60 overflow-y-auto"
        >
          <button
            v-for="country in countryCodes"
            :key="country.code"
            type="button"
            @click="selectCountry(country)"
            class="w-full flex items-center gap-3 px-4 py-2 text-left hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-white"
            :class="{ 'bg-zinc-50 dark:bg-zinc-900/30': selectedCountry?.code === country.code }"
          >
            <span class="text-lg">{{ country.flag }}</span>
            <span class="flex-1 text-sm">{{ country.name }}</span>
            <span class="text-sm text-gray-500 dark:text-gray-400">{{ country.dialCode }}</span>
          </button>
        </div>
      </div>

      <!-- Number Input -->
      <input
        type="tel"
        :value="localNumber"
        @input="handleNumberInput"
        :placeholder="placeholder"
        :required="required"
        :disabled="disabled"
        class="flex-1 min-w-0 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-r-md bg-white dark:bg-gray-800 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-[#27272a] focus:border-zinc-500 disabled:opacity-50 disabled:cursor-not-allowed"
      />
    </div>

    <!-- Display formatted E.164 (for debugging/preview) -->
    <div v-if="formatE164" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
      Format: {{ formatE164 }}
    </div>

    <!-- Click outside handler -->
    <div
      v-if="showDropdown"
      class="fixed inset-0 z-40"
      @click="closeDropdown"
    />
  </div>
</template>

<style scoped>
.phone-input-container {
  width: 100%;
}
</style>
