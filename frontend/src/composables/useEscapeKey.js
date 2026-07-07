import { onMounted, onUnmounted } from 'vue'
import { isTourActive } from '@/composables/useTour.js'

/**
 * Composable for handling Escape key to close modals
 *
 * @param {Function} callback - Function to call when Escape is pressed
 * @param {import('vue').Ref<boolean>} [isActive] - Optional ref to check if handler should be active
 *
 * @example
 * // Basic usage - always active
 * useEscapeKey(() => emit('close'))
 *
 * @example
 * // With visibility check
 * useEscapeKey(() => emit('close'), toRef(props, 'show'))
 *
 * @example
 * // With local ref
 * const isOpen = ref(false)
 * useEscapeKey(() => { isOpen.value = false }, isOpen)
 */
export function useEscapeKey(callback, isActive = null) {
  const handleKeydown = (e) => {
    if (e.key === 'Escape') {
      if (isTourActive()) return  // let tour system handle Escape
      if (isActive === null || isActive.value) {
        callback()
      }
    }
  }

  onMounted(() => {
    window.addEventListener('keydown', handleKeydown)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', handleKeydown)
  })
}
