<script setup lang="ts">
import { RouterLink } from 'vue-router'
import {
  mockDaemonStatus,
  mockAgentSummaries,
  mockSessions,
  mockSkills,
  mockRunRecords,
} from '../mock/data'

function formatUptime(seconds: number): string {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  return `${hours}h ${minutes}m`
}

const recentRuns = mockRunRecords.slice(0, 5)
</script>

<template>
  <div>
    <h1 class="mb-6 text-2xl font-bold">Dashboard</h1>

    <div class="mb-8 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
      <div class="rounded border border-black p-4 dark:border-white">
        <div class="text-sm text-gray-500 dark:text-gray-400">Agents</div>
        <div class="text-3xl font-bold">{{ mockAgentSummaries.length }}</div>
        <RouterLink
          to="/agents"
          class="text-sm text-blue-500 hover:underline"
        >
          View all
        </RouterLink>
      </div>

      <div class="rounded border border-black p-4 dark:border-white">
        <div class="text-sm text-gray-500 dark:text-gray-400">Sessions</div>
        <div class="text-3xl font-bold">{{ mockSessions.length }}</div>
        <RouterLink
          to="/sessions"
          class="text-sm text-blue-500 hover:underline"
        >
          View all
        </RouterLink>
      </div>

      <div class="rounded border border-black p-4 dark:border-white">
        <div class="text-sm text-gray-500 dark:text-gray-400">Skills</div>
        <div class="text-3xl font-bold">{{ mockSkills.length }}</div>
        <RouterLink
          to="/skills"
          class="text-sm text-blue-500 hover:underline"
        >
          View all
        </RouterLink>
      </div>

      <div class="rounded border border-black p-4 dark:border-white">
        <div class="text-sm text-gray-500 dark:text-gray-400">Uptime</div>
        <div class="text-3xl font-bold">
          {{ formatUptime(mockDaemonStatus.uptimeSeconds) }}
        </div>
        <div class="text-sm text-gray-500">
          Since {{ new Date(mockDaemonStatus.startedAtRfc3339).toLocaleTimeString() }}
        </div>
      </div>
    </div>

    <div class="mb-8">
      <h2 class="mb-4 text-xl font-bold">System Status</h2>
      <div class="rounded border border-black dark:border-white">
        <table class="w-full text-sm">
          <tbody>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">gRPC Bind</td>
              <td class="px-4 py-2 font-mono">{{ mockDaemonStatus.bind }}</td>
            </tr>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">LLM Endpoint</td>
              <td class="px-4 py-2 font-mono">{{ mockDaemonStatus.endpoint }}</td>
            </tr>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">Model Directory</td>
              <td class="px-4 py-2 font-mono">{{ mockDaemonStatus.modelDir }}</td>
            </tr>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">Data Directory</td>
              <td class="px-4 py-2 font-mono">{{ mockDaemonStatus.dataDir }}</td>
            </tr>
            <tr>
              <td class="px-4 py-2 font-medium">LLama Server PID</td>
              <td class="px-4 py-2 font-mono">{{ mockDaemonStatus.llamaServerPid }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div>
      <h2 class="mb-4 text-xl font-bold">Recent Runs</h2>
      <div class="rounded border border-black dark:border-white">
        <table class="w-full text-sm">
          <thead class="border-b border-black bg-gray-50 dark:border-white dark:bg-gray-900">
            <tr>
              <th class="px-4 py-2 text-left font-medium">Run ID</th>
              <th class="px-4 py-2 text-left font-medium">Session</th>
              <th class="px-4 py-2 text-left font-medium">Status</th>
              <th class="px-4 py-2 text-left font-medium">Time</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="run in recentRuns"
              :key="run.runId"
              class="border-b border-gray-200 last:border-b-0 dark:border-gray-700"
            >
              <td class="px-4 py-2 font-mono">{{ run.runId }}</td>
              <td class="px-4 py-2">
                <RouterLink
                  :to="`/sessions/${run.sessionId}`"
                  class="text-blue-500 hover:underline"
                >
                  {{ run.sessionId }}
                </RouterLink>
              </td>
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
              <td class="px-4 py-2">
                {{ new Date(run.timestamp).toLocaleString() }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
