<script setup lang="ts">
import { computed, ref } from 'vue'
import SidebarNav from '../components/SidebarNav.vue'
import NoteList from '../components/NoteList.vue'
import NoteEditor from '../components/NoteEditor.vue'
import { state, toggleZenMode } from '../stores/notes'
import type { Note } from '../stores/notes'

const mediaDark = window.matchMedia('(prefers-color-scheme: dark)')
const systemDark = ref(mediaDark.matches)
mediaDark.addEventListener?.('change', (e) => {
  systemDark.value = e.matches
})

const themeClass = computed(() =>
  systemDark.value ? 'notes-app-dark' : 'notes-app-light',
)

function selectNote(note: Note) {
  state.activeNote = note
}

function deselectNote() {
  state.activeNote = null
}

function handleDeleted() {
  state.activeNote = null
}

function handleNew() {
  state.activeNote = null
}
</script>

<template>
  <main :class="['notes-app', themeClass]">
    <header class="notes-topbar">
      <button class="btn-sidebar-toggle" @click="state.sidebarOpen = !state.sidebarOpen">☰</button>
      <h1 class="notes-title">{{ $gettext('Notes') }}</h1>
      <span class="topbar-spacer"></span>
      <button class="btn-zen" :title="$gettext('Zen mode')" @click="toggleZenMode">◻</button>
    </header>

    <div class="notes-body">
      <aside v-if="!state.zenMode && state.sidebarOpen" class="notes-sidebar">
        <SidebarNav @new-note="handleNew" />
      </aside>

      <main class="notes-main">
        <NoteList v-if="!state.activeNote" @select-note="selectNote" @new-note="handleNew" />
        <NoteEditor
          v-else
          :note="state.activeNote"
          @back="deselectNote"
          @deleted="handleDeleted"
        />
      </main>
    </div>
  </main>
</template>

<style scoped>
.notes-main {
  flex: 1;
  overflow-y: auto;
  min-width: 0;
}
</style>
