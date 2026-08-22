import { isValidElement } from 'react'
import { CreditCard } from 'lucide-react'
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { getPaymentIcon } from './ui'

function elementPart<TProps>(node: unknown) {
  assert.equal(isValidElement(node), true)
  return node as { type: unknown; props: TProps }
}

describe('getPaymentIcon', () => {
  test('upgrades a backend http icon URL to https before rendering', () => {
    const icon = getPaymentIcon(
      'custom-gateway',
      'h-5 w-5',
      'http://pay.example.com/logo.png?theme=light',
      'Custom Pay'
    )

    const element = elementPart<{ src: string; alt: string }>(icon)

    assert.equal(element.type, 'img')
    assert.equal(
      element.props.src,
      'https://pay.example.com/logo.png?theme=light'
    )
    assert.equal(element.props.alt, 'Custom Pay')
  })

  test('falls back to the default icon for unsafe backend icon URLs', () => {
    const icon = getPaymentIcon(
      'unknown-gateway',
      'h-5 w-5',
      'javascript:alert(1)'
    )

    assert.equal(elementPart(icon).type, CreditCard)
  })

  test('does not render URLs with embedded credentials', () => {
    const icon = getPaymentIcon(
      'unknown-gateway',
      'h-5 w-5',
      'https://user:pass@pay.example.com/logo.png'
    )

    assert.equal(elementPart(icon).type, CreditCard)
  })
})
