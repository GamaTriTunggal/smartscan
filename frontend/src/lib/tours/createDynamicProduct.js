import {
  getNextTutorialProductName,
  tourSetValue,
} from './tourUtils.js'

export const createDynamicProductTour = {
  id: 'create-dynamic-product',
  name: 'Create Your First Dynamic QR Product',
  description: 'Learn how to create a product and generate QR code batches — step by step.',
  estimatedMinutes: 3,
  steps: [
    // ── Step 0: Sidebar → Dynamic QR ──
    {
      id: 'sidebar-dynamic-qr',
      expectedRoute: null, // any tenant page
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

    // ── Step 1: Click "+ Add New Product" ──
    {
      id: 'add-product-btn',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="add-product-btn"]',
      popover: {
        title: 'Create a New Product',
        description: 'Click this button to open the product creation form.',
        side: 'bottom',
        align: 'end',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 2: Auto-fill Product Name ──
    {
      id: 'product-name-input',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="product-name-input"]',
      popover: {
        title: 'Product Name',
        description: 'Enter your product name. This is the only required field.',
        side: 'bottom',
      },
      type: 'auto-fill',
      insideModal: true,
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        const productName = await getNextTutorialProductName('Product Example')
        tourSetValue('product_name', productName)
      },
    },

    // ── Step 3: Auto-fill Description ──
    {
      id: 'product-description',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="product-description"]',
      popover: {
        title: 'Description',
        description: 'Add a description so customers understand your product.',
        side: 'bottom',
      },
      type: 'auto-fill',
      insideModal: true,
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('description', 'This is a product example with dynamic qr')
      },
    },

    // ── Step 4: Click "Create" product button ──
    {
      id: 'create-product-btn',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="create-product-btn"]',
      popover: {
        title: 'Create Product',
        description: 'Everything is set! Click "Create" to create your product.',
        side: 'top',
      },
      type: 'interactive',
      insideModal: true,
      waitForEl: true,
    },

    // ── Step 13: Click "New Batch" on first product ──
    {
      id: 'new-batch-btn',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="new-batch-btn"]',
      popover: {
        title: 'Create a QR Batch',
        description: 'Now create a batch of QR codes for your new product. Click "New Batch".',
        side: 'bottom',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 14: Auto-fill Batch Name ──
    {
      id: 'batch-name-input',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="batch-name-input"]',
      popover: {
        title: 'Batch Name',
        description: 'Give your batch a descriptive name, e.g. the production month.',
        side: 'bottom',
      },
      type: 'auto-fill',
      insideModal: true,
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('batch_name', 'March 2026')
      },
    },

    // ── Step 15: Auto-fill Production Date ──
    {
      id: 'batch-production-date',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="batch-production-date"]',
      popover: {
        title: 'Production Date',
        description: 'Set the production date. Customers will see this when they scan the QR code.',
        side: 'bottom',
      },
      type: 'auto-fill',
      insideModal: true,
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('production_date', '2026-03-07')
      },
    },

    // ── Step 16: Click "Create Batch" (final step) ──
    {
      id: 'create-batch-btn',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="create-batch-btn"]',
      popover: {
        title: 'Create Batch',
        description: 'Click "Create Batch" to generate QR codes.',
        side: 'top',
      },
      type: 'interactive',
      insideModal: true,
      waitForEl: true,
    },
  ],
}
