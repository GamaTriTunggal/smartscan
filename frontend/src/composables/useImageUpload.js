import { ref, computed } from 'vue'
import { useAPI } from '@/composables/useAPI'

// Image upload constraints
const MAX_FILE_SIZE = 2 * 1024 * 1024 // 2MB
const MIN_DIMENSION = 720 // Smallest dimension must be at least 720px
const ALLOWED_TYPES = ['image/jpeg', 'image/png', 'image/webp']

export function useImageUpload() {
  const { api } = useAPI()

  const isUploading = ref(false)
  const uploadProgress = ref(0)
  const uploadError = ref(null)
  const uploadedUrl = ref(null)

  const isValid = computed(() => !uploadError.value && !isUploading.value)

  /**
   * Validate file before upload
   * @param {File} file - The file to validate
   * @returns {Promise<{valid: boolean, error?: string}>}
   */
  const validateFile = async (file) => {
    // Check file type
    if (!ALLOWED_TYPES.includes(file.type)) {
      return {
        valid: false,
        error: `Invalid file type. Allowed: JPEG, PNG, WebP`
      }
    }

    // Check file size
    if (file.size > MAX_FILE_SIZE) {
      return {
        valid: false,
        error: `File size exceeds ${MAX_FILE_SIZE / (1024 * 1024)}MB limit`
      }
    }

    // Check dimensions - smallest dimension must be at least MIN_DIMENSION
    try {
      const dimensions = await getImageDimensions(file)
      const smallerDim = Math.min(dimensions.width, dimensions.height)
      if (smallerDim < MIN_DIMENSION) {
        return {
          valid: false,
          error: `Image smallest dimension must be at least ${MIN_DIMENSION}px. Current: ${smallerDim}px (${dimensions.width}x${dimensions.height})`
        }
      }
    } catch (err) {
      return {
        valid: false,
        error: 'Failed to read image dimensions'
      }
    }

    return { valid: true }
  }

  /**
   * Get image dimensions from file
   * @param {File} file - The image file
   * @returns {Promise<{width: number, height: number}>}
   */
  const getImageDimensions = (file) => {
    return new Promise((resolve, reject) => {
      const img = new Image()
      img.onload = () => {
        resolve({ width: img.width, height: img.height })
        URL.revokeObjectURL(img.src)
      }
      img.onerror = () => {
        reject(new Error('Failed to load image'))
        URL.revokeObjectURL(img.src)
      }
      img.src = URL.createObjectURL(file)
    })
  }

  /**
   * Upload background image for tenant
   * @param {File} file - The file to upload
   * @param {string} type - Upload type: 'landing' (only supported type)
   * @returns {Promise<{success: boolean, url?: string, error?: string}>}
   */
  const uploadBackground = async (file, type = 'landing') => {
    uploadError.value = null
    uploadedUrl.value = null
    uploadProgress.value = 0

    // Validate file first
    const validation = await validateFile(file)
    if (!validation.valid) {
      uploadError.value = validation.error
      return { success: false, error: validation.error }
    }

    isUploading.value = true

    try {
      const formData = new FormData()
      formData.append('image', file)
      formData.append('type', type)

      const response = await api.post('/tenant/uploads/background', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        },
        onUploadProgress: (progressEvent) => {
          // total is undefined when the browser can't determine upload size
          if (progressEvent.total) {
            uploadProgress.value = Math.round(
              (progressEvent.loaded * 100) / progressEvent.total
            )
          }
        }
      })

      if (response.data.success) {
        uploadedUrl.value = response.data.data.url
        return { success: true, url: response.data.data.url }
      } else {
        throw new Error(response.data.message || 'Upload failed')
      }
    } catch (err) {
      const errorMessage = err.response?.data?.message || err.message || 'Upload failed'
      uploadError.value = errorMessage
      return { success: false, error: errorMessage }
    } finally {
      isUploading.value = false
    }
  }

  /**
   * Delete uploaded background image
   * @param {string} filename - The filename to delete
   * @param {string} type - Upload type: 'landing' (only supported type)
   * @returns {Promise<{success: boolean, error?: string}>}
   */
  const deleteBackground = async (filename, type = 'landing') => {
    try {
      const response = await api.delete(`/tenant/uploads/background/${filename}?type=${type}`)
      if (response.data.success) {
        uploadedUrl.value = null
        return { success: true }
      }
      throw new Error(response.data.message || 'Delete failed')
    } catch (err) {
      const errorMessage = err.response?.data?.message || err.message || 'Delete failed'
      return { success: false, error: errorMessage }
    }
  }

  /**
   * Upload preset background image (admin only)
   * @param {File} file - The file to upload
   * @returns {Promise<{success: boolean, url?: string, error?: string}>}
   */
  const uploadPresetBackground = async (file) => {
    uploadError.value = null
    uploadedUrl.value = null
    uploadProgress.value = 0

    // Validate file first
    const validation = await validateFile(file)
    if (!validation.valid) {
      uploadError.value = validation.error
      return { success: false, error: validation.error }
    }

    isUploading.value = true

    try {
      const formData = new FormData()
      formData.append('image', file)

      const response = await api.post('/tenant/theme-presets/upload', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        },
        onUploadProgress: (progressEvent) => {
          // total is undefined when the browser can't determine upload size
          if (progressEvent.total) {
            uploadProgress.value = Math.round(
              (progressEvent.loaded * 100) / progressEvent.total
            )
          }
        }
      })

      if (response.data.success) {
        uploadedUrl.value = response.data.data.url
        return { success: true, url: response.data.data.url }
      } else {
        throw new Error(response.data.message || 'Upload failed')
      }
    } catch (err) {
      const errorMessage = err.response?.data?.message || err.message || 'Upload failed'
      uploadError.value = errorMessage
      return { success: false, error: errorMessage }
    } finally {
      isUploading.value = false
    }
  }

  /**
   * Reset upload state
   */
  const reset = () => {
    isUploading.value = false
    uploadProgress.value = 0
    uploadError.value = null
    uploadedUrl.value = null
  }

  return {
    // State
    isUploading,
    uploadProgress,
    uploadError,
    uploadedUrl,
    isValid,

    // Methods
    validateFile,
    uploadBackground,
    deleteBackground,
    uploadPresetBackground,
    reset,

    // Constants
    MAX_FILE_SIZE,
    MIN_DIMENSION,
    ALLOWED_TYPES
  }
}
