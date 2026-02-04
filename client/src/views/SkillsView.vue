<script setup lang="ts">
import { RouterLink } from 'vue-router'
import DataTable from '../components/DataTable.vue'
import { mockSkills, deleteSkill } from '../mock/data'

function handleDelete(name: string) {
  if (confirm(`Delete skill "${name}"?`)) {
    deleteSkill(name)
  }
}
</script>

<template>
  <div>
    <h1 class="mb-6 text-2xl font-bold">Skills</h1>

    <DataTable :columns="['Name', 'Description', 'User', 'Model']">
      <tr
        v-for="skill in mockSkills"
        :key="skill.name"
        class="border-b border-gray-200 last:border-b-0 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-900"
      >
        <td class="px-4 py-2">
          <RouterLink
            :to="`/skills/${skill.name}`"
            class="font-mono text-blue-500 hover:underline"
          >
            {{ skill.name }}
          </RouterLink>
        </td>
        <td class="px-4 py-2 text-gray-600 dark:text-gray-400">
          {{ skill.description }}
        </td>
        <td class="px-4 py-2">
          <span
            v-if="skill.userInvocable"
            class="rounded bg-green-100 px-2 py-0.5 text-xs text-green-800 dark:bg-green-900 dark:text-green-200"
          >
            Yes
          </span>
          <span v-else class="text-gray-400">No</span>
        </td>
        <td class="px-4 py-2">
          <span
            v-if="!skill.disableModelInvocation"
            class="rounded bg-blue-100 px-2 py-0.5 text-xs text-blue-800 dark:bg-blue-900 dark:text-blue-200"
          >
            Yes
          </span>
          <span v-else class="text-gray-400">No</span>
        </td>
        <td class="px-4 py-2 text-right">
          <button
            @click="handleDelete(skill.name)"
            class="rounded border border-red-500 px-2 py-1 text-xs text-red-500 hover:bg-red-50 dark:hover:bg-red-950"
          >
            Delete
          </button>
        </td>
      </tr>
    </DataTable>
  </div>
</template>
