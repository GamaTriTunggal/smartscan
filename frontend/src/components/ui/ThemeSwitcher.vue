<script setup>
import { useDarkMode } from '@/composables/useDarkMode'

const { themeMode, setThemeMode } = useDarkMode()

const themes = [
  { value: 'system', label: 'System', icon: 'computer' },
  { value: 'light', label: 'Light', icon: 'sun' },
  { value: 'dark', label: 'Dark', icon: 'moon' },
]
</script>

<template>
  <div class="relative inline-flex items-center bg-gray-100 dark:bg-gray-700 rounded-lg p-1">
    <button
      v-for="theme in themes"
      :key="theme.value"
      @click="setThemeMode(theme.value)"
      class="p-1.5 rounded-md transition-all duration-200"
      :class="themeMode === theme.value
        ? 'bg-white dark:bg-gray-600 shadow-sm text-zinc-600 dark:text-zinc-400'
        : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'"
      :title="theme.label"
    >
      <!-- Computer/System icon -->
      <svg v-if="theme.icon === 'computer'" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
      </svg>
      <!-- Sun icon -->
      <svg v-else-if="theme.icon === 'sun'" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
      </svg>
      <!-- Moon icon -->
      <svg v-else-if="theme.icon === 'moon'" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
      </svg>
    </button>
  </div>
</template>
