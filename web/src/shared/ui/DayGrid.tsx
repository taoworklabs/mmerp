import { ActionIcon, Group, SegmentedControl, Text, Tooltip } from '@mmerp/ui'
import { IconChevronLeft, IconChevronRight } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import classes from './DayGrid.module.css'
import { icon } from './theme'

export type DayCell = {
  key: string
  day: string // the day's number
  weekday: string // shown inside the cell on a phone, where the header row is gone
  tone?: 'rest' | 'holiday' // a tint; the content must say the same in words
  children?: ReactNode
}

// DayGrid is a calendar: weekdays across, one cell per day, null for a blank cell. On a phone the
// days stack in one column.
export function DayGrid({ label, weekdays, cells }: { label: string; weekdays: string[]; cells: (DayCell | null)[] }) {
  return (
    <div className={classes.grid} role="group" aria-label={label}>
      {weekdays.map((w) => (
        <div key={w} className={classes.head} aria-hidden>
          {w}
        </div>
      ))}
      {cells.map((c, i) =>
        c ? (
          <div key={c.key} className={classes.cell} data-tone={c.tone}>
            <Text size="sm" className={classes.day}>
              {c.day} <span className={classes.weekday}>{c.weekday}</span>
            </Text>
            {c.children}
          </div>
        ) : (
          <div key={`blank-${i}`} className={classes.cell} data-empty="" aria-hidden />
        ),
      )}
    </div>
  )
}

// PeriodNav switches between periods of time (e.g. Month / Week) and steps to the previous or
// next one; `title` names the period in view.
export function PeriodNav<M extends string>(props: {
  modes: { value: M; label: string }[]
  mode: M
  onMode: (mode: M) => void
  modeLabel: string
  title: string
  onPrev: () => void
  onNext: () => void
  prevLabel: string
  nextLabel: string
}) {
  const { modes, mode, onMode, modeLabel, title, onPrev, onNext, prevLabel, nextLabel } = props
  return (
    <Group gap="sm" justify="space-between">
      <SegmentedControl aria-label={modeLabel} data={modes} value={mode} onChange={(v) => onMode(v as M)} />
      <Group gap="xs">
        <Tooltip label={prevLabel}>
          <ActionIcon variant="subtle" color="gray" aria-label={prevLabel} onClick={onPrev}>
            <IconChevronLeft {...icon.button} />
          </ActionIcon>
        </Tooltip>
        <Text weight={600} aria-live="polite">
          {title}
        </Text>
        <Tooltip label={nextLabel}>
          <ActionIcon variant="subtle" color="gray" aria-label={nextLabel} onClick={onNext}>
            <IconChevronRight {...icon.button} />
          </ActionIcon>
        </Tooltip>
      </Group>
    </Group>
  )
}
