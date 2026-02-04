<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import { getSkill } from '../mock/data'

const route = useRoute()
const skillName = computed(() => route.params.name as string)
const skill = computed(() => getSkill(skillName.value))
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
        <div class="flex gap-2">
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
        <div class="p-4">
          <pre class="whitespace-pre-wrap text-sm">{{ skill.content }}</pre>
        </div>
      </div>
    </div>

    <div v-else class="rounded border border-red-500 bg-red-50 p-4 dark:bg-red-950">
      <p class="text-red-700 dark:text-red-300">
        Skill "{{ skillName }}" not found.
      </p>
    </div>
  </div>
</template>
