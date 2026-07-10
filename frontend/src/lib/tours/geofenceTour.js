import {
  delay,
  tourSetValue,
  getNextGeofenceBatchName,
} from './tourUtils.js'

export const geofenceTour = {
  id: 'geofence',
  name: 'Using Geofence',
  description: 'Learn how to set up distribution zone geofencing for grey market detection.',
  estimatedMinutes: 5,
  requires: 'create-dynamic-product',
  steps: [
    // ── Step 0: Sidebar → Dynamic QR ──
    {
      id: 'sidebar-dynamic-qr',
      expectedRoute: null,
      selector: '[data-tour="sidebar-dynamic-qr"]',
      popover: {
        title: 'Dynamic QR Products',
        description: 'Navigate to the Dynamic QR page. Click this menu item.',
        side: 'right',
        align: 'center',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 1: Search product ──
    {
      id: 'search-product',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="search-product"]',
      popover: {
        title: 'Find Your Product',
        description: 'Search for "Product Example 1" to find the product you want to create a geofenced batch for.',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('search_query', 'Product Example 1')
      },
    },

    // ── Step 2: Click "+ New Batch" ──
    {
      id: 'new-batch-btn',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="new-batch-btn"]',
      popover: {
        title: 'Create a New Batch',
        description: 'Click this button to create a new QR batch with geofencing enabled.',
        side: 'bottom',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 3: Auto-fill Batch Name ──
    {
      id: 'batch-name-input',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="batch-name-input"]',
      popover: {
        title: 'Batch Name',
        description: 'Enter a descriptive name for this batch. We\'ll name it to identify it as a geofenced batch.',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      insideModal: true,
      beforeHighlight: async ({ setTourData }) => {
        const name = await getNextGeofenceBatchName('Geofence Example')
        setTourData('geofenceBatchName', name)
        tourSetValue('batch_name', name)
      },
    },

    // ── Step 4: Toggle Geofence ON ──
    {
      id: 'geofence-toggle',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="geofence-toggle"]',
      popover: {
        title: 'Enable Distribution Zone',
        description: 'Turn on the Distribution Zone (Geofence) feature. Scans outside the defined zone will be flagged as potential grey market activity.',
        side: 'right',
      },
      type: 'auto-click',
      waitForNext: true,
      waitForEl: true,
      insideModal: true,
      beforeHighlight: async () => {
        tourSetValue('geofence_enabled', true)
        await delay(300)
      },
    },

    // ── Step 5: Zone Template info (skip if no templates exist) ──
    {
      id: 'geofence-zone-template',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="geofence-zone-template"]',
      popover: {
        title: 'Zone Templates',
        description: 'If you have saved zone templates, you can quickly load them here instead of setting up coordinates manually. For now, we\'ll set up a zone from scratch.',
        side: 'bottom',
      },
      type: 'info',
      skipIfNoElement: true,
      insideModal: true,
    },

    // ── Step 6: Search Semarang ──
    {
      id: 'geofence-search-input',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="geofence-search-input"]',
      popover: {
        title: 'Search Location',
        description: 'Search for a city or location to center your distribution zone. We\'ll search for "Semarang" and select it from the results.',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      insideModal: true,
      beforeHighlight: async () => {
        tourSetValue('geofence_search', 'Semarang')
        // Wait for Nominatim API to return results and auto-select
        await delay(3000)
      },
    },

    // ── Step 7: Set Radius 60 km ──
    {
      id: 'geofence-radius',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="geofence-radius"]',
      popover: {
        title: 'Distribution Radius',
        description: 'Set the radius of your distribution zone to 60 km. Any QR scan outside this radius will be recorded as a geofence violation.',
        side: 'top',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      insideModal: true,
      beforeHighlight: async () => {
        tourSetValue('geofence_radius', 60)
      },
    },

    // ── Step 8: Set Zone Label ──
    {
      id: 'geofence-zone-label',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="geofence-zone-label"]',
      popover: {
        title: 'Zone Label',
        description: 'Give your zone a descriptive label. This helps identify the distribution area in reports.',
        side: 'top',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      insideModal: true,
      beforeHighlight: async () => {
        tourSetValue('geofence_label', 'Semarang')
      },
    },

    // ── Step 9: Click Create Batch ──
    {
      id: 'create-batch-btn',
      expectedRoute: '/tenant/products/dynamic',
      selector: '[data-tour="create-batch-btn"]',
      popover: {
        title: 'Create the Batch',
        description: 'Click "Create Batch" to generate QR codes with geofencing enabled. You\'ll be redirected to the batch history page.',
        side: 'top',
      },
      type: 'interactive',
      waitForEl: true,
      insideModal: true,
    },

    // ── Step 10: Search batch on ProductBatchHistoryPage ──
    {
      id: 'batch-search-input',
      expectedRoute: '/tenant/products/:productId/batches',
      selector: '[data-tour="batch-search-input"]',
      popover: {
        title: 'Find Your Batch',
        description: 'Search for the geofenced batch you just created to view its details.',
        side: 'bottom',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async ({ getTourData }) => {
        const batchName = getTourData('geofenceBatchName') || 'Geofence Example'
        tourSetValue('batch_search_query', batchName)
      },
    },

    // ── Step 11: Click Insights ──
    {
      id: 'batch-insights-btn',
      expectedRoute: '/tenant/products/:productId/batches',
      selector: '[data-tour="batch-insights-btn"]',
      popover: {
        title: 'View Batch Insights',
        description: 'Click "Insights" to view the batch detail page where you can see the scan distribution map and edit the geofence zone.',
        side: 'left',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 12: Click "Edit Geofence Zone" ──
    {
      id: 'edit-geofence-btn',
      expectedRoute: '/tenant/qr-batches/:id',
      selector: '[data-tour="edit-geofence-btn"]',
      popover: {
        title: 'Edit Geofence Zone',
        description: 'Click here to modify the distribution zone settings for this batch. You can adjust the center, radius, and label.',
        side: 'bottom',
      },
      type: 'interactive',
      waitForEl: true,
    },

    // ── Step 13: Change Radius to 70 km ──
    {
      id: 'geofence-edit-radius',
      expectedRoute: '/tenant/qr-batches/:id',
      selector: '[data-tour="geofence-edit-radius"]',
      popover: {
        title: 'Adjust the Radius',
        description: 'Let\'s expand the distribution zone to 70 km. This gives more coverage area for authorized distribution.',
        side: 'top',
      },
      type: 'auto-fill',
      waitForNext: true,
      waitForEl: true,
      beforeHighlight: async () => {
        tourSetValue('geofence_edit_radius', 70)
      },
    },

    // ── Step 14: Click Save ──
    {
      id: 'save-geofence-btn',
      expectedRoute: '/tenant/qr-batches/:id',
      selector: '[data-tour="save-geofence-btn"]',
      popover: {
        title: 'Save Changes',
        description: 'Click "Save" to apply the updated geofence zone. Future scans will be checked against this new zone.',
        side: 'top',
      },
      type: 'interactive',
      waitForEl: true,
    },
  ],
}
