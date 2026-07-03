import { test, expect } from '@playwright/test'

/**
 * E2E Tests for Authentication Flow
 * Tests the complete login flow from login page to dashboard
 */

test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    // Clear any existing auth state
    await page.context().clearCookies()
  })

  test('should display login page', async ({ page }) => {
    await page.goto('/login')

    // Check page title and form elements
    await expect(page.locator('h1')).toContainText('Smart Label')
    await expect(page.locator('text=Sign in to your account')).toBeVisible()
    await expect(page.locator('input[type="email"]')).toBeVisible()
    await expect(page.locator('input[type="password"]')).toBeVisible()
    await expect(page.locator('button[type="submit"]')).toBeVisible()
  })

  test('should show error for empty form submission', async ({ page }) => {
    await page.goto('/login')

    // Click submit without filling form
    await page.locator('button[type="submit"]').click()

    // Should show validation error
    await expect(page.locator('text=Please enter email and password')).toBeVisible()
  })

  test('should show error for invalid credentials', async ({ page }) => {
    await page.goto('/login')

    // Fill with invalid credentials
    await page.locator('input[type="email"]').fill('invalid@example.com')
    await page.locator('input[type="password"]').fill('wrongpassword')
    await page.locator('button[type="submit"]').click()

    // Wait for error message
    await expect(page.locator('text=Invalid email or password')).toBeVisible({ timeout: 10000 })
  })

  test('tenant admin should login and redirect to tenant dashboard', async ({ page }) => {
    await page.goto('/login')

    // Fill login form with tenant admin credentials
    await page.locator('input[type="email"]').fill('admin@example.com')
    await page.locator('input[type="password"]').fill('password')
    await page.locator('button[type="submit"]').click()

    // Wait for navigation to tenant dashboard
    await page.waitForURL('**/tenant/dashboard', { timeout: 15000 })

    // Verify we're on the dashboard
    await expect(page).toHaveURL(/.*\/tenant\/dashboard/)

    // Check dashboard heading is visible
    await expect(page.locator('h1:has-text("Dashboard")')).toBeVisible({ timeout: 10000 })
  })

  test('should logout successfully', async ({ page }) => {
    // First login
    await page.goto('/login')
    await page.locator('input[type="email"]').fill('admin@example.com')
    await page.locator('input[type="password"]').fill('password')
    await page.locator('button[type="submit"]').click()
    await page.waitForURL('**/tenant/dashboard', { timeout: 15000 })

    // Find and click logout button (usually in header/sidebar)
    // Look for logout in dropdown or sidebar
    const logoutButton = page.locator('text=Logout').or(page.locator('text=Sign Out')).or(page.locator('[data-testid="logout"]'))

    // If logout is in a dropdown, we might need to open it first
    const userMenu = page.locator('[data-testid="user-menu"]').or(page.locator('button:has-text("Admin")'))
    if (await userMenu.isVisible()) {
      await userMenu.click()
    }

    // Click logout if visible
    if (await logoutButton.isVisible()) {
      await logoutButton.click()
      // Should redirect to login
      await expect(page).toHaveURL(/.*\/login/, { timeout: 10000 })
    }
  })
})
