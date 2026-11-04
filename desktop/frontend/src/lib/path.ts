// How a folder is written on a card, a row or a title: its name, its
// path with ~ for the person's home, and a long path kept to its end,
// where the name is (the whole is the tooltip).

/** baseName is a folder's last name ("/a/b/" is "b"); a path with none is itself. */
export const baseName = (p: string): string => p.split('/').filter(Boolean).pop() || p

/** SHORT_PATH is how long a path on a home card or row gets before it keeps only its end. */
export const SHORT_PATH = 34

/** shortPath is p, or "…/" and its last two names when it is longer than max and deeper than three. */
export function shortPath(p: string, max = SHORT_PATH): string {
  const parts = p.split('/')
  return p.length > max && parts.length > 3 ? '…/' + parts.slice(-2).join('/') : p
}

/** tildePath writes a path under home with ~ in its place. */
export const tildePath = (p: string, home: string): string => (home && p.startsWith(home + '/') ? '~' + p.slice(home.length) : p)

/** APP_DIR is the app's folder inside the person's home. */
export const APP_DIR = '/.stickypane'

/** homeOf is the person's home from the app's folder in it, or '' when dir is not that folder. */
export const homeOf = (dir: string): string => (dir.endsWith(APP_DIR) ? dir.slice(0, -APP_DIR.length) : '')
