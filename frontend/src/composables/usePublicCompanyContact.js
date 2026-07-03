import { ref } from 'vue'
import axios from 'axios'

const apiBase = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'

// Module-level cache — the public company contact is fetched at most once per
// app load and shared by every consumer (contact card, warranty consent label).
const companyName = ref('')
const contact = ref(null)
let fetchPromise = null

/**
 * Public company contact info (no auth required).
 * Backed by GET /public/company-contact with a module-level cache so multiple
 * components on the same page trigger a single request.
 */
export function usePublicCompanyContact() {
  function fetchOnce() {
    if (!fetchPromise) {
      fetchPromise = axios
        .get(`${apiBase}/public/company-contact`)
        .then((response) => {
          if (response.data?.success && response.data.data) {
            companyName.value = response.data.data.company_name || ''
            contact.value = response.data.data.contact || null
          }
        })
        .catch(() => {
          // Decorative public info only — fail silently.
        })
    }
    return fetchPromise
  }

  return { companyName, contact, fetchOnce }
}
