<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { mockAgentSummaries } from '../mock/data'
</script>

<template>
  <div>
    <h1 class="mb-6 text-2xl font-bold">Agents</h1>

    <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      <RouterLink
        v-for="agent in mockAgentSummaries"
        :key="agent.name"
        :to="`/agents/${agent.name}`"
        class="rounded border border-black p-4 transition-colors hover:bg-gray-50 dark:border-white dark:hover:bg-gray-900"
      >
        <div class="mb-2 flex items-center justify-between">
          <h2 class="text-lg font-bold">{{ agent.displayName }}</h2>
          <span class="font-mono text-sm text-gray-500">{{ agent.name }}</span>
        </div>

        <div class="flex gap-2">
          <span
            v-if="agent.hasPrompt"
            class="rounded bg-green-100 px-2 py-0.5 text-xs text-green-800 dark:bg-green-900 dark:text-green-200"
          >
            Prompt
          </span>
          <span
            v-if="agent.hasConfig"
            class="rounded bg-blue-100 px-2 py-0.5 text-xs text-blue-800 dark:bg-blue-900 dark:text-blue-200"
          >
            Config
          </span>
        </div>
      </RouterLink>
    </div>
  </div>
</template>
