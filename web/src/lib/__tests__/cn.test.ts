import { describe, it, expect } from 'vitest'
import { cn } from '../cn'

describe('cn', () => {
  it('merges classes correctly', () => {
    expect(cn('foo', 'bar')).toBe('foo bar')
  })

  it('handles conditional classes', () => {
    expect(cn('base', false && 'not-included', 'included')).toBe('base included')
    expect(cn('base', true && 'yes')).toBe('base yes')
  })

  it('deduplicates conflicting Tailwind classes', () => {
    // twMerge resolves conflicts: last wins
    expect(cn('p-2', 'p-4')).toBe('p-4')
    expect(cn('text-sm', 'text-lg')).toBe('text-lg')
  })

  it('handles undefined and null gracefully', () => {
    expect(cn('foo', undefined, null as unknown as string, 'bar')).toBe('foo bar')
  })

  it('returns empty string with no arguments', () => {
    expect(cn()).toBe('')
  })
})
