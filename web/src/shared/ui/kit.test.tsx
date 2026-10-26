import { Button } from '@mmerp/ui'
import { useState } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { expect, test } from 'vitest'

function Wrapper() {
  const [label] = useState('Lưu')
  return <Button>{label}</Button>
}

// Server render: the web has no DOM test environment, and a second React copy fails on hooks here too.
test('renders a kit Button inside a component that uses a hook', () => {
  expect(renderToStaticMarkup(<Wrapper />)).toMatch(/<button[^>]*>.*Lưu/)
})
