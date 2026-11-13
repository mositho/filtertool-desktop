import { deletePath, getPath, setPath, stableStringify } from "./path"

const originals = new WeakMap<object, Map<string, { value: unknown; effective: unknown }>>()
const clone = (value: unknown): unknown => (value === undefined ? undefined : JSON.parse(JSON.stringify(value)))

/** Return-to-original edits restore the override that existed before editing,
 * including inheritance, explicit empty selections and uncapped values. */
export function setEditedValue(root: Record<string, unknown>, path: string, value: unknown, inherited: unknown, unordered = false): void {
  let paths = originals.get(root)
  if (!paths) originals.set(root, (paths = new Map()))
  if (!paths.has(path)) {
    const initial = getPath(root, path)
    paths.set(path, { value: clone(initial), effective: clone(initial === undefined ? inherited : initial) })
  }
  const original = paths.get(path)!
  const comparable = (entry: unknown) => {
    if (unordered && Array.isArray(entry)) return stableStringify([...entry].sort())
    if (entry && inherited && typeof entry === "object" && typeof inherited === "object" && !Array.isArray(entry)) {
      const effective: Record<string, unknown> = { ...inherited, ...entry }
      for (const key of ["text", "background", "border"]) {
        if (typeof effective[key] === "string") effective[key] = effective[key].toLowerCase()
      }
      return stableStringify(effective)
    }
    if (typeof entry === "string" && /(?:^|\.)(text|background|border)$/.test(path)) return entry.toLowerCase()
    return stableStringify(entry)
  }
  const restored = original.value === undefined ? inherited : original.value
  if (value !== undefined && comparable(value) === comparable(original.effective) && comparable(value) === comparable(restored)) {
    value = clone(original.value)
  }
  if (value === undefined) deletePath(root, path)
  else setPath(root, path, value)
}
