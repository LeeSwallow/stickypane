// The remote hosts: computers whose boards open over the person's own ssh
// (the shell's remote.go, docs/desktop/02 §2.9). The app keeps only this
// list, in overlay.json; signing in is ~/.ssh/config and ssh-agent.
import { Call } from '@wailsio/runtime'
import { home } from '../lib/shell'

/** A host: its name in the app, what ssh is given, its stickypane, its boards' folders there. */
export interface Host {
  name: string
  target: string
  command?: string
  boards?: string[] | null
}

/** What a host's stickypane said, or why ssh could not ask it. */
export interface HostCheck {
  ok: boolean
  version?: string
  bridge: boolean
  error?: string
}

/** A host of the ssh config not in the list yet, and where ssh would go (user@hostname:port). */
export interface SshAlias {
  name: string
  resolved?: string
}

export interface HostsApi {
  hosts(): Promise<Host[] | null>
  /** adds a host (old "") or changes the one named old */
  saveHost(h: Host, old: string): Promise<void>
  removeHost(name: string): Promise<void>
  checkHost(name: string): Promise<HostCheck>
  addBoard(name: string, path: string): Promise<void>
  removeBoard(name: string, path: string): Promise<void>
  openRemote(name: string, path: string): Promise<unknown>
  aliases(): Promise<SshAlias[] | null>
}

const call = <T>(method: string, ...args: unknown[]) => Call.ByName(home(method), ...args) as Promise<T>

export const hostsService: HostsApi = {
  hosts: () => call('Hosts'),
  saveHost: (h, old) => call('SaveHost', h, old),
  removeHost: (name) => call('RemoveHost', name),
  checkHost: (name) => call('CheckHost', name),
  addBoard: (name, path) => call('AddBoard', name, path),
  removeBoard: (name, path) => call('RemoveBoard', name, path),
  openRemote: (name, path) => call('OpenRemote', name, path),
  aliases: () => call('SSHAliases'),
}

/** The boards of every host, in the list's order: what the home's 원격 group shows. */
export const remoteBoards = (hosts: Host[] | null | undefined): { host: string; path: string }[] =>
  (hosts ?? []).flatMap((h) => (h.boards ?? []).map((path) => ({ host: h.name, path })))
