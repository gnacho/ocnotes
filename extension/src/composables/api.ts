import type { Note } from '../stores/notes'

const BASE = '/index.php/apps/notes/api/v1/'

export async function fetchNotes(
  category?: string,
  exclude?: string[],
  pruneBefore?: number,
): Promise<Note[]> {
  const params = new URLSearchParams()
  if (category) params.set('category', category)
  if (exclude) params.set('exclude', exclude.join(','))
  if (pruneBefore) params.set('pruneBefore', String(pruneBefore))

  const res = await fetch(`${BASE}notes?${params}`)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}

export async function fetchNote(id: number): Promise<Note> {
  const res = await fetch(`${BASE}notes/${id}`)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}

export async function createNote(payload: Partial<Note>): Promise<Note> {
  const res = await fetch(`${BASE}notes`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      title: payload.title || 'New note',
      content: payload.content || '',
      category: payload.category || '',
    }),
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}

export async function updateNote(
  id: number,
  payload: Partial<Note>,
  etag: string,
): Promise<Note> {
  const fields: Record<string, unknown> = {}
  if (payload.title !== undefined) fields.title = payload.title
  if (payload.content !== undefined) fields.content = payload.content
  if (payload.category !== undefined) fields.category = payload.category
  if (payload.favorite !== undefined) fields.favorite = payload.favorite

  const res = await fetch(`${BASE}notes/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      'If-Match': `"${etag}"`,
    },
    body: JSON.stringify(fields),
  })
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    const err: Error & { note?: Note } = new Error(data.error || `HTTP ${res.status}`)
    if (res.status === 412 && data.id) {
      (err as any).note = data
    }
    throw err
  }
  return res.json()
}

export async function deleteNote(id: number): Promise<void> {
  const res = await fetch(`${BASE}notes/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
}

export async function fetchSettings(): Promise<{ notesPath: string; fileSuffix: string }> {
  const res = await fetch(`${BASE}settings`)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}

export async function updateSettings(
  updates: Record<string, string>,
): Promise<Record<string, string>> {
  const res = await fetch(`${BASE}settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}
