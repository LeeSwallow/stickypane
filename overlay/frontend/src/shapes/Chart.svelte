<script lang="ts">
  // A chart (03 §3.8, 10 §10.4): the terminal's three views, drawn — bars
  // scaled to the largest value with the number as written, a trend line
  // with its range, first and last labels and the latest value marked, a
  // calendar of days shaded by rank with its total and peak. Hand-drawn
  // in doodle, straight in neat. The numbers come from chart.ts, which
  // follows the terminal's view.go; a chart takes no action.
  import type { Note } from '../board/model'
  import { tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import { boxPaths, ellipsePaths, linePaths, polylinePaths } from '../lib/doodle/rough'
  import Strokes from '../lib/doodle/Strokes.svelte'
  import { seedOf } from '../lib/theme/tokens'
  import { bars, calendar, grouped, spark } from './chart'
  import { dataOf, type ChartData } from './data'

  let { note }: { note: Note } = $props()
  const data = $derived(dataOf<ChartData>(note.data, { view: 'bar', prose: [], points: [] }))
  const points = $derived(data.points ?? [])
  const seed = $derived(seedOf(note.name))
  let width = $state(0)

  // --- bars
  const BAR_H = 14
  const rows = $derived(bars(points))
  let track = $state(0)

  // --- trend
  const SPARK_H = 72
  const trend = $derived(width ? spark(points, Math.max(width - 8, 40), SPARK_H) : null)
  const trendSegs = $derived(
    trend
      ? [
          ...linePaths(0, SPARK_H, trend.line.at(-1)![0], SPARK_H, { stroke: 'var(--line-strong)', strokeWidth: 1, seed, style: look.style }),
          ...polylinePaths(trend.line, { stroke: 'var(--accent)', strokeWidth: 2, seed: seed + 1, style: look.style }),
          ...ellipsePaths(trend.line.at(-1)![0], trend.line.at(-1)![1], 8, 8, { stroke: 'var(--accent)', fill: 'var(--accent)', seed: seed + 2, style: look.style }),
        ]
      : [],
  )

  // --- calendar
  const CELL = 11
  const GAP = 2
  const GUTTER = 30
  const fit = $derived(width ? Math.max(1, Math.floor((width - GUTTER) / (CELL + GAP))) : 53)
  const cal = $derived(data.view === 'heat' ? calendar(points, fit) : null)
  const DAYS = ['Mon', '', 'Wed', '', 'Fri', '', '']
  const shade = (level: number) => [0.3, 0.5, 0.75, 1][level]
</script>

<div class="chart" bind:clientWidth={width}>
  {#each data.prose ?? [] as line, i (i)}
    <p class="prose">{line}</p>
  {/each}
  {#if !points.length}
    <p class="empty">{$tr('chart.empty')}</p>
  {:else if cal}
    <svg
      class="heat"
      role="img"
      aria-label={$tr('chart.heat', { total: grouped(cal.total), peak: cal.peak.text, date: cal.peak.date })}
      width={GUTTER + cal.weeks.length * (CELL + GAP)}
      height={16 + 7 * (CELL + GAP)}
    >
      {#each cal.months as m, w (w)}
        {#if m}<text class="axis" x={GUTTER + w * (CELL + GAP)} y="10">{m}</text>{/if}
      {/each}
      {#each DAYS as d, r (r)}
        {#if d}<text class="axis" x="0" y={16 + r * (CELL + GAP) + CELL - 2}>{d}</text>{/if}
      {/each}
      {#each cal.weeks as week, w (w)}
        {#each week as day, r (r)}
          {@const x = GUTTER + w * (CELL + GAP)}
          {@const y = 16 + r * (CELL + GAP)}
          {#if day.level === 'zero'}
            <circle class="zero" cx={x + CELL / 2} cy={y + CELL / 2} r="1.5"><title>{day.date}: {day.text}</title></circle>
          {:else if day.level !== null}
            <rect
              class="cell"
              {x}
              {y}
              width={CELL}
              height={CELL}
              rx={look.style === 'neat' ? 2 : 3}
              opacity={shade(day.level)}
              data-level={day.level}
              transform={look.style === 'doodle' ? `rotate(${((seedOf(day.date) % 7) - 3) * 0.6} ${x + CELL / 2} ${y + CELL / 2})` : undefined}
              ><title>{day.date}: {day.text}</title></rect
            >
          {/if}
        {/each}
      {/each}
    </svg>
    <p class="legend">
      {$tr('chart.total', { total: grouped(cal.total) })} · {$tr('chart.peak', { peak: cal.peak.text, date: cal.peak.date })}
      <span class="scale" aria-hidden="true">
        {$tr('chart.less')}
        {#each [0, 1, 2, 3] as l (l)}<i style:opacity={shade(l)}></i>{/each}
        {$tr('chart.more')}
      </span>
    </p>
  {:else if data.view === 'spark' && (trend || !width)}
    <div class="spark" style:height="{SPARK_H}px" role="img" aria-label={trend ? $tr('chart.trend', { low: grouped(trend.low), high: grouped(trend.high), latest: trend.latest }) : ''}>
      {#if trend}<Strokes segs={trendSegs} viewBox="0 0 {width - 8} {SPARK_H}" />{/if}
    </div>
    {#if trend}
      <p class="axis-row">
        <span>{trend.first}</span>
        <span class="range">{grouped(trend.low)} – {grouped(trend.high)}</span>
        <span>{trend.last} <b class="latest">{trend.latest}</b></span>
      </p>
    {/if}
  {:else}
    <div class="bars" bind:clientWidth={track}>
      {#each rows as row, i (i)}
        <span class="label" title={row.label}>{row.label}</span>
        <span class="track">
          {#if row.ratio > 0 && track}
            {@const w = Math.max(4, row.ratio * (track * 0.55))}
            <span class="bar" style:width="{w}px" style:height="{BAR_H}px" aria-hidden="true">
              <Strokes
                segs={boxPaths(w, BAR_H, {
                  stroke: 'var(--accent)',
                  fill: 'color-mix(in srgb, var(--accent) 45%, transparent)',
                  strokeWidth: 1.2,
                  roughness: 0.9,
                  seed: seed + i,
                  style: look.style,
                })}
                viewBox="0 0 {w} {BAR_H}"
              />
            </span>
          {/if}
        </span>
        <span class="value">{row.text}</span>
      {/each}
    </div>
  {/if}
</div>

<style>
  .chart {
    color: var(--ink);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .prose {
    margin: 0 0 var(--space-1);
    color: var(--pencil);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .empty {
    margin: 0;
    color: var(--pencil);
  }
  .bars {
    display: grid;
    grid-template-columns: minmax(0, max-content) 1fr max-content;
    gap: var(--space-1) var(--space-2);
    align-items: center;
  }
  .label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 14ch;
  }
  .track {
    position: relative;
    display: block;
    height: 14px;
  }
  .bar {
    position: relative;
    display: block;
  }
  .value,
  .latest {
    font: var(--weight-mono-md) var(--text-mono-md) / 1 var(--font-mono);
    font-variant-numeric: tabular-nums;
    text-align: right;
  }
  .spark {
    position: relative;
    margin: var(--space-2) var(--space-1) 0;
  }
  .axis-row {
    display: flex;
    justify-content: space-between;
    gap: var(--space-2);
    margin: var(--space-1) 0 0;
    color: var(--pencil);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .latest {
    color: var(--accent);
  }
  .heat {
    display: block;
    max-width: 100%;
  }
  .axis {
    fill: var(--pencil);
    font: var(--weight-mono-sm) 9px / 1 var(--font-mono);
  }
  .cell {
    fill: var(--good);
  }
  .zero {
    fill: var(--pencil);
  }
  .legend {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    gap: var(--space-2);
    margin: var(--space-1) 0 0;
    color: var(--pencil);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .scale {
    display: inline-flex;
    align-items: center;
    gap: 2px;
  }
  .scale i {
    display: inline-block;
    width: 9px;
    height: 9px;
    border-radius: 2px;
    background: var(--good);
  }
</style>
