import { useDisclosure } from '../../hooks/useDisclosure.ts'
import { ActionIcon } from './ActionIcon.tsx'
import { TextInput, type TextInputProps } from './TextInput.tsx'

export interface PasswordInputProps extends TextInputProps {
  visibilityLabels: { show: string; hide: string }
}

export function PasswordInput({ visibilityLabels, ...props }: PasswordInputProps) {
  const [visible, { toggle }] = useDisclosure(false)
  return (
    <TextInput
      {...props}
      type={visible ? 'text' : 'password'}
      rightSection={
        <ActionIcon
          variant="subtle"
          color="gray"
          size="xs"
          disabled={props.disabled}
          aria-label={visible ? visibilityLabels.hide : visibilityLabels.show}
          onClick={toggle}
        >
          {visible ? <EyeOffGlyph /> : <EyeGlyph />}
        </ActionIcon>
      }
    />
  )
}

function EyeGlyph() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
      <path
        d="M1 7C2.2 4 4.4 2.3 7 2.3S11.8 4 13 7c-1.2 3-3.4 4.7-6 4.7S2.2 10 1 7Z"
        stroke="currentColor"
        strokeWidth="1.2"
        strokeLinejoin="round"
      />
      <circle cx="7" cy="7" r="2" stroke="currentColor" strokeWidth="1.2" />
    </svg>
  )
}

function EyeOffGlyph() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
      <path
        d="M1 7C2.2 4 4.4 2.3 7 2.3S11.8 4 13 7c-1.2 3-3.4 4.7-6 4.7S2.2 10 1 7Z"
        stroke="currentColor"
        strokeWidth="1.2"
        strokeLinejoin="round"
      />
      <circle cx="7" cy="7" r="2" stroke="currentColor" strokeWidth="1.2" />
      <path d="M1.5 12.5 12.5 1.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
    </svg>
  )
}
