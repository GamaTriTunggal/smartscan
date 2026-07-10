import { createDynamicProductTour } from './createDynamicProduct.js'
import { productSettingsTour } from './productSettings.js'
import { createLandingTemplateTour } from './createLandingTemplate.js'
import { geofenceTour } from './geofenceTour.js'

export const allTours = [
  createDynamicProductTour,
  productSettingsTour,
  createLandingTemplateTour,
  geofenceTour,
]
