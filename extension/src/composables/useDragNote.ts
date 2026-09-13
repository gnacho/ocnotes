import type { Note } from '../stores/notes'
import { state } from '../stores/notes'
import { useNotesApi } from './api'

export function useDragNote() {
  const api = useNotesApi()

  function topLevel(note: Note): string {
    return note.category ? note.category.split('/')[0] : ''
  }

  function isRealChange(note: Note, target: string): boolean {
    return topLevel(note) !== target
  }

  function onDragStart(e: DragEvent, note: Note) {
    state.draggedNoteId = note.id
    state.dropError = null
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move'
      e.dataTransfer.setData('text/plain', String(note.id))
    }
  }

  function onDragEnd() {
    state.draggedNoteId = null
    state.dragOverCategory = null
  }

  function onDragOverCategory(e: DragEvent, cat: string) {
    if (!state.draggedNoteId) return
    const note = state.notes.find((n) => n.id === state.draggedNoteId)
    if (!note || !isRealChange(note, cat)) return
    e.preventDefault()
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
    state.dragOverCategory = cat
  }

  function onDragLeaveCategory(e: DragEvent) {
    const target = e.currentTarget as HTMLElement | null
    const related = e.relatedTarget as Node | null
    if (target && related && target.contains(related)) return
    state.dragOverCategory = null
  }

  async function onDropOnCategory(e: DragEvent, cat: string) {
    e.preventDefault()
    const id = state.draggedNoteId
    state.draggedNoteId = null
    state.dragOverCategory = null
    if (!id) return
    const note = state.notes.find((n) => n.id === id)
    if (!note || !isRealChange(note, cat)) return
    try {
      const resp = await api.updateNote(note.id, { category: cat }, note.etag)
      const updated = (resp as { data?: Note }).data
      note.category = cat
      if (updated?.etag) note.etag = updated.etag
      state.dropError = null
    } catch (e) {
      state.dropError = e instanceof Error ? e.message : String(e)
    }
  }

  return {
    onDragStart,
    onDragEnd,
    onDragOverCategory,
    onDragLeaveCategory,
    onDropOnCategory,
    isRealChange,
  }
}
