<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  AlignLeft,
  Eye,
  Minimize2,
  Paperclip,
  Star,
  Trash2,
  Type,
} from 'lucide-vue-next'
import {
  useTextEditor,
  TextEditorProvider,
  TextEditorContent,
  TextEditorToolbar,
} from '@opencloud-eu/web-pkg/editor'
import type { Note } from '../stores/notes'
import { state, toggleZenMode } from '../stores/notes'
import { useNotesApi } from '../composables/api'
import { useGettext } from 'vue3-gettext'
import ModalDialog from './ModalDialog.vue'

const { $gettext } = useGettext()

const saveClasses = computed(() => [
  'btn-save',
  'oc-button',
  'oc-button-primary',
  'oc-button-filled',
  'shadow-md',
  'hover:brightness-110',
  'duration-150',
  'ease-out',
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
const showConfirmDelete = ref(false)

const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const attachmentUrls = new Map<string, string>()

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

/* ----- WYSIWYG editor (host tiptap, markdown round-trip) ----- */
const editor = useTextEditor({
  contentType: 'markdown',
  modelValue: content,
  onUpdate: (c) => {
    content.value = c
  },
  placeholder: $gettext('Content'),
  ariaLabel: $gettext('Content'),
  // Images stay in Preview (proxy + attachments); source mode is driven by
  // the header "plain" toggle, not by the toolbar's own button.
  excludeActions: ['source-mode', 'image', 'image-url', 'image-upload', 'image-cloud'],
})

// Keep the header "rich/plain" toggle in sync with the editor's source mode.
watch(
  () => state.displayMode,
  (mode) => {
    editor.state.sourceMode.value = mode === 'plain'
  },
)

const isDirty = computed(
  () => title.value !== props.note.title || content.value !== props.note.content,
)

function currentContent(): string {
  return editor.getContent()
}

// Live dirty check for the note-switch / new-note guards in NotesApp: reads the
// editor content directly instead of the debounced `content` ref, so a switch
// right after typing cannot drop the last keystrokes.
function isDirtyNow(): boolean {
  return title.value !== props.note.title || currentContent() !== props.note.content
}

const categories = computed(() => {
  const set = new Set<string>()
  for (const n of state.notes) {
    if (n.category) set.add(n.category.split('/')[0])
  }
  for (const c of state.pendingCategories) set.add(c)
  return [...set].sort((a, b) => a.localeCompare(b))
})

function resetTo(n: Note) {
  revokeAttachments()
  title.value = n.title
  content.value = n.content
  editor.setContent(n.content)
  error.value = null
  showConfirmDelete.value = false
}

watch(() => props.note, resetTo)

// Clear the pending edits, used when the user chooses to discard them.
function discard() {
  resetTo(props.note)
}

/* ----- attachments ----- */
const ATTACHMENT_ESCAPE_RE = /[ ()%#?]/g

function encodeAttachmentPath(path: string): string {
  return path.replace(ATTACHMENT_ESCAPE_RE, (ch) => {
    const hex = ch.charCodeAt(0).toString(16).toUpperCase().padStart(2, '0')
    return `%${hex}`
  })
}

function decodeAttachmentPath(raw: string): string {
  const path = raw.startsWith('./') ? raw.slice(2) : raw
  try {
    return decodeURIComponent(path)
  } catch {
    return path
  }
}

function attachmentError(e: unknown): string {
  const status = (e as { response?: { status?: number } } | null)?.response?.status
  if (status === 413) {
    return $gettext('The file is too large. The maximum size is 32 MiB.')
  }
  return $gettext('Failed to attach the file.')
}

function openAttachmentPicker() {
  fileInput.value?.click()
}

function insertAttachmentRef(text: string) {
  const ed = editor.editor.value
  if (!ed) return
  // The host editor's Markdown extension accepts markdown strings via
  // insertContent({ contentType: 'markdown' }); that option type lives in the
  // @tiptap/markdown module augmentation, which is a transitive dep of web-pkg
  // and not part of this program's type graph.
  ed.chain()
    .focus()
    .insertContent(text, { contentType: 'markdown' } as any)
    .run()
}

async function onAttachmentSelected(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  uploading.value = true
  error.value = null
  try {
    const filename = await api.uploadAttachment(props.note.id, file)
    const target = encodeAttachmentPath(filename)
    const reference = file.type.startsWith('image/')
      ? `![${file.name}](${target})`
      : `[${file.name}](${target})`
    insertAttachmentRef(reference)
  } catch (err) {
    error.value = attachmentError(err)
  } finally {
    uploading.value = false
  }
}

/* ----- keyboard ----- */
function onWindowKeydown(e: KeyboardEvent) {
  const mod = e.ctrlKey || e.metaKey
  if (mod && e.key.toLowerCase() === 's') {
    e.preventDefault()
    void save()
  }
}

/* ----- api actions ----- */
async function save() {
  saving.value = true
  error.value = null
  try {
    const current = editor.getContent()
    const payload: Partial<Note> = {}
    if (title.value !== props.note.title) payload.title = title.value
    if (current !== props.note.content) payload.content = current
    if (!Object.keys(payload).length) return
    const resp = await api.updateNote(props.note.id, payload, props.note.etag)
    const updated = (resp as { data?: Note }).data
    Object.assign(props.note, payload, {
      etag: updated?.etag ?? props.note.etag,
      modified: updated?.modified ?? Date.now() / 1000,
    })
    content.value = current
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

function confirmDelete() {
  showConfirmDelete.value = false
  void remove()
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

onMounted(() => window.addEventListener('keydown', onWindowKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onWindowKeydown))
onBeforeUnmount(revokeAttachments)

function onBeforeUnload(e: BeforeUnloadEvent) {
  e.preventDefault()
  e.returnValue = ''
}

watch(isDirty, (dirty) => {
  if (dirty) window.addEventListener('beforeunload', onBeforeUnload)
  else window.removeEventListener('beforeunload', onBeforeUnload)
})

onBeforeUnmount(() => window.removeEventListener('beforeunload', onBeforeUnload))

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

function revokeAttachments() {
  for (const url of attachmentUrls.values()) URL.revokeObjectURL(url)
  attachmentUrls.clear()
}

async function resolveAttachments(html: string): Promise<string> {
  const doc = new DOMParser().parseFromString(html, 'text/html')
  const nodes = [
    ...doc.querySelectorAll<HTMLImageElement>('img[src^=".attachments."]'),
    ...doc.querySelectorAll<HTMLAnchorElement>('a[href^=".attachments."]'),
  ]
  await Promise.all(
    nodes.map(async (node) => {
      const raw =
        node.tagName === 'IMG' ? node.getAttribute('src') : node.getAttribute('href')
      if (!raw) return
      const path = decodeAttachmentPath(raw)
      try {
        let url = attachmentUrls.get(path)
        if (!url) {
          const blob = await api.getAttachmentBlob(props.note.id, path)
          url = URL.createObjectURL(blob)
          attachmentUrls.set(path, url)
        }
        if (node.tagName === 'IMG') node.setAttribute('src', url)
        else node.setAttribute('href', url)
      } catch {
        // keep the relative reference as-is on failure
      }
    }),
  )
  return doc.body.innerHTML
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
  const html = DOMPurify.sanitize(md.render(text), { ADD_ATTR: ['target'] })
  return resolveAttachments(html)
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
  ([, p]) => {
    if (!p) return
    window.clearTimeout(previewTimer)
    previewTimer = window.setTimeout(() => {
      void buildPreview(currentContent()).then((h) => {
        previewHtml.value = h
      })
    }, 250)
  },
  { immediate: true },
)

defineExpose({ isDirty: isDirtyNow, save, discard })
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
        v-if="!isPreview"
        class="oc-button oc-button-raw icon-btn"
        :disabled="uploading"
        :aria-label="$gettext('Attach file')"
        :title="$gettext('Attach file')"
        @click="openAttachmentPicker"
      >
        <Paperclip :size="18" />
      </button>
      <input ref="fileInput" type="file" hidden @change="onAttachmentSelected" />
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
        @click="showConfirmDelete = true"
      >
        <Trash2 :size="18" />
      </button>
    </header>

    <main class="editor-body">
      <input v-model="title" :placeholder="$gettext('Title')" class="title-input" />
      <div v-show="!isPreview" class="editor-wrap">
        <TextEditorProvider :editor="editor">
          <TextEditorToolbar />
          <div class="editor-content-wrap" :style="editorStyle">
            <TextEditorContent />
          </div>
        </TextEditorProvider>
      </div>
      <div v-show="isPreview" :style="editorStyle" class="preview-pane" v-html="previewHtml" />
      <p v-if="error" class="error-msg">{{ error }}</p>
    </main>

    <ModalDialog
      v-if="showConfirmDelete"
      :title="$gettext('Delete note')"
      @close="showConfirmDelete = false"
    >
      <p>{{ $gettext('Are you sure you want to delete this note? This action cannot be undone.') }}</p>
      <footer class="modal-actions">
        <button type="button" class="oc-button oc-button-outline" @click="showConfirmDelete = false">
          {{ $gettext('Cancel') }}
        </button>
        <button type="button" class="oc-button oc-button-filled btn-danger" @click="confirmDelete">
          {{ $gettext('Delete') }}
        </button>
      </footer>
    </ModalDialog>
  </div>
</template>
