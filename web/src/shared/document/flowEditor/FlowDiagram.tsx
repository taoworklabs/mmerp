import { Menu, Text, UnstyledButton } from '@mantine/core'
import { IconPlus } from '@tabler/icons-react'
import { useQueries } from '@tanstack/react-query'
import { createContext, Fragment, useContext, type ReactNode } from 'react'
import { get, useFormState, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap } from '@/shared/api/core'
import { fieldId, rowLabel, useOrgUnits } from '@/shared/ui/form'
import { newSplit, newStep, type CondValues, type ElementValues, type FlowValues, type RuleValues } from '../flowForm'
import { flowProblems } from '../flowProblems'
import { approverText, condText, stepDetail, type ValueLabel } from '../flowText'
import { documentKeys } from '../keys'
import { useFlowEditor } from './editor'
import c from './FlowDiagram.module.css'

// The server allows at most this many splits nested in one another.
const maxDepth = 5

type Look = { values: RuleValues; issueAt: (at: string) => string | undefined; valueLabel: ValueLabel }
const LookContext = createContext<Look | null>(null)
const useLook = () => useContext(LookContext)!

// searchedIds lists, per searchable field, the ids its conditions hold anywhere in the flow.
function searchedIds(flow: FlowValues, searchable: Set<string>, out = new Map<string, Set<number>>()) {
  const fromCond = (cond: CondValues) => {
    for (const r of cond.rows) {
      if (r.group) fromCond(r.group)
      else if (r.field && searchable.has(r.field) && Number(r.value) > 0) {
        if (!out.has(r.field)) out.set(r.field, new Set())
        out.get(r.field)!.add(Number(r.value))
      }
    }
  }
  for (const el of flow) {
    for (const b of el.branches) {
      fromCond(b.cond)
      searchedIds(b.body, searchable, out)
    }
    searchedIds(el.otherwise, searchable, out)
  }
  return out
}

// useValueLabel names the ids conditions hold: org units from the tree, searched values from the
// server, fetched once for the saved ids so a condition reads in full before any search.
function useValueLabel(values: RuleValues): ValueLabel {
  const { ctx } = useFlowEditor()
  const docType = ctx.rule.doc_type
  const units = useOrgUnits({}).data ?? []
  const searchable = new Set(ctx.rule.fields.filter((f) => f.searchable).map((f) => f.key))
  const wanted = [...searchedIds(values.flow, searchable)].map(([key, ids]) => ({ key, ids: [...ids].sort((a, b) => a - b) }))
  const found = useQueries({
    queries: wanted.map(({ key, ids }) => ({
      queryKey: documentKeys.approvalFieldOptions(docType, key, { ids }),
      queryFn: async () => (await unwrap(api.GET('/approval-rules/{doc_type}/fields/{key}/options', { params: { path: { doc_type: docType, key }, query: { ids } } }))) ?? [],
    })),
  })
  return (field, value) => {
    if (field.kind === 'org_unit') return units.find((u) => String(u.id) === value)?.name
    if (!field.searchable) return undefined
    const at = wanted.findIndex((w) => w.key === field.key)
    const row = found[at]?.data?.find((r) => String(r.id) === value)
    return row ? rowLabel(row) : '…'
  }
}

// FlowDiagram draws the rule's flow: a start box, the elements top to bottom, an end box. A box
// with a problem is red as soon as the problem exists; a ⊕ on every connector inserts there.
export function FlowDiagram() {
  const { t } = useTranslation()
  const { select } = useFlowEditor()
  const values = useWatch() as RuleValues
  const { errors } = useFormState()
  const problems = new Map(flowProblems(values).map((p) => [p.at, p.code]))
  const issueAt = (at: string) => {
    const code = problems.get(at)
    return code ? t(`shared.rule.reason.${code}`) : (get(errors, at)?.message as string | undefined)
  }
  const valueLabel = useValueLabel(values)
  const empty = values.flow.length === 0
  return (
    <LookContext.Provider value={{ values, issueAt, valueLabel }}>
      <div
        className={c.frame}
        // Clicking the background, not a box, puts the panel back to its hint.
        onClick={(e) => (e.target === e.currentTarget || e.target === e.currentTarget.firstChild) && select(null)}
      >
        <div className={c.lane}>
          <div className={c.start}>
            <Text size="sm" fw={600}>
              {t('shared.rule.start')}
            </Text>
          </div>
          <Lane path="flow" depth={0} top />
          <div className={c.end}>
            <Text size="sm" fw={600}>
              {t(empty ? 'shared.rule.end_now' : 'shared.rule.end')}
            </Text>
          </div>
          {empty && (
            <Text size="sm" c="dimmed" mt="md">
              {t('shared.rule.empty_hint')}
            </Text>
          )}
        </div>
      </div>
    </LookContext.Provider>
  )
}

function Lane({ path, depth, top }: { path: string; depth: number; top?: boolean }) {
  const { t } = useTranslation()
  const items: ElementValues[] = get(useLook().values, path) ?? []
  return (
    <div className={c.lane}>
      {items.map((el, i) => (
        <Fragment key={i}>
          <Edge list={path} at={i} depth={depth} />
          {el.type === 'step' ? <StepBox path={`${path}.${i}`} el={el} /> : <SplitBox path={`${path}.${i}`} el={el} depth={depth} />}
        </Fragment>
      ))}
      {items.length === 0 && !top && (
        <>
          <Edge list={path} at={0} depth={depth} />
          <div className={c.skip}>
            <Text size="xs">{t('shared.rule.skip')}</Text>
          </div>
        </>
      )}
      {(items.length > 0 || top) && <Edge list={path} at={items.length} depth={depth} root={top && items.length === 0} />}
    </div>
  )
}

