// @vitest-environment jsdom
import { afterEach, describe, expect, it } from "vitest"
import { createApp, defineComponent, h, nextTick, reactive, type App } from "vue"
import HighlightCard from "../../frontend/src/components/HighlightCard.vue"
import WeaponPhaseTemplateEditor from "../../frontend/src/components/WeaponPhaseTemplateEditor.vue"
import type { ReferenceData } from "../../frontend/src/api"
import { TEST_BASE_TYPE } from "../fixtures/base-types"
import { emptyHighlight, emptyWeaponHighlight } from "../../src/config/form-schema"
import { pruneEmpty } from "../../frontend/src/path"

let app: App | undefined
let container: HTMLDivElement | undefined

afterEach(() => {
  app?.unmount()
  app = undefined
  container?.remove()
  container = undefined
})

describe("highlight modules", () => {
  it("restores inherited highlight defaults and module metadata after a round trip", async () => {
    const highlight = reactive<Record<string, unknown>>({ baseTypes: [TEST_BASE_TYPE] })
    const reference = {
      itemClasses: [],
      weaponClasses: [],
      baseTypesByClass: {},
      sounds: [],
      styles: {},
      styleNames: [],
    } as unknown as ReferenceData
    container = document.createElement("div")
    document.body.append(container)
    app = createApp({ setup: () => () => h(HighlightCard, { highlight, reference }) })
    app.mount(container)
    const mode = container.querySelector<HTMLButtonElement>('[role="switch"]')!
    mode.click()
    await nextTick()
    const rare = container.querySelector('[data-rarity="Rare"]')!
    rare.querySelector<HTMLButtonElement>('[title="Square"]')!.click()
    await nextTick()
    rare.querySelector<HTMLButtonElement>('[title="Star"]')!.click()
    await nextTick()
    mode.click()
    await nextTick()
    expect(pruneEmpty(highlight)).toEqual({ baseTypes: [TEST_BASE_TYPE] })
  })
  it.each([false, true])("offers every applicable phase module before selecting targets, perRarity=%s", async (perRarityCustomization) => {
    const modelValue = reactive<Record<string, unknown>>({ perRarityCustomization })
    const reference = {
      itemClasses: [],
      weaponClasses: [],
      baseTypesByClass: {},
      sounds: [],
      styles: {},
      styleNames: [],
    } as unknown as ReferenceData
    container = document.createElement("div")
    document.body.append(container)
    app = createApp({ setup: () => () => h(WeaponPhaseTemplateEditor, { modelValue, reference }) })
    app.mount(container)
    ;[...container.querySelectorAll<HTMLButtonElement>("button")].find((button) => button.textContent?.includes("Add module"))!.click()
    await nextTick()
    const buttons = [...container.querySelectorAll<HTMLButtonElement>("button")]
    for (const label of [
      "Minimum Attack Speed",
      "Minimum Linked Sockets",
      "Minimum Sockets",
      "Item Level",
      "Weapon Cutoff",
      "Size",
      "Style",
    ]) {
      expect(buttons.some((button) => button.textContent?.trim() === label)).toBe(true)
    }
    expect(buttons.some((button) => button.textContent?.trim() === "Area Level")).toBe(false)
    for (const label of ["Icon", "Sound"])
      expect(buttons.some((button) => button.textContent?.trim() === label)).toBe(!perRarityCustomization)
    buttons.find((button) => button.textContent?.trim() === "Weapon Cutoff")!.click()
    await nextTick()
    expect(modelValue.weaponCutoffOverlap).toBe(10)
    expect(modelValue.modules).toContain("weaponCutoff")
  })

  it("enables per-rarity customization on newly created highlights and weapon phases", () => {
    expect(emptyHighlight()).toEqual({ perRarityCustomization: true })
    expect(emptyWeaponHighlight()).toEqual({ perRarityCustomization: true })
  })

  it("does not add optional defaults when creating a weapon phase", () => {
    expect(emptyWeaponHighlight()).toEqual({ perRarityCustomization: true })
  })

  it("shows inherited rarity defaults and preserves explicit no-icon/no-sound overrides", async () => {
    const highlight = reactive<Record<string, unknown>>({ baseTypes: [TEST_BASE_TYPE], modules: ["weaponCutoff"] })
    const reference = {
      itemClasses: [],
      weaponClasses: [],
      baseTypesByClass: {},
      sounds: [],
      styles: {},
      styleNames: [],
    } as unknown as ReferenceData
    container = document.createElement("div")
    document.body.append(container)
    app = createApp(defineComponent({ setup: () => () => h(HighlightCard, { highlight, reference }) }))
    app.mount(container)
    container.querySelector<HTMLButtonElement>('[role="switch"]')!.click()
    await nextTick()
    expect(highlight.modules).toContain("weaponCutoff")
    expect(highlight.weaponCutoffOverlap).toBeUndefined()
    for (const [rarity, color] of [
      ["Normal", "Grey"],
      ["Magic", "Blue"],
      ["Rare", "Yellow"],
    ]) {
      const row = container.querySelector(`[data-rarity="${rarity}"]`)!
      expect(row.querySelector('[title="Star"]')!.classList.contains("border-white")).toBe(true)
      expect(row.querySelector(`[title="${color}"]`)!.classList.contains("border-white")).toBe(true)
    }
    expect(highlight.normal).toBeUndefined()
    expect(highlight.rare).toBeUndefined()
    const rare = container.querySelector('[data-rarity="Rare"]')!
    const soundType = rare.querySelector<HTMLSelectElement>("select")!
    expect(soundType.value).toBe("alert")
    expect(rare.querySelectorAll<HTMLSelectElement>("select")[1].value).toBe("3")
    soundType.value = "none"
    soundType.dispatchEvent(new Event("change", { bubbles: true }))
    await nextTick()
    expect(highlight.rare).toEqual({ soundEnabled: false })
    container.querySelector<HTMLButtonElement>('[data-rarity="Normal"] [title="No Icon"]')!.click()
    await nextTick()
    expect(highlight.normal).toEqual({ iconShape: "" })
    container.querySelector<HTMLButtonElement>('[role="switch"]')!.click()
    await nextTick()
    container.querySelector<HTMLButtonElement>('[role="switch"]')!.click()
    await nextTick()
    expect(container.querySelector<HTMLSelectElement>('[data-rarity="Rare"] select')!.value).toBe("none")
    expect(container.querySelector('[data-rarity="Normal"] [title="No Icon"]')!.classList.contains("border-white")).toBe(true)
  })

  it.each([
    ["minAps", "Minimum Attack Speed"],
    ["linkedSockets", "Minimum Linked Sockets"],
    ["minSockets", "Minimum Sockets"],
    ["areaLevel", "Area Level"],
    ["itemLevel", "Item Level"],
    ["weaponCutoff", "Weapon Cutoff"],
    ["size", "Size"],
    ["style", "Style"],
    ["icon", "Icon"],
    ["sound", "Sound"],
  ])("adds the %s module and persists its module id", async (moduleId, label) => {
    const highlight = reactive<Record<string, unknown>>({ baseTypes: [TEST_BASE_TYPE] })
    const reference = {
      itemClasses: ["Synthetic Weapon Class"],
      weaponClasses: ["Synthetic Weapon Class"],
      baseTypesByClass: { "Synthetic Weapon Class": [TEST_BASE_TYPE] },
      sounds: [],
      styles: {},
      styleNames: [],
    } as unknown as ReferenceData
    container = document.createElement("div")
    document.body.append(container)
    app = createApp(defineComponent({ setup: () => () => h(HighlightCard, { highlight, reference }) }))
    app.mount(container)

    Array.from(container.querySelectorAll<HTMLButtonElement>("button"))
      .find((button) => button.textContent?.includes("Add module"))!
      .click()
    await nextTick()
    Array.from(container.querySelectorAll<HTMLButtonElement>("button"))
      .find((button) => button.textContent?.trim() === label)!
      .click()
    await nextTick()

    expect(highlight.modules).toContain(moduleId)
    expect(container.textContent).toContain(label)
    if (moduleId === "weaponCutoff") expect(highlight.weaponCutoffOverlap).toBe(10)
    container.querySelector<HTMLButtonElement>(`[title="Remove ${label}"]`)!.click()
    await nextTick()
    expect(highlight.modules).toBeUndefined()
    if (moduleId === "weaponCutoff") expect(highlight.weaponCutoffOverlap).toBeUndefined()
    app.unmount()
    app = undefined
    container.remove()
    container = undefined
  })
})
