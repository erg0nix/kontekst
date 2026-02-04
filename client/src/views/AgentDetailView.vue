<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import { getAgentConfig } from '../mock/data'

const route = useRoute()
const agentName = computed(() => route.params.name as string)
const agent = computed(() => getAgentConfig(agentName.value))
</script>

<template>
  <div>
    <div class="mb-6 flex items-center gap-2">
      <RouterLink to="/agents" class="text-blue-500 hover:underline">
        Agents
      </RouterLink>
      <span class="text-gray-400">/</span>
      <span>{{ agentName }}</span>
    </div>

    <div v-if="agent">
      <h1 class="mb-2 text-2xl font-bold">{{ agent.displayName }}</h1>
      <p class="mb-6 font-mono text-gray-500">{{ agent.name }}</p>

      <div class="mb-6 rounded border border-black dark:border-white">
        <div class="border-b border-black bg-gray-50 px-4 py-2 font-medium dark:border-white dark:bg-gray-900">
          Configuration
        </div>
        <table class="w-full text-sm">
          <tbody>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">Model</td>
              <td class="px-4 py-2 font-mono">{{ agent.model }}</td>
            </tr>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">Tool Role</td>
              <td class="px-4 py-2">
                <span
                  class="rounded px-2 py-0.5 text-xs"
                  :class="
                    agent.toolRole
                      ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                      : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
                  "
                >
                  {{ agent.toolRole ? 'Enabled' : 'Disabled' }}
                </span>
              </td>
            </tr>
            <template v-if="agent.sampling">
              <tr class="border-b border-gray-200 dark:border-gray-700">
                <td class="px-4 py-2 font-medium">Temperature</td>
                <td class="px-4 py-2 font-mono">
                  {{ agent.sampling.temperature ?? 'default' }}
                </td>
              </tr>
              <tr class="border-b border-gray-200 dark:border-gray-700">
                <td class="px-4 py-2 font-medium">Top P</td>
                <td class="px-4 py-2 font-mono">
                  {{ agent.sampling.topP ?? 'default' }}
                </td>
              </tr>
              <tr class="border-b border-gray-200 dark:border-gray-700">
                <td class="px-4 py-2 font-medium">Top K</td>
                <td class="px-4 py-2 font-mono">
                  {{ agent.sampling.topK ?? 'default' }}
                </td>
              </tr>
              <tr class="border-b border-gray-200 dark:border-gray-700">
                <td class="px-4 py-2 font-medium">Repeat Penalty</td>
                <td class="px-4 py-2 font-mono">
                  {{ agent.sampling.repeatPenalty ?? 'default' }}
                </td>
              </tr>
              <tr>
                <td class="px-4 py-2 font-medium">Max Tokens</td>
                <td class="px-4 py-2 font-mono">
                  {{ agent.sampling.maxTokens ?? 'default' }}
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <div class="rounded border border-black dark:border-white">
        <div class="border-b border-black bg-gray-50 px-4 py-2 font-medium dark:border-white dark:bg-gray-900">
          System Prompt
        </div>
        <div class="p-4">
          <pre class="whitespace-pre-wrap text-sm">{{ agent.systemPrompt }}</pre>
        </div>
      </div>
    </div>

    <div v-else class="rounded border border-red-500 bg-red-50 p-4 dark:bg-red-950">
      <p class="text-red-700 dark:text-red-300">
        Agent "{{ agentName }}" not found.
      </p>
    </div>
  </div>
</template>
