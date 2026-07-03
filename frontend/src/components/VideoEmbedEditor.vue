<script setup>
import { ref, computed, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { Plus, Trash2, Play, GripVertical } from 'lucide-vue-next'

const props = defineProps({
  modelValue: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue'])

const videos = ref([...(props.modelValue || [])])
const MAX_VIDEOS = 5

const platforms = [
  { value: 'youtube', label: 'YouTube', icon: '🎬', placeholder: 'dQw4w9WgXcQ or full URL', defaultAspect: 'landscape' },
  { value: 'tiktok', label: 'TikTok', icon: '🎵', placeholder: '7123456789012345678 or full URL', defaultAspect: 'portrait' },
  { value: 'instagram', label: 'Instagram Reels', icon: '📸', placeholder: 'ABC123xyz or full URL', defaultAspect: 'portrait' }
]

// Get default aspect ratio for a platform
function getDefaultAspectRatio(platform) {
  const p = platforms.find(pl => pl.value === platform)
  return p ? p.defaultAspect : 'landscape'
}

const canAddVideo = computed(() => videos.value.length < MAX_VIDEOS)

// Watch for external changes
watch(() => props.modelValue, (newVal) => {
  if (JSON.stringify(newVal) !== JSON.stringify(videos.value)) {
    videos.value = [...(newVal || [])]
  }
}, { deep: true })

// Emit changes
function emitUpdate() {
  emit('update:modelValue', [...videos.value])
}

function addVideo() {
  if (!canAddVideo.value) return
  videos.value.push({
    platform: 'youtube',
    video_id: '',
    autoplay: false,
    caption: '',
    aspect_ratio: 'landscape' // Default for YouTube
  })
  emitUpdate()
}

function removeVideo(index) {
  videos.value.splice(index, 1)
  emitUpdate()
}

function updateVideo(index, field, value) {
  videos.value[index][field] = value
  // When platform changes, update aspect_ratio to platform default
  if (field === 'platform') {
    videos.value[index].aspect_ratio = getDefaultAspectRatio(value)
  }
  emitUpdate()
}

// Get preview aspect class based on aspect_ratio setting
function getPreviewAspectClass(video) {
  const aspectRatio = video.aspect_ratio || getDefaultAspectRatio(video.platform)
  if (aspectRatio === 'portrait') {
    return 'aspect-[9/20] max-w-[200px]'
  }
  return 'aspect-video max-w-sm'
}

// Parse video URL to extract ID
function parseVideoUrl(url, platform) {
  if (!url) return url

  // If it's already just an ID (no slashes or dots), return as-is
  if (!/[\/\.]/.test(url)) return url

  try {
    if (platform === 'youtube') {
      // youtube.com/watch?v=ID
      const watchMatch = url.match(/[?&]v=([^&]+)/)
      if (watchMatch) return watchMatch[1]
      // youtu.be/ID
      const shortMatch = url.match(/youtu\.be\/([^?&]+)/)
      if (shortMatch) return shortMatch[1]
      // youtube.com/embed/ID
      const embedMatch = url.match(/embed\/([^?&]+)/)
      if (embedMatch) return embedMatch[1]
    } else if (platform === 'tiktok') {
      // tiktok.com/@user/video/ID
      const match = url.match(/video\/(\d+)/)
      if (match) return match[1]
    } else if (platform === 'instagram') {
      // instagram.com/reel/ID/
      const match = url.match(/reel\/([^\/\?]+)/)
      if (match) return match[1]
      // instagram.com/p/ID/
      const pMatch = url.match(/\/p\/([^\/\?]+)/)
      if (pMatch) return pMatch[1]
    }
  } catch (e) {
    console.error('Failed to parse URL:', e)
  }

  return url
}

function handleVideoIdInput(index, event) {
  const input = event.target.value
  const platform = videos.value[index].platform
  const parsed = parseVideoUrl(input, platform)
  updateVideo(index, 'video_id', parsed)
}

function getEmbedUrl(video) {
  if (!video.video_id) return null

  switch (video.platform) {
    case 'youtube':
      return `https://www.youtube.com/embed/${video.video_id}?autoplay=${video.autoplay ? 1 : 0}&mute=1`
    case 'tiktok':
      return `https://www.tiktok.com/embed/v2/${video.video_id}`
    case 'instagram':
      return `https://www.instagram.com/reel/${video.video_id}/embed`
    default:
      return null
  }
}

function getPlatformInfo(platformCode) {
  return platforms.find(p => p.value === platformCode) || platforms[0]
}
</script>

<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h3 class="text-lg font-medium text-gray-900 dark:text-gray-100">Video Embeds</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          Add up to {{ MAX_VIDEOS }} videos from YouTube, TikTok, or Instagram Reels.
        </p>
      </div>
      <span class="text-sm text-gray-500">
        {{ videos.length }} / {{ MAX_VIDEOS }} videos
      </span>
    </div>

    <!-- Video List -->
    <div class="space-y-4">
      <div
        v-for="(video, index) in videos"
        :key="index"
        class="border border-gray-200 dark:border-gray-700 rounded-lg p-4"
      >
        <div class="flex items-start gap-4">
          <!-- Drag handle (future enhancement) -->
          <div class="hidden sm:flex pt-2 text-gray-400">
            <GripVertical class="w-5 h-5" />
          </div>

          <!-- Video config -->
          <div class="flex-1 space-y-3">
            <!-- Platform select -->
            <div class="flex flex-wrap gap-3">
              <select
                :value="video.platform"
                @change="updateVideo(index, 'platform', $event.target.value)"
                class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-sm"
              >
                <option v-for="p in platforms" :key="p.value" :value="p.value">
                  {{ p.icon }} {{ p.label }}
                </option>
              </select>

              <div class="flex items-center gap-2">
                <input
                  type="checkbox"
                  :id="`autoplay-${index}`"
                  :checked="video.autoplay"
                  @change="updateVideo(index, 'autoplay', $event.target.checked)"
                  class="h-4 w-4 text-zinc-600 rounded border-gray-300 dark:border-gray-600"
                />
                <label :for="`autoplay-${index}`" class="text-sm text-gray-600 dark:text-gray-400">
                  Autoplay (muted)
                </label>
              </div>
            </div>

            <!-- Video ID input -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Video ID or URL
              </label>
              <Input
                :value="video.video_id"
                @input="handleVideoIdInput(index, $event)"
                :placeholder="getPlatformInfo(video.platform).placeholder"
              />
            </div>

            <!-- Caption -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Caption (optional)
              </label>
              <Input
                :value="video.caption"
                @input="updateVideo(index, 'caption', $event.target.value)"
                placeholder="e.g., Product Demo"
                maxlength="255"
              />
            </div>

            <!-- Aspect Ratio -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Aspect Ratio
              </label>
              <div class="flex gap-4">
                <label class="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    :name="`aspect-${index}`"
                    value="landscape"
                    :checked="(video.aspect_ratio || getDefaultAspectRatio(video.platform)) === 'landscape'"
                    @change="updateVideo(index, 'aspect_ratio', 'landscape')"
                    class="h-4 w-4 text-zinc-600 border-gray-300 dark:border-gray-600"
                  />
                  <span class="text-sm text-gray-600 dark:text-gray-400">
                    Landscape (16:9)
                  </span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    :name="`aspect-${index}`"
                    value="portrait"
                    :checked="(video.aspect_ratio || getDefaultAspectRatio(video.platform)) === 'portrait'"
                    @change="updateVideo(index, 'aspect_ratio', 'portrait')"
                    class="h-4 w-4 text-zinc-600 border-gray-300 dark:border-gray-600"
                  />
                  <span class="text-sm text-gray-600 dark:text-gray-400">
                    Portrait (9:16)
                  </span>
                </label>
              </div>
            </div>

            <!-- Preview -->
            <div v-if="video.video_id" class="mt-3">
              <p class="text-sm text-gray-500 mb-2">Preview:</p>
              <div :class="[getPreviewAspectClass(video), 'rounded-lg overflow-hidden bg-gray-100 dark:bg-gray-800']">
                <iframe
                  :src="getEmbedUrl(video)"
                  class="w-full h-full"
                  frameborder="0"
                  allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                  allowfullscreen
                ></iframe>
              </div>
            </div>
          </div>

          <!-- Delete button -->
          <button
            @click="removeVideo(index)"
            class="p-2 text-gray-400 hover:text-red-500 transition-colors"
            title="Remove video"
          >
            <Trash2 class="w-5 h-5" />
          </button>
        </div>
      </div>
    </div>

    <!-- Add button -->
    <Button
      v-if="canAddVideo"
      variant="outline"
      @click="addVideo"
      class="w-full"
    >
      <Plus class="w-4 h-4 mr-2" />
      Add Video
    </Button>

    <!-- Empty state -->
    <div v-if="videos.length === 0" class="text-center py-8 text-gray-500 border-2 border-dashed border-gray-200 dark:border-gray-700 rounded-lg">
      <Play class="w-12 h-12 mx-auto mb-2 text-gray-300" />
      <p>No videos added yet</p>
      <Button variant="outline" @click="addVideo" class="mt-3">
        <Plus class="w-4 h-4 mr-2" />
        Add Your First Video
      </Button>
    </div>
  </div>
</template>
