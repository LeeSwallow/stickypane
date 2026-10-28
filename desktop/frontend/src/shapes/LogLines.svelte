<script lang="ts">
  // Lines of a log in the mono face, coloured as the terminal colours
  // them (logs.ts toneOf): the command faint, checks green or red, the
  // closing "[…]" line bold. A log note, a script's output and a
  // request's response all draw with it.
  import { toneOf } from './logs'

  let { lines, label }: { lines: string[]; label?: string } = $props()
</script>

<pre class="log" aria-label={label}>{#each lines as line, i (i)}<span class="line {toneOf(line)}">{line || ' '}</span>{/each}</pre>

<style>
  .log {
    margin: 0;
    font: var(--weight-mono-md) var(--text-mono-md) / var(--leading-mono-md) var(--font-mono);
    color: var(--ink);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    font-variant-numeric: tabular-nums;
  }
  .line {
    display: block;
  }
  .command {
    color: var(--pencil);
  }
  .good,
  .end-good {
    color: var(--good-text, var(--good));
  }
  .bad,
  .end-bad {
    color: var(--danger-text, var(--bad));
  }
  .end-good,
  .end-bad {
    font-weight: 700;
  }
</style>
