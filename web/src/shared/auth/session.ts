import { QueryClient } from '@tanstack/react-query'

// The cache belongs to one session, one locale and one authz version; ending any of them reloads the page.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      // API errors are answers, not glitches; only network failures are retried.
      retry: (count, err) => count < 2 && !('status' in err),
    },
  },
})

let authzVersion: string | null = null
let ending = false

// Aborts every API request of this page when the session ends, not only the cached queries.
export const sessionSignal = new AbortController()

const channel = new BroadcastChannel('mmerp.session')
// Another tab started or ended the session: reload, never re-announce.
channel.onmessage = (e) => {
  if (e.data === 'session_ended' || e.data === 'session_started') location.reload()
}

export function startSession(version: string) {
  authzVersion = version
}

export function announceSessionStarted() {
  channel.postMessage('session_started')
}

function stopEverything() {
  sessionSignal.abort()
  void queryClient.cancelQueries()
  queryClient.clear()
}

// endSession runs once per page: the reload lands anonymous, where a 401 only shows the login screen.
export function endSession() {
  if (ending || authzVersion === null) return
  ending = true
  stopEverything()
  channel.postMessage('session_ended')
  location.assign(`/login?next=${encodeURIComponent(location.pathname + location.search)}`)
}

// A changed authz version means cached data may no longer be visible to this user.
export function onAuthzVersion(version: string | null) {
  if (ending || authzVersion === null || version === null || version === authzVersion) return
  ending = true
  stopEverything()
  location.reload()
}
