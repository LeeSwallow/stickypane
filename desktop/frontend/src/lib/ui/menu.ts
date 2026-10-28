import type { IconName } from './icons/icons'

/** An item of a Menu: an action, or a separator line. */
export type MenuItem =
  | {
      kind?: 'item'
      id: string
      label: string
      icon?: IconName
      shortcut?: string
      danger?: boolean
      disabled?: boolean
    }
  | { kind: 'separator'; id: string }

export const isAction = (m: MenuItem): m is Extract<MenuItem, { label: string }> => m.kind !== 'separator'
