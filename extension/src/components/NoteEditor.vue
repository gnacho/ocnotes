<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  AlignLeft,
  Bold,
  Code,
  Eye,
  Heading2,
  Italic,
  Link2,
  List,
  ListOrdered,
  Minimize2,
  Paperclip,
  Redo2,
  Smile,
  Star,
  Strikethrough,
  Table,
  TextQuote,
  Trash2,
  Type,
  Undo2,
} from 'lucide-vue-next'
import type { Note } from '../stores/notes'
import { state, toggleZenMode } from '../stores/notes'
import { useNotesApi } from '../composables/api'
import { useIsDark } from '../composables/theme'

const isDark = useIsDark()

const saveClasses = computed(() => [
  'btn-save',
  'shadow-md',
  'hover:brightness-110',
  'duration-150',
  'ease-out',
  isDark.value
    ? 'oc-button-filled oc-button-primary-container !bg-gradient-to-r !from-role-secondary-container !to-role-primary-container'
    : 'oc-button-filled oc-button-primary !bg-gradient-to-r !from-role-secondary !to-role-primary',
])

const emit = defineEmits<{
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
const showEmoji = ref(false)

const ta = ref<HTMLTextAreaElement | null>(null)

const isPreview = computed(() => state.displayMode === 'preview')

const FONT_STACKS: Record<string, string> = {
  default: 'inherit',
  sans: "system-ui, -apple-system, 'Segoe UI', sans-serif",
  serif: "Georgia, 'Times New Roman', serif",
  mono: "'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, monospace",
}

const editorStyle = computed(() => ({
  fontFamily: FONT_STACKS[state.editorFont] ?? 'inherit',
  fontSize: `${state.editorFontSize}px`,
}))

const categories = computed(() => {
  const set = new Set<string>()
  for (const n of state.notes) {
    if (n.category) set.add(n.category.split('/')[0])
  }
  for (const c of state.pendingCategories) set.add(c)
  return [...set].sort((a, b) => a.localeCompare(b))
})

/* ----- undo/redo history for the content ----- */
const history = ref<string[]>([props.note.content])
let hIndex = 0
let suppressHistory = false
let pushTimer: number | undefined

function pushHistory(snapshot: string) {
  if (snapshot === history.value[hIndex]) return
  history.value = history.value.slice(0, hIndex + 1)
  history.value.push(snapshot)
  if (history.value.length > 100) history.value.shift()
  hIndex = history.value.length - 1
  canUndo.value = hIndex > 0
  canRedo.value = hIndex < history.value.length - 1
}

const canUndo = ref(false)
const canRedo = ref(false)

watch(content, (v) => {
  if (suppressHistory) return
  window.clearTimeout(pushTimer)
  pushTimer = window.setTimeout(() => pushHistory(v), 500)
})

function undo() {
  if (hIndex <= 0) return
  hIndex--
  suppressHistory = true
  content.value = history.value[hIndex]
  canUndo.value = hIndex > 0
  canRedo.value = hIndex < history.value.length - 1
  void nextTick(() => {
    suppressHistory = false
    ta.value?.focus()
  })
}

function redo() {
  if (hIndex >= history.value.length - 1) return
  hIndex++
  suppressHistory = true
  content.value = history.value[hIndex]
  canUndo.value = hIndex > 0
  canRedo.value = hIndex < history.value.length - 1
  void nextTick(() => {
    suppressHistory = false
    ta.value?.focus()
  })
}

watch(
  () => props.note,
  (n) => {
    title.value = n.title
    content.value = n.content
    history.value = [n.content]
    hIndex = 0
    canUndo.value = false
    canRedo.value = false
    error.value = null
    showEmoji.value = false
  },
)

/* ----- markdown editing helpers ----- */
function applyEdit(next: string, selStart: number, selEnd: number) {
  pushHistory(content.value)
  window.clearTimeout(pushTimer)
  content.value = next
  void nextTick(() => {
    const el = ta.value
    if (el) {
      el.selectionStart = selStart
      el.selectionEnd = selEnd
      el.focus()
    }
  })
}

function wrap(before: string, after: string, placeholder: string) {
  const el = ta.value
  if (!el) return
  const { selectionStart: s, selectionEnd: e, value } = el
  const sel = value.slice(s, e) || placeholder
  applyEdit(
    value.slice(0, s) + before + sel + after + value.slice(e),
    s + before.length,
    s + before.length + sel.length,
  )
}

function toggleLinePrefix(prefix: string) {
  const el = ta.value
  if (!el) return
  const { selectionStart: s, selectionEnd: e, value } = el
  const ls = value.lastIndexOf('\n', s - 1) + 1
  const nl = value.indexOf('\n', e)
  const le = nl === -1 ? value.length : nl
  const block = value.slice(ls, le)
  const lines = block.split('\n')
  const numbered = /^\d+\.\s/.test(prefix)
  const all = lines.every((l) =>
    numbered ? /^\d+\.\s/.test(l) : l.startsWith(prefix),
  )
  const next = lines
    .map((l, i) => {
      if (all) {
        if (numbered) return l.replace(/^\d+\.\s/, '')
        return l.slice(prefix.length)
      }
      return numbered ? `${i + 1}. ${l}` : prefix + l
    })
    .join('\n')
  applyEdit(value.slice(0, ls) + next + value.slice(le), ls, ls + next.length)
}

function insertBlock(text: string) {
  const el = ta.value
  if (!el) return
  const { selectionStart: s, selectionEnd: e, value } = el
  const atLineStart = s === 0 || value[s - 1] === '\n'
  const block = atLineStart ? text : `\n${text}`
  applyEdit(
    value.slice(0, s) + block + value.slice(e),
    s + block.length,
    s + block.length,
  )
}

function insertAtCursor(text: string) {
  const el = ta.value
  if (!el) return
  const { selectionStart: s, selectionEnd: e, value } = el
  applyEdit(
    value.slice(0, s) + text + value.slice(e),
    s + text.length,
    s + text.length,
  )
}

const TABLE = '| Column | Column |\n| --- | --- |\n| Text | Text |'
const EMOJIS = [
  '😀', '😂', '🙂', '😍', '🤔', '😎', '🥳', '🙄',
  '😴', '😢', '😭', '🥺', '😡', '🤯', '😇', '🥰',
  '👍', '👏', '🙏', '💪', '✅', '⚠️', '❌', '⭐',
  '❤️', '🔥', '✨', '🎉', '💡', '🚀', '📌', '🔗',
  '📎', '🤖', '🐱', '🐶', '☕', '🍕', '⚽', '🌙',
]

/* ----- keyboard ----- */
function onKeydown(e: KeyboardEvent) {
  const mod = e.ctrlKey || e.metaKey
  if (!mod) return
  const k = e.key.toLowerCase()
  if (k === 's') {
    e.preventDefault()
    void save()
  } else if (k === 'z' && !e.shiftKey) {
    e.preventDefault()
    undo()
  } else if (k === 'y' || (k === 'z' && e.shiftKey)) {
    e.preventDefault()
    redo()
  }
}

/* ----- api actions ----- */
async function save() {
  saving.value = true
  error.value = null
  try {
    const payload: Partial<Note> = {}
    if (title.value !== props.note.title) payload.title = title.value
    if (content.value !== props.note.content) payload.content = content.value
    if (!Object.keys(payload).length) return
    const resp = await api.updateNote(props.note.id, payload, props.note.etag)
    const updated = (resp as { data?: Note }).data
    Object.assign(props.note, payload, {
      etag: updated?.etag ?? props.note.etag,
      modified: updated?.modified ?? Date.now() / 1000,
    })
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
  error.value = null
  try {
    const resp = await api.updateNote(props.note.id, { favorite: fav }, props.note.etag)
    const updated = (resp as { data?: Note }).data
    props.note.favorite = fav
    if (updated?.etag) props.note.etag = updated.etag
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  }
}

async function changeCategory(cat: string) {
  error.value = null
  try {
    const resp = await api.updateNote(
      props.note.id,
      { category: cat },
      props.note.etag,
    )
    const updated = (resp as { data?: Note }).data
    props.note.category = cat
    if (updated?.etag) props.note.etag = updated.etag
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  }
}

function closeEmoji() {
  showEmoji.value = false
}

onMounted(() => window.addEventListener('click', closeEmoji))
onBeforeUnmount(() => window.removeEventListener('click', closeEmoji))

import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'

const md = new MarkdownIt({
  html: true,
  linkify: true,
  typographer: true,
  breaks: true,
})

DOMPurify.addHook('afterSanitizeAttributes', (node) => {
  if (node.tagName === 'A') {
    node.setAttribute('target', '_blank')
    node.setAttribute('rel', 'noopener noreferrer')
  }
})

const IMG_URL_RE = [
  /!\[[^\]]*\]\((https?:\/\/[^)\s]+)[^)]*\)/g,
  /src=["'](https?:\/\/[^"']+)["']/g,
]

