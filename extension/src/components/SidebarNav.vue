<script setup lang="ts">
import { computed } from 'vue'
import type { Note } from '../stores/notes'
import { state } from '../stores/notes'

interface Props {
  onSelectNote?: (note: Note) => void
  onNewNote?: () => void
}
const props = defineProps<Props>()

async function handleNew() {
  // skeleton - will be wired to API later
  props.onNewNote?.()
}
</script>

<template>
  <nav class="sidebar-nav" aria-label="Notes navigation">
    <button class="btn btn-new-note" @click="handleNew()">+ New note</button>

    <input v-model="state.searchQuery" type="search" placeholder="Search..." aria-label="Search" class="search-input" />

    <section class="nav-section">
      <h3>Categories</h3>
      <ul class="category-list">
        <li :class="{ active: !state.currentCategory && !state.filterFavorites }">
          <button @click="state.currentCategory = ''; state.filterFavorites = false">Uncategorized ({{ state.notes.filter(n => !n.category).length }})</button>
        </li>
        <li v-for="cat in ['Work', 'Personal', 'Ideas']" :key="cat" :class="{ active: state.currentCategory === cat }">
          <button @click="state.currentCategory = cat">{{ cat }}</button>
        </li>
      </ul>
    </section>

    <section class="nav-section">
      <h3>Favorites</h3>
      <button :class="{ active: state.filterFavorites }" @click="state.filterFavorites = !state.filterFavorites">★ Favorites ({{ state.notes.filter(n => n.favorite).length }})</button>
    </section>
  </nav>
</template>
