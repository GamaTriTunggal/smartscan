import { ref, watch } from 'vue'
import { useDebounceFn } from '@vueuse/core'

/**
 * Reusable composable for list pages with search, pagination, and filters.
 *
 * @param {Function} fetchFn - The function to call when filters change or page navigates
 * @param {Object} [options]
 * @param {number} [options.limit] - Items per page (default 20)
 * @param {number} [options.debounceMs] - Debounce delay for search input (default 300)
 * @returns {{ search, pagination, watchFilter, prevPage, nextPage }}
 *
 * Usage:
 *   const { search, pagination, watchFilter, prevPage, nextPage } = useListFilter(fetchItems)
 *   const statusFilter = ref('active')
 *   watchFilter(statusFilter)
 */
export function useListFilter(fetchFn, options = {}) {
  const { limit = 20, debounceMs = 300 } = options

  const search = ref('')
  const pagination = ref({ page: 1, limit, total: 0, total_page: 0 })

  const debouncedFetch = useDebounceFn(() => {
    pagination.value.page = 1
    fetchFn()
  }, debounceMs)

  watch(search, debouncedFetch)

  function watchFilter(...refs) {
    watch(refs, () => {
      pagination.value.page = 1
      fetchFn()
    })
  }

  function prevPage() {
    if (pagination.value.page > 1) {
      pagination.value.page--
      fetchFn()
    }
  }

  function nextPage() {
    if (pagination.value.page < pagination.value.total_page) {
      pagination.value.page++
      fetchFn()
    }
  }

  return { search, pagination, watchFilter, prevPage, nextPage }
}
