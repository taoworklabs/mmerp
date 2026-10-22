const COMBINING_MARKS = /[̀-ͯ]/g

// Vietnamese đ/Đ don't decompose under NFD like the other diacritics, so they're mapped by hand.
function normalize(input: string): string {
  return input.replace(/đ/g, 'd').replace(/Đ/g, 'D').normalize('NFD').replace(COMBINING_MARKS, '').toLowerCase()
}

export function matchesQuery(label: string, query: string): boolean {
  const words = normalize(query).trim().split(/\s+/).filter(Boolean)
  if (words.length === 0) return true
  const target = normalize(label)
  return words.every((word) => target.includes(word))
}
