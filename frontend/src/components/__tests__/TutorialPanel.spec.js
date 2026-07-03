import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import TutorialPanel from '../TutorialPanel.vue'

// Mock vue-router
vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
    currentRoute: { value: { path: '/tenant/dashboard' } },
  }),
}))

// Mock driver.js
vi.mock('driver.js', () => ({
  driver: () => ({
    highlight: vi.fn(),
    destroy: vi.fn(),
  }),
}))
vi.mock('driver.js/dist/driver.css', () => ({}))

// Mock the tours registry
vi.mock('@/lib/tours/index.js', () => ({
  allTours: [
    {
      id: 'create-dynamic-product',
      name: 'Create Your First Dynamic QR Product',
      description: 'Learn how to create a product and generate QR codes.',
      estimatedMinutes: 3,
      requiredTier: ['intermediate', 'pro'],
      steps: [],
    },
    {
      id: 'product-settings',
      name: 'Configure Product Settings',
      description: 'Learn how to configure product settings.',
      estimatedMinutes: 5,
      requiredTier: ['intermediate', 'pro'],
      requires: 'create-dynamic-product',
      steps: [],
    },
    {
      id: 'create-landing-template',
      name: 'Create a Landing Page Template',
      description: 'Learn how to design a landing page template.',
      estimatedMinutes: 4,
      requiredTier: ['intermediate', 'pro'],
      steps: [],
    },
    {
      id: 'geofence-intermediate',
      name: 'Using Geofence (Intermediate Tier)',
      description: 'Learn how to set up distribution zone geofencing for grey market detection.',
      estimatedMinutes: 5,
      requiredTier: ['intermediate'],
      requires: 'create-dynamic-product',
      steps: [],
    },
    {
      id: 'geofence-pro',
      name: 'Using Geofence (Pro Tier)',
      description: 'Learn how to set up geofencing for grey market detection.',
      estimatedMinutes: 5,
      requiredTier: ['pro'],
      requires: 'create-dynamic-product',
      steps: [],
    },
  ],
}))

// Control isTourCompleted behavior per test
let mockCompletedTours = []
vi.mock('@/composables/useTour.js', () => ({
  useTour: () => ({
    isTourCompleted: (id) => mockCompletedTours.includes(id),
  }),
}))

describe('TutorialPanel', () => {
  beforeEach(() => {
    mockCompletedTours = []
  })

  function createWrapper(props = {}) {
    return mount(TutorialPanel, {
      props: {
        show: true,
        ...props,
      },
    })
  }

  it('renders panel when show is true', () => {
    const wrapper = createWrapper()
    expect(wrapper.find('h2').text()).toBe('Tutorials')
  })

  it('renders tour cards', () => {
    const wrapper = createWrapper()
    expect(wrapper.text()).toContain('Create Your First Dynamic QR Product')
    expect(wrapper.text()).toContain('Learn how to create a product')
    expect(wrapper.text()).toContain('~3 min')
  })

  it('shows "Start Tour" button for uncompleted tour', () => {
    const wrapper = createWrapper()
    const btn = wrapper.findAll('button').find(b => b.text().includes('Start Tour'))
    expect(btn).toBeTruthy()
  })

  it('shows "Restart" button for completed tour', () => {
    mockCompletedTours = ['create-dynamic-product']
    const wrapper = createWrapper()
    expect(wrapper.text()).toContain('Completed')
    const btn = wrapper.findAll('button').find(b => b.text().includes('Restart'))
    expect(btn).toBeTruthy()
  })

  it('emits close when close button is clicked', async () => {
    const wrapper = createWrapper()
    // Find the close button (X icon)
    const closeBtn = wrapper.findAll('button').find(b => b.find('svg path[d*="M6 18L18 6"]'))
    await closeBtn.trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('emits close when backdrop is clicked', async () => {
    const wrapper = createWrapper()
    const backdrop = wrapper.find('.fixed.inset-0.bg-black\\/30')
    await backdrop.trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('emits start-tour with tour ID', async () => {
    const wrapper = createWrapper()
    const startBtn = wrapper.findAll('button').find(b => b.text().includes('Start Tour'))
    await startBtn.trigger('click')
    expect(wrapper.emitted('start-tour')).toBeTruthy()
    expect(wrapper.emitted('start-tour')[0]).toEqual(['create-dynamic-product'])
  })

  it('has slide-out animation class when hidden', () => {
    const wrapper = createWrapper({ show: false })
    const panel = wrapper.findAll('.fixed.inset-y-0.right-0')[0]
    expect(panel.classes()).toContain('translate-x-full')
  })

  it('does not show "More tutorials coming soon" with multiple tours', () => {
    const wrapper = createWrapper()
    expect(wrapper.text()).not.toContain('More tutorials coming soon.')
  })

  it('renders correct tour cards for pro tier (no intermediate-only tours)', () => {
    const wrapper = createWrapper()
    expect(wrapper.text()).toContain('Create Your First Dynamic QR Product')
    expect(wrapper.text()).toContain('Configure Product Settings')
    expect(wrapper.text()).toContain('Create a Landing Page Template')
    expect(wrapper.text()).not.toContain('Using Geofence (Intermediate Tier)')
    expect(wrapper.text()).toContain('Using Geofence (Pro Tier)')
  })

  it('shows tier badge for pro-tier tour', () => {
    const wrapper = createWrapper()
    expect(wrapper.text()).toContain('pro')
  })

  it('shows comma-separated tier badge for multi-tier tours', () => {
    const wrapper = createWrapper()
    expect(wrapper.text()).toContain('intermediate, pro')
  })

  it('shows locked state when prerequisite not completed', () => {
    const wrapper = createWrapper()
    const cards = wrapper.findAll('.rounded-lg.p-4')
    // product-settings is the 2nd card (index 1) for pro tier
    const settingsCard = cards[1]
    expect(settingsCard.text()).toContain('Locked')
    expect(settingsCard.text()).toContain('Complete "Create Your First Dynamic QR Product" first')
    expect(settingsCard.classes()).toContain('opacity-50')
  })

  it('unlocks tour when prerequisite is completed', () => {
    mockCompletedTours = ['create-dynamic-product']
    const wrapper = createWrapper()
    const cards = wrapper.findAll('.rounded-lg.p-4')
    const settingsCard = cards[1]
    expect(settingsCard.text()).not.toContain('Complete "Create Your First Dynamic QR Product" first')
    expect(settingsCard.classes()).not.toContain('opacity-50')
    const btn = settingsCard.findAll('button').find(b => b.text().includes('Start Tour'))
    expect(btn).toBeTruthy()
  })

  it('create-dynamic-product and create-landing-template are always unlocked', () => {
    const wrapper = createWrapper()
    const cards = wrapper.findAll('.rounded-lg.p-4')
    // create-dynamic-product is index 0, create-landing-template is index 2
    expect(cards[0].classes()).not.toContain('opacity-50')
    expect(cards[0].text()).not.toContain('Complete "Create Your First Dynamic QR Product" first')
    expect(cards[2].classes()).not.toContain('opacity-50')
    expect(cards[2].text()).not.toContain('Complete "Create Your First Dynamic QR Product" first')
  })

  it('locked tour button is disabled', () => {
    const wrapper = createWrapper()
    const cards = wrapper.findAll('.rounded-lg.p-4')
    const settingsCard = cards[1]
    const btn = settingsCard.findAll('button').find(b => b.text().includes('Locked'))
    expect(btn).toBeTruthy()
    expect(btn.attributes('disabled')).toBeDefined()
  })
})
