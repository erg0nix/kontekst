<script setup lang="ts">
import { RouterLink } from 'vue-router'
import DataTable from '../components/DataTable.vue'
import { mockSessions, getRunsForSession, deleteSession } from '../mock/data'

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString()
}

function handleDelete(id: string) {
  if (confirm(`Delete session ${id}?`)) {
    deleteSession(id)
  }
}
</script>

<template>
  <div>
    <h1 class="mb-6 text-2xl font-bold">Sessions</h1>

    <DataTable
      :columns="['Session ID', 'Agent', 'Messages', 'Runs', 'Created', 'Last Active']"
    >
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
        <td class="px-4 py-2 text-right">
          <button
            @click="handleDelete(session.id)"
            class="rounded border border-red-500 px-2 py-1 text-xs text-red-500 hover:bg-red-50 dark:hover:bg-red-950"
          >
            Delete
          </button>
        </td>
      </tr>
    </DataTable>
  </div>
</template>
