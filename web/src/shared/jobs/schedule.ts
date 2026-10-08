// pollDelay is how long to wait before asking about a job again: every 2 s while it is
// fresh, then every 10 s, so a long job does not keep the server busy.
export function pollDelay(elapsedMs: number): number {
  return elapsedMs < 30_000 ? 2_000 : 10_000
}
