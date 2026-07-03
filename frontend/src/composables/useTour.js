import { ref, computed, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { driver } from 'driver.js'
import 'driver.js/dist/driver.css'
import { waitForElement, delay, setTourNonce, getTourNonce } from '@/lib/tours/tourUtils.js'
import { useAuthStore } from '@/stores/auth'

export { getTourNonce }

// ── Singleton state (shared across all components) ──

const activeTourId = ref(null)
const currentStepIndex = ref(0)
const tourData = ref({})          // arbitrary data passed between steps (e.g. productId)
const tourRegistry = {}           // tourId -> tour definition
let driverInstance = null
let interactiveCleanup = null     // cleanup function for interactive step listeners
let escapeHandler = null          // keydown handler for Escape during tour
let executingStepIndex = -1       // guards against duplicate executeCurrentStep for same step
let advancePending = false        // guards resumeIfActive when advanceStep is scheduled
let dispatcherPending = false     // guards against double-click while dispatcher tour is resolving

/**
 * Check if a tour is currently active (usable outside composable context).
 */
export function isTourActive() {
  return !!activeTourId.value
}

/**
 * Clear active (in-progress) tour state on logout.
 *
 * This is a standalone export (no Vue component context required) so it can
 * be called from:
 *   - auth.js `logout()` via dynamic import (SPA navigation to /login)
 *   - useAPI.js `clearAuthStorage()` inline (full page reload to /login)
 *
 * It resets module-level state, tears down the driver.js overlay (preventing
 * zombie overlays after SPA navigation), and removes the active-tour
 * localStorage key. Completed-tours history is intentionally kept — users
 * should not be forced to re-take tours they've already finished.
 *
 * Reads the user ID directly from localStorage to avoid importing the auth
 * store (which would create a circular dependency: auth → useTour → auth).
 */
export function clearActiveTourState(explicitUserId) {
  // Reset module-level state
  executingStepIndex = -1
  advancePending = false
  dispatcherPending = false
  setTourNonce(null)
  activeTourId.value = null
  currentStepIndex.value = 0
  tourData.value = {}

  // Tear down driver.js overlay and event listeners to prevent zombie UI
  // after SPA navigation (full page reloads don't need this, but it's safe).
  // Each call is try/catch'd because the DOM element backing the overlay or
  // interactive listener may have already been removed by Vue unmount.
  if (escapeHandler) {
    window.removeEventListener('keydown', escapeHandler)
    escapeHandler = null
  }
  if (interactiveCleanup) {
    try { interactiveCleanup() } catch { /* element already removed */ }
    interactiveCleanup = null
  }
  if (driverInstance) {
    try { driverInstance.destroy() } catch { /* DOM already gone */ }
    driverInstance = null
  }

  // Determine the user ID for the localStorage key.
  // - auth.js passes `explicitUserId` because it clears `localStorage['user']`
  //   BEFORE this function runs (preventing the fallback from finding it).
  // - useAPI.js reads userId before clearing, then inlines the removeItem.
  let userId = explicitUserId || 'anonymous'
  if (!explicitUserId) {
    try {
      const stored = localStorage.getItem('user')
      if (stored) {
        const u = JSON.parse(stored)
        if (u?.id) userId = u.id
      }
    } catch { /* ignore parse errors */ }
  }
  localStorage.removeItem(`smartscan_active_tour_${userId}`)
}

// ── Public API ──
export function useTour() {
  const router = useRouter()

  function getStorageKey() {
    const authStore = useAuthStore()
    const userId = authStore.user?.id || 'anonymous'
    return `smartscan_active_tour_${userId}`
  }

  function getCompletedKey() {
    const authStore = useAuthStore()
    const userId = authStore.user?.id || 'anonymous'
    return `smartscan_completed_tours_${userId}`
  }

  const isActive = computed(() => !!activeTourId.value)
  const currentTour = computed(() => activeTourId.value ? tourRegistry[activeTourId.value] : null)
  const currentStep = computed(() => {
    if (!currentTour.value) return null
    return currentTour.value.steps[currentStepIndex.value] || null
  })
  const totalSteps = computed(() => currentTour.value?.steps.length || 0)

  /**
   * Register a tour definition. Call this at app startup.
   */
  function registerTour(tourDef) {
    tourRegistry[tourDef.id] = tourDef
  }

  /**
   * Start a tour from step 0.
   *
   * Supports two tour types:
   * - Normal tour: has `steps` array, executes step-by-step
   * - Dispatcher tour: has `dispatch()` async function that decides which
   *   sub-tour to start based on runtime state (e.g., API call). Dispatcher
   *   tours do NOT persist state to localStorage — they just delegate to a
   *   sub-tour via another startTour() call. This keeps reload-mid-dispatch
   *   safe (no stale dispatcher state left behind).
   *
   * Re-entry guards:
   * - If another tour is already active, cancel it first so the new tour
   *   starts clean (no overlapping overlays or zombie click listeners).
   * - If a dispatcher is currently resolving (API in flight), ignore repeat
   *   invocations to prevent a double-dispatch race when the user double-
   *   clicks "Start Tour".
   */
  function startTour(tourId) {
    const tourDef = tourRegistry[tourId]
    if (!tourDef) {
      console.warn(`Tour "${tourId}" not registered`)
      return
    }

    // Dispatcher tour: delegate to sub-tour without persisting state.
    // If the dispatch function fails or never resolves, localStorage stays
    // clean so a page reload leaves the app in a normal state.
    if (typeof tourDef.dispatch === 'function') {
      // Ignore repeat invocations while a dispatcher is still resolving
      if (dispatcherPending) return
      dispatcherPending = true
      Promise.resolve(
        tourDef.dispatch({ startTour, router, delay })
      )
        .catch((err) => {
          console.error(`Tour dispatcher "${tourId}" failed:`, err)
        })
        .finally(() => {
          dispatcherPending = false
        })
      return
    }

    // If another tour is already running, cancel it first to avoid overlap.
    // This can happen when a dispatcher calls startTour on a sub-tour while
    // a previous sub-tour is still mid-flight.
    if (activeTourId.value && activeTourId.value !== tourId) {
      cancelTour()
    }

    // Normal tour: reset state and persist
    executingStepIndex = -1
    advancePending = false
    setTourNonce(crypto.randomUUID())
    activeTourId.value = tourId
    currentStepIndex.value = 0
    tourData.value = {}
    saveState()

    // Navigate to the correct page if needed, then execute
    const firstStep = tourDef.steps[0]
    if (firstStep.expectedRoute && !routeMatches(firstStep.expectedRoute)) {
      router.push(firstStep.expectedRoute).then(() => {
        // Page component will call resumeIfActive() on mount
      })
    } else {
      nextTick(() => executeCurrentStep())
    }
  }

  /**
   * Called by page components in onMounted to resume an active tour.
   */
  async function resumeIfActive() {
    // If advanceStep is pending from an interactive click, let it handle the transition
    if (advancePending) return

    loadState()
    if (!activeTourId.value) return
    if (!tourRegistry[activeTourId.value]) return

    const step = tourRegistry[activeTourId.value].steps[currentStepIndex.value]
    if (!step) {
      completeTour()
      return
    }

    // Check if current route matches expected route
    if (step.expectedRoute && !routeMatches(step.expectedRoute)) {
      return // wrong page, tour stays paused
    }

    // Small delay for DOM to settle after mount
    await delay(400)
    await executeCurrentStep()
  }

  /**
   * Advance to next step.
   */
  async function advanceStep() {
    executingStepIndex = -1
    advancePending = false
    cleanupInteractive()
    destroyDriver()

    currentStepIndex.value++
    saveState()

    const step = currentTour.value?.steps[currentStepIndex.value]
    if (!step) {
      completeTour()
      return
    }

    // If next step is on a different page, wait for navigation
    if (step.expectedRoute && !routeMatches(step.expectedRoute)) {
      // The target page's onMounted -> resumeIfActive() will pick it up
      return
    }

    await delay(300)
    await executeCurrentStep()
  }

  /**
   * Cancel the tour.
   */
  function cancelTour() {
    executingStepIndex = -1
    advancePending = false
    setTourNonce(null)
    cleanupInteractive()
    destroyDriver()
    activeTourId.value = null
    currentStepIndex.value = 0
    tourData.value = {}
    clearState()
    window.dispatchEvent(new CustomEvent('tour-cancelled'))
  }

  /**
   * Complete the tour (mark as finished).
   *
   * If the completed tour has a `parentTour` field, the parent chain is walked
   * iteratively and every ancestor is marked complete (up to any depth).
   * Safeguards:
   * - Only marks ancestors that are actually registered in the tour registry
   *   (prevents leaving orphan completion records from stale IDs).
   * - Uses a visited Set to short-circuit circular parentTour references.
   */
  function completeTour() {
    const tourId = activeTourId.value
    executingStepIndex = -1
    advancePending = false
    setTourNonce(null)
    cleanupInteractive()
    destroyDriver()
    activeTourId.value = null
    currentStepIndex.value = 0
    tourData.value = {}
    clearState()

    if (tourId) {
      markCompleted(tourId)
      // Walk the parentTour chain, marking each registered ancestor complete.
      const visited = new Set([tourId])
      let cursor = tourRegistry[tourId]?.parentTour
      while (cursor && !visited.has(cursor) && tourRegistry[cursor]) {
        visited.add(cursor)
        markCompleted(cursor)
        cursor = tourRegistry[cursor]?.parentTour
      }
    }
  }

  /**
   * Check if a tour has been completed before.
   */
  function isTourCompleted(tourId) {
    return getCompletedTours().includes(tourId)
  }

  /**
   * Get list of all completed tour IDs.
   */
  function getCompletedTours() {
    try {
      return JSON.parse(localStorage.getItem(getCompletedKey()) || '[]')
    } catch {
      return []
    }
  }

  /**
   * Get all registered tours.
   */
  function getRegisteredTours() {
    return Object.values(tourRegistry)
  }

  /**
   * Get the expected route for the current step (used by TourIndicator).
   */
  function getExpectedRoute() {
    return currentStep.value?.expectedRoute || null
  }

  /**
   * Store arbitrary data for cross-step communication.
   */
  function setTourData(key, value) {
    tourData.value[key] = value
    saveState()
  }

  function getTourData(key) {
    return tourData.value[key]
  }

  // ── Internal functions ──

  async function executeCurrentStep() {
    const tourDef = tourRegistry[activeTourId.value]
    if (!tourDef) return

    const stepIdx = currentStepIndex.value
    const step = tourDef.steps[stepIdx]
    if (!step) {
      completeTour()
      return
    }

    // Guard: prevent duplicate execution of the same step (race between advanceStep & resumeIfActive)
    if (executingStepIndex === stepIdx) return
    executingStepIndex = stepIdx

    // Wait for element if needed
    if (step.waitForEl !== false) {
      try {
        await waitForElement(step.selector, step.skipIfNoElement ? 1000 : 5000)
      } catch (err) {
        if (step.skipIfNoElement) {
          // Element not found but step is optional — skip to next
          executingStepIndex = -1
          advanceStep()
          return
        }
        console.warn(err.message)
        cancelTour()
        return
      }
    }

    // Scroll element into view
    const targetEl = document.querySelector(step.selector)
    if (targetEl) {
      targetEl.scrollIntoView({ behavior: 'smooth', block: 'center' })
      await delay(200)
    }

    // Run beforeHighlight callback (auto-fill, auto-click, etc.)
    if (step.beforeHighlight) {
      try {
        await step.beforeHighlight({ setTourData, getTourData, advanceStep })
      } catch (err) {
        console.warn('Tour beforeHighlight error:', err)
      }
    }

    // Elevate modal z-index if step is inside a modal
    if (step.insideModal) {
      elevateModalZIndex(step.selector)
    }

    // Create driver and highlight
    destroyDriver()

    const popoverButtons = buildPopoverButtons(step)

    driverInstance = driver({
      showProgress: false,
      showButtons: [],
      allowClose: true,
      overlayColor: 'black',
      overlayOpacity: 0.5,
      stagePadding: 8,
      stageRadius: 8,
      popoverClass: 'smartscan-tour-popover',
      allowKeyboardControl: false,
      onCloseClick: () => cancelTour(),
      onOverlayClick: () => {}, // prevent accidental close
    })

    driverInstance.highlight({
      element: step.selector,
      popover: {
        title: step.popover.title,
        description: buildDescription(step, popoverButtons),
        side: step.popover.side || 'bottom',
        align: step.popover.align || 'center',
        popoverClass: step.insideModal
          ? 'smartscan-tour-popover smartscan-tour-modal-popover'
          : 'smartscan-tour-popover',
      },
    })

    // Register Escape handler so tour owns the key (useEscapeKey defers to us)
    if (!escapeHandler) {
      escapeHandler = (e) => {
        if (e.key === 'Escape') cancelTour()
      }
      window.addEventListener('keydown', escapeHandler)
    }

    // Handle step type
    if (step.type === 'auto-fill' || step.type === 'auto-click') {
      if (step.waitForNext) {
        setupInfoStep(step)
      } else {
        const advanceMs = step.autoAdvanceMs || 1500
        setTimeout(() => {
          if (activeTourId.value) advanceStep()
        }, advanceMs)
      }
    } else if (step.type === 'interactive') {
      setupInteractiveStep(step)
    } else if (step.type === 'info') {
      // Info step: show Next/Finish button via click handler on popover
      setupInfoStep(step)
    }
  }

  function buildDescription(step, buttons) {
    const stepNum = `<div class="smartscan-tour-step-counter">Step ${currentStepIndex.value + 1} of ${totalSteps.value}</div>`
    const desc = step.popover.description || ''
    return `${stepNum}${desc}${buttons}`
  }

  function buildPopoverButtons(step) {
    if (step.type === 'auto-fill' || step.type === 'auto-click') {
      if (step.waitForNext) {
        const isLast = currentStepIndex.value === totalSteps.value - 1
        const btnText = isLast ? 'Finish Tour' : 'Next'
        return `<div class="smartscan-tour-buttons"><button class="smartscan-tour-btn" data-tour-action="${isLast ? 'finish' : 'next'}">${btnText}</button></div>`
      }
      return '<div class="smartscan-tour-hint">Auto-advancing...</div>'
    }
    if (step.type === 'interactive') {
      return '<div class="smartscan-tour-hint">Click the highlighted element to continue</div>'
    }
    if (step.type === 'info') {
      const isLast = currentStepIndex.value === totalSteps.value - 1
      const btnText = isLast ? 'Finish Tour' : 'Next'
      return `<div class="smartscan-tour-buttons"><button class="smartscan-tour-btn" data-tour-action="${isLast ? 'finish' : 'next'}">${btnText}</button></div>`
    }
    return ''
  }

  function setupInteractiveStep(step) {
    const el = document.querySelector(step.selector)
    if (!el) return

    // Allow clicking the element through the overlay
    el.style.pointerEvents = 'auto'
    el.style.position = 'relative'
    el.style.zIndex = '10001'

    const handler = () => {
      el.removeEventListener('click', handler, true)
      el.style.pointerEvents = ''
      el.style.position = ''
      el.style.zIndex = ''
      // Small delay to let the click action take effect (modal open, navigation, etc.)
      advancePending = true
      setTimeout(() => advanceStep(), 200)
    }

    el.addEventListener('click', handler, true)
    interactiveCleanup = () => {
      el.removeEventListener('click', handler, true)
      el.style.pointerEvents = ''
      el.style.position = ''
      el.style.zIndex = ''
    }
  }

  function setupInfoStep() {
    // Use event delegation for the popover button
    const handler = (e) => {
      const btn = e.target.closest('[data-tour-action]')
      if (!btn) return
      document.removeEventListener('click', handler, true)
      const action = btn.dataset.tourAction
      if (action === 'finish') {
        completeTour()
      } else {
        advanceStep()
      }
    }
    document.addEventListener('click', handler, true)
    interactiveCleanup = () => document.removeEventListener('click', handler, true)
  }

  function elevateModalZIndex(selector) {
    const el = document.querySelector(selector)
    if (!el) return
    // Find the modal container (fixed inset-0 z-50)
    const modal = el.closest('.fixed.inset-0')
    if (modal) {
      modal.style.zIndex = '10001'
      // Also elevate the modal content
      const content = modal.querySelector('.relative')
      if (content) content.style.zIndex = '10001'
    }
  }

  function routeMatches(expectedRoute) {
    const current = router.currentRoute.value.path
    if (expectedRoute.includes(':')) {
      // Pattern matching for dynamic routes like /tenant/products/:id/batches
      const regex = new RegExp('^' + expectedRoute.replace(/:[^/]+/g, '[^/]+') + '$')
      return regex.test(current)
    }
    return current === expectedRoute
  }

  function destroyDriver() {
    if (escapeHandler) {
      window.removeEventListener('keydown', escapeHandler)
      escapeHandler = null
    }
    if (driverInstance) {
      driverInstance.destroy()
      driverInstance = null
    }
    // Reset any elevated modal z-index
    document.querySelectorAll('.fixed.inset-0').forEach(el => {
      el.style.zIndex = ''
      const content = el.querySelector('.relative')
      if (content) content.style.zIndex = ''
    })
  }

  function cleanupInteractive() {
    if (interactiveCleanup) {
      interactiveCleanup()
      interactiveCleanup = null
    }
  }

  function saveState() {
    localStorage.setItem(getStorageKey(), JSON.stringify({
      tourId: activeTourId.value,
      stepIndex: currentStepIndex.value,
      data: tourData.value,
    }))
  }

  function loadState() {
    try {
      const stored = localStorage.getItem(getStorageKey())
      if (stored) {
        const parsed = JSON.parse(stored)
        activeTourId.value = parsed.tourId
        currentStepIndex.value = parsed.stepIndex || 0
        tourData.value = parsed.data || {}
        // Generate fresh nonce on reload if tour is active
        if (activeTourId.value && !getTourNonce()) {
          setTourNonce(crypto.randomUUID())
        }
      }
    } catch {
      clearState()
    }
  }

  function clearState() {
    localStorage.removeItem(getStorageKey())
  }

  function markCompleted(tourId) {
    const completed = getCompletedTours()
    if (!completed.includes(tourId)) {
      completed.push(tourId)
      localStorage.setItem(getCompletedKey(), JSON.stringify(completed))
    }
  }

  return {
    // State
    isActive,
    activeTourId,
    currentStepIndex,
    currentStep,
    totalSteps,
    currentTour,

    // Actions
    registerTour,
    startTour,
    resumeIfActive,
    advanceStep,
    cancelTour,
    completeTour,

    // Queries
    isTourCompleted,
    getCompletedTours,
    getRegisteredTours,
    getExpectedRoute,

    // Tour data
    setTourData,
    getTourData,
  }
}
