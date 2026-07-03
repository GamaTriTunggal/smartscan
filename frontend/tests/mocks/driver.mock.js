// Mock for driver.js — used by useTour.js
export function driver() {
  return {
    highlight: () => {},
    destroy: () => {},
    moveNext: () => {},
    movePrevious: () => {},
    isActive: () => false,
  }
}
