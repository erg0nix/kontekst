<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { mockSkills } from '../mock/data'
</script>

<template>
  <div>
    <h1 class="mb-6 text-2xl font-bold">Skills</h1>

    <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      <RouterLink
        v-for="skill in mockSkills"
        :key="skill.name"
        :to="`/skills/${skill.name}`"
        class="rounded border border-black p-4 transition-colors hover:bg-gray-50 dark:border-white dark:hover:bg-gray-900"
      >
        <div class="mb-2 flex items-center justify-between">
          <h2 class="text-lg font-bold">{{ skill.name }}</h2>
          <div class="flex gap-1">
            <span
              v-if="skill.userInvocable"
              class="rounded bg-green-100 px-2 py-0.5 text-xs text-green-800 dark:bg-green-900 dark:text-green-200"
            >
              User
            </span>
            <span
              v-if="!skill.disableModelInvocation"
              class="rounded bg-blue-100 px-2 py-0.5 text-xs text-blue-800 dark:bg-blue-900 dark:text-blue-200"
            >
              Model
            </span>
          </div>
        </div>

        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ skill.description }}
        </p>
      </RouterLink>
    </div>
  </div>
</template>
