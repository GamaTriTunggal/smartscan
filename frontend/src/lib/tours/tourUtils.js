/**
 * DOM helper utilities for Driver.js product tours.
 * Handles element waiting, auto-fill, and auto-click operations.
 */

// ── Per-tour-session nonce (prevents CustomEvent forgery) ──
let _tourNonce = null

export function setTourNonce(nonce) {
  _tourNonce = nonce
}

export function getTourNonce() {
  return _tourNonce
}

/**
 * Poll for an element to appear in the DOM.
 * @param {string} selector - CSS selector
 * @param {number} timeout - Max wait time in ms (default 5000)
 * @returns {Promise<Element>}
 */
export function waitForElement(selector, timeout = 5000) {
  return new Promise((resolve, reject) => {
    const el = document.querySelector(selector)
    if (el) return resolve(el)

    const interval = 100
    let elapsed = 0
    const timer = setInterval(() => {
      const found = document.querySelector(selector)
      if (found) {
        clearInterval(timer)
        return resolve(found)
      }
      elapsed += interval
      if (elapsed >= timeout) {
        clearInterval(timer)
        reject(new Error(`Tour: element "${selector}" not found after ${timeout}ms`))
      }
    }, interval)
  })
}

/**
 * Auto-fill a text input and trigger v-model sync.
 * @param {string} selector
 * @param {string} value
 */
export async function autoFillInput(selector, value) {
  const el = await waitForElement(selector)
  const input = el.tagName === 'INPUT' ? el : el.querySelector('input')
  if (!input) return

  // Use native setter to ensure Vue v-model picks it up
  const nativeSetter = Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype, 'value'
  ).set
  nativeSetter.call(input, value)
  input.dispatchEvent(new Event('input', { bubbles: true }))
}

/**
 * Auto-fill a textarea and trigger v-model sync.
 * @param {string} selector
 * @param {string} value
 */
export async function autoFillTextarea(selector, value) {
  const el = await waitForElement(selector)
  const textarea = el.tagName === 'TEXTAREA' ? el : el.querySelector('textarea')
  if (!textarea) return

  const nativeSetter = Object.getOwnPropertyDescriptor(
    HTMLTextAreaElement.prototype, 'value'
  ).set
  nativeSetter.call(textarea, value)
  textarea.dispatchEvent(new Event('input', { bubbles: true }))
}

/**
 * Auto-fill a select dropdown and trigger v-model sync.
 * @param {string} selector
 * @param {string|function} valueFinder - Exact value string, or function(options) => value
 */
export async function autoFillSelect(selector, valueFinder) {
  const el = await waitForElement(selector)
  const select = el.tagName === 'SELECT' ? el : el.querySelector('select')
  if (!select) return

  let value = valueFinder
  if (typeof valueFinder === 'function') {
    value = valueFinder(Array.from(select.options))
  }

  if (value !== undefined && value !== null) {
    const nativeSetter = Object.getOwnPropertyDescriptor(
      HTMLSelectElement.prototype, 'value'
    ).set
    nativeSetter.call(select, value)
    select.dispatchEvent(new Event('change', { bubbles: true }))
  }
}

/**
 * Auto-fill a date input.
 * @param {string} selector
 * @param {string} dateString - YYYY-MM-DD format
 */
export async function autoFillDate(selector, dateString) {
  const el = await waitForElement(selector)
  const input = el.tagName === 'INPUT' ? el : el.querySelector('input[type="date"]')
  if (!input) return

  const nativeSetter = Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype, 'value'
  ).set
  nativeSetter.call(input, dateString)
  input.dispatchEvent(new Event('input', { bubbles: true }))
}

/**
 * Auto-fill a number input.
 * @param {string} selector
 * @param {number} value
 */
