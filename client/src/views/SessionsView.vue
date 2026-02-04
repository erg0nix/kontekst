<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { mockSessions, getRunsForSession } from '../mock/data'

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString()
}
</script>

<template>
  <div>
    <h1 class="mb-6 text-2xl font-bold">Sessions</h1>

    <div class="rounded border border-black dark:border-white">
      <table class="w-full text-sm">
        <thead class="border-b border-black bg-gray-50 dark:border-white dark:bg-gray-900">
          <tr>
            <th class="px-4 py-2 text-left font-medium">Session ID</th>
            <th class="px-4 py-2 text-left font-medium">Agent</th>
            <th class="px-4 py-2 text-left font-medium">Messages</th>
            <th class="px-4 py-2 text-left font-medium">Runs</th>
            <th class="px-4 py-2 text-left font-medium">Created</th>
            <th class="px-4 py-2 text-left font-medium">Last Active</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="session in mockSessions"
            :key="session.id"
            class="border-b border-gray-200 last:border-b-0 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-900"
          >
            <td class="px-4 py-2">
              <RouterLink
                :to="`/sessions/${session.id}`"
                class="font-mono text-blue-500 hover:underline"
              >
                {{ session.id }}
              </RouterLink>
            </td>
            <td class="px-4 py-2">
              <RouterLink
                :to="`/agents/${session.agentName}`"
                class="text-blue-500 hover:underline"
              >
                {{ session.agentName }}
              </RouterLink>
            </td>
            <td class="px-4 py-2">{{ session.messages.length }}</td>
            <td class="px-4 py-2">{{ getRunsForSession(session.id).length }}</td>
            <td class="px-4 py-2">{{ formatDate(session.createdAt) }}</td>
            <td class="px-4 py-2">{{ formatDate(session.lastActiveAt) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
