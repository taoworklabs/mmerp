// A mention is @login; a trailing dot or comma ends the sentence, not the login. The server reads it the same way.
export const mentionPattern = /@([\p{L}\p{N}_.-]*[\p{L}\p{N}_-])/gu

// splitMentions cuts a comment into text and the logins it mentions, in order.
export function splitMentions(body: string): { text: string; mention: boolean }[] {
  const out: { text: string; mention: boolean }[] = []
  let at = 0
  for (const m of body.matchAll(mentionPattern)) {
    if (m.index > at) out.push({ text: body.slice(at, m.index), mention: false })
    out.push({ text: m[0], mention: true })
    at = m.index + m[0].length
  }
  if (at < body.length) out.push({ text: body.slice(at), mention: false })
  return out
}

// mentionAt is the @word being typed just before the caret, if any.
export function mentionAt(body: string, caret: number): { start: number; query: string } | null {
  const query = /(?:^|\s)@([\p{L}\p{N}_.-]*)$/u.exec(body.slice(0, caret))?.[1]
  return query === undefined ? null : { start: caret - query.length - 1, query }
}
