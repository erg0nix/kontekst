<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import { getSession, getRunsForSession } from '../mock/data'

const route = useRoute()
const sessionId = computed(() => route.params.id as string)
const session = computed(() => getSession(sessionId.value))
const runs = computed(() => getRunsForSession(sessionId.value))

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString()
}

function getRoleColor(role: string): string {
  switch (role) {
    case 'system':
      return 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200'
    case 'user':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
    case 'assistant':
      return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
    case 'tool':
      return 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
  }
}
</script>

<template>
  <div>
    <div class="mb-6 flex items-center gap-2">
      <RouterLink to="/sessions" class="text-blue-500 hover:underline">
        Sessions
      </RouterLink>
      <span class="text-gray-400">/</span>
      <span class="font-mono">{{ sessionId }}</span>
    </div>

    <div v-if="session">
      <div class="mb-6 flex items-center justify-between">
        <h1 class="text-2xl font-bold">Session Details</h1>
        <RouterLink
          :to="`/agents/${session.agentName}`"
          class="text-blue-500 hover:underline"
        >
          Agent: {{ session.agentName }}
        </RouterLink>
      </div>

      <div class="mb-6 grid grid-cols-1 gap-4 md:grid-cols-3">
        <div class="rounded border border-black p-4 dark:border-white">
          <div class="text-sm text-gray-500 dark:text-gray-400">Created</div>
          <div class="font-mono">{{ formatDate(session.createdAt) }}</div>
        </div>
        <div class="rounded border border-black p-4 dark:border-white">
          <div class="text-sm text-gray-500 dark:text-gray-400">Last Active</div>
          <div class="font-mono">{{ formatDate(session.lastActiveAt) }}</div>
        </div>
        <div class="rounded border border-black p-4 dark:border-white">
          <div class="text-sm text-gray-500 dark:text-gray-400">Messages</div>
          <div class="text-2xl font-bold">{{ session.messages.length }}</div>
        </div>
      </div>

      <div v-if="runs.length > 0" class="mb-6">
        <h2 class="mb-4 text-xl font-bold">Runs</h2>
        <div class="rounded border border-black dark:border-white">
          <table class="w-full text-sm">
            <thead class="border-b border-black bg-gray-50 dark:border-white dark:bg-gray-900">
              <tr>
                <th class="px-4 py-2 text-left font-medium">Run ID</th>
                <th class="px-4 py-2 text-left font-medium">Status</th>
                <th class="px-4 py-2 text-left font-medium">Time</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="run in runs"
                :key="run.runId"
                class="border-b border-gray-200 last:border-b-0 dark:border-gray-700"
              >
                <td class="px-4 py-2 font-mono">{{ run.runId }}</td>
                <td class="px-4 py-2">
                  <span
                    class="rounded px-2 py-0.5 text-xs"
                    :class="{
                      'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200':
                        run.status === 'completed',
                      'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200':
                        run.status === 'started',
                      'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200':
                        run.status === 'failed',
                      'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200':
                        run.status === 'cancelled',
                    }"
                  >
                    {{ run.status }}
                  </span>
                </td>
                <td class="px-4 py-2">{{ formatDate(run.timestamp) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div>
        <h2 class="mb-4 text-xl font-bold">Messages</h2>
        <div class="space-y-4">
          <div
            v-for="(message, index) in session.messages"
            :key="index"
            class="rounded border border-black dark:border-white"
          >
            <div
              class="flex items-center justify-between border-b border-black px-4 py-2 dark:border-white"
            >
              <span
                class="rounded px-2 py-0.5 text-xs"
                :class="getRoleColor(message.role)"
              >
                {{ message.role }}
              </span>
              <span class="text-sm text-gray-500">
                {{ message.tokens }} tokens
              </span>
            </div>
            <div class="p-4">
              <pre class="whitespace-pre-wrap text-sm">{{ message.content }}</pre>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="rounded border border-red-500 bg-red-50 p-4 dark:bg-red-950">
      <p class="text-red-700 dark:text-red-300">
        Session "{{ sessionId }}" not found.
      </p>
    </div>
  </div>
</template>
