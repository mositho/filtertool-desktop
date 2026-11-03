// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest"
import { createApp, h, nextTick, ref, type App, type Ref } from "vue"
import { forgetFilterViewState, renameFilterViewState, useCollapsedState } from "../../frontend/src/viewState"
import ConfigForm from "../../frontend/src/components/ConfigForm.vue"
import type { ReferenceData } from "../../frontend/src/api"
import reference from "../../internal/data/reference.json"
import { ruleButton } from "./ruleButtons"

let app: App | undefined
let root: HTMLDivElement | undefined
afterEach(() => {
  app?.unmount()
  root?.remove()
  localStorage.clear()
})

function mountState(key: Ref<string>) {
  let state!: Ref<Record<string, boolean>>
  app = createApp({
    setup() {
      state = useCollapsedState(key, () => ({ parent: true, child: false }))
      return () => h("div")
    },
  })
  root = document.createElement("div")
  app.mount(root)
  return state
}

it("saves immediately and isolates state when the storage key changes", () => {
  const key = ref("filter-a")
  const state = mountState(key)
  state.value.parent = false
  state.value.child = true
  expect(JSON.parse(localStorage.getItem("filter-a")!)).toEqual({ parent: false, child: true })
  key.value = "filter-b"
  expect(state.value).toEqual({ parent: true, child: false })
  state.value.child = true
  key.value = "filter-a"
  expect(state.value).toEqual({ parent: false, child: true })
  app!.unmount()
  root!.remove()
  expect(mountState(key).value).toEqual({ parent: false, child: true })
})

it("keeps valid saved choices while new keys receive defaults and invalid storage is ignored", () => {
  const key = ref("view")
  localStorage.setItem("view", JSON.stringify({ parent: false, child: "wrong", another: true }))
  const state = mountState(key)
  expect(state.value).toEqual({ parent: false, child: false, another: true })
  for (const invalid of ["not json", "null", "[]", "42"]) {
    localStorage.setItem(invalid, invalid)
    key.value = invalid
    expect(state.value).toEqual({ parent: true, child: false })
  }
})

it("retains top-level, nested, Advanced and currency-tier preferences on rename", () => {
  localStorage.setItem(
    "filtertool:collapsed:old",
    JSON.stringify({ "equipment": false, "equipment:lifeFlasks": false, "early:early.rules.armour2:advanced": false }),
  )
  localStorage.setItem("filtertool:currency-tiers:old", JSON.stringify({ a: false }))
  localStorage.setItem("filtertool:styles:old", JSON.stringify({ styles: false, Currency: false }))
  renameFilterViewState("old", "new")
  expect(localStorage.getItem("filtertool:collapsed:new")).toContain('"equipment:lifeFlasks":false')
  expect(localStorage.getItem("filtertool:currency-tiers:new")).toBe('{"a":false}')
  expect(localStorage.getItem("filtertool:styles:new")).toBe('{"styles":false,"Currency":false}')
  expect(localStorage.getItem("filtertool:collapsed:old")).toBeNull()
  expect(localStorage.getItem("filtertool:currency-tiers:old")).toBeNull()
  expect(localStorage.getItem("filtertool:styles:old")).toBeNull()
  forgetFilterViewState("new")
  expect(localStorage.getItem("filtertool:collapsed:new")).toBeNull()
  expect(localStorage.getItem("filtertool:currency-tiers:new")).toBeNull()
  expect(localStorage.getItem("filtertool:styles:new")).toBeNull()
})

it("remembers every subsection per filter without modifying filter data", async () => {
  const name = ref("first")
  const modelValue = { buildProfile: {}, buildSpecificOptions: {} }
  app = createApp({
    setup: () => () =>
      h(ConfigForm, {
        modelValue,
        filterName: name.value,
        reference: { ...reference, styles: {}, sounds: [] } as unknown as ReferenceData,
      }),
  })
  root = document.createElement("div")
  document.body.append(root)
  app.mount(root)
  const button = (key: string) => ruleButton(root!, key)
  expect(root.querySelector('[data-editor-subsection="lifeFlasks"]')).toBeNull()
  button("flasks").click()
  await nextTick()
  for (const key of ["lifeFlasks", "manaFlasks", "rarityItems"]) {
    expect(button(key).getAttribute("aria-expanded")).toBe("false")
  }
  expect(button("tinctures").getAttribute("aria-expanded")).toBe("false")
  button("lifeFlasks").click()
  button("tinctures").click()
  await nextTick()
  button("Collapse all").click()
  button("Expand all").click()
  await nextTick()
  expect(button("lifeFlasks").getAttribute("aria-expanded")).toBe("true")
  expect(button("tinctures").getAttribute("aria-expanded")).toBe("true")
  name.value = "second"
  await nextTick()
  button("flasks").click()
  await nextTick()
  expect(button("lifeFlasks").getAttribute("aria-expanded")).toBe("false")
  expect(button("tinctures").getAttribute("aria-expanded")).toBe("false")
  name.value = "first"
  await nextTick()
  expect(button("lifeFlasks").getAttribute("aria-expanded")).toBe("true")
  expect(button("tinctures").getAttribute("aria-expanded")).toBe("true")
  expect(modelValue).toEqual({ buildProfile: {}, buildSpecificOptions: {} })
})