export async function autoFillNumber(selector, value) {
  const el = await waitForElement(selector)
  const input = el.tagName === 'INPUT' ? el : el.querySelector('input[type="number"]')
  if (!input) return

  const nativeSetter = Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype, 'value'
  ).set
  nativeSetter.call(input, String(value))
  input.dispatchEvent(new Event('input', { bubbles: true }))
}

/**
 * Click an element.
 * @param {string} selector
 */
export async function autoClick(selector) {
  const el = await waitForElement(selector)
  el.click()
}

/**
 * Scroll an element into view within its scroll container.
 * @param {string} selector
 */
export async function scrollIntoView(selector) {
  const el = await waitForElement(selector)
  el.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

/**
 * Small delay utility.
 * @param {number} ms
 */
export function delay(ms) {
  return new Promise(resolve => setTimeout(resolve, ms))
}

/**
 * Set a reactive value directly via CustomEvent.
 * Components listen for 'tour-set-value' and update their own refs.
 * This bypasses DOM manipulation entirely — no synthetic events, no native setters.
 * @param {string} field - Field identifier the component recognizes
 * @param {*} value - Value to set
 */
export function tourSetValue(field, value) {
  window.dispatchEvent(new CustomEvent('tour-set-value', { detail: { field, value, _nonce: _tourNonce } }))
}

/**
 * Generate a unique tutorial product name by checking existing products via API.
 * Pattern: "Product Example 1", "Product Example 2", etc.
 * @param {string} baseName - Base name prefix (default "Product Example")
 * @returns {Promise<string>} Next available product name
 */
export async function getNextTutorialProductName(baseName = 'Product Example') {
  try {
    // Use relative path so requests go through Vite proxy (same-origin, no CORS issues)
    const apiUrl = '/api/v1'
    const res = await fetch(`${apiUrl}/tenant/products?search=${encodeURIComponent(baseName)}&limit=100`, {
      credentials: 'include',
    })
    if (!res.ok) return `${baseName} 1`

    const json = await res.json()
    const products = json.data?.products || []
    const names = products.map(p => p.product_name)

    let maxNum = 0
    const pattern = new RegExp(`^${baseName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(?: (\\d+))?$`)
    for (const name of names) {
      const match = name.match(pattern)
      if (match) {
        const num = match[1] ? parseInt(match[1], 10) : 1
        if (num > maxNum) maxNum = num
      }
    }
    return `${baseName} ${maxNum + 1}`
  } catch {
    return `${baseName} 1`
  }
}

/**
 * Generate a unique geofence batch name by checking existing batches for the first "Product Example" product.
 * Pattern: "Geofence Example 1", "Geofence Example 2", etc.
 * @param {string} baseName - Base name prefix (default "Geofence Example")
 * @returns {Promise<string>} Next available batch name
 */
export async function getNextGeofenceBatchName(baseName = 'Geofence Example') {
  try {
    const apiUrl = '/api/v1'
    const productRes = await fetch(`${apiUrl}/tenant/products?search=${encodeURIComponent('Product Example')}&limit=1`, {
      credentials: 'include',
    })
    if (!productRes.ok) return `${baseName} 1`
    const productJson = await productRes.json()
    const products = productJson.data?.products || []
    if (products.length === 0) return `${baseName} 1`

    const productId = products[0].id
    const batchRes = await fetch(`${apiUrl}/tenant/qr-batches?product_id=${productId}&limit=100`, {
      credentials: 'include',
    })
    if (!batchRes.ok) return `${baseName} 1`
    const batchJson = await batchRes.json()
    const batches = batchJson.data?.batches || []

    let maxNum = 0
    const pattern = new RegExp(`^${baseName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(?: (\\d+))?$`)
    for (const batch of batches) {
      const match = batch.batch_name.match(pattern)
      if (match) {
        const num = match[1] ? parseInt(match[1], 10) : 1
        if (num > maxNum) maxNum = num
      }
    }
    return `${baseName} ${maxNum + 1}`
  } catch {
    return `${baseName} 1`
  }
}
