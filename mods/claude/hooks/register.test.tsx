import { expect, mock, test } from 'claude-code/testing'

const INDEX = JSON.stringify([
  { tab: '02-작업', tab_title: '작업', name: '02-작업/02-checklist.md', kind: 'checklist', icon: '☐', title: '체크리스트', gist: '4/4', open: true },
])
const PANE = {
  component: 'Pane',
  requestId: 'stickypane',
  props: { title: 'stickypane', isFocused: true, bodyColumns: 60, placement: 'dock' },
} as const


test('/stickypane lists the notes and a press shows one', async ($, on) => {
  on('process.run', ($, e) => {
    const [, command] = e.argv
    const stdout = command === 'index' ? INDEX : command === 'cat' ? '- [x] 리뷰 요청\n' : ''
    return { value: { exitCode: 0, stdout, stderr: '', isStdoutTruncated: false, isStderrTruncated: false } }
  })
  on('ui.open', () => ({ value: { isPlaced: true } }))

  const ran = await $.command.run({ command: 'stickypane' })
  expect(ran.text).toContain('open')

  for (const surface of ['terminal', 'desktop'] as const) {
    const ui = await $.ui.mount({ plugin: 'board-mod', surface, ...PANE })
    expect(await ui.find({ text: /체크리스트/ })).toBeDefined()
    await ui.press({ key: '02-작업/02-checklist.md' })
    expect(await ui.find({ type: 'Markdown', text: /리뷰 요청/ })).toBeDefined()
    await ui.press({ key: 'back' })
    await ui.unmount()
  }
})

test('/stickypane says so where there is no board', async ($, on) => {
  on('process.run', () => ({
    value: { exitCode: 3, stdout: '', stderr: 'no board', isStdoutTruncated: false, isStderrTruncated: false },
  }))
  const ran = await $.command.run({ command: 'stickypane' })
  expect(ran.text).toContain('No stickypane board')
})

test('a change the user makes on the board is a toast', async ($, on) => {
  const toasts: string[] = []
  mock.clock(on, { now: 10_000 })
  on('process.run', () => ({
    value: { exitCode: 0, stdout: INDEX, stderr: '', isStdoutTruncated: false, isStderrTruncated: false },
  }))
  on('process.spawn', async function* () {
    yield { stream: 'stdout', text: '{"note":"02-작업/02-checklist.md","type":"item.ticked","item":"리뷰 요청"}\n' }
    return { value: { code: 0, signal: null } }
  })
  on('ui.toast', ($, e) => {
    toasts.push(e.text)
    return { value: undefined }
  })

  on('session.start', ($, e) => e)
  on('command.register', () => ({ value: undefined }))

  await $.session.start({ cwd: '.', surface: 'terminal', isInteractive: true })
  for (let i = 0; i < 2000 && toasts.length === 0; i++) await Promise.resolve()
  expect(toasts).toEqual(['☑ 체크리스트: 리뷰 요청'])
})

test('a change the agent made a moment ago is not a toast', async ($, on) => {
  const toasts: string[] = []
  mock.clock(on, { now: 10_000 })
  on('process.run', () => ({
    value: { exitCode: 0, stdout: INDEX, stderr: '', isStdoutTruncated: false, isStderrTruncated: false },
  }))
  on('tool.call', () => ({ result: '02-작업/02-checklist.md: 4/4' }))
  on('process.spawn', async function* () {
    yield { stream: 'stdout', text: '{"note":"02-작업/02-checklist.md","type":"item.ticked","item":"리뷰 요청"}\n' }
    return { value: { code: 0, signal: null } }
  })
  on('ui.toast', ($, e) => {
    toasts.push(e.text)
    return { value: undefined }
  })
  on('session.start', ($, e) => e)
  on('command.register', () => ({ value: undefined }))

  await $.tool.call({ tool: 'Bash', command: 'stickypane todo 02-작업/02-checklist check 리뷰' })
  await $.session.start({ cwd: '.', surface: 'terminal', isInteractive: true })
  for (let i = 0; i < 2000; i++) await Promise.resolve()
  expect(toasts).toEqual([])
})
