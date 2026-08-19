<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { Note } from '../stores/notes'
import { state } from '../stores/notes'
import { useNotesApi } from '../composables/api'

const emit = defineEmits<{
  selectNote: [note: Note]
}>()

const api = useNotesApi()
const loading = ref(false)
const error = ref<string | null>(null)

async function loadNotes() {
  loading.value = true
  error.value = null
  try {
    state.notes = await api.listNotes(state.currentCategory || undefined)
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

onMounted(loadNotes)

function formatDate(ts: number): string {
  const d = new Date(ts * 1000)
  return d.toLocaleDateString(undefined, { day: 'numeric', month: 'short' })
}

const filteredNotes = computed(() => {
  let notes = state.notes
  if (state.searchQuery) {
    const q = state.searchQuery.toLowerCase()
    notes = notes.filter(
      (n) => n.title.toLowerCase().includes(q) || n.content.toLowerCase().includes(q),
    )
  }
  if (state.filterFavorites) {
    notes = notes.filter((n) => n.favorite)
  }
  return [...notes].sort((a, b) => b.modified - a.modified)
})

const categories = computed(() => {
  const set = new Set<string>()
  for (const n of state.notes) {
    if (n.category) set.add(n.category)
  }
  return [...set].sort()
})
</script>

<template>
  <div class="note-list">
    <header class="note-list-header">
      <button
        class="btn-new"
        @click="
          api
            .createNote('New note', '', state.currentCategory)
            .then((n) => {
              state.notes = [n, ...state.notes]
              emit('selectNote', n)
            })
        "
      >
        + {{ $gettext('New note') }}
      </button>
    </header>

    <p v-if="loading" class="empty-msg">Loading…</p>
    <p v-else-if="error" class="error-msg">{{ error }}</p>
    <p v-else-if="!filteredNotes.length" class="empty-msg">
      {{ $gettext('No notes yet') }}. {{ $gettext('Start by creating your first note.') }}
    </p>

    <ul v-else class="note-items">
      <li
        v-for="note in filteredNotes"
        :key="note.id"
        class="note-item"
        @click="emit('selectNote', note)"
      >
        <span class="title">{{ note.title }}</span>
        <span class="preview">
          {{ note.content.slice(0, 100).replace(/\n/g, ' ') }}{{
            note.content.length > 100 ? '...' : ''
          }}
        </span>
        <span class="meta">
          {{ formatDate(note.modified) }}
          <span v-if="note.category" class="category">{{ note.category.split('/').pop() }}</span>
          <span v-if="note.favorite" class="fav">★</span>
        </span>
      </li>
    </ul>
  </div>
</template>
