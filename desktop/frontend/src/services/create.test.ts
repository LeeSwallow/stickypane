import { describe, expect, it } from 'vitest'
import kinds from '../demo/kinds.json'
import { fieldOf, initialValues, reason, toOptions, type Shape } from './create'

const shapes = kinds as unknown as Shape[]
const shape = (name: string) => shapes.find((s) => s.name === name)!

describe('the create form and note.create', () => {
  it('starts from the defaults the shape gives', () => {
    expect(initialValues(shape('board'))).toEqual({ columns: 'To do\nDoing\nDone', cards: '' })
    const b: Shape = { ...shape('note'), options: [{ name: 'on', type: 'bool', label: 'On', default: 'true' }] }
    expect(initialValues(b)).toEqual({ on: true })
  })
  it('sends lines as a list, numbers as numbers, the rest as written', () => {
    expect(toOptions(shape('board'), { columns: 'A\nB', cards: '' })).toEqual({ columns: ['A', 'B'], cards: [''] })
    expect(toOptions(shape('api'), { method: 'POST', url: 'http://x', status: ' 201 ' })).toEqual({ method: 'POST', url: 'http://x', status: 201 })
    // a number that is not one is sent as it is: the shape says what is wrong
    expect(toOptions(shape('api'), { method: 'GET', url: '', status: 'two' }).status).toBe('two')
    const b: Shape = { ...shape('note'), options: [{ name: 'on', type: 'bool', label: 'On', default: '' }] }
    expect(toOptions(b, { on: false })).toEqual({ on: false })
  })
  it('sends only the shape\'s own options', () => {
    expect(toOptions(shape('checklist'), { items: 'a', stray: 'x' })).toEqual({ items: ['a'] })
  })
  it('finds the field a bridge error is about', () => {
    expect(fieldOf('bad option: view of a chart: "pie" is not one of bar, spark, heat', shape('chart'))).toBe('view')
    expect(fieldOf('option "status" is {}: give a string, a number, true or false, or a list of strings', shape('api'))).toBe('status')
    expect(fieldOf('bad option: status of an api: "x" is not a number', shape('api'))).toBe('status')
    expect(fieldOf('there is no tab "x"', shape('chart'))).toBeNull()
    expect(reason('bad option: view of a chart: "pie" is not one of bar')).toBe('view of a chart: "pie" is not one of bar')
  })
})
