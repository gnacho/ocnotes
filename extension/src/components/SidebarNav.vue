<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Note } from '../stores/notes'
import { state } from '../stores/notes'
import { useNotesApi } from '../composables/api'

const emit = defineEmits<{
  newNote: []
}>()

const api = useNotesApi()
const creating = ref(false)
const error = ref<string | null>(null)

async function handleNew() {
  creating.value = true
  error.value = null
  try {
    const n = await api.createNote('New note', '', state.currentCategory)
    state.notes = [n, ...state.notes]
    emit('newNote')
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    creating.value = false
  }
}

const categories = computed(() => {
  const set = new Set<string>()
  for (const n of state.notes) {
    if (n.category) set.add(n.category)
  }
  return [...set].sort()
})

const uncategorizedCount = computed(() => state.notes.filter((n) => !n.category).length)
const favoritesCount = computed(() => state.notes.filter((n) => n.favorite).length)
</script>

<template>
  <nav class="sidebar-nav" aria-label="Notes navigation">
    <button class="btn btn-new-note" :disabled="creating" @click="handleNew">
      + {{ $gettext('New note') }}
    </button>
    <p v-if="error" class="error-msg">{{ error }}</p>

    <input
      v-model="state.searchQuery"
      type="search"
      :placeholder="$gettext('Search...')"
      aria-label="Search"
      class="search-input"
    />

    <section class="nav-section">
      <h3>{{ $gettext('Categories') }}</h3>
      <ul class="category-list">
        <li :class="{ active: !state.currentCategory && !state.filterFavorites }">
          <button @click="state.currentCategory = ''; state.filterFavorites = false">
            {{ $gettext('Uncategorized') }} ({{ uncategorizedCount }})
          </button>
        </li>
        <li
          v-for="cat in categories"
          :key="cat"
          :class="{ active: state.currentCategory === cat }"
        >
          <button @click="state.currentCategory = cat">{{ cat }}</button>
        </li>
      </ul>
    </section>

    <section class="nav-section">
      <h3>{{ $gettext('Favorites') }}</h3>
      <button :class="{ active: state.filterFavorites }" @click="state.filterFavorites = !state.filterFavorites">
        ★ {{ $gettext('Favorites') }} ({{ favoritesCount }})
      </button>
    </section>
  </nav>
</template>
