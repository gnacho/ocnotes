<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { PanelLeft, StickyNote } from 'lucide-vue-next'
import SidebarNav from '../components/SidebarNav.vue'
import NoteList from '../components/NoteList.vue'
import NoteEditor from '../components/NoteEditor.vue'
import NewNoteDialog from '../components/NewNoteDialog.vue'
import ModalDialog from '../components/ModalDialog.vue'
import { state, toggleZenMode } from '../stores/notes'
import type { Note } from '../stores/notes'
import { useNotesApi } from '../composables/api'

const api = useNotesApi()

const editorRef = ref<InstanceType<typeof NoteEditor> | null>(null)

type PendingAction = { kind: 'select'; note: Note } | { kind: 'new' }
const pendingAction = ref<PendingAction | null>(null)
const showUnsavedDialog = ref(false)

function selectNote(note: Note) {
  if (note === state.activeNote) return
  if (editorRef.value?.isDirty()) {
    pendingAction.value = { kind: 'select', note }
    showUnsavedDialog.value = true
    return
  }
  state.activeNote = note
}

function handleDeleted() {
  const remaining = [...state.notes].sort((a, b) => b.modified - a.modified)
  state.activeNote = remaining[0] ?? null
}

const showNewNote = ref(false)

function newNote() {
  if (editorRef.value?.isDirty()) {
    pendingAction.value = { kind: 'new' }
    showUnsavedDialog.value = true
    return
  }
  showNewNote.value = true
}

function proceed() {
  const action = pendingAction.value
  pendingAction.value = null
  showUnsavedDialog.value = false
  if (!action) return
  if (action.kind === 'select') {
    state.activeNote = action.note
  } else {
    showNewNote.value = true
  }
}

function discardChanges() {
  proceed()
}

async function saveChanges() {
  const editor = editorRef.value
  if (!editor) return
  await editor.save()
  if (editor.isDirty()) return
  proceed()
}

function cancelUnsaved() {
  pendingAction.value = null
  showUnsavedDialog.value = false
}

async function onNewNoteCreated(title: string) {
  showNewNote.value = false
  const category = state.currentCategory === '__none__' ? '' : state.currentCategory
  try {
    const n = await api.createNote(title, '', category)
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
    }
  }
}

onMounted(() => {
  window.addEventListener('keydown', onGlobalKey)
  void api
    .getSettings()
    .then((s) => {
      const fonts = ['default', 'sans', 'serif', 'mono']
      if (fonts.includes(s.editorFont)) state.editorFont = s.editorFont
      const size = Number.parseInt(s.editorFontSize, 10)
      if (Number.isFinite(size) && size >= 12 && size <= 22) state.editorFontSize = size
    })
    .catch(() => {})
})
onBeforeUnmount(() => window.removeEventListener('keydown', onGlobalKey))
</script>

<template>
  <main :class="['notes-app', { 'zen-mode': state.zenMode }]">
    <header class="notes-topbar">
      <button
        class="oc-button oc-button-raw icon-btn"
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
        <NoteList @select-note="selectNote" @request-new-note="newNote" />
      </section>

      <div v-if="!state.zenMode" class="col-resizer" @pointerdown="startResize" />

      <main class="notes-main">
        <NoteEditor
          ref="editorRef"
          v-if="state.activeNote"
          :note="state.activeNote"
          @deleted="handleDeleted"
        />
        <div v-else class="editor-empty">
          <StickyNote :size="56" />
          <p class="empty-title">{{ $gettext('Select a note') }}</p>
          <p class="empty-hint">{{ $gettext('Choose a note from the list to read or edit it.') }}</p>
        </div>
      </main>
    </div>

    <NewNoteDialog
      v-if="showNewNote"
      @create="onNewNoteCreated"
      @close="showNewNote = false"
    />

    <ModalDialog
      v-if="showUnsavedDialog"
      :title="$gettext('Unsaved changes')"
      @close="cancelUnsaved"
    >
      <p>{{ $gettext('You have unsaved changes. Save them before leaving this note?') }}</p>
      <footer class="modal-actions">
        <button type="button" class="oc-button oc-button-outline" @click="cancelUnsaved">
          {{ $gettext('Cancel') }}
        </button>
        <button type="button" class="oc-button oc-button-outline" @click="discardChanges">
          {{ $gettext('Discard') }}
        </button>
        <button type="button" class="oc-button oc-button-primary oc-button-filled" @click="saveChanges">
          {{ $gettext('Save') }}
        </button>
      </footer>
    </ModalDialog>
  </main>
</template>
