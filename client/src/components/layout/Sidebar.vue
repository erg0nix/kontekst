<script setup lang="ts">
import { RouterLink } from 'vue-router'
import {
  mockAgentSummaries,
  mockSessions,
  mockSkills,
} from '../../mock/data'

import { computed } from 'vue'

const navItems = computed(() => [
  {
    name: 'Dashboard',
    path: '/',
    icon: 'home',
    count: null,
  },
  {
    name: 'Agents',
    path: '/agents',
    icon: 'agents',
    count: mockAgentSummaries.value.length,
  },
  {
    name: 'Sessions',
    path: '/sessions',
    icon: 'sessions',
    count: mockSessions.value.length,
  },
  {
    name: 'Skills',
    path: '/skills',
    icon: 'skills',
    count: mockSkills.value.length,
  },
])
</script>

<template>
  <aside
    class="fixed top-14 left-0 bottom-0 w-48 border-r border-gray-300 bg-white dark:border-gray-600 dark:bg-black"
  >
    <nav class="p-2">
      <RouterLink
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="mb-1 flex items-center justify-between rounded px-3 py-2 text-black hover:bg-gray-100 dark:text-white dark:hover:bg-gray-800"
        active-class="bg-gray-200 dark:bg-gray-700"
      >
        <div class="flex items-center gap-2">
          <svg
            v-if="item.icon === 'home'"
            xmlns="http://www.w3.org/2000/svg"
            class="h-5 w-5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
            />
          </svg>
          <svg
            v-else-if="item.icon === 'agents'"
            xmlns="http://www.w3.org/2000/svg"
            class="h-5 w-5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
            />
          </svg>
          <svg
            v-else-if="item.icon === 'sessions'"
            xmlns="http://www.w3.org/2000/svg"
            class="h-5 w-5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
            />
          </svg>
          <svg
            v-else-if="item.icon === 'skills'"
            xmlns="http://www.w3.org/2000/svg"
            class="h-5 w-5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M13 10V3L4 14h7v7l9-11h-7z"
            />
          </svg>
          <span>{{ item.name }}</span>
        </div>
        <span
          v-if="item.count !== null"
          class="rounded-full bg-gray-200 px-2 py-0.5 text-xs dark:bg-gray-700"
        >
          {{ item.count }}
        </span>
      </RouterLink>
    </nav>
  </aside>
</template>
