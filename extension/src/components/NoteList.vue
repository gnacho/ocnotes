<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useGettext } from 'vue3-gettext'
import { Plus, Star } from 'lucide-vue-next'
import type { Note } from '../stores/notes'
import { state } from '../stores/notes'
import { useNotesApi } from '../composables/api'

const { $gettext } = useGettext()

const emit = defineEmits<{
  selectNote: [note: Note]
}>()

const api = useNotesApi()
const loading = ref(false)
const creating = ref(false)
const error = ref<string | null>(null)

async function loadNotes() {
  loading.value = true
  error.value = null
  try {
    state.notes = await api.listNotes()
    if (!state.activeNote) {
      const newest = [...state.notes].sort((a, b) => b.modified - a.modified)[0]
      if (newest) {
        state.activeNote = newest
      } else {
        const n = await api.createNote('New note', '', '')
        state.notes = [n]
        state.activeNote = n
      }
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

onMounted(loadNotes)

async function createNew() {
  creating.value = true
  error.value = null
  try {
    const category = state.currentCategory === '__none__' ? '' : state.currentCategory
    const n = await api.createNote('New note', '', category)
    if (category) {
      state.pendingCategories = state.pendingCategories.filter((c) => c !== category)
    }
    state.notes = [n, ...state.notes]
    emit('selectNote', n)
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    creating.value = false
  }
}

function inCurrentCategory(n: Note): boolean {
  if (state.currentCategory === '') return true
  if (state.currentCategory === '__none__') return !n.category
  return !!n.category && n.category.split('/')[0] === state.currentCategory
}

function sortedDesc(notes: Note[]): Note[] {
  return [...notes].sort((a, b) => b.modified - a.modified)
}

interface Group {
  key: string
  label: string
  favorites: boolean
  notes: Note[]
}

const groups = computed<Group[]>(() => {
  let pool = state.notes.filter(inCurrentCategory)
  const q = state.searchQuery.trim().toLowerCase()
  if (q) {
    pool = pool.filter(
      (n) =>
        n.title.toLowerCase().includes(q) || n.content.toLowerCase().includes(q),
    )
    return [{ key: 'results', label: '', favorites: false, notes: sortedDesc(pool) }]
  }
  const result: Group[] = []
  const favs = sortedDesc(pool.filter((n) => n.favorite))
  if (favs.length) result.push({ key: 'favorites', label: $gettext('Favorites'), favorites: true, notes: favs })
  const byBucket = new Map<string, Note[]>()
  for (const n of sortedDesc(pool.filter((n) => !n.favorite))) {
    const label = bucketLabel(n.modified)
    if (!byBucket.has(label)) byBucket.set(label, [])
    byBucket.get(label)!.push(n)
  }
  for (const [label, notes] of byBucket) {
    result.push({ key: label, label, favorites: false, notes })
  }
  return result
})

function bucketLabel(ts: number): string {
  const d = new Date(ts * 1000)
  const now = new Date()
  if (d.toDateString() === now.toDateString()) return $gettext('Today')
  const monday = new Date(now)
  monday.setHours(0, 0, 0, 0)
  monday.setDate(monday.getDate() - ((monday.getDay() + 6) % 7))
  if (d >= monday) return $gettext('This week')
  const month = d.toLocaleDateString(undefined, { month: 'long' })
  return `${month.charAt(0).toUpperCase() + month.slice(1)} ${d.getFullYear()}`
}

function formatShort(ts: number): string {
  const d = new Date(ts * 1000)
  const now = new Date()
  if (d.toDateString() === now.toDateString()) {
    return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
  }
  if (d.getFullYear() === now.getFullYear()) {
    return d.toLocaleDateString(undefined, { day: 'numeric', month: 'short' })
  }
  return d.toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })
}
</script>

<template>
  <div class="note-list">
    <header class="note-list-header">
      <button class="oc-button oc-button-primary oc-button-filled btn-new" :disabled="creating" @click="createNew">
        <Plus :size="16" />
        {{ $gettext('New note') }}
      </button>
      <input
        id="notes-search"
        v-model="state.searchQuery"
        type="search"
        class="search-input"
        :placeholder="$gettext('Search notes')"
        :aria-label="$gettext('Search notes')"
      />
    </header>

    <p v-if="loading" class="empty-msg">…</p>
    <p v-else-if="error" class="error-msg">{{ error }}</p>
    <p v-else-if="!groups.length" class="empty-msg">
      {{ $gettext('No notes yet') }}. {{ $gettext('Start by creating your first note.') }}
    </p>

    <template v-else>
      <section v-for="g in groups" :key="g.key" class="note-group">
        <h3 v-if="g.label" :class="['group-header', { favorites: g.favorites }]">
          <Star v-if="g.favorites" :size="13" class="fav-star" />
          {{ g.label }}
        </h3>
        <ul class="note-items">
          <li
            v-for="note in g.notes"
            :key="note.id"
            :class="['note-item', { selected: state.activeNote?.id === note.id }]"
            @click="emit('selectNote', note)"
          >
            <span class="title">
              <Star v-if="note.favorite" :size="12" class="fav-star" />
              {{ note.title }}
            </span>
            <span class="preview">
              {{ note.content.slice(0, 120).replace(/\n/g, ' ') }}{{
                note.content.length > 120 ? '…' : ''
              }}
            </span>
            <span class="meta">
              {{ formatShort(note.modified) }}
              <span v-if="note.category" class="category">{{ note.category.split('/').pop() }}</span>
            </span>
          </li>
        </ul>
      </section>
    </template>
  </div>
</template>
