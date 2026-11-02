import { BUILD_PROFILE_SCHEMA, BUILD_SPECIFIC_SCHEMA, type FieldSubsection, type SchemaField } from "../../src/config/form-schema"
import { getPath } from "./path"

const subsectionFields = (section: FieldSubsection): SchemaField[] => [
  ...section.fields,
  ...(section.subsections ?? []).flatMap(subsectionFields),
]
const fields = [...BUILD_PROFILE_SCHEMA, ...BUILD_SPECIFIC_SCHEMA].flatMap((group) => [
  ...group.fields,
  ...(group.subsections ?? []).flatMap(subsectionFields),
])

/** Resolve linked defaults without materializing them into the saved configuration. */
export function fieldDefault(
  field: SchemaField,
  local: Record<string, unknown>,
  shared: Record<string, unknown> = {},
  styles: Record<string, Record<string, unknown>> = {},
): unknown {
  const value = getPath(shared, field.path)
  if (value !== undefined) return value
  if (field.sizePreset) return styles[field.sizePreset]?.size ?? field.defaultValue
  if (field.defaultFrom) {
    const { path, value: fallback } = field.defaultFrom
    const explicit = getPath(local, path) ?? getPath(shared, path)
    if (explicit !== undefined) return explicit
    const source = fields.find((candidate) => candidate.path === path)
    return source ? fieldDefault(source, local, shared, styles) : fallback
  }
  if (field.defaultWhen) {
    const { path, equals, defaultValue, value: whenValue } = field.defaultWhen
    return (getPath(local, path) ?? defaultValue) === equals ? whenValue : undefined
  }
  return field.defaultValue
}

export function defaultSource(field: SchemaField, shared: Record<string, unknown> = {}): string | undefined {
  return getPath(shared, field.path) === undefined ? field.defaultFrom?.label : undefined
}
