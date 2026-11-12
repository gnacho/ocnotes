import { describe, expect, it } from 'vitest'
import {
  createDirtyBaseline,
  hasUnsavedChanges,
} from './dirty'

describe('dirty baseline', () => {
  it('does not mark an untouched note dirty when the editor normalizes markdown', () => {
    const stored = '# Weekend hiking plan\n\n**Route:** La Pedriza loop, ~12 km.\n'
    const serialized = '# Weekend hiking plan\n\n**Route:** La Pedriza loop, \\~12 km.\n\n'
    const baseline = createDirtyBaseline('Weekend hiking plan', serialized)

    expect(serialized).not.toBe(stored)
    expect(hasUnsavedChanges(baseline, 'Weekend hiking plan', serialized)).toBe(false)
  })

  it('detects a title edit', () => {
    const baseline = createDirtyBaseline('Original title', 'Body')
    expect(hasUnsavedChanges(baseline, 'Changed title', 'Body')).toBe(true)
  })

  it('detects a content edit', () => {
    const baseline = createDirtyBaseline('Title', 'Body')
    expect(hasUnsavedChanges(baseline, 'Title', 'Changed body')).toBe(true)
  })

  it('becomes clean again when the content returns to the baseline', () => {
    const baseline = createDirtyBaseline('Title', 'Body')
    expect(hasUnsavedChanges(baseline, 'Title', 'Changed body')).toBe(true)
    expect(hasUnsavedChanges(baseline, 'Title', 'Body')).toBe(false)
  })
})
