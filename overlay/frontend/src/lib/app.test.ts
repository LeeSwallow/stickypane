import { expect, test } from 'vitest'
import { appName } from './app'

test('the app has its name', () => {
  expect(appName).toBe('stickypane')
})
