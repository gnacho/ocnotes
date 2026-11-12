export interface DirtyBaseline {
  title: string
  content: string
}

export function createDirtyBaseline(title: string, content: string): DirtyBaseline {
  return { title, content }
}

export function hasUnsavedChanges(
  baseline: DirtyBaseline,
  title: string,
  content: string,
): boolean {
  return title !== baseline.title || content !== baseline.content
}
