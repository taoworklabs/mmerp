import { Button, Group, Popover, Stack, Text, Textarea } from '@mmerp/ui'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, unwrap } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { errorText } from '@/shared/i18n'
import { PersonAvatar } from '@/shared/ui/avatar'
import classes from './DiscussionPanel.module.css'
import { documentKeys } from './keys'
import { mentionAt, splitMentions } from './mentions'

export function useDiscussion(docType: string, id: number) {
  return useQuery({
    queryKey: documentKeys.discussion(docType, id),
    queryFn: () => unwrap(api.GET('/records/{type}/{id}/comments', { params: { path: { type: docType, id } } })),
  })
}

// CommentText is a comment's text, the people it mentions in bold.
export function CommentText({ body }: { body: string }) {
  return (
    <Text size="sm" className={classes.body}>
      {splitMentions(body).map((part, i) =>
        part.mention ? (
          <Text key={i} render={<span />} weight={600} tone="primary">
            {part.text}
          </Text>
        ) : (
          part.text
        ),
      )}
    </Text>
  )
}

// CommentBox writes a comment beside the writer's own avatar. Its button and the mention hint
// show once something is typed; Ctrl+Enter (⌘+Enter) sends.
export function CommentBox({ docType, id, maxLength }: { docType: string; id: number; maxLength: number }) {
  const { t } = useTranslation()
  const me = useMe()
  const qc = useQueryClient()
  const [body, setBody] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const typed = body.trim() !== ''

  async function send() {
    if (!typed || busy) return
    setError(null)
    setBusy(true)
    try {
      await unwrap(api.POST('/records/{type}/{id}/comments', { params: { path: { type: docType, id } }, body: { body } }))
      setBody('')
      await qc.invalidateQueries({ queryKey: documentKeys.document(docType, id) })
    } catch (err) {
      setError(errorText(t, err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Group align="start" gap="sm" wrap={false}>
      <PersonAvatar name={me.name} size="md" />
      <Stack gap="xs" fill>
        <MentionTextarea docType={docType} id={id} maxLength={maxLength} value={body} error={error} hint={typed} onChange={setBody} onSubmit={() => void send()} />
        {typed && (
          <Group justify="end">
            <Button variant="default" loading={busy} onClick={() => void send()}>
              {t('shared.discussion.send')}
            </Button>
          </Group>
        )}
      </Stack>
    </Group>
  )
}

// MentionTextarea is the comment box; typing @ offers the users who may view the record,
// and picking one writes @login.
function MentionTextarea({
  docType,
  id,
  maxLength,
  value,
  error,
  hint,
  onChange,
  onSubmit,
}: {
  docType: string
  id: number
  maxLength: number
  value: string
  error: string | null
  // Shows how to mention someone, once the writer has started.
  hint: boolean
  onChange: (value: string) => void
  onSubmit: () => void
}) {
  const { t } = useTranslation()
  const listId = useId()
  const ref = useRef<HTMLTextAreaElement>(null)
  const [token, setToken] = useState<{ start: number; query: string } | null>(null)
  // The highlighted option; the first one whenever the typed name changes.
  const [active, setActive] = useState(0)
  // One line until the writer comes to it, so the box does not push the feed down.
  const [focused, setFocused] = useState(false)
  const users = useQuery({
    queryKey: documentKeys.mentionable(docType, id),
    queryFn: () => unwrap(api.GET('/records/{type}/{id}/mentionable', { params: { path: { type: docType, id } } })),
    enabled: token !== null,
  })
  const query = token?.query.toLowerCase() ?? ''
  const options = (users.data ?? []).filter((u) => u.login.toLowerCase().includes(query) || u.name.toLowerCase().includes(query)).slice(0, 8)
  const open = token !== null && options.length > 0
  const highlighted = Math.min(active, options.length - 1)

  const follow = (text: string, caret: number) => {
    onChange(text)
    setToken(mentionAt(text, caret))
    setActive(0)
  }
  const pick = (login: string) => {
    if (!token) return
    const caret = token.start + 1 + token.query.length
    const text = `${value.slice(0, token.start)}@${login} ${value.slice(caret)}`
    const at = token.start + login.length + 2
    onChange(text)
    setToken(null)
    requestAnimationFrame(() => ref.current?.setSelectionRange(at, at))
  }

  return (
    // Escape or a click elsewhere closes the list; the typing stays in the box throughout.
    <Popover.Root open={open} onOpenChange={(next) => !next && setToken(null)}>
      <Textarea
        ref={ref}
        aria-label={t('shared.discussion.input')}
        aria-controls={open ? listId : undefined}
        aria-activedescendant={open ? `${listId}-${highlighted}` : undefined}
        placeholder={t('shared.discussion.placeholder')}
        description={hint ? t('shared.discussion.mention_hint') : undefined}
        autosize
        minRows={focused || value ? 3 : 1}
        maxRows={8}
        maxLength={maxLength}
        value={value}
        error={error ?? undefined}
        onChange={(e) => follow(e.currentTarget.value, e.currentTarget.selectionStart)}
        onClick={(e) => setToken(mentionAt(value, e.currentTarget.selectionStart))}
        onFocus={() => setFocused(true)}
        onBlur={() => {
          setToken(null)
          setFocused(false)
        }}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
            e.preventDefault()
            onSubmit()
          } else if (open && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
            e.preventDefault()
            setActive((highlighted + (e.key === 'ArrowDown' ? 1 : options.length - 1)) % options.length)
          } else if (open && e.key === 'Enter') {
            e.preventDefault()
            pick(options[highlighted]!.login)
          }
        }}
      />
      <Popover.Popup anchor={ref} side="bottom" align="start" initialFocus={false} finalFocus={false} className={classes.mentions}>
        <div id={listId} role="listbox" aria-label={t('shared.discussion.mention')}>
          {options.map((u, i) => (
            <div
              key={u.login}
              id={`${listId}-${i}`}
              role="option"
              aria-selected={i === highlighted}
              className={classes.mention}
              // Keeps the focus, and the caret, in the comment box.
              onMouseDown={(e) => e.preventDefault()}
              onMouseEnter={() => setActive(i)}
              onClick={() => pick(u.login)}
            >
              <Text size="sm">
                {u.name}{' '}
                <Text render={<span />} size="xs" tone="dimmed">
                  @{u.login}
                </Text>
              </Text>
            </div>
          ))}
        </div>
      </Popover.Popup>
    </Popover.Root>
  )
}
