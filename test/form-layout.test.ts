import { expect, it } from "vitest"
import { layoutFields } from "../frontend/src/formLayout"
import type { SchemaField } from "../src/config/form-schema"

function layout(...controls: SchemaField["control"][]) {
  return layoutFields(controls.map((control, index) => ({ path: String(index), label: String(index), control }))).map(
    (entry) => entry.fullWidth,
  )
}

it("fills the row for a lone compact control", () => {
  expect(layout("number")).toEqual([true])
  expect(layout("select")).toEqual([true])
})

it("pairs compact controls without leaving an unused half-row", () => {
  expect(layout("number", "number")).toEqual([false, false])
  expect(layout("number", "number", "number")).toEqual([false, false, true])
  expect(layout("number", "number", "number", "number")).toEqual([false, false, false, false])
})

it("starts a fresh row around wide controls", () => {
  expect(layout("multiselect", "number")).toEqual([true, true])
  expect(layout("number", "multiselect")).toEqual([true, true])
  expect(layout("number", "multiselect", "number")).toEqual([true, true, true])
  expect(layout("multiselect", "number", "number", "icon", "select")).toEqual([true, false, false, true, true])
})

it("honours explicitly wide fields", () => {
  const fields: SchemaField[] = [
    { path: "one", label: "One", control: "number", fullWidth: true },
    { path: "two", label: "Two", control: "number" },
  ]
  expect(layoutFields(fields).map((entry) => entry.fullWidth)).toEqual([true, true])
  expect(layoutFields([])).toEqual([])
})
