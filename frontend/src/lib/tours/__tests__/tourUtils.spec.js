import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { getNextTutorialProductName } from '../tourUtils.js'

describe('getNextTutorialProductName', () => {
  let fetchSpy

  beforeEach(() => {
    fetchSpy = vi.spyOn(globalThis, 'fetch')
  })

  afterEach(() => {
    fetchSpy.mockRestore()
  })

  function mockFetchProducts(productNames) {
    fetchSpy.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        data: {
          products: productNames.map((name, i) => ({
            id: `uuid-${i}`,
            product_name: name,
          })),
        },
      }),
    })
  }

  it('returns "Product Example 1" when no products exist', async () => {
    mockFetchProducts([])
    const name = await getNextTutorialProductName('Product Example')
    expect(name).toBe('Product Example 1')
  })

  it('returns "Product Example 2" when "Product Example 1" exists', async () => {
    mockFetchProducts(['Product Example 1'])
    const name = await getNextTutorialProductName('Product Example')
    expect(name).toBe('Product Example 2')
  })

  it('finds the max number and increments', async () => {
    mockFetchProducts(['Product Example 1', 'Product Example 3', 'Product Example 5'])
    const name = await getNextTutorialProductName('Product Example')
    expect(name).toBe('Product Example 6')
  })

  it('handles bare "Product Example" without number as 1', async () => {
    // "Product Example" (no number) should be treated as 1
    mockFetchProducts(['Product Example'])
    const name = await getNextTutorialProductName('Product Example')
    expect(name).toBe('Product Example 2')
  })

  it('ignores unrelated product names', async () => {
    mockFetchProducts(['Product Example 2', 'Some Other Product', 'Product Example ABC'])
    const name = await getNextTutorialProductName('Product Example')
    expect(name).toBe('Product Example 3')
  })

  it('returns fallback on fetch error', async () => {
    fetchSpy.mockRejectedValueOnce(new Error('Network error'))
    const name = await getNextTutorialProductName('Product Example')
    expect(name).toBe('Product Example 1')
  })

  it('returns fallback on non-ok response', async () => {
    fetchSpy.mockResolvedValueOnce({ ok: false })
    const name = await getNextTutorialProductName('Product Example')
    expect(name).toBe('Product Example 1')
  })

  it('works with custom base name', async () => {
    mockFetchProducts(['Test Product 1', 'Test Product 2'])
    const name = await getNextTutorialProductName('Test Product')
    expect(name).toBe('Test Product 3')
  })

  it('calls API with correct URL and credentials', async () => {
    mockFetchProducts([])
    await getNextTutorialProductName('Product Example')
    expect(fetchSpy).toHaveBeenCalledWith(
      expect.stringContaining('/tenant/products?search=Product%20Example&limit=100'),
      { credentials: 'include' }
    )
  })
})
