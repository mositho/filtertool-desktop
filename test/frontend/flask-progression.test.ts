// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest"
import { createApp, nextTick, reactive, type App } from "vue"
import ConfigForm from "../../frontend/src/components/ConfigForm.vue"
import type { ReferenceData } from "../../frontend/src/api"
import { getPath } from "../../frontend/src/path"
import { BUILD_SPECIFIC_SCHEMA, DEFAULTS_SCHEMA } from "../../src/config/form-schema"
import progression from "../../internal/data/flask-progression.json"
import reference from "../../internal/data/reference.json"
import styles from "../../internal/data/styles.json"
import sounds from "../../internal/data/manifest.json"
import { ruleButton } from "./ruleButtons"

let app: App
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
  localStorage.clear()
})

async function mount(defaults: Record<string, unknown> = {}) {
  const modelValue = reactive({ buildProfile: {}, buildSpecificOptions: {} })
  root = document.createElement("div")
  document.body.append(root)
  app = createApp(ConfigForm, {
    modelValue,
    defaults,
    filterName: "Flask progression test",
    reference: {
      ...reference,
      styles,
      sounds,
      styleNames: Object.keys(styles),
      ttsLocales: [],
      ttsSpeed: { min: 0.5, max: 2 },
    } as ReferenceData,
  })
  app.mount(root)
  ruleButton(root, "flasks").click()
  await nextTick()
  return modelValue
}

it("shares all flask cutoffs and Advanced Rules titles between filter and global schemas", () => {
  const equipment = BUILD_SPECIFIC_SCHEMA.find((group) => group.key === "equipment")!
  const global = DEFAULTS_SCHEMA.find((group) => group.key === "equipment")!
  for (const kind of ["life", "mana"] as const) {
    const section = equipment
      .subsections!.find((section) => section.key === "flasks")!
      .subsections!.find((section) => section.key === `${kind}Flasks`)!
    expect(
      global.subsections!.find((candidate) => candidate.key === "flasks")!.subsections!.find((candidate) => candidate.key === section.key),
    ).toEqual(section)
    expect(section.title).toBe(`${kind === "life" ? "Life" : "Mana"} Flasks`)
    expect(section.fields).toHaveLength(progression[kind].length)
    for (const field of section.fields) {
      expect(field.emptyMeansUncapped).toBe(true)
      expect(field.control).toBe("number")
    }
  }
  for (const schema of [BUILD_SPECIFIC_SCHEMA, DEFAULTS_SCHEMA]) {
    for (const key of ["socketRules", "linkRules", "genericEquipment"]) {
      expect(schema.flatMap((group) => group.subsections ?? []).find((section) => section.key === key)?.title).toBe("Advanced Rules")
    }
  }
})

it("uses compact paired fields with existing defaults and no per-flask sound controls", async () => {
  await mount()
  for (const kind of ["life", "mana"] as const) {
    const section = root.querySelector(`[data-editor-subsection="${kind}Flasks"]`)!
    expect(section.querySelector(".grid")?.classList.contains("sm:grid-cols-2")).toBe(true)
    expect(section.querySelector("details")).toBeNull()
    for (const flask of progression[kind]) {
      const field = section.querySelector(`[data-field-path="flasks.${kind}MaxAreaLevels.${flask.key}"]`)!
      expect(field.querySelector("input")?.value).toBe(flask.maxAreaLevel === null ? "" : String(flask.maxAreaLevel))
      expect(field.parentElement?.classList.contains("sm:col-span-2")).toBe(false)
    }
  }
  expect(root.textContent).not.toContain("Socket Rules")
  expect(root.textContent).not.toContain("Link Rules")
  expect(root.textContent).not.toContain("Equipment Rules")
})

it("distinguishes inherited, explicit cutoff, and explicitly uncapped values", async () => {
  const defaults = reactive({ flasks: { lifeMaxAreaLevels: { small: 18 } } })
  const model = await mount(defaults)
  const field = root.querySelector('[data-field-path="flasks.lifeMaxAreaLevels.small"]')!
  const input = field.querySelector("input")!
  expect(input.value).toBe("18")
  const update = async (value: string) => {
    input.value = value
    input.dispatchEvent(new Event("input", { bubbles: true }))
    await nextTick()
  }
  await update("25")
  expect(getPath(model.buildSpecificOptions, "flasks.lifeMaxAreaLevels.small")).toBe(25)
  await update("")
  expect(getPath(model.buildSpecificOptions, "flasks.lifeMaxAreaLevels.small")).toBe(25)
  input.dispatchEvent(new FocusEvent("blur"))
  await nextTick()
  expect(getPath(model.buildSpecificOptions, "flasks.lifeMaxAreaLevels.small")).toBeNull()
  expect(input.value).toBe("")
  expect(input.placeholder).toBe("No cutoff")
  defaults.flasks.lifeMaxAreaLevels.small = 22
  await nextTick()
  expect(input.value).toBe("")
  field.querySelector<HTMLButtonElement>('button[title="Use default"]')!.click()
  await nextTick()
  expect(getPath(model.buildSpecificOptions, "flasks.lifeMaxAreaLevels.small")).toBeUndefined()
  expect(input.value).toBe("22")
})
