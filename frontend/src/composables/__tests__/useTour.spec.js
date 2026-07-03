import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Mock vue-router
const mockRouterPush = vi.fn()
const mockCurrentRoute = { value: { path: '/tenant/dashboard' } }
vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockRouterPush,
    currentRoute: mockCurrentRoute,
  }),
}))

// driver.js is mocked via resolve alias in vitest.config.js → tests/mocks/driver.mock.js

// Mock auth store
const mockUser = { id: 'user-123' }
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user: mockUser }),
}))

import { useTour, clearActiveTourState } from '../useTour'

const TEST_USER_ID = 'user-123'
const STORAGE_KEY = `smartscan_active_tour_${TEST_USER_ID}`
const COMPLETED_KEY = `smartscan_completed_tours_${TEST_USER_ID}`

const sampleTour = {
  id: 'test-tour',
  name: 'Test Tour',
  description: 'A test tour',
  estimatedMinutes: 1,
  steps: [
    {
      id: 'step-1',
      expectedRoute: '/tenant/dashboard',
      selector: '[data-tour="step-1"]',
      popover: { title: 'Step 1', description: 'First step' },
      type: 'info',
      waitForEl: false,
    },
    {
      id: 'step-2',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="step-2"]',
      popover: { title: 'Step 2', description: 'Second step' },
      type: 'interactive',
      waitForEl: false,
    },
  ],
}

