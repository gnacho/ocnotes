import { reactive } from 'vue'

export interface Note {
  id: number
  etag: string
  title: string
  content: string
  category: string
  favorite: boolean
  modified: number
  readonly?: boolean
}

export type DisplayMode = 'rich' | 'plain' | 'preview'

interface AppState {
  notes: Note[]
  activeNote: Note | null
  categories: string[]
  currentCategory: string
  filterFavorites: boolean
  searchQuery: string
  displayMode: DisplayMode
  zenMode: boolean
  loading: boolean
  error: string | null
  sidebarOpen: boolean
}

export const state = reactive<AppState>({
  notes: [],
  activeNote: null,
  categories: [],
  currentCategory: '',
  filterFavorites: false,
  searchQuery: '',
  displayMode: 'rich',
  zenMode: false,
  loading: false,
  error: null,
  sidebarOpen: true,
})

export function setActiveNote(note: Note | null) {
  state.activeNote = note
}

export function clearActiveNote() {
  state.activeNote = null
}

export function setDisplayMode(mode: DisplayMode) {
  state.displayMode = mode
}

export function toggleZenMode() {
  state.zenMode = !state.zenMode
}

export function isZenMode() {
  return state.zenMode
}

export function setCurrentCategory(cat: string) {
  state.currentCategory = cat
  state.filterFavorites = false
}

export function toggleFavoritesFilter() {
  state.filterFavorites = !state.filterFavorites
  if (state.filterFavorites) {
    state.currentCategory = ''
  }
}

export function setSearchQuery(q: string) {
  state.searchQuery = q
}

export function updateSettings(newSettings: Record<string, string>) {
  // handled via API call
  void newSettings
}
