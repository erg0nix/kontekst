<script setup lang="ts">
defineProps<{
  columns: string[]
}>()

defineEmits<{
  delete: [key: string]
}>()
</script>

<template>
  <div class="rounded border border-black dark:border-white">
    <table class="w-full text-sm">
      <thead class="border-b border-black bg-gray-50 dark:border-white dark:bg-gray-900">
        <tr>
          <th
            v-for="col in columns"
            :key="col"
            class="px-4 py-2 text-left font-medium"
          >
            {{ col }}
          </th>
          <th class="px-4 py-2 text-right font-medium"></th>
        </tr>
      </thead>
      <tbody>
        <slot />
      </tbody>
    </table>
  </div>
</template>
