<script setup lang="ts">
import Navbar from './Navbar.vue'
import Sidebar from './Sidebar.vue'
</script>

<template>
  <div class="min-h-screen bg-white text-black dark:bg-black dark:text-white">
    <Navbar />
    <Sidebar />
    <main class="ml-48 pt-14">
      <div class="p-6">
        <RouterView />
      </div>
    </main>
  </div>
</template>