function extractImageUrls(src: string): string[] {
  const urls = new Set<string>()
  for (const re of IMG_URL_RE) {
    for (const m of src.matchAll(re)) {
      if (m[1].includes('/api/v1/img?')) continue
      urls.add(m[1])
    }
  }
  return [...urls].slice(0, 32)
}

async function buildPreview(src: string): Promise<string> {
  let text = src
  const urls = extractImageUrls(src)
  if (urls.length) {
    try {
      const map = await api.signImages(urls)
      for (const [u, proxied] of Object.entries(map)) {
        text = text.split(u).join(proxied)
      }
    } catch {
      // sin firma las imágenes no cargarán, pero el resto del markdown sí
    }
  }
  return DOMPurify.sanitize(md.render(text), { ADD_ATTR: ['target'] })
}

const defaultLink =
  md.renderer.rules.link_open ||
  ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))
md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  token.attrSet('target', '_blank')
  token.attrSet('rel', 'noopener noreferrer')
  return defaultLink(tokens, idx, options, env, self)
}

const previewHtml = ref('')

let previewTimer: number | undefined
watch(
  [content, isPreview],
  ([c, p]) => {
    if (!p) return
    window.clearTimeout(previewTimer)
    previewTimer = window.setTimeout(() => {
      void buildPreview(c).then((h) => {
        previewHtml.value = h
      })
    }, 250)
  },
  { immediate: true },
)
</script>