describe('useTour', () => {
  let tour

  beforeEach(() => {
    localStorage.clear()
    mockRouterPush.mockClear()
    mockCurrentRoute.value = { path: '/tenant/dashboard' }
    tour = useTour()
    tour.cancelTour()
    tour.registerTour(sampleTour)
  })

  afterEach(() => {
    tour.cancelTour()
  })

  describe('registerTour', () => {
    it('registers a tour definition', () => {
      const tours = tour.getRegisteredTours()
      expect(tours.find(t => t.id === 'test-tour')).toBeTruthy()
    })
  })

  describe('startTour', () => {
    it('sets active tour state', () => {
      tour.startTour('test-tour')
      expect(tour.isActive.value).toBe(true)
      expect(tour.activeTourId.value).toBe('test-tour')
      expect(tour.currentStepIndex.value).toBe(0)
    })

    it('does nothing for unregistered tour', () => {
      tour.startTour('non-existent')
      expect(tour.isActive.value).toBe(false)
    })
  })

  describe('cancelTour', () => {
    it('clears active tour state', () => {
      tour.startTour('test-tour')
      tour.cancelTour()
      expect(tour.isActive.value).toBe(false)
      expect(tour.activeTourId.value).toBeNull()
    })
  })

  describe('completeTour', () => {
    it('clears active state', () => {
      tour.startTour('test-tour')
      tour.completeTour()
      expect(tour.isActive.value).toBe(false)
    })
  })

  describe('isTourCompleted', () => {
    it('returns false for uncompleted tour', () => {
      expect(tour.isTourCompleted('test-tour')).toBe(false)
    })
  })

  describe('getCompletedTours', () => {
    it('returns empty array when no tours completed', () => {
      expect(tour.getCompletedTours()).toEqual([])
    })

    it('handles corrupt localStorage', () => {
      localStorage.setItem(COMPLETED_KEY, 'not-json')
      expect(tour.getCompletedTours()).toEqual([])
    })
  })

  describe('tourData', () => {
    it('stores and retrieves tour data', () => {
      tour.startTour('test-tour')
      tour.setTourData('productId', '123')
      expect(tour.getTourData('productId')).toBe('123')
    })
  })

  describe('computed properties', () => {
    it('currentTour returns tour definition when active', () => {
      tour.startTour('test-tour')
      expect(tour.currentTour.value.id).toBe('test-tour')
    })

    it('currentTour returns null when no active tour', () => {
      expect(tour.currentTour.value).toBeNull()
    })

    it('currentStep returns step definition', () => {
      tour.startTour('test-tour')
      expect(tour.currentStep.value.id).toBe('step-1')
    })

    it('totalSteps returns correct count', () => {
      tour.startTour('test-tour')
      expect(tour.totalSteps.value).toBe(2)
    })
  })

  describe('resumeIfActive', () => {
    it('does nothing when no stored state', async () => {
      await tour.resumeIfActive()
      expect(tour.isActive.value).toBe(false)
    })

    it('handles corrupt localStorage gracefully', async () => {
      localStorage.setItem(STORAGE_KEY, 'bad-json')
      await tour.resumeIfActive()
      expect(tour.isActive.value).toBe(false)
    })
  })

  describe('getExpectedRoute', () => {
    it('returns expected route for current step', () => {
      tour.startTour('test-tour')
      expect(tour.getExpectedRoute()).toBe('/tenant/dashboard')
    })

    it('returns null when no active tour', () => {
      expect(tour.getExpectedRoute()).toBeNull()
    })
  })

  describe('dispatcher tours', () => {
    it('invokes dispatch function instead of executing steps', () => {
      const dispatchSpy = vi.fn()
      const dispatcherTour = {
        id: 'dispatcher-tour',
        name: 'Dispatcher',
        description: 'Test dispatcher',
        estimatedMinutes: 1,
        dispatch: dispatchSpy,
      }
      tour.registerTour(dispatcherTour)
      tour.startTour('dispatcher-tour')
      expect(dispatchSpy).toHaveBeenCalledTimes(1)
      // Dispatcher tour must NOT persist its own state
      expect(tour.isActive.value).toBe(false)
      expect(localStorage.getItem(STORAGE_KEY)).toBeNull()
    })

    it('passes startTour helper into dispatch so it can delegate', async () => {
      const subTour = {
        id: 'sub-tour',
        name: 'Sub',
        description: 'Sub tour',
        estimatedMinutes: 1,
        steps: [
          {
            id: 'sub-step',
            expectedRoute: '/tenant/dashboard',
            selector: '[data-tour="sub-step"]',
            popover: { title: 'Sub', description: 'Sub' },
            type: 'info',
            waitForEl: false,
          },
        ],
      }
      const dispatcherTour = {
        id: 'dispatcher-delegates',
        name: 'Dispatcher',
        description: 'Delegates',
        estimatedMinutes: 1,
        async dispatch({ startTour }) {
          startTour('sub-tour')
        },
      }
      tour.registerTour(subTour)
      tour.registerTour(dispatcherTour)
      tour.startTour('dispatcher-delegates')
      // Drain the microtask queue so the async dispatch() fully resolves.
      // This makes the test robust against future async work inside dispatch.
      await Promise.resolve()
      await Promise.resolve()
      // Sub-tour is the active one after dispatch
      expect(tour.activeTourId.value).toBe('sub-tour')
    })

    it('swallows errors from dispatch and leaves state clean', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      const dispatcherTour = {
        id: 'dispatcher-throws',
        name: 'Dispatcher',
        description: 'Throws',
        estimatedMinutes: 1,
        async dispatch() {
          throw new Error('boom')
        },
      }
      tour.registerTour(dispatcherTour)
      tour.startTour('dispatcher-throws')
      // Let the promise chain resolve
      await Promise.resolve()
      await Promise.resolve()
      expect(tour.isActive.value).toBe(false)
      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })

    it('ignores rapid double-click on dispatcher tour while first resolves', async () => {
      let dispatchCount = 0
      let resolveDispatch
      const pendingPromise = new Promise((resolve) => { resolveDispatch = resolve })
      const dispatcherTour = {
        id: 'dispatcher-reentry',
        name: 'Dispatcher',
        description: 'Re-entry test',
        estimatedMinutes: 1,
        async dispatch() {
          dispatchCount += 1
          await pendingPromise
        },
      }
      tour.registerTour(dispatcherTour)
      tour.startTour('dispatcher-reentry')
      // Second click while first dispatch is still awaiting
      tour.startTour('dispatcher-reentry')
      tour.startTour('dispatcher-reentry')
      expect(dispatchCount).toBe(1)
      // Resolve the pending promise so the dispatcher finally completes.
      // Drain several microtasks to let .catch() and .finally() run before
      // asserting the guard is released.
      resolveDispatch()
      for (let i = 0; i < 5; i++) {
        // eslint-disable-next-line no-await-in-loop
        await Promise.resolve()
      }
      // After completion, a new click is allowed
      tour.startTour('dispatcher-reentry')
      expect(dispatchCount).toBe(2)
    })
  })

  describe('parentTour completion', () => {
    it('writes both sub-tour and parent tour IDs to completed list', () => {
      const parentTour = {
        id: 'parent-tour',
        name: 'Parent',
        description: 'Parent',
        estimatedMinutes: 1,
        dispatch: vi.fn(),
      }
      const subTour = {
        id: 'child-tour',
        name: 'Child',
        description: 'Child',
        estimatedMinutes: 1,
        parentTour: 'parent-tour',
        steps: [
          {
            id: 'child-step',
            expectedRoute: '/tenant/dashboard',
            selector: '[data-tour="child-step"]',
            popover: { title: 'Child', description: 'Child' },
            type: 'info',
            waitForEl: false,
          },
        ],
      }
      tour.registerTour(parentTour)
      tour.registerTour(subTour)
      tour.startTour('child-tour')
      tour.completeTour()

      // localStorage.setItem should have been called twice (sub-tour, then parent).
      // We assert on setItem invocations because localStorage is mocked in
      // tests/setup.js and doesn't actually persist values.
      const setItemCalls = localStorage.setItem.mock.calls.filter(
        ([key]) => key === COMPLETED_KEY
      )
      expect(setItemCalls.length).toBe(2)
      // First setItem should include 'child-tour'
      const firstStoredList = JSON.parse(setItemCalls[0][1])
      expect(firstStoredList).toContain('child-tour')
      // Second setItem should include 'parent-tour'
      const secondStoredList = JSON.parse(setItemCalls[1][1])
      expect(secondStoredList).toContain('parent-tour')
    })

    it('does NOT write parent entry when sub-tour has no parentTour field', () => {
      const subTour = {
        id: 'orphan-child',
        name: 'Orphan Child',
        description: 'Child',
        estimatedMinutes: 1,
        // No parentTour field
        steps: [
          {
            id: 'orphan-step',
            expectedRoute: '/tenant/dashboard',
            selector: '[data-tour="orphan-step"]',
            popover: { title: 'Orphan', description: 'Orphan' },
            type: 'info',
            waitForEl: false,
          },
        ],
      }
      tour.registerTour(subTour)
      tour.startTour('orphan-child')
      tour.completeTour()

      const setItemCalls = localStorage.setItem.mock.calls.filter(
        ([key]) => key === COMPLETED_KEY
      )
      // Only one setItem — for orphan-child, no parent
      expect(setItemCalls.length).toBe(1)
      const storedList = JSON.parse(setItemCalls[0][1])
      expect(storedList).toContain('orphan-child')
      expect(storedList).not.toContain('orphan-parent')
    })
  })

  describe('clearActiveTourState', () => {
    it('resets active tour state and clears localStorage using explicit userId', () => {
      tour.startTour('test-tour')
      expect(tour.isActive.value).toBe(true)

      // Pass explicit userId (as auth.js does — localStorage['user'] is already gone)
      clearActiveTourState('user-123')

      expect(tour.isActive.value).toBe(false)
      expect(tour.activeTourId.value).toBeNull()
      expect(tour.currentStepIndex.value).toBe(0)
      expect(localStorage.removeItem).toHaveBeenCalledWith(
        'smartscan_active_tour_user-123'
      )
    })

    it('falls back to reading userId from localStorage when no explicit ID', () => {
      localStorage.getItem.mockImplementation((key) => {
        if (key === 'user') return JSON.stringify({ id: 'user-456' })
        return null
      })

      tour.startTour('test-tour')
      clearActiveTourState()

      expect(localStorage.removeItem).toHaveBeenCalledWith(
        'smartscan_active_tour_user-456'
      )
    })

    it('uses "anonymous" if no explicit ID and user key is not in localStorage', () => {
      localStorage.getItem.mockReturnValue(null)

      tour.startTour('test-tour')
      clearActiveTourState()

      expect(localStorage.removeItem).toHaveBeenCalledWith(
        'smartscan_active_tour_anonymous'
      )
    })

    it('does not clear completed tours', () => {
      clearActiveTourState('user-123')

      const removedKeys = localStorage.removeItem.mock.calls.map(([k]) => k)
      expect(removedKeys).not.toContain(COMPLETED_KEY)
    })
  })
})
