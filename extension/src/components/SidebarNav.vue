<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  FolderPlus,
  Maximize2,
  Minimize2,
  Settings,
  Keyboard,
  FileText,
  Folder,
} from 'lucide-vue-next'
import { state, setCurrentCategory, toggleZenMode } from '../stores/notes'
import { useNotesApi } from '../composables/api'
import { useIsDark } from '../composables/theme'
import { useDragNote } from '../composables/useDragNote'
import ModalDialog from './ModalDialog.vue'

const api = useNotesApi()
const isDark = useIsDark()
const { onDragOverCategory, onDragLeaveCategory, onDropOnCategory } = useDragNote()

const createCategoryClasses = computed(() => [
  'btn-new-category',
  'shadow-md',
  'duration-150',
  'ease-out',
  'hover:brightness-110',
  'w-full',
  isDark.value
    ? 'oc-button-filled oc-button-primary-container !bg-gradient-to-r !from-role-secondary-container !to-role-primary-container'
    : 'oc-button-filled oc-button-primary !bg-gradient-to-r !from-role-secondary !to-role-primary',
])

const showNewCategory = ref(false)
const showSettings = ref(false)
const showShortcuts = ref(false)
const newCategoryName = ref('')
const categoryError = ref<string | null>(null)

const settingsNotesPath = ref('')
const settingsFileSuffix = ref('')
const settingsView = ref<'rich' | 'plain' | 'preview'>('rich')
const settingsFont = ref('default')
const settingsFontSize = ref(15)
const settingsError = ref<string | null>(null)
const settingsSaved = ref(false)
const settingsLoading = ref(false)

const allCategories = computed(() => {
  const set = new Set<string>()
  for (const n of state.notes) {
    if (n.category) set.add(n.category.split('/')[0])
  }
  for (const c of state.pendingCategories) set.add(c)
  return [...set].sort((a, b) => a.localeCompare(b))
})

function categoryCount(cat: string): number {
  return state.notes.filter((n) => n.category && n.category.split('/')[0] === cat).length
}

const uncategorizedCount = computed(() => state.notes.filter((n) => !n.category).length)
const totalCount = computed(() => state.notes.length)

function createCategory() {
  const name = newCategoryName.value.trim()
  if (!name) return
  if (allCategories.value.some((c) => c.toLowerCase() === name.toLowerCase())) {
    categoryError.value = name
    return
  }
  state.pendingCategories = [...state.pendingCategories, name]
  setCurrentCategory(name)
  showNewCategory.value = false
  newCategoryName.value = ''
  categoryError.value = null
}

watch(showNewCategory, (open) => {
  if (open) {
    newCategoryName.value = ''
    categoryError.value = null
  }
})

watch(showSettings, async (open) => {
  if (!open) return
  settingsError.value = null
  settingsSaved.value = false
  settingsLoading.value = true
  try {
    const s = await api.getSettings()
    settingsNotesPath.value = s.notesPath
    settingsFileSuffix.value = s.fileSuffix
    const fonts = ['default', 'sans', 'serif', 'mono']
    if (fonts.includes(s.editorFont)) settingsFont.value = s.editorFont
    const size = Number.parseInt(s.editorFontSize ?? '', 10)
    if (Number.isFinite(size) && size >= 12 && size <= 22) settingsFontSize.value = size
  } catch {
    settingsError.value = 'load'
  } finally {
    settingsLoading.value = false
  }
})

async function saveSettings() {
  settingsError.value = null
  settingsSaved.value = false
  try {
    await api.updateSettings({
      notesPath: settingsNotesPath.value.trim(),
      fileSuffix: settingsFileSuffix.value.trim(),
      editorFont: settingsFont.value,
      editorFontSize: String(settingsFontSize.value),
    })
    state.displayMode = settingsView.value
    state.editorFont = settingsFont.value
    state.editorFontSize = settingsFontSize.value
    settingsSaved.value = true
  } catch {
    settingsError.value = 'save'
  }
}
</script>

