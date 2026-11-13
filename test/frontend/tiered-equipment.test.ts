// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest"
import { createApp, nextTick, reactive, type App } from "vue"
import ConfigForm from "../../frontend/src/components/ConfigForm.vue"
import reference from "../../internal/data/reference.json"
import styles from "../../internal/data/styles.json"
import defaults from "../../internal/data/defaults.json"
import { DEFAULTS_SCHEMA, BUILD_SPECIFIC_SCHEMA } from "../../src/config/form-schema"
import { getPath, pruneEmpty } from "../../frontend/src/path"
import type { FilterConfig, ReferenceData } from "../../frontend/src/api"
import { ruleButton } from "./ruleButtons"
import { fieldStyleKeys } from "../../frontend/src/namedStyles"

let app: App
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
  localStorage.clear()
})

it("shares Tiered controls across defaults and filters and restores inherited toggles/styles", async () => {
  const tiered = (groups: typeof DEFAULTS_SCHEMA) =>
    groups.find((g) => g.key === "equipment")!.subsections!.find((s) => s.key === "tiered")!
  expect(tiered(DEFAULTS_SCHEMA)).toEqual(tiered(BUILD_SPECIFIC_SCHEMA))
  const config = reactive<FilterConfig>({ buildProfile: {}, buildSpecificOptions: {} })
  root = document.createElement("div")
  document.body.append(root)
  app = createApp(ConfigForm, {
    modelValue: config,
    filterName: "tiered-test",
    defaults,
    reference: { ...reference, styles, sounds: [] } as unknown as ReferenceData,
  })
  app.mount(root)
  for (const key of ["Tiered", "Rare"]) {
    ruleButton(root, key).click()
    await nextTick()
  }
  const toggle = root.querySelector<HTMLButtonElement>('[data-field-path="rareItems.tiers.2.enabled"] button[role="switch"]')!
  expect(toggle.getAttribute("aria-checked")).toBe("true")
  toggle.click()
  await nextTick()
  expect(getPath(config.buildSpecificOptions, "rareItems.tiers.2.enabled")).toBe(false)
  toggle.click()
  await nextTick()
  expect(getPath(config.buildSpecificOptions, "rareItems.tiers.2.enabled")).toBeUndefined()
  const shortcut = root.querySelector<HTMLButtonElement>('[data-field-path="rareItems.tiers.2.enabled"] button[title*="style" i]')!
  shortcut.click()
  await nextTick()
  const size = document.querySelector<HTMLInputElement>('[data-style-key="rareTier2"] input[inputmode="decimal"]')!
  for (const value of ["30", "45"]) {
    size.value = value
    size.dispatchEvent(new Event("input", { bubbles: true }))
    await nextTick()
  }
  expect(pruneEmpty(config.buildSpecificOptions)).toEqual({})
  expect(pruneEmpty(config.localStyles ?? {})).toEqual({})
  const magic = tiered(BUILD_SPECIFIC_SCHEMA).subsections!.find((s) => s.key === "magicTiers")!
  for (const tier of magic.fields) {
    expect(tier.defaultValue).toBe(false)
    expect(styles).toHaveProperty(fieldStyleKeys(tier)[0])
  }
})
