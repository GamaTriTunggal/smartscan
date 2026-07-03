<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { onClickOutside } from '@vueuse/core'
import { useAPI } from '@/composables/useAPI'
import { useDateTime } from '@/composables/useDateTime'
import { Bell } from 'lucide-vue-next'

const router = useRouter()
const { get, post } = useAPI()
const { formatRelative } = useDateTime()

const open = ref(false)
const loading = ref(false)
const notifications = ref([])
const unreadCount = ref(0)
const markingAll = ref(false)

const containerRef = ref(null)
onClickOutside(containerRef, () => { open.value = false })

let pollTimer = null

async function fetchUnreadCount() {
  try {
    const response = await get('/tenant/notifications', { limit: 1, unread_only: true })
    if (response.success && response.data) {
      unreadCount.value = response.data.unread_count || 0
    }
  } catch (error) {
    // Silent failure - polling should never surface errors to the user
    console.error('Failed to fetch unread notification count:', error)
  }
}

async function fetchNotifications() {
  try {
    loading.value = true
    const response = await get('/tenant/notifications', { limit: 20, offset: 0 })
    if (response.success && response.data) {
      notifications.value = response.data.notifications || []
      unreadCount.value = response.data.unread_count || 0
    }
  } catch (error) {
    console.error('Failed to fetch notifications:', error)
  } finally {
    loading.value = false
  }
}

function toggleDropdown() {
  open.value = !open.value
  if (open.value) {
    // Fetch list lazily when the dropdown opens
    fetchNotifications()
  }
}

async function handleItemClick(item) {
  if (!item.read_at) {
    // Optimistic update; the mark-read call is fire-and-forget from UI perspective
    item.read_at = new Date().toISOString()
    unreadCount.value = Math.max(0, unreadCount.value - 1)
    try {
      await post(`/tenant/notifications/${item.id}/read`)
    } catch (error) {
      console.error('Failed to mark notification as read:', error)
    }
  }
  if (item.link) {
    open.value = false
    router.push(item.link)
  }
}

async function markAllRead() {
  if (markingAll.value) return
  try {
    markingAll.value = true
    const response = await post('/tenant/notifications/read-all')
    if (response.success) {
      const now = new Date().toISOString()
      notifications.value.forEach(n => {
        if (!n.read_at) n.read_at = now
      })
      unreadCount.value = 0
    }
  } catch (error) {
    console.error('Failed to mark all notifications as read:', error)
  } finally {
    markingAll.value = false
  }
}

onMounted(() => {
  fetchUnreadCount()
  // Poll unread count every 60s while mounted
  pollTimer = setInterval(fetchUnreadCount, 60000)
})

onUnmounted(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
})
</script>

<template>
  <div ref="containerRef" class="relative">
    <!-- Bell button -->
    <button
      type="button"
      class="relative p-2 rounded-full text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 focus:outline-none transition-colors duration-200"
      :aria-label="unreadCount > 0 ? `Notifications (${unreadCount} unread)` : 'Notifications'"
      @click="toggleDropdown"
    >
      <Bell class="w-5 h-5" />
      <span
        v-if="unreadCount > 0"
        class="absolute -top-0.5 -right-0.5 min-w-[18px] h-[18px] px-1 flex items-center justify-center text-[10px] font-bold text-white bg-red-500 rounded-full"
      >
        {{ unreadCount > 99 ? '99+' : unreadCount }}
      </span>
    </button>

    <!-- Dropdown panel -->
    <div
      v-if="open"
      class="absolute right-0 mt-2 w-80 sm:w-96 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg z-50 overflow-hidden"
    >
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">Notifications</h3>
        <button
          v-if="unreadCount > 0"
          type="button"
          class="text-xs font-medium text-zinc-600 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300 disabled:opacity-50"
          :disabled="markingAll"
          @click="markAllRead"
        >
          {{ markingAll ? 'Marking...' : 'Mark all read' }}
        </button>
      </div>

      <!-- List -->
      <div class="max-h-96 overflow-y-auto">
        <div v-if="loading" class="flex justify-center py-8">
          <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-zinc-500"></div>
        </div>

        <div v-else-if="notifications.length === 0" class="px-4 py-8 text-center">
          <Bell class="w-8 h-8 mx-auto mb-2 text-gray-300 dark:text-gray-600" />
          <p class="text-sm text-gray-500 dark:text-gray-400">No notifications yet</p>
        </div>

        <template v-else>
          <button
            v-for="item in notifications"
            :key="item.id"
            type="button"
            :class="[
              'w-full text-left px-4 py-3 border-b border-gray-100 dark:border-gray-700/50 last:border-b-0 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors duration-150',
              !item.read_at && 'bg-zinc-50/60 dark:bg-zinc-900/10'
            ]"
            @click="handleItemClick(item)"
          >
            <div class="flex items-start gap-2">
              <span
                v-if="!item.read_at"
                class="mt-1.5 w-2 h-2 shrink-0 rounded-full bg-zinc-500"
                aria-hidden="true"
              ></span>
              <div class="min-w-0 flex-1">
                <p
                  :class="[
                    'text-sm font-bold truncate',
                    item.read_at ? 'text-gray-700 dark:text-gray-300' : 'text-gray-900 dark:text-white'
                  ]"
                >
                  {{ item.title }}
                </p>
                <p v-if="item.body" class="text-xs text-gray-500 dark:text-gray-400 line-clamp-2 mt-0.5">
                  {{ item.body }}
                </p>
                <p class="text-[11px] text-gray-400 dark:text-gray-500 mt-1">
                  {{ formatRelative(item.created_at) }}
                </p>
              </div>
            </div>
          </button>
        </template>
      </div>
    </div>
  </div>
</template>
