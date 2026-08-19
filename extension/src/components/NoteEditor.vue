<script setup lang="ts">
import { ref } from 'vue'
import type { Note } from '../stores/notes'
import { state } from '../stores/notes'

const emit = defineEmits<{
  back: []
}>()

interface Props {
  note: Note
}
defineProps<Props>()

const title = ref(props.note.title)
const content = ref(props.note.content)
</script>

<template>
  <div class="note-editor">
    <header class="editor-header">
      <button class="btn-back" @click="emit('back')">← Back</button>
      <select v-model="state.displayMode" aria-label="Display mode">
        <option value="rich">Rich text</option>
        <option value="plain">Plain text</option>
        <option value="preview">Preview</option>
      </select>
      <button class="btn-save" @click="/* save */">Save</button>
      <button class="btn-delete">Delete</button>
    </header>
    <main class="editor-body">
      <input v-model="title" placeholder="Title" class="title-input" />
      <textarea v-if="state.displayMode !== 'preview'" v-model="content" placeholder="Content..." class="content-textarea" />
      <div v-else class="preview-pane" v-html="content.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>').replace(/\n/g, '<br>')" />
    </main>
  </div>
</template>
