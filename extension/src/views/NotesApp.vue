<script setup lang="ts">
import { defineComponent } from 'vue'
import SidebarNav from '../components/SidebarNav.vue'
import NoteList from '../components/NoteList.vue'
import NoteEditor from '../components/NoteEditor.vue'
import { state } from '../stores/notes'

// Note selection flow
const selectNote = (note: any) => { state.activeNote = note }
const deselectNote = () => { state.activeNote = null }
</script>

<template>
  <div class="notes-app">
    <!-- Top bar -->
    <header class="topbar">
      <button class="btn-sidebar" @click="state.sidebarOpen = !state.sidebarOpen">☰</button>
      <h1 class="title">{{ $gettext('Notes') }}</h1>
      <div></div>
      <button class="zen-toggle" title="Zen mode" @click="$emit('toggleZen')">◻</button>
    </header>

    <div class="body">
      <aside v-if="!state.zenMode && state.sidebarOpen" class="sidebar">
        <SidebarNav />
      </aside>
      
      <main class="main">
        <NoteList v-if="!state.activeNote" @select-note="selectNote" />
        <NoteEditor v-else :note="state.activeNote" @back="deselectNote" @saved="selectNote" />
      </main>
    </div>
  </div>
</template>

<style scoped>
.notes-app { height: 100%; display: flex; flex-direction: column; }
.topbar { display: flex; align-items: center; gap: .5rem; padding: .5rem 1rem; border-bottom: 1px solid var(--oc-role-secondary); background: var(--oc-role-surface); min-height: 48px; }
.title { margin: 0; font-size: 1.1rem; white-space: nowrap; }
.btn-sidebar, .zen-toggle { background: none; border: none; cursor: pointer; font-size: 1.2rem; padding: .25rem; }
.body { display: flex; flex: 1; overflow: hidden; }
.sidebar { width: 240px; min-width: 240px; border-right: 1px solid var(--oc-role-secondary); overflow-y: auto; padding: .5rem; background: var(--oc-role-surface); }
.main { flex: 1; overflow-y: auto; }
</style>
