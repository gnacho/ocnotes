<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { PanelLeft, StickyNote } from 'lucide-vue-next'
import SidebarNav from '../components/SidebarNav.vue'
import NoteList from '../components/NoteList.vue'
import NoteEditor from '../components/NoteEditor.vue'
import { state, toggleZenMode } from '../stores/notes'
import type { Note } from '../stores/notes'
import { useNotesApi } from '../composables/api'

const api = useNotesApi()

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

async function newNote() {
  const category = state.currentCategory === '__none__' ? '' : state.currentCategory
  try {
    const n = await api.createNote('New note', '', category)
    if (category) {
      state.pendingCategories = state.pendingCategories.filter((c) => c !== category)
    }
    state.notes = [n, ...state.notes]
    state.activeNote = n
  } catch (e) {
    state.error = e instanceof Error ? e.message : String(e)
  }
}

/* ----- resizable middle column (persisted) ----- */
const WIDTH_KEY = 'ocnotes.listWidth'
const listWidth = ref((() => {
  const stored = Number.parseFloat(window.localStorage.getItem(WIDTH_KEY) || '')
  return Number.isFinite(stored) ? Math.min(560, Math.max(260, stored)) : 360
})())

function startResize(e: PointerEvent) {
  const el = e.target as HTMLElement
  el.setPointerCapture(e.pointerId)
  const startX = e.clientX
  const startW = listWidth.value
  const move = (ev: PointerEvent) => {
    listWidth.value = Math.min(560, Math.max(260, startW + ev.clientX - startX))
  }
  const up = () => {
    el.removeEventListener('pointermove', move)
    el.removeEventListener('pointerup', up)
    window.localStorage.setItem(WIDTH_KEY, String(listWidth.value))
  }
  el.addEventListener('pointermove', move)
  el.addEventListener('pointerup', up)
}

/* ----- global keyboard shortcuts ----- */
function onGlobalKey(e: KeyboardEvent) {
  const mod = e.ctrlKey || e.metaKey
  if (mod && e.key.toLowerCase() === 'n') {
    e.preventDefault()
    void newNote()
  } else if (mod && e.key.toLowerCase() === 'f') {
    e.preventDefault()
    document.getElementById('notes-search')?.focus()
  } else if (e.key === 'Escape') {
    if (state.zenMode) {
      toggleZenMode()
    } else if (state.activeNote) {
      state.activeNote = null
    }
  }
}

onMounted(() => window.addEventListener('keydown', onGlobalKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onGlobalKey))
</script>

<template>
  <main :class="['notes-app', themeClass, { 'zen-mode': state.zenMode }]">
    <header class="notes-topbar">
      <button
        class="icon-btn"
        :aria-label="$gettext('Toggle sidebar')"
        @click="state.sidebarOpen = !state.sidebarOpen"
      >
        <PanelLeft :size="18" />
      </button>
      <h1 class="notes-title">{{ $gettext('Notes') }}</h1>
    </header>

    <div class="notes-body">
      <aside v-if="state.sidebarOpen && !state.zenMode" class="notes-sidebar">
        <SidebarNav />
      </aside>

      <section
        v-if="!state.zenMode"
        class="notes-list-col"
        :style="{ width: `${listWidth}px` }"
      >
        <NoteList @select-note="selectNote" />
      </section>

      <div v-if="!state.zenMode" class="col-resizer" @pointerdown="startResize" />

      <main class="notes-main">
        <NoteEditor
          v-if="state.activeNote"
          :note="state.activeNote"
          @back="deselectNote"
          @deleted="handleDeleted"
        />
        <div v-else class="editor-empty">
          <StickyNote :size="56" />
          <p class="empty-title">{{ $gettext('Select a note') }}</p>
          <p class="empty-hint">{{ $gettext('Choose a note from the list to read or edit it.') }}</p>
        </div>
      </main>
    </div>
  </main>
</template>

<style scoped>
.notes-main {
  flex: 1;
  overflow: hidden;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
</style>
