<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { Codemirror } from 'vue-codemirror'
import { markdown } from '@codemirror/lang-markdown'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorView } from '@codemirror/view'
import { useTheme } from '../composables/useTheme'
import { getAgentConfig, deleteAgent } from '../mock/data'

const route = useRoute()
const router = useRouter()
const { theme } = useTheme()
const agentName = computed(() => route.params.name as string)
const agent = computed(() => getAgentConfig(agentName.value))

const availableModels = [
  'gpt-oss-20b-Q4_K_M.gguf',
  'codestral-22b-Q4_K_M.gguf',
  'llama-3.1-8b-Q4_K_M.gguf',
  'mistral-7b-Q4_K_M.gguf',
]

const configModel = ref('')
const configToolRole = ref(false)
const configTemperature = ref<number | undefined>()
const configTopP = ref<number | undefined>()
const configTopK = ref<number | undefined>()
const configRepeatPenalty = ref<number | undefined>()
const configMaxTokens = ref<number | undefined>()
const promptContent = ref('')

function resetConfig() {
  const a = agent.value
  if (!a) return
  configModel.value = a.model
  configToolRole.value = a.toolRole
  configTemperature.value = a.sampling?.temperature
  configTopP.value = a.sampling?.topP
  configTopK.value = a.sampling?.topK
  configRepeatPenalty.value = a.sampling?.repeatPenalty
  configMaxTokens.value = a.sampling?.maxTokens
  promptContent.value = a.systemPrompt
}

watch(agent, () => resetConfig(), { immediate: true })

const cmExtensions = computed(() => {
  const exts = [markdown(), EditorView.lineWrapping]
  if (theme.value === 'dark') exts.push(oneDark)
  return exts
})

function handleConfigSave() {
  alert('Configuration saved (mock)')
}

function handleConfigCancel() {
  resetConfig()
}

function handlePromptSave() {
  alert('System prompt saved (mock)')
}

function handlePromptCancel() {
  if (agent.value) promptContent.value = agent.value.systemPrompt
}

function handleDelete() {
  if (confirm(`Delete agent "${agentName.value}"?`)) {
    deleteAgent(agentName.value)
    router.push('/agents')
  }
}
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
      <div class="mb-6 flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">{{ agent.displayName }}</h1>
          <p class="font-mono text-gray-500">{{ agent.name }}</p>
        </div>
        <button
          @click="handleDelete"
          class="rounded border border-red-500 px-3 py-1 text-sm text-red-500 hover:bg-red-50 dark:hover:bg-red-950"
        >
          Delete
        </button>
      </div>

      <div class="mb-6 rounded border border-gray-300 dark:border-gray-600">
        <div class="border-b border-gray-300 bg-gray-50 px-4 py-2 font-medium dark:border-gray-600 dark:bg-gray-900">
          Configuration
        </div>
        <table class="w-full text-sm">
          <tbody>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="w-40 px-4 py-2 font-medium">Model</td>
              <td class="px-4 py-2">
                <select
                  v-model="configModel"
                  class="w-full rounded border border-gray-300 bg-white px-2 py-1 font-mono dark:border-gray-600 dark:bg-black"
                >
                  <option v-for="m in availableModels" :key="m" :value="m">
                    {{ m }}
                  </option>
                </select>
              </td>
            </tr>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">Tool Role</td>
              <td class="px-4 py-2">
                <input
                  type="checkbox"
                  v-model="configToolRole"
                  class="h-4 w-4 accent-green-500"
                />
                <span class="ml-2 text-gray-500">{{ configToolRole ? 'Enabled' : 'Disabled' }}</span>
              </td>
            </tr>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">Temperature</td>
              <td class="px-4 py-2">
                <input
                  type="number"
                  v-model.number="configTemperature"
                  step="0.1"
                  min="0"
                  max="2"
                  placeholder="default"
                  class="w-32 rounded border border-gray-300 bg-white px-2 py-1 font-mono dark:border-gray-600 dark:bg-black"
                />
              </td>
            </tr>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">Top P</td>
              <td class="px-4 py-2">
                <input
                  type="number"
                  v-model.number="configTopP"
                  step="0.05"
                  min="0"
                  max="1"
                  placeholder="default"
                  class="w-32 rounded border border-gray-300 bg-white px-2 py-1 font-mono dark:border-gray-600 dark:bg-black"
                />
              </td>
            </tr>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">Top K</td>
              <td class="px-4 py-2">
                <input
                  type="number"
                  v-model.number="configTopK"
                  step="1"
                  min="0"
                  placeholder="default"
                  class="w-32 rounded border border-gray-300 bg-white px-2 py-1 font-mono dark:border-gray-600 dark:bg-black"
                />
              </td>
            </tr>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">Repeat Penalty</td>
              <td class="px-4 py-2">
                <input
                  type="number"
                  v-model.number="configRepeatPenalty"
                  step="0.1"
                  min="0"
                  placeholder="default"
                  class="w-32 rounded border border-gray-300 bg-white px-2 py-1 font-mono dark:border-gray-600 dark:bg-black"
                />
              </td>
            </tr>
            <tr>
              <td class="px-4 py-2 font-medium">Max Tokens</td>
              <td class="px-4 py-2">
                <input
                  type="number"
                  v-model.number="configMaxTokens"
                  step="256"
                  min="1"
                  placeholder="default"
                  class="w-32 rounded border border-gray-300 bg-white px-2 py-1 font-mono dark:border-gray-600 dark:bg-black"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mb-6 flex gap-2">
        <button
          @click="handleConfigSave"
          class="rounded border border-green-500 px-3 py-1 text-sm text-green-600 hover:bg-green-50 dark:text-green-400 dark:hover:bg-green-950"
        >
          Save
        </button>
        <button
          @click="handleConfigCancel"
          class="rounded border border-gray-300 px-3 py-1 text-sm hover:bg-gray-100 dark:border-gray-600 dark:hover:bg-gray-800"
        >
          Cancel
        </button>
      </div>

      <div class="rounded border border-gray-300 dark:border-gray-600">
        <div class="border-b border-gray-300 bg-gray-50 px-4 py-2 font-medium dark:border-gray-600 dark:bg-gray-900">
          System Prompt
        </div>
        <Codemirror
          v-model="promptContent"
          :extensions="cmExtensions"
          :style="{ fontSize: '13px' }"
        />
      </div>
      <div class="mt-3 flex gap-2">
        <button
          @click="handlePromptSave"
          class="rounded border border-green-500 px-3 py-1 text-sm text-green-600 hover:bg-green-50 dark:text-green-400 dark:hover:bg-green-950"
        >
          Save
        </button>
        <button
          @click="handlePromptCancel"
          class="rounded border border-gray-300 px-3 py-1 text-sm hover:bg-gray-100 dark:border-gray-600 dark:hover:bg-gray-800"
        >
          Cancel
        </button>
      </div>
    </div>

    <div v-else class="rounded border border-red-500 bg-red-50 p-4 dark:bg-red-950">
      <p class="text-red-700 dark:text-red-300">
        Agent "{{ agentName }}" not found.
      </p>
    </div>
  </div>
</template>