// Edge is a connector; editable, it carries the ⊕ that inserts an element at its place.
// root marks the only ⊕ of an empty flow, where an empty flow's problem is shown.
function Edge({ list, at, depth, root }: { list: string; at: number; depth: number; root?: boolean }) {
  const { t } = useTranslation()
  const { ctx, insert } = useFlowEditor()
  const { issueAt } = useLook()
  if (ctx.readOnly)
    return (
      <div className={c.edge}>
        <div className={c.line} />
      </div>
    )
  const m = ctx.rule.module_approvers
  const item = (title: string, hint: string, make: () => ElementValues) => (
    <Menu.Item onClick={() => insert(list, at, make())}>
      <Text size="sm" fw={500}>
        {title}
      </Text>
      <Text size="xs" c="dimmed">
        {hint}
      </Text>
    </Menu.Item>
  )
  return (
    <div className={c.edge}>
      <div className={c.line} />
      <Menu position="right-start">
        <Menu.Target>
          <UnstyledButton
            className={c.plus}
            id={root ? fieldId('flow.root') : undefined}
            data-big={root || undefined}
            aria-invalid={root && issueAt('flow.root') ? true : undefined}
            aria-label={t('shared.rule.add_here')}
          >
            <IconPlus size={root ? 18 : 12} aria-hidden />
          </UnstyledButton>
        </Menu.Target>
        <Menu.Dropdown>
          {item(t('shared.rule.step'), t('shared.rule.step_hint'), () => newStep(m))}
          {depth < maxDepth && item(t('shared.rule.split.one'), t('shared.rule.split.one_hint'), () => newSplit('one', m))}
          {depth < maxDepth && item(t('shared.rule.split.all'), t('shared.rule.split.all_hint'), () => newSplit('all', m))}
        </Menu.Dropdown>
      </Menu>
      <div className={c.line} />
    </div>
  )
}

// Box is one selectable thing on the diagram. It owns its element's error name as its id, so the
// form's "next field to fix" lands on it, and focusing it selects it.
function Box({ path, at, label, className, children, ...rest }: { path: string; at: string; label: string; className?: string; children: ReactNode; 'data-mode'?: string; 'data-quiet'?: boolean }) {
  const { selected, select } = useFlowEditor()
  const issue = useLook().issueAt(at)
  return (
    <UnstyledButton
      id={fieldId(at)}
      className={className}
      aria-label={label}
      aria-pressed={selected === path}
      aria-invalid={issue ? true : undefined}
      onClick={() => select(path)}
      onFocus={() => select(path)}
      {...rest}
    >
      {children}
    </UnstyledButton>
  )
}

function StepBox({ path, el }: { path: string; el: ElementValues }) {
  const { t } = useTranslation()
  const { ctx } = useFlowEditor()
  const issue = useLook().issueAt(`${path}.issue`)
  const text = approverText(t, el, ctx.roleOptions, ctx.userOptions)
  const detail = stepDetail(t, el)
  return (
    <Box path={path} at={`${path}.issue`} className={c.node} label={t('shared.rule.box.named', { name: t('shared.rule.step'), text })}>
      <Text size="sm" fw={600}>
        {text}
      </Text>
      {issue ? (
        <Text size="xs" c="danger">
          {issue}
        </Text>
      ) : (
        detail && (
          <Text size="xs" c="dimmed">
            {detail}
          </Text>
        )
      )}
    </Box>
  )
}

function SplitBox({ path, el, depth }: { path: string; el: ElementValues; depth: number }) {
  const { t } = useTranslation()
  const { ctx } = useFlowEditor()
  const { valueLabel } = useLook()
  const one = el.type === 'one'
  const name = t(`shared.rule.split.${el.type}`)
  const column = (key: string, label: string, chip: ReactNode, lane: string) => (
    <div className={c.branch} key={key} role="group" aria-label={label}>
      <div className={c.join} data-at="top" />
      {chip}
      <Lane path={lane} depth={depth + 1} />
      <div className={c.fill} />
      <div className={c.join} data-at="bottom" />
    </div>
  )
  return (
    <div className={c.split}>
      <Box path={path} at={`${path}.issue`} className={c.pill} label={name} data-mode={el.type}>
        <Text size="sm" fw={600}>
          {name}
        </Text>
      </Box>
      <div className={c.line} />
      <div className={c.branches}>
        {el.branches.map((b, j) => {
          const bp = `${path}.branches.${j}`
          const text = condText(t, b.cond, ctx.rule.fields, valueLabel)
          let chip: string
          if (text) chip = t(one ? 'shared.rule.chip.if' : 'shared.rule.chip.when', { text })
          else chip = t(one ? 'shared.rule.chip.always_one' : 'shared.rule.chip.always_all')
          const label = t('shared.rule.box.named', { name: t(one ? 'shared.rule.case_n' : 'shared.rule.branch', { n: j + 1 }), text: chip })
          return column(
            String(j),
            label,
            <Box path={bp} at={`${bp}.issue`} className={c.cond} label={label} data-quiet={!text || undefined}>
              {chip}
            </Box>,
            `${bp}.body`,
          )
        })}
        {one &&
          column(
            'otherwise',
            t('shared.rule.otherwise'),
            <Box path={`${path}.otherwise`} at={`${path}.otherwise.issue`} className={c.cond} label={t('shared.rule.otherwise')} data-quiet>
              {t('shared.rule.otherwise')}
            </Box>,
            `${path}.otherwise`,
          )}
      </div>
      {!one && (
        <Text size="xs" c="dimmed" mt="xxs">
          {t('shared.rule.parallel_join')}
        </Text>
      )}
    </div>
  )
}
