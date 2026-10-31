export function getPath(obj: Record<string, unknown>, path: string): unknown {
  return path
    .split(".")
    .reduce<unknown>((acc, key) => (acc && typeof acc === "object" ? (acc as Record<string, unknown>)[key] : undefined), obj)
}

export function setPath(obj: Record<string, unknown>, path: string, value: unknown): void {
  const keys = path.split(".")
  let cursor: Record<string, unknown> = obj
  for (let i = 0; i < keys.length - 1; i++) {
    const key = keys[i]
    const existing = cursor[key]
    if (typeof existing !== "object" || existing === null || Array.isArray(existing)) {
      cursor[key] = {}
    }
    cursor = cursor[key] as Record<string, unknown>
  }
  cursor[keys[keys.length - 1]] = value
}

export function deletePath(obj: Record<string, unknown>, path: string): void {
  const keys = path.split(".")
  let cursor: Record<string, unknown> = obj
  for (let i = 0; i < keys.length - 1; i++) {
    const next = cursor[keys[i]]
    if (typeof next !== "object" || next === null) return
    cursor = next as Record<string, unknown>
  }
  delete cursor[keys[keys.length - 1]]
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function isEmptyContainer(value: unknown): boolean {
  if (Array.isArray(value)) return value.length === 0
  return isPlainObject(value) && Object.keys(value).length === 0
}

/**
 * Remove empty object residue without losing explicit empty lists or nulls.
 * Both can intentionally override inherited defaults or styles.
 */
export function pruneEmpty(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(pruneEmpty)
  if (isPlainObject(value)) {
    const result: Record<string, unknown> = {}
    for (const [key, entry] of Object.entries(value)) {
      if (entry === undefined) continue
      const pruned = pruneEmpty(entry)
      if (isPlainObject(pruned) && isEmptyContainer(pruned)) continue
      result[key] = pruned
    }
    return result
  }
  return value
}

function sortValue(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(sortValue)
  if (value && typeof value === "object") {
    const sorted: Record<string, unknown> = {}
    for (const key of Object.keys(value as Record<string, unknown>).sort()) {
      sorted[key] = sortValue((value as Record<string, unknown>)[key])
    }
    return sorted
  }
  return value
}

/** Stringifies with object keys sorted so key order alone never looks like a change. */
export function stableStringify(value: unknown): string {
  return JSON.stringify(sortValue(value))
}
