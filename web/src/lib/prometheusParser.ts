// Parses Prometheus text format (exposition format v0.0.4) into structured metrics.

export interface MetricSample {
  name: string
  labels: Record<string, string>
  value: number
  timestamp?: number
}

export interface ParsedMetrics {
  samples: MetricSample[]
  byName: Map<string, MetricSample[]>
}

/**
 * Parse a single label set string like `{foo="bar",baz="qux"}` into a Record.
 */
function parseLabels(raw: string): Record<string, string> {
  const labels: Record<string, string> = {}
  if (!raw) return labels
  // Strip braces
  const inner = raw.replace(/^\{/, '').replace(/\}$/, '').trim()
  if (!inner) return labels

  // Match key="value" pairs, handling escaped quotes inside values
  const re = /(\w+)="((?:[^"\\]|\\.)*)"/g
  let m: RegExpExecArray | null
  while ((m = re.exec(inner)) !== null) {
    labels[m[1]] = m[2].replace(/\\"/g, '"').replace(/\\\\/g, '\\')
  }
  return labels
}

/**
 * Parse Prometheus text format into structured samples.
 * Skips HELP and TYPE lines. Skips comment-only lines.
 */
export function parsePrometheusText(text: string): ParsedMetrics {
  const samples: MetricSample[] = []
  const byName = new Map<string, MetricSample[]>()

  for (const rawLine of text.split('\n')) {
    const line = rawLine.trim()
    if (!line || line.startsWith('#')) continue

    // Format: metric_name[{labels}] value [timestamp]
    // Use a regex that handles optional label block
    const match = line.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{[^}]*\})?\s+([^\s]+)(?:\s+(\d+))?$/)
    if (!match) continue

    const [, name, labelStr, valueStr, tsStr] = match
    const value = parseFloat(valueStr)
    if (isNaN(value)) continue

    const sample: MetricSample = {
      name,
      labels: parseLabels(labelStr ?? ''),
      value,
    }
    if (tsStr) {
      sample.timestamp = parseInt(tsStr, 10)
    }

    samples.push(sample)
    if (!byName.has(name)) byName.set(name, [])
    byName.get(name)!.push(sample)
  }

  return { samples, byName }
}

/**
 * Convenience: get the first numeric value for a named metric, or null.
 */
export function getMetricValue(parsed: ParsedMetrics, name: string): number | null {
  const samples = parsed.byName.get(name)
  if (!samples || samples.length === 0) return null
  return samples[0].value
}
