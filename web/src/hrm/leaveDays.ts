// leaveDays is the count the server will compute for a request: the working days of its
// range less half a day for each half day. Null when there is none to take.
export function leaveDays(working: number, startHalf: boolean, endHalf: boolean): string | null {
  if (working <= 0 || (working === 1 && startHalf && endHalf)) return null
  return String(working - (Number(startHalf) + Number(endHalf)) / 2)
}
