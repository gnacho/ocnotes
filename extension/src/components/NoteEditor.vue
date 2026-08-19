<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Note } from '../stores/notes'
import { state } from '../stores/notes'
import { useNotesApi } from '../composables/api'

const emit = defineEmits<{
  back: []
  deleted: [id: number]
}>()

const props = defineProps<{
  note: Note
}>()

const api = useNotesApi()
const title = ref(props.note.title)
const content = ref(props.note.content)
const saving = ref(false)
const error = ref<string | null>(null)

watch(
  () => props.note,
  (n) => {
    title.value = n.title
    content.value = n.content
  },
)

function renderPreview(md: string): string {
  const escaped = md
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
  return escaped
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]*)`/g, '<code>$1</code>')
    .replace(/\n/g, '<br>')
}

async function save() {
  saving.value = true
  error.value = null
  try {
    const payload: Partial<Note> = {}
    if (title.value !== props.note.title) payload.title = title.value
    if (content.value !== props.note.content) payload.content = content.value
    if (!Object.keys(payload).length) return
    await api.updateNote(props.note.id, payload, props.note.etag)
    state.notes = state.notes.map((n) =>
      n.id === props.note.id ? { ...n, ...payload, modified: Date.now() / 1000 } : n,
    )
    emit('back')
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    saving.value = false
  }
}

async function remove() {
  error.value = null
  try {
    await api.deleteNote(props.note.id)
    state.notes = state.notes.filter((n) => n.id !== props.note.id)
    emit('deleted', props.note.id)
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  }
}

async function toggleFavorite() {
  const fav = !props.note.favorite
  try {
    await api.updateNote(props.note.id, { favorite: fav }, props.note.etag)
    state.notes = state.notes.map((n) => (n.id === props.note.id ? { ...n, favorite: fav } : n))
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  }
}
</script>

<template>
  <div class="note-editor">
    <header class="editor-header">
      <button class="btn-back" @click="emit('back')">← {{ $gettext('Back') }}</button>
      <select v-model="state.displayMode" aria-label="Display mode">
        <option value="rich">{{ $gettext('Rich text') }}</option>
        <option value="plain">{{ $gettext('Plain text') }}</option>
        <option value="preview">{{ $gettext('Preview') }}</option>
      </select>
      <button class="btn-fav" @click="toggleFavorite" :title="props.note.favorite ? 'Unstar' : 'Star'">
        {{ props.note.favorite ? '★' : '☆' }}
      </button>
      <button class="btn-save" :disabled="saving" @click="save">
        {{ saving ? '…' : $gettext('Save') }}
      </button>
      <button class="btn-delete" @click="remove">{{ $gettext('Delete') }}</button>
    </header>
    <main class="editor-body">
      <input v-model="title" :placeholder="$gettext('Title')" class="title-input" />
      <textarea
        v-if="state.displayMode !== 'preview'"
        v-model="content"
        :placeholder="$gettext('Content') + '...'"
        class="content-textarea"
      />
      <div v-else class="preview-pane" v-html="renderPreview(content)" />
      <p v-if="error" class="error-msg">{{ error }}</p>
    </main>
  </div>
</template>
