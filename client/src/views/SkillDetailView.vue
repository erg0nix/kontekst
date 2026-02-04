<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { Codemirror } from 'vue-codemirror'
import { markdown } from '@codemirror/lang-markdown'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorView } from '@codemirror/view'
import { useTheme } from '../composables/useTheme'
import { getSkill, deleteSkill } from '../mock/data'

const route = useRoute()
const router = useRouter()
const { theme } = useTheme()
const skillName = computed(() => route.params.name as string)
const skill = computed(() => getSkill(skillName.value))

const skillContent = ref('')
watch(skill, (s) => {
  if (s) skillContent.value = s.content
}, { immediate: true })

const cmExtensions = computed(() => {
  const exts = [markdown(), EditorView.lineWrapping]
  if (theme.value === 'dark') exts.push(oneDark)
  return exts
})

function handleSave() {
  alert('Saved (mock)')
}

function handleCancel() {
  if (skill.value) skillContent.value = skill.value.content
}

function handleDelete() {
  if (confirm(`Delete skill "${skillName.value}"?`)) {
    deleteSkill(skillName.value)
    router.push('/skills')
  }
}
</script>

<template>
  <div>
    <div class="mb-6 flex items-center gap-2">
      <RouterLink to="/skills" class="text-blue-500 hover:underline">
        Skills
      </RouterLink>
      <span class="text-gray-400">/</span>
      <span>{{ skillName }}</span>
    </div>

    <div v-if="skill">
      <div class="mb-6 flex items-center justify-between">
        <h1 class="text-2xl font-bold">{{ skill.name }}</h1>
        <div class="flex items-center gap-2">
          <span
            v-if="skill.userInvocable"
            class="rounded bg-green-100 px-2 py-0.5 text-sm text-green-800 dark:bg-green-900 dark:text-green-200"
          >
            User Invocable
          </span>
          <span
            v-if="!skill.disableModelInvocation"
            class="rounded bg-blue-100 px-2 py-0.5 text-sm text-blue-800 dark:bg-blue-900 dark:text-blue-200"
          >
            Model Invocable
          </span>
          <button
            @click="handleDelete"
            class="rounded border border-red-500 px-3 py-1 text-sm text-red-500 hover:bg-red-50 dark:hover:bg-red-950"
          >
            Delete
          </button>
        </div>
      </div>

      <p class="mb-6 text-gray-600 dark:text-gray-400">{{ skill.description }}</p>

      <div class="mb-6 rounded border border-black dark:border-white">
        <div class="border-b border-black bg-gray-50 px-4 py-2 font-medium dark:border-white dark:bg-gray-900">
          Details
        </div>
        <table class="w-full text-sm">
          <tbody>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">Path</td>
              <td class="px-4 py-2 font-mono">{{ skill.path }}</td>
            </tr>
            <tr class="border-b border-gray-200 dark:border-gray-700">
              <td class="px-4 py-2 font-medium">User Invocable</td>
              <td class="px-4 py-2">{{ skill.userInvocable ? 'Yes' : 'No' }}</td>
            </tr>
            <tr>
              <td class="px-4 py-2 font-medium">Model Invocation</td>
              <td class="px-4 py-2">
                {{ skill.disableModelInvocation ? 'Disabled' : 'Enabled' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="rounded border border-black dark:border-white">
        <div class="border-b border-black bg-gray-50 px-4 py-2 font-medium dark:border-white dark:bg-gray-900">
          Content
        </div>
        <Codemirror
          v-model="skillContent"
          :extensions="cmExtensions"
          :style="{ fontSize: '13px' }"
        />
      </div>
      <div class="mt-3 flex gap-2">
        <button
          @click="handleSave"
          class="rounded border border-green-500 px-3 py-1 text-sm text-green-600 hover:bg-green-50 dark:text-green-400 dark:hover:bg-green-950"
        >
          Save
        </button>
        <button
          @click="handleCancel"
          class="rounded border border-black px-3 py-1 text-sm hover:bg-gray-100 dark:border-white dark:hover:bg-gray-800"
        >
          Cancel
        </button>
      </div>
    </div>

    <div v-else class="rounded border border-red-500 bg-red-50 p-4 dark:bg-red-950">
      <p class="text-red-700 dark:text-red-300">
        Skill "{{ skillName }}" not found.
      </p>
    </div>
  </div>
</template>
