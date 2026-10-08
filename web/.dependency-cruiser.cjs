// Boundaries between areas. Rules match folder patterns, so a new area needs no change here.
// Run from web/ on src (must be clean) and from web/lint-fixtures (must report every planted violation).
const P = '^src/'
const area = { path: `${P}([^/]+)/`, pathNot: `${P}(app|shared)/` }

module.exports = {
  forbidden: [
    {
      name: 'area-to-other-area',
      comment: 'An area depends only on itself and shared; never on another area or app.',
      severity: 'error',
      from: area,
      to: { path: `${P}[^/]+/`, pathNot: `${P}($1|shared)/` },
    },
    {
      name: 'area-only-through-index',
      comment: 'Outside an area, only its index.ts may be imported.',
      severity: 'error',
      from: { path: `${P}([^/]+)/` },
      to: { path: `${P}(?!$1/)(?!app/|shared/)[^/]+/(?!index\\.ts$)` },
    },
    {
      name: 'shared-not-to-areas',
      comment: 'shared knows no area (core included) and not app.',
      severity: 'error',
      from: { path: `${P}shared/` },
      to: { path: P, pathNot: `${P}shared/` },
    },
    {
      name: 'area-api-only-core-and-own',
      comment: 'An area calls only shared/api/core and its own product client.',
      severity: 'error',
      from: area,
      to: { path: `${P}shared/api/(?!(core|$1)\\.)` },
    },
    {
      name: 'entry-only-to-app',
      comment: 'Files directly under src (main.tsx) only start app; areas are reached through app.',
      severity: 'error',
      from: { path: `${P}[^/]+$` },
      to: { path: P, pathNot: `${P}app/` },
    },
    {
      name: 'no-css-in-areas',
      comment: 'Styling comes from shared/ui; areas import no CSS.',
      severity: 'error',
      from: area,
      to: { path: '\\.css$' },
    },
  ],
  options: {
    doNotFollow: { path: 'node_modules' },
    tsPreCompilationDeps: true,
    enhancedResolveOptions: { extensions: ['.ts', '.tsx', '.js', '.json'] },
  },
}
