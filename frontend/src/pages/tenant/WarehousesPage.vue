<script setup>
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-900">Warehouses</h1>
      <Button v-if="!['SettingsPage', 'SubscriptionPage'].includes('WarehousesPage')">Add New</Button>
    </div>
    <Card class="p-6">
      <p class="text-gray-500">Content coming soon.</p>
    </Card>
  </div>
</template>
