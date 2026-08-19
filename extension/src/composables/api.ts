import { useClientService } from '@opencloud-eu/web-pkg'
import type { Note } from '../stores/notes'

const BASE = '/index.php/apps/notes/api/v1'

export interface Settings {
  notesPath: string
  fileSuffix: string
}

export function useNotesApi() {
  const client = useClientService()

  async function get<T>(path: string, params: Record<string, string> = {}): Promise<T> {
    const usp = new URLSearchParams(params)
    const qs = usp.toString()
    const { data } = await client.httpAuthenticated.get(`${BASE}${path}${qs ? `?${qs}` : ''}`)
    return data as T
  }

  async function post<T>(path: string, body?: unknown): Promise<T> {
    const { data } = await client.httpAuthenticated.post(`${BASE}${path}`, body ?? {})
    return data as T
  }

  async function put<T>(path: string, body?: unknown): Promise<T> {
    const { data } = await client.httpAuthenticated.put(`${BASE}${path}`, body ?? {})
    return data as T
  }

  return {
    listNotes: (category?: string, exclude?: string[], pruneBefore?: number) => {
      const params: Record<string, string> = {}
      if (category) params.category = category
      if (exclude?.length) params.exclude = exclude.join(',')
      if (pruneBefore) params.pruneBefore = String(pruneBefore)
      return get<Note[]>('/notes', params)
    },
    getNote: (id: number) => get<Note>(`/notes/${id}`),
    createNote: (title: string, content: string, category: string) =>
      post<Note>('/notes', { title, content, category }),
    updateNote: (id: number, payload: Partial<Note>, etag: string) =>
      client.httpAuthenticated.put(`${BASE}/notes/${id}`, payload, {
        headers: { 'If-Match': `"${etag}"` },
      }),
    deleteNote: (id: number) => client.httpAuthenticated.delete(`${BASE}/notes/${id}`),
    getSettings: () => get<Settings>('/settings'),
    updateSettings: (updates: Partial<Settings>) => put<Settings>('/settings', updates),
  }
}
