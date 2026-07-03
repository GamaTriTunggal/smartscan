<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAPI } from '@/composables/useAPI'
import { useDateTime } from '@/composables/useDateTime'
import { useTour } from '@/composables/useTour'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'

const router = useRouter()
const route = useRoute()
const { get, del, post } = useAPI()
const { formatDate } = useDateTime()

const loading = ref(true)
const templates = ref([])
const pagination = ref({ page: 1, limit: 50, total: 0, total_page: 0 })
const typeFilter = ref('validation')
const statusFilter = ref('active')
const tenantDefaults = ref({
  default_validation_template_id: null,
  default_warranty_template_id: null
})
const settingDefault = ref(null) // Track which template is being set as default

// Page titles based on template type
const pageTitles = {
  validation: {
    title: 'Template - Landing',
    description: 'Design your product validation landing page that customers see when scanning QR codes',
    addLabel: '+ Add Template'
  },
  warranty: {
    title: 'Template - Warranty',
    description: 'Design warranty registration forms for your products',
    addLabel: '+ Add Template'
  },
}

const currentPageInfo = computed(() => {
  return pageTitles[typeFilter.value] || pageTitles.validation
})

const fetchTenantDefaults = async () => {
  try {
    const response = await get('/tenant/templates/defaults')
    if (response.success && response.data) {
      tenantDefaults.value = response.data
    }
  } catch (error) {
    console.error('Failed to fetch tenant defaults:', error)
  }
}

const fetchTemplates = async () => {
  try {
    loading.value = true
    const params = {
      page: pagination.value.page,
      limit: pagination.value.limit,
      status: statusFilter.value,
      type: typeFilter.value,
    }

    const response = await get('/tenant/templates', params)
    if (response.success && response.data) {
      templates.value = response.data.templates || []
      pagination.value = response.data.pagination
    }
  } catch (error) {
    console.error('Failed to fetch templates:', error)
  } finally {
    loading.value = false
  }
}

// Check if a template is the tenant default for its type
const isDefault = (template) => {
  if (template.template_type === 'validation') {
    return template.id === tenantDefaults.value.default_validation_template_id
  } else if (template.template_type === 'warranty') {
    return template.id === tenantDefaults.value.default_warranty_template_id
  }
  return false
}

// Set a template as the tenant default
const setAsDefault = async (template) => {

  try {
    settingDefault.value = template.id
    const response = await post(`/tenant/templates/${template.id}/set-default`)
    if (response.success) {
      alert(`"${template.template_name}" is now the default ${template.template_type} template`)
      await fetchTenantDefaults()
    } else {
      alert(response.message || 'Failed to set template as default')
    }
  } catch (error) {
    alert('Failed to set template as default')
    console.error('Failed to set template as default:', error)
  } finally {
    settingDefault.value = null
  }
}

const deleteTemplate = async (template) => {
  if (!confirm(`Are you sure you want to delete "${template.template_name}"?`)) return

  try {
    const response = await del(`/tenant/templates/${template.id}`)
    if (response.success) {
      fetchTemplates()
    }
  } catch (error) {
    console.error('Failed to delete template:', error)
  }
}

const editTemplate = (id) => {
  router.push(`/tenant/templates/${id}`)
}

const createTemplate = (type) => {
  router.push(`/tenant/templates/new?type=${type}`)
}

// Watch for route query changes
watch(() => route.query.type, (newType) => {
  if (newType && typeof newType === 'string' && ['validation', 'warranty'].includes(newType)) {
    typeFilter.value = newType
    fetchTemplates()
  }
}, { immediate: false })

const { resumeIfActive } = useTour()

