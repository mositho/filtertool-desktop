// @vitest-environment jsdom
import { afterEach, describe, expect, it } from "vitest"
import { createApp, defineComponent, h, nextTick, reactive, type App } from "vue"
import HighlightCard from "../../frontend/src/components/HighlightCard.vue"
import type { ReferenceData } from "../../frontend/src/api"
import { TEST_BASE_TYPE } from "../fixtures/base-types"
import { emptyWeaponHighlight } from "../../src/config/form-schema"

let app: App | undefined
let container: HTMLDivElement | undefined

afterEach(() => {
  app?.unmount()
  app = undefined
  container?.remove()
  container = undefined
})

describe("highlight modules", () => {
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
