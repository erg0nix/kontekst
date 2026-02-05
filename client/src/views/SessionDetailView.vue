<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { Codemirror } from 'vue-codemirror'
import { json } from '@codemirror/lang-json'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorView } from '@codemirror/view'
import { useTheme } from '../composables/useTheme'
import { getSession, getRunsForSession, deleteSession } from '../mock/data'

const route = useRoute()
const router = useRouter()
const sessionId = computed(() => route.params.id as string)
const session = computed(() => getSession(sessionId.value))
const runs = computed(() => getRunsForSession(sessionId.value))
const { theme } = useTheme()

const jsonlContent = computed(() => {
  if (!session.value) return ''
  return session.value.messages
    .map((msg) => JSON.stringify(msg))
    .join('\n')
})

const cmExtensions = computed(() => {
  const exts = [json(), EditorView.lineWrapping, EditorView.editable.of(false)]
  if (theme.value === 'dark') exts.push(oneDark)
  return exts
})

function handleDelete() {
  if (confirm(`Delete session ${sessionId.value}?`)) {
    deleteSession(sessionId.value)
    router.push('/sessions')
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString()
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
        <div class="flex items-center gap-4">
          <RouterLink
            :to="`/agents/${session.agentName}`"
            class="text-blue-500 hover:underline"
          >
            Agent: {{ session.agentName }}
          </RouterLink>
          <button
            @click="handleDelete"
            class="rounded border border-red-500 px-3 py-1 text-sm text-red-500 hover:bg-red-50 dark:hover:bg-red-950"
          >
            Delete
          </button>
        </div>
      </div>

      <div class="mb-6 grid grid-cols-1 gap-4 md:grid-cols-3">
        <div class="rounded border border-gray-300 p-4 dark:border-gray-600">
          <div class="text-sm text-gray-500 dark:text-gray-400">Created</div>
          <div class="font-mono">{{ formatDate(session.createdAt) }}</div>
        </div>
        <div class="rounded border border-gray-300 p-4 dark:border-gray-600">
          <div class="text-sm text-gray-500 dark:text-gray-400">Last Active</div>
          <div class="font-mono">{{ formatDate(session.lastActiveAt) }}</div>
        </div>
        <div class="rounded border border-gray-300 p-4 dark:border-gray-600">
          <div class="text-sm text-gray-500 dark:text-gray-400">Messages</div>
          <div class="text-2xl font-bold">{{ session.messages.length }}</div>
        </div>
      </div>

      <div v-if="runs.length > 0" class="mb-6">
        <h2 class="mb-4 text-xl font-bold">Runs</h2>
        <div class="rounded border border-gray-300 dark:border-gray-600">
          <table class="w-full text-sm">
            <thead class="border-b border-gray-300 bg-gray-50 dark:border-gray-600 dark:bg-gray-900">
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
        <h2 class="mb-4 text-xl font-bold">History (JSONL)</h2>
        <div class="overflow-hidden rounded border border-gray-300 dark:border-gray-600">
          <Codemirror
            :model-value="jsonlContent"
            :extensions="cmExtensions"
            :style="{ fontSize: '13px' }"
          />
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