<template>
  <nav class="sidebar-nav" aria-label="Notes navigation">
    <div class="sidebar-top">
      <button :class="createCategoryClasses" @click="showNewCategory = true">
        <FolderPlus :size="16" />
        {{ $gettext('New category') }}
      </button>

      <ul class="nav-list">
        <li :class="{ active: state.currentCategory === '' }">
          <button @click="setCurrentCategory('')">
            <span class="nav-label"><FileText :size="15" /> {{ $gettext('All notes') }}</span>
            <span class="count">{{ totalCount }}</span>
          </button>
        </li>
      </ul>

      <h3 class="nav-heading">{{ $gettext('Categories') }}</h3>
      <ul class="nav-list">
        <li
          v-for="cat in allCategories"
          :key="cat"
          :class="{ active: state.currentCategory === cat, 'drop-target': state.dragOverCategory === cat }"
          @dragover="onDragOverCategory($event, cat)"
          @dragleave="onDragLeaveCategory($event)"
          @drop="onDropOnCategory($event, cat)"
        >
          <button @click="setCurrentCategory(cat)">
            <span class="nav-label"><Folder :size="15" /> {{ cat }}</span>
            <span class="count">{{ categoryCount(cat) }}</span>
          </button>
        </li>
        <li
          :class="{ active: state.currentCategory === '__none__', 'drop-target': state.dragOverCategory === '' }"
          @dragover="onDragOverCategory($event, '')"
          @dragleave="onDragLeaveCategory($event)"
          @drop="onDropOnCategory($event, '')"
        >
          <button @click="setCurrentCategory('__none__')">
            <span class="nav-label">{{ $gettext('Uncategorized') }}</span>
            <span class="count">{{ uncategorizedCount }}</span>
          </button>
        </li>
      </ul>
    </div>

    <div class="sidebar-bottom">
      <button :disabled="!state.activeNote" @click="state.activeNote && toggleZenMode()">
        <Maximize2 v-if="!state.zenMode" :size="15" />
        <Minimize2 v-else :size="15" />
        {{ $gettext('Zen mode') }}
      </button>
      <button @click="showSettings = true">
        <Settings :size="15" />
        {{ $gettext('Settings') }}
      </button>
      <button @click="showShortcuts = true">
        <Keyboard :size="15" />
        {{ $gettext('Keyboard shortcuts') }}
      </button>
    </div>

    <ModalDialog
      v-if="showNewCategory"
      :title="$gettext('New category')"
      @close="showNewCategory = false"
    >
      <form class="modal-form" @submit.prevent="createCategory">
        <input
          v-model="newCategoryName"
          type="text"
          class="modal-input"
              :placeholder="$gettext('Category name')"
          autofocus
        />
        <p v-if="categoryError" class="error-msg">{{ categoryError }}</p>
        <footer class="modal-actions">
          <button type="button" class="oc-button oc-button-outline" @click="showNewCategory = false">
            {{ $gettext('Cancel') }}
          </button>
          <button type="submit" class="oc-button oc-button-primary oc-button-filled">{{ $gettext('Create') }}</button>
        </footer>
      </form>
    </ModalDialog>

    <ModalDialog v-if="showSettings" :title="$gettext('Settings')" @close="showSettings = false">
      <div v-if="settingsLoading" class="empty-msg">…</div>
      <form v-else class="modal-form" @submit.prevent="saveSettings">
        <label class="modal-label">
          {{ $gettext('View type') }}
          <select v-model="settingsView" class="modal-input">
            <option value="rich">{{ $gettext('Rich text') }}</option>
            <option value="plain">{{ $gettext('Plain text') }}</option>
            <option value="preview">{{ $gettext('Preview') }}</option>
          </select>
        </label>
        <label class="modal-label">
          {{ $gettext('Editor font') }}
          <select v-model="settingsFont" class="modal-input">
            <option value="default">{{ $gettext('Default') }}</option>
            <option value="sans">{{ $gettext('Sans') }}</option>
            <option value="serif">{{ $gettext('Serif') }}</option>
            <option value="mono">{{ $gettext('Mono') }}</option>
          </select>
        </label>
        <label class="modal-label">
          {{ $gettext('Font size') }}
          <select v-model.number="settingsFontSize" class="modal-input">
            <option v-for="n in [12, 13, 14, 15, 16, 17, 18, 20, 22]" :key="n" :value="n">
              {{ n }} px
            </option>
          </select>
        </label>
        <label class="modal-label">
          {{ $gettext('Notes folder') }}
          <input v-model="settingsNotesPath" type="text" class="modal-input" />
        </label>
        <label class="modal-label">
          {{ $gettext('File suffix') }}
          <input v-model="settingsFileSuffix" type="text" class="modal-input" />
        </label>
        <p v-if="settingsSaved" class="ok-msg">{{ $gettext('Settings saved') }}</p>
        <p v-else-if="settingsError" class="error-msg">{{ $gettext('Error') }}</p>
        <footer class="modal-actions">
          <button type="button" class="oc-button oc-button-outline" @click="showSettings = false">
            {{ $gettext('Close') }}
          </button>
          <button type="submit" class="oc-button oc-button-primary oc-button-filled">{{ $gettext('Save') }}</button>
        </footer>
      </form>
    </ModalDialog>

    <ModalDialog
      v-if="showShortcuts"
      :title="$gettext('Keyboard shortcuts')"
      @close="showShortcuts = false"
    >
      <dl class="shortcut-list">
        <div><dt><kbd>Ctrl</kbd> + <kbd>N</kbd></dt><dd>{{ $gettext('New note') }}</dd></div>
        <div><dt><kbd>Ctrl</kbd> + <kbd>F</kbd></dt><dd>{{ $gettext('Search notes') }}</dd></div>
        <div><dt><kbd>Ctrl</kbd> + <kbd>S</kbd></dt><dd>{{ $gettext('Save note') }}</dd></div>
        <div><dt><kbd>Ctrl</kbd> + <kbd>Z</kbd></dt><dd>{{ $gettext('Undo') }}</dd></div>
        <div><dt><kbd>Ctrl</kbd> + <kbd>Y</kbd></dt><dd>{{ $gettext('Redo') }}</dd></div>
        <div><dt><kbd>Esc</kbd></dt><dd>{{ $gettext('Back or exit zen mode') }}</dd></div>
      </dl>
    </ModalDialog>
  </nav>
</template>
