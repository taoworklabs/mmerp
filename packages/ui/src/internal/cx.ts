// Joins class names, dropping falsy ones; no dependency for one line.
export const cx = (...parts: (string | false | null | undefined)[]): string => parts.filter(Boolean).join(' ')
