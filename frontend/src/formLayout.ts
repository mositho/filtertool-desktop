import type { SchemaField } from "../../src/config/form-schema"

/** Pair adjacent compact controls; wide controls and unpaired fields fill a row. */
export function layoutFields(fields: SchemaField[]): { field: SchemaField; fullWidth: boolean }[] {
  const wide = (field: SchemaField) =>
    field.fullWidth ?? ["multiselect", "link-colors", "style", "icon", "rule-sound"].includes(field.control)
  let column = 0
  return fields.map((field, index) => {
    let fullWidth = wide(field)
    if (!fullWidth) {
      const next = fields[index + 1]
      fullWidth = column === 0 && (!next || wide(next))
    }
    column = fullWidth ? 0 : 1 - column
    return { field, fullWidth }
  })
}
