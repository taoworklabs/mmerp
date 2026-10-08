// safeNext keeps the post-login redirect on this site. The browser's own URL parser
// decides, since it strips tabs and newlines that a prefix check would miss.
export function safeNext(next: string | null, origin = location.origin): string | null {
  if (!next?.startsWith('/')) return null
  let url: URL
  try {
    url = new URL(next, origin)
  } catch {
    return null
  }
  if (url.origin !== origin) return null
  // Dot segments can leave "//host" in the pathname, which the caller would read as another site.
  const path = url.pathname.replace(/^\/+/, '/')
  return path.startsWith('/login') ? '/' : path + url.search + url.hash
}

// nextFromLocation is the safe ?next= of the current page.
export function nextFromLocation(): string | null {
  return safeNext(new URLSearchParams(location.search).get('next'))
}
