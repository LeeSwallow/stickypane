// How much the app moves (04 §4.6): the app's own setting (overlay.json
// "motion": "" follows the system, "reduce", "off") and the system's
// prefers-reduced-motion. Every window reads it; the settings change it.
import { levelOf, type Level } from './keyframes'

export const motion = $state<{ setting: string; systemReduced: boolean }>({ setting: '', systemReduced: false })

/** level is the motion the window plays now. */
export const level = (): Level => levelOf(motion.setting, motion.systemReduced)

/** watchReducedMotion follows the system's setting; it returns the way to stop. */
export function watchReducedMotion(): () => void {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return () => {}
  const q = window.matchMedia('(prefers-reduced-motion: reduce)')
  motion.systemReduced = q.matches
  const on = (e: MediaQueryListEvent) => (motion.systemReduced = e.matches)
  q.addEventListener?.('change', on)
  return () => q.removeEventListener?.('change', on)
}
