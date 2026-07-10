import {
  autoClick,
  delay,
  tourSetValue,
} from './tourUtils.js'

/**
 * Generate a unique tutorial template name by checking existing templates via API.
 * Pattern: "Template Example 1", "Template Example 2", etc.
 */
async function getNextTutorialTemplateName(baseName = 'Template Example') {
  try {
    const apiUrl = '/api/v1'
    const res = await fetch(`${apiUrl}/tenant/templates?type=validation&status=active&limit=100`, {
      credentials: 'include',
    })
    if (!res.ok) return `${baseName} 1`

    const json = await res.json()
    const templates = json.data?.templates || []
    const names = templates.map(t => t.template_name)

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

export const createLandingTemplateTour = {
  id: 'create-landing-template',
  name: 'Create a Landing Page Template',
  description: 'Learn how to design a landing page template that customers see when scanning your QR codes.',
  estimatedMinutes: 4,
  steps: [
    // ── Step 0: Sidebar → Landing ──
    {
      id: 'sidebar-landing',
      expectedRoute: null,
      selector: '[data-tour="sidebar-landing"]',
      popover: {
        title: 'Landing Page Templates',
        description: 'Navigate to the Landing Page Templates section. Click this menu item.',
        side: 'right',
        align: 'center',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 1: Click "+ Add Template" ──
    {
      id: 'add-template-btn',
      expectedRoute: '/tenant/templates',
      selector: '[data-tour="add-template-btn"]',
      popover: {
        title: 'Create a New Template',
        description: 'Click this button to create a new landing page template.',
        side: 'bottom',
        align: 'end',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 2: Auto-fill Template Name ──
    {
      id: 'template-name-input',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="template-name-input"]',
      popover: {
        title: 'Template Name',
        description: 'Give your template a descriptive name. Each template can be assigned to different products.',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        const name = await getNextTutorialTemplateName('Template Example')
        tourSetValue('template_name', name)
      },
    },

    // ── Step 3: Auto-fill header bg color ──
    {
      id: 'header-bg-color',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="header-bg-color"]',
      popover: {
        title: 'Header Background Color',
        description: 'Set the background color for the landing page header. We\'ll set it to a purple shade (#8a007e).',
        side: 'right',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('header_bg_color', '#8a007e')
      },
    },

    // ── Step 4: Company Logo section info ──
    {
      id: 'company-logo-section',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="company-logo-section"]',
      popover: {
        title: 'Company Logo',
        description: 'Upload your company logo (JPG or PNG, max 2MB). The logo appears at the top of the landing page header. You can upload it after saving the template.',
        side: 'right',
      },
      type: 'info',
      waitForEl: true,
    },

    // ── Step 5: Auto-fill Badge Text ──
    {
      id: 'badge-text-input',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="badge-text-input"]',
      popover: {
        title: 'Badge Text',
        description: 'This badge appears on the header to signal product authenticity. We\'ll set it to "Original Product".',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('badge_text', 'Original Product')
      },
    },

    // ── Step 6: Click Preset tab ──
    {
      id: 'bg-preset-tab',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="bg-preset-tab"]',
      popover: {
        title: 'Background Image — Preset',
        description: 'Choose a preset background image for your landing page. Click "Preset" to see available options.',
        side: 'bottom',
      },
      type: 'auto-click',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('bg_type', 'preset')
        await delay(500)
      },
    },

    // ── Step 7: Select last preset ──
    {
      id: 'bg-preset-last',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="bg-preset-last"]',
      popover: {
        title: 'Select a Preset',
        description: 'Pick a background image. We\'ll select this one. You can always change it later.',
        side: 'left',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('bg_preset_last', true)
      },
    },

    // ── Step 8: Overlay Opacity 0% ──
    {
      id: 'appearance-overlay-opacity',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="appearance-overlay-opacity"]',
      popover: {
        title: 'Overlay Opacity',
        description: 'Controls the dark overlay on top of the background image. Set to 0% for a clean, bright look.',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('overlay_opacity', 0)
      },
    },

    // ── Step 9: Card Opacity 50% ──
    {
      id: 'appearance-card-opacity',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="appearance-card-opacity"]',
      popover: {
        title: 'Card Opacity',
        description: 'Controls the transparency of the product info card. At 50%, the background image shows through for a glass effect.',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('card_opacity', 50)
      },
    },

    // ── Step 10: Card Blur 0px ──
    {
      id: 'appearance-card-blur',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="appearance-card-blur"]',
      popover: {
        title: 'Card Blur',
        description: 'Adds a blur effect to the card background. Set to 0px for a clear see-through effect.',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('card_blur', 0)
      },
    },

    // ── Step 11: Warranty Button section info ──
    {
      id: 'warranty-button-section',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="warranty-button-section"]',
      popover: {
        title: 'Warranty Activation Button',
        description: 'Customize the "Activate Warranty" button that end-users see on the landing page. Change the text, background color, and text color to match your brand.',
        side: 'right',
      },
      type: 'info',
      waitForEl: true,
    },

    // ── Step 14: Social Media Section info ──
    {
      id: 'social-media-section',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="social-media-section"]',
      popover: {
        title: 'Social Media Section',
        description: 'When the Sticky Bottom Bar is ON, social media icons are pinned to the bottom of the screen. When OFF, the icons follow the section order you define below.',
        side: 'right',
      },
      type: 'info',
      waitForEl: true,
    },

    // ── Step 15: Section Order info ──
    {
      id: 'section-order-section',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="section-order-section"]',
      popover: {
        title: 'Section Order',
        description: 'Drag and drop to arrange the order of sections on the landing page. You can position gallery, videos, certifications, and other sections exactly where you want them.',
        side: 'right',
      },
      type: 'info',
      waitForEl: true,
    },

    // ── Step 16: Click "Create Template" ──
    {
      id: 'create-template-btn',
      expectedRoute: '/tenant/templates/new',
      selector: '[data-tour="create-template-btn"]',
      popover: {
        title: 'Create Template',
        description: 'All set! Click "Create Template" to save your new landing page template. You can then assign it to products in their settings.',
        side: 'bottom',
      },
      type: 'interactive',
      waitForEl: true,
    },
  ],
}
