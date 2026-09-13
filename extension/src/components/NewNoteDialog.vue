<script setup lang="ts">
import { onMounted, ref } from 'vue'
import ModalDialog from './ModalDialog.vue'

const emit = defineEmits<{
  create: [title: string]
  close: []
}>()

const title = ref('New note')
const input = ref<HTMLInputElement | null>(null)

function submit() {
  const trimmed = title.value.trim()
  if (!trimmed) return
  emit('create', trimmed)
}

onMounted(() => input.value?.focus())
</script>

<template>
  <ModalDialog :title="$gettext('New note')" @close="emit('close')">
    <form class="modal-form" @submit.prevent="submit">
      <input
        ref="input"
        v-model="title"
        type="text"
        class="modal-input"
        :placeholder="$gettext('Note title')"
        :aria-label="$gettext('Note title')"
      />
      <footer class="modal-actions">
        <button type="button" class="oc-button oc-button-outline" @click="emit('close')">
          {{ $gettext('Cancel') }}
        </button>
        <button type="submit" class="oc-button oc-button-primary oc-button-filled">
          {{ $gettext('Create') }}
        </button>
      </footer>
    </form>
  </ModalDialog>
</template>