onMounted(async () => {
  // Resume tour if active (cross-page navigation)
  resumeIfActive()

  // Read type from query param
  const queryType = route.query.type
  if (queryType && ['validation', 'warranty'].includes(queryType)) {
    typeFilter.value = queryType
  }

  // Fetch templates and tenant defaults in parallel
  await Promise.all([fetchTemplates(), fetchTenantDefaults()])
})
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ currentPageInfo.title }}</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
          {{ currentPageInfo.description }}
        </p>
      </div>
      <Button data-tour="add-template-btn" @click="createTemplate(typeFilter)">
        {{ currentPageInfo.addLabel }}
      </Button>
    </div>

    <!-- Filters -->
    <div class="flex gap-4 mb-6">
      <select
        v-model="statusFilter"
        @change="fetchTemplates"
        class="px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-[#27272a]"
      >
        <option value="active">Active</option>
        <option value="all">All</option>
        <option value="deleted">Deleted</option>
      </select>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-zinc-500"></div>
    </div>

    <!-- Template Grid -->
    <div v-else-if="templates.length > 0" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <Card
        v-for="template in templates"
        :key="template.id"
        :class="[
          'p-4 transition-colors',
          template.deleted_at
            ? 'opacity-60 bg-gray-50 dark:bg-gray-800/50 border-dashed'
            : 'hover:border-zinc-300 dark:hover:border-zinc-600 cursor-pointer'
        ]"
        @click="!template.deleted_at && editTemplate(template.id)"
      >
        <div class="flex items-start justify-between mb-2">
          <div class="flex items-center gap-2 flex-1 min-w-0">
            <h3
              :class="[
                'font-semibold truncate',
                template.deleted_at
                  ? 'text-gray-500 dark:text-gray-400'
                  : 'text-gray-900 dark:text-white'
              ]"
            >
              {{ template.template_name }}
            </h3>
            <!-- Default badge -->
            <span
              v-if="isDefault(template) && !template.deleted_at"
              class="px-2 py-0.5 text-xs rounded-full bg-zinc-100 text-zinc-800 dark:bg-zinc-900/30 dark:text-zinc-400 flex-shrink-0"
            >
              Default
            </span>
          </div>
          <span
            :class="[
              'px-2 py-0.5 text-xs rounded-full flex-shrink-0 ml-2',
              template.deleted_at
                ? 'bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
                : template.is_active
                  ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                  : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
            ]"
          >
            {{ template.deleted_at ? 'Deleted' : template.is_active ? 'Active' : 'Inactive' }}
          </span>
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
          Updated: {{ formatDate(template.updated_at) }}
        </p>
        <div class="flex flex-wrap gap-2" @click.stop>
          <Button
            v-if="!template.deleted_at"
            variant="outline"
            size="sm"
            @click="editTemplate(template.id)"
          >
            Edit
          </Button>
          <!-- Set as Default button (only for validation/warranty, not already default) -->
          <Button
            v-if="!template.deleted_at && !isDefault(template)"
            variant="outline"
            size="sm"
            :disabled="settingDefault === template.id"
            @click="setAsDefault(template)"
          >
            {{ settingDefault === template.id ? 'Setting...' : 'Set as Default' }}
          </Button>
          <!-- Show Delete button unless it's the only template of its type -->
          <Button
            v-if="!template.deleted_at && templates.length > 1"
            variant="outline"
            size="sm"
            class="text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
            @click="deleteTemplate(template)"
          >
            Delete
          </Button>
          <span
            v-if="!template.deleted_at && templates.length === 1"
            class="text-xs text-gray-400 self-center"
            title="Cannot delete the last template"
          >
            (required)
          </span>
        </div>
      </Card>
    </div>

    <!-- Empty state -->
    <Card v-else class="p-6">
      <div class="text-center py-8">
        <svg class="w-16 h-16 text-gray-300 dark:text-gray-600 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
        </svg>
        <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">
          No templates found
        </h3>
        <p class="text-gray-500 dark:text-gray-400 mb-4">
          Create your first template.
        </p>
        <Button @click="createTemplate(typeFilter)">
          {{ currentPageInfo.addLabel }}
        </Button>
      </div>
    </Card>
  </div>
</template>