<template>
  <div class="note-editor">
    <header class="editor-header">
      <select
        class="category-select"
        :value="props.note.category?.split('/')[0] || ''"
        :aria-label="$gettext('Category')"
        @change="changeCategory(($event.target as HTMLSelectElement).value)"
      >
        <option value="">{{ $gettext('Uncategorized') }}</option>
        <option v-for="cat in categories" :key="cat" :value="cat">{{ cat }}</option>
      </select>
      <span class="editor-spacer" />
      <button
        class="oc-button oc-button-raw icon-btn"
        :class="{ starred: props.note.favorite }"
        :aria-label="props.note.favorite ? $gettext('Unstar') : $gettext('Star')"
        @click="toggleFavorite"
      >
        <Star :size="18" />
      </button>
      <button
        :class="['oc-button oc-button-raw icon-btn', { active: state.displayMode === 'rich' }]"
        :aria-label="$gettext('Rich text')"
        :title="$gettext('Rich text')"
        @click="state.displayMode = 'rich'"
      >
        <Type :size="18" />
      </button>
      <button
        :class="['oc-button oc-button-raw icon-btn', { active: state.displayMode === 'plain' }]"
        :aria-label="$gettext('Plain text')"
        :title="$gettext('Plain text')"
        @click="state.displayMode = 'plain'"
      >
        <AlignLeft :size="18" />
      </button>
      <button
        :class="['oc-button oc-button-raw icon-btn', { active: isPreview }]"
        :aria-label="$gettext('Preview')"
        :title="$gettext('Preview')"
        @click="state.displayMode = 'preview'"
      >
        <Eye :size="18" />
      </button>
      <button
        v-if="state.zenMode"
        class="oc-button oc-button-raw icon-btn"
        :aria-label="$gettext('Exit zen mode')"
        :title="$gettext('Exit zen mode')"
        @click="toggleZenMode()"
      >
        <Minimize2 :size="18" />
      </button>
      <button :class="saveClasses" :disabled="saving" @click="save">
        {{ saving ? '…' : $gettext('Save') }}
      </button>
      <button
        class="oc-button oc-button-raw icon-btn danger"
        :aria-label="$gettext('Delete')"
        @click="remove"
      >
        <Trash2 :size="18" />
      </button>
    </header>

    <div v-if="!isPreview" class="md-toolbar">
      <button class="oc-button oc-button-raw md-btn" :disabled="!canUndo" :title="$gettext('Undo')" @click="undo">
        <Undo2 :size="16" />
      </button>
      <button class="oc-button oc-button-raw md-btn" :disabled="!canRedo" :title="$gettext('Redo')" @click="redo">
        <Redo2 :size="16" />
      </button>
      <span class="md-sep" />
      <button class="oc-button oc-button-raw md-btn" :title="$gettext('Bold')" @click="wrap('**', '**', 'text')">
        <Bold :size="16" />
      </button>
      <button class="oc-button oc-button-raw md-btn" :title="$gettext('Italic')" @click="wrap('*', '*', 'text')">
        <Italic :size="16" />
      </button>
      <button
        class="oc-button oc-button-raw md-btn"
        :title="$gettext('Strikethrough')"
        @click="wrap('~~', '~~', 'text')"
      >
        <Strikethrough :size="16" />
      </button>
      <button class="oc-button oc-button-raw md-btn" :title="$gettext('Heading')" @click="toggleLinePrefix('## ')">
        <Heading2 :size="16" />
      </button>
      <span class="md-sep" />
      <button class="oc-button oc-button-raw md-btn" :title="$gettext('Bulleted list')" @click="toggleLinePrefix('- ')">
        <List :size="16" />
      </button>
      <button
        class="oc-button oc-button-raw md-btn"
        :title="$gettext('Numbered list')"
        @click="toggleLinePrefix('1. ')"
      >
        <ListOrdered :size="16" />
      </button>
      <button class="oc-button oc-button-raw md-btn" :title="$gettext('Quote')" @click="toggleLinePrefix('> ')">
        <TextQuote :size="16" />
      </button>
      <button class="oc-button oc-button-raw md-btn" :title="$gettext('Code')" @click="wrap('`', '`', 'code')">
        <Code :size="16" />
      </button>
      <span class="md-sep" />
      <button class="oc-button oc-button-raw md-btn" :title="$gettext('Table')" @click="insertBlock(TABLE)">
        <Table :size="16" />
      </button>
      <button
        class="oc-button oc-button-raw md-btn"
        :title="$gettext('Link')"
        @click="wrap('[', '](url)', 'text')"
      >
        <Link2 :size="16" />
      </button>
      <button
        class="oc-button oc-button-raw md-btn"
        disabled
        :title="$gettext('Attachments are not available yet')"
      >
        <Paperclip :size="16" />
      </button>
      <div class="emoji-wrap">
        <button
          class="oc-button oc-button-raw md-btn"
          :title="$gettext('Emoji')"
          @click.stop="showEmoji = !showEmoji"
        >
          <Smile :size="16" />
        </button>
        <div v-if="showEmoji" class="emoji-pop" @click.stop>
          <button
            v-for="e in EMOJIS"
            :key="e"
            class="emoji-item"
            @click="insertAtCursor(e)"
          >
            {{ e }}
          </button>
        </div>
      </div>
    </div>

    <main class="editor-body">
      <input v-model="title" :placeholder="$gettext('Title')" class="title-input" />
      <textarea
        v-if="!isPreview"
        ref="ta"
        v-model="content"
        :style="editorStyle"
        :placeholder="$gettext('Content')"
        class="content-textarea"
        @keydown="onKeydown"
      />
      <div v-else :style="editorStyle" class="preview-pane" v-html="previewHtml" />
      <p v-if="error" class="error-msg">{{ error }}</p>
    </main>
  </div>
</template>
