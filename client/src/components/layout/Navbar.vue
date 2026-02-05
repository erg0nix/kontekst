<script setup lang="ts">
import { ref } from 'vue'
import StatusBadge from '../StatusBadge.vue'
import ThemeToggle from '../ThemeToggle.vue'
import { mockDaemonStatus } from '../../mock/data'

const daemonRunning = ref(true)

function toggleDaemon() {
  daemonRunning.value = !daemonRunning.value
}
</script>

<template>
  <nav
    class="fixed top-0 left-0 right-0 z-50 flex h-14 items-center justify-between border-b border-gray-300 bg-white px-4 dark:border-gray-600 dark:bg-black"
  >
    <div class="flex items-center gap-2">
      <span class="text-xl font-bold text-black dark:text-white">Kontekst</span>
    </div>

    <div class="flex items-center gap-4">
      <StatusBadge
        label="Daemon"
        :status="daemonRunning ? 'healthy' : 'unhealthy'"
      />
      <StatusBadge
        label="LLM"
        :status="
          mockDaemonStatus.llamaServerHealthy ? 'healthy' : 'unhealthy'
        "
      />

      <button
        @click="toggleDaemon"
        class="rounded border px-3 py-1 text-sm"
        :class="
          daemonRunning
            ? 'border-red-500 text-red-500 hover:bg-red-50 dark:hover:bg-red-950'
            : 'border-green-500 text-green-500 hover:bg-green-50 dark:hover:bg-green-950'
        "
      >
        {{ daemonRunning ? 'Stop' : 'Start' }}
      </button>

      <ThemeToggle />
    </div>
  </nav>
</template>
