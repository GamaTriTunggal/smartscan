/**
 * Normalize the API's pagination object into a frontend-canonical shape.
 *
 * Every paginated backend response embeds `pagination: {page, limit, total,
 * total_page}` (singular total_page — see utils.PaginationMeta on the Go
 * side). This helper is the one place in the frontend that knows that wire
 * shape: read pagination through it so a backend contract change fails
 * loudly in dev instead of silently zeroing pagers.
 *
 * @param {Object} data - The envelope's `data` payload (`response.data` for
 *   useAPI consumers, `response.data.data` for raw axios).
 * @returns {{ page: number, limit: number, total: number, totalPages: number }}
 */
export function getPagination(data) {
  const p = data?.pagination
  if (!p && import.meta.env.DEV) {
    console.warn('[pagination] response payload has no pagination object:', data)
  }
  return {
    page: p?.page || 1,
    limit: p?.limit || 20,
    total: p?.total || 0,
    totalPages: p?.total_page || 0,
  }
}
