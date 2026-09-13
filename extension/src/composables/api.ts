import { useClientService } from '@opencloud-eu/web-pkg'
import type { Note } from '../stores/notes'

const BASE = '/index.php/apps/notes/api/v1'
const BASE14 = '/index.php/apps/notes/api/v1.4'

export interface Settings {
  notesPath: string
  fileSuffix: string
  editorFont?: string
  editorFontSize?: string
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
    signImages: async (urls: string[]): Promise<Record<string, string>> => {
      if (!urls.length) return {}
      const { data } = await client.httpAuthenticated.post(`${BASE}/img/sign`, { urls })
      return (data as { signed?: Record<string, string> }).signed ?? {}
    },
    uploadAttachment: async (noteId: number, file: File): Promise<string> => {
      const form = new FormData()
      form.append('file', file)
      const { data } = await client.httpAuthenticated.post<{ filename: string }>(
        `${BASE14}/attachment/${noteId}`,
        form,
      )
      return data.filename
    },
    getAttachmentBlob: async (noteId: number, path: string): Promise<Blob> => {
      const { data } = await client.httpAuthenticated.get<Blob>(`${BASE14}/attachment/${noteId}`, {
        params: { path },
        responseType: 'blob',
      })
      return data
    },
  }
}
