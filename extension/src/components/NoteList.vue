<script setup lang="ts">
import { computed } from 'vue'
import type { Note } from '../stores/notes'
import { state } from '../stores/notes'
import { fetchNotes } from '../composables/api'

const emit = defineEmits<{
  selectNote: [note: Note]
  newNote: []
}>()

fetchNotes(state.currentCategory || undefined).then(notes => {
  Object.assign(state, { notes: notes as unknown as Note[] })
})

const filteredNotes = computed(() => {
  let notes = state.notes
  if (state.searchQuery) {
    const q = state.searchQuery.toLowerCase()
    notes = notes.filter(n => n.title.toLowerCase().includes(q) || n.content.toLowerCase().includes(q))
  }
  return notes.sort((a, b) => b.modified - a.modified)
})
</script>

<template>
  <div class="note-list">
    <header class="note-list-header">
      <button class="btn-new" @click="emit('newNote')">+ New note</button>
    </header>

    <p v-if="!filteredNotes.length" class="empty-msg">No notes yet. Start by creating your first note.</p>

    <ul v-else class="note-items">
      <li v-for="note in filteredNotes" :key="note.id" class="note-item" @click="emit('selectNote', note)">
        <span class="title">{{ note.title }}</span>
        <span class="preview">{{ note.content.slice(0, 100).replace(/\n/g, ' ') }}{{ note.content.length > 100 ? '...' : '' }}</span>
        <span class="meta">
          {{ formatDate(note.modified) }}
          <span v-if="note.category" class="category">{{ note.category.split('/').pop() }}</span>
        </span>
      </li>
    </ul>
  </div>
</template>
