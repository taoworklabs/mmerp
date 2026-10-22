import type * as React from 'react'

// Keeps the caller's ref working while the component holds its own handle on the same node. Returns
// a cleanup so React 19 detaches each ref properly (running its own cleanup, or calling it with
// null) instead of discarding what a callback ref returned, which would leak e.g. an observer.
export function mergeRefs<T>(...refs: (React.Ref<T> | undefined)[]): React.RefCallback<T> {
  return (node) => {
    const cleanups = refs.map((ref) => {
      if (typeof ref === 'function') {
        const cleanup = ref(node)
        return typeof cleanup === 'function' ? cleanup : () => ref(null)
      }
      if (ref) {
        ref.current = node
        return () => {
          ref.current = null
        }
      }
      return undefined
    })
    return () => {
      for (const cleanup of cleanups) cleanup?.()
    }
  }
}
