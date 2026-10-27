// Order matters: Edge and Chrome both say Safari, Android says Linux, iOS says Mac OS X.
const browsers: [RegExp, string][] = [
  [/Edg(e|A|iOS)?\//, 'Edge'],
  [/OPR\/|Opera/, 'Opera'],
  [/Firefox\/|FxiOS\//, 'Firefox'],
  [/Chrome\/|CriOS\//, 'Chrome'],
  [/Safari\//, 'Safari'],
]
const systems: [RegExp, string][] = [
  [/Windows/, 'Windows'],
  [/iPhone|iPad|iPod/, 'iOS'],
  [/Android/, 'Android'],
  [/Mac OS X|Macintosh/, 'macOS'],
  [/CrOS/, 'ChromeOS'],
  [/Linux|X11/, 'Linux'],
]

// deviceLabel names a session's device from its User-Agent, e.g. "Chrome · Windows".
export function deviceLabel(ua: string): string {
  const parts = [browsers, systems].map((list) => list.find(([re]) => re.test(ua))?.[1])
  const known = parts.filter(Boolean).join(' · ')
  return known || ua.slice(0, 40)
}
