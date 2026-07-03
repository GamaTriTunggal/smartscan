import {
  autoClick,
  delay,
  tourSetValue,
} from './tourUtils.js'

export const productSettingsTour = {
  id: 'product-settings',
  name: 'Configure Product Settings',
  description: 'Learn how to edit product info, add certifications, customize the landing page template, and set up warranty.',
  estimatedMinutes: 5,
  requiredTier: ['intermediate', 'pro'],
  requires: 'create-dynamic-product',
  steps: [
    // ── Step 0: Sidebar → Dynamic QR ──
    {
      id: 'sidebar-dynamic-qr',
      expectedRoute: null,
      selector: '[data-tour="sidebar-dynamic-qr"]',
      popover: {
        title: 'Dynamic QR Products',
        description: 'Start by navigating to the Dynamic QR page. Click this menu item.',
        side: 'right',
        align: 'center',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 1: Search for product ──
    {
      id: 'search-product',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="search-product"]',
      popover: {
        title: 'Search Products',
        description: 'Use the search bar to find products. We\'ll search for "Product Example".',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('search_query', 'Product Example')
      },
    },

    // ── Step 2: Click Settings button ──
    {
      id: 'product-settings-btn',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="product-settings-btn"]',
      popover: {
        title: 'Product Settings',
        description: 'Click "Settings" to open the product detail page where you can configure all product settings.',
        side: 'bottom',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 3: Auto-fill Product Code (Basic Info tab) ──
    {
      id: 'product-code-input',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="product-code-input"]',
      popover: {
        title: 'Product Code',
        description: 'Add a product code (SKU) for easier identification. This appears on the landing page if enabled.',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('product_code', 'PE1')
      },
    },

    // ── Step 4: Save Basic Info ──
    {
      id: 'save-basic-info',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="save-basic-info"]',
      popover: {
        title: 'Save Changes',
        description: 'Click "Save Changes" to save the product code.',
        side: 'top',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 5: Switch to Certifications tab ──
    {
      id: 'tab-certifications',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="tab-certifications"]',
      popover: {
        title: 'Certifications Tab',
        description: 'Add certifications like Halal, BPOM, SNI, or ISO to build consumer trust. These appear on the landing page.',
        side: 'bottom',
      },
      type: 'auto-click',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('tab_switch', 'certifications')
        await delay(300)
      },
    },

    // ── Step 6: Click Add Certification button ──
    {
      id: 'add-cert-btn-detail',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="add-cert-btn-detail"]',
      popover: {
        title: 'Add Certification',
        description: 'Click to add a certification to this product.',
        side: 'bottom',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 7: Auto-fill cert type (modal) ──
    {
      id: 'cert-type-select-detail',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="cert-type-select-detail"]',
      popover: {
        title: 'Select Certification Type',
        description: 'Choose a certification type. We\'ll select "HACCP" as an example.',
        side: 'bottom',
      },
      type: 'auto-fill',
      insideModal: true,
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('detail_cert_type_id', 'haccp')
      },
    },

    // ── Step 8: Auto-fill registration number (modal) ──
    {
      id: 'cert-reg-detail',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="cert-reg-detail"]',
      popover: {
        title: 'Registration Number',
        description: 'Enter the certification registration number.',
        side: 'bottom',
      },
      type: 'auto-fill',
      insideModal: true,
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('detail_cert_reg_number', 'HACCP/ID/2023/1234')
      },
    },

    // ── Step 9: Click Add button (modal) ──
    {
      id: 'cert-submit-detail',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="cert-submit-detail"]',
      popover: {
        title: 'Add Certification',
        description: 'Click "Add" to save this certification to the product.',
        side: 'top',
      },
      type: 'interactive',
      insideModal: true,
      waitForEl: true,
    },

    // ── Step 10: Gallery tab info ──
    {
      id: 'tab-gallery',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="tab-gallery"]',
      popover: {
        title: 'Gallery Tab',
        description: 'Upload up to 15 PNG/JPG product images. The image marked as "Main" becomes the product\'s profile picture on the landing page.',
        side: 'bottom',
      },
      type: 'auto-click',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('tab_switch', 'gallery')
        await delay(300)
      },
    },

    // ── Step 11: Videos tab info ──
    {
      id: 'tab-videos',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="tab-videos"]',
      popover: {
        title: 'Videos Tab',
        description: 'Embed videos from Instagram, TikTok, and YouTube on your landing page.',
        side: 'bottom',
      },
      type: 'auto-click',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('tab_switch', 'videos')
        await delay(300)
      },
    },

    // ── Step 12: Social Media tab info ──
    {
      id: 'tab-social',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="tab-social"]',
      popover: {
        title: 'Social Media Tab',
        description: 'Add your company website and social media accounts (Instagram, TikTok, etc.) to display on the landing page.',
        side: 'bottom',
      },
      type: 'auto-click',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('tab_switch', 'links')
        await delay(300)
      },
    },

    // ── Step 13: Switch to Landing Page Template tab ──
    {
      id: 'tab-template',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="tab-template"]',
      popover: {
        title: 'Landing Page Template',
        description: 'Customize the landing page that customers see when they scan a QR code.',
        side: 'bottom',
      },
      type: 'auto-click',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('tab_switch', 'template')
        await delay(300)
      },
    },

    // ── Step 14: Select template ──
    {
      id: 'template-select',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="template-select"]',
      popover: {
        title: 'Select Template',
        description: 'Choose a validation page template. We\'ll select "Default Validation Template".',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('validation_template', 'Default Validation Template')
      },
    },

    // ── Step 15: Toggle Product Code display field ──
    {
      id: 'display-fields',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="display-fields"]',
      popover: {
        title: 'Display Fields',
        description: 'Enable "Product Code" to show the SKU on the landing page. You can also drag fields to reorder them.',
        side: 'right',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('display_product_code', true)
      },
    },

    // ── Step 16: Reorder display fields ──
    {
      id: 'display-fields-reorder',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="display-fields"]',
      popover: {
        title: 'Reorder Display Fields',
        description: 'We\'ll move Verification Count above Product Code, so scan count appears more prominently.',
        side: 'right',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        // Reorder: product_name, brand_name, show_verification_count, product_code, batch_code, production_date, expiry_date
        tourSetValue('display_field_order', ['product_name', 'brand_name', 'show_verification_count', 'product_code', 'batch_code', 'production_date', 'expiry_date'])
      },
    },

    // ── Step 17: Reorder sections ──
    {
      id: 'section-order',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="section-order"]',
      popover: {
        title: 'Section Order',
        description: 'We\'ll move Certifications to the top so it\'s the first thing customers see after the product info.',
        side: 'right',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('section_order', ['certifications', 'images', 'videos', 'social_accounts', 'website_link', 'description', 'warranty_button'])
      },
    },

    // ── Step 18: Advanced Customization info ──
    {
      id: 'advanced-customization',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="advanced-customization"]',
      popover: {
        title: 'Advanced Customization',
        description: 'Customize colors, backgrounds, badges, and more for the landing page. Each group can be expanded to fine-tune the design.',
        side: 'right',
      },
      type: 'info',
      waitForEl: true,
    },

    // ── Step 19: Save Landing Page Settings ──
    {
      id: 'save-template',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="save-template"]',
      popover: {
        title: 'Save Landing Page Settings',
        description: 'Click to save your template, display fields, and section order changes.',
        side: 'top',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 20: Switch to Warranty tab ──
    {
      id: 'tab-warranty',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="tab-warranty"]',
      popover: {
        title: 'Warranty Settings',
        description: 'Configure warranty registration for this product. Customers can register warranties after scanning.',
        side: 'bottom',
      },
      type: 'auto-click',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('tab_switch', 'warranty')
        await delay(300)
      },
    },

    // ── Step 21: Toggle warranty on ──
    {
      id: 'warranty-toggle',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="warranty-toggle"]',
      popover: {
        title: 'Enable Warranty',
        description: 'Turn on warranty registration so customers can activate warranty after scanning the QR code.',
        side: 'left',
      },
      type: 'auto-click',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('warranty_toggle', true)
        await delay(300)
      },
    },

    // ── Step 22: Set warranty period ──
    {
      id: 'warranty-period',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="warranty-period"]',
      popover: {
        title: 'Warranty Period',
        description: 'Set how many months the warranty lasts after purchase. We\'ll set it to 24 months.',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('warranty_period', 24)
      },
    },

    // ── Step 23: Set max registration days ──
    {
      id: 'warranty-reg-days',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="warranty-reg-days"]',
      popover: {
        title: 'Max Registration Days',
        description: 'Set the window (in days) after purchase within which warranty can be registered. We\'ll set 90 days.',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('warranty_reg_days', 90)
      },
    },

    // ── Step 24: Select warranty template ──
    {
      id: 'warranty-template-select',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="warranty-template-select"]',
      popover: {
        title: 'Warranty Page Template',
        description: 'Choose a template for the warranty registration page. We\'ll select "Default Warranty Template".',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('warranty_template', 'Default Warranty Template')
      },
    },

    // ── Step 25: Warranty preview - Form tab ──
    {
      id: 'warranty-preview-form',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="warranty-preview-form"]',
      popover: {
        title: 'Warranty Form Preview',
        description: 'This preview shows what customers see when registering warranty. The form collects their details.',
        side: 'left',
      },
      type: 'auto-click',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        await autoClick('[data-tour="warranty-preview-form"]')
        await delay(200)
      },
    },

    // ── Step 26: Warranty preview - Success tab ──
    {
      id: 'warranty-preview-success',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="warranty-preview-success"]',
      popover: {
        title: 'Success Page Preview',
        description: 'This is what customers see after successfully registering their warranty.',
        side: 'left',
      },
      type: 'auto-click',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        await autoClick('[data-tour="warranty-preview-success"]')
        await delay(200)
      },
    },

    // ── Step 27: Warranty preview - Error tab (FINAL) ──
    {
      id: 'warranty-preview-error',
      expectedRoute: '/tenant/products/:id',
      selector: '[data-tour="warranty-preview-error"]',
      popover: {
        title: 'Error Page Preview',
        description: 'This appears when a customer tries to register warranty on a product that\'s already been activated. You\'ve completed the Product Settings tour!',
        side: 'left',
      },
      type: 'info',
      waitForEl: true,
      beforeHighlight: async () => {
        await autoClick('[data-tour="warranty-preview-error"]')
        await delay(200)
      },
    },
  ],
}
