<script setup lang="ts">
defineProps<{
  label: string
  status: 'healthy' | 'unhealthy' | 'unknown'
}>()
</script>

<template>
  <div class="flex items-center gap-2 text-sm">
    <span
      class="h-2 w-2 rounded-full"
      :class="{
        'bg-green-500': status === 'healthy',
        'bg-red-500': status === 'unhealthy',
        'bg-gray-400': status === 'unknown',
      }"
    ></span>
    <span class="text-black dark:text-white">{{ label }}</span>
  </div>
</template>
