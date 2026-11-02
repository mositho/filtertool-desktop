// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest"
import { createApp, defineComponent, h, nextTick, ref, type App } from "vue"

const originalColor = "#112233"

vi.mock("../../frontend/src/api", () => ({
  api: {
    getSettings: async () => ({ filterPath: "", soundsFolder: "", tts: { locale: "en-US", speed: 1.6 } }),
    getDefaults: async () => ({ userDefaults: {}, baseDefaults: {}, defaults: {} }),
    getStyles: async () => ({ baseStyles: { lifeFlask: { text: originalColor } }, userStyles: {} }),
    reference: async () => ({
      ttsLocales: [],
      ttsSpeed: { min: 0.5, max: 2 },
      styleNames: [],
      styles: {},
      baseTypesByClass: {},
      amulets: [],
    }),
  },
}))

import GlobalSettings from "../../frontend/src/components/GlobalSettings.vue"
import { ruleButton } from "./ruleButtons"

let app: App | undefined
let container: HTMLDivElement | undefined

afterEach(() => {
  app?.unmount()
  app = undefined
  container?.remove()
  container = undefined
  localStorage.clear()
})

describe("global settings style history", () => {
  it("aligns default section headers and keeps nested state across top-level expand/collapse all", async () => {
    container = document.createElement("div")
    document.body.append(container)
    app = createApp(GlobalSettings)
    app.mount(container)
    await nextTick()
    await nextTick()
    const defaults = [...container.querySelectorAll("section")].find((section) => section.querySelector("h2")?.textContent === "Defaults")!
    const button = (name: string) => ruleButton(defaults, name)
    expect(button("Early").classList.contains("px-4")).toBe(false)
    button("Early").click()
    await nextTick()
    button("socketRules").click()
    await nextTick()
    button("2 Sockets · Armour").click()
    await nextTick()
    expect(button("2 Sockets · Armour").parentElement!.parentElement!.querySelector("summary")).toBeNull()
    expect(button("2 Sockets · Armour").classList.contains("py-1")).toBe(true)
    button("Collapse all").click()
    await nextTick()
    button("Expand all").click()
    await nextTick()
    expect(button("Early").getAttribute("aria-expanded")).toBe("true")
    expect(button("socketRules").getAttribute("aria-expanded")).toBe("true")
    expect(button("2 Sockets · Armour").getAttribute("aria-expanded")).toBe("true")
    expect(button("3 Sockets · Armour").getAttribute("aria-expanded")).toBe("false")
    expect(button("linkRules").getAttribute("aria-expanded")).toBe("false")
  })

  it("aligns non-rare jewellery cutoffs and sizes style rows by their container", async () => {
    container = document.createElement("div")
    document.body.append(container)
    app = createApp(GlobalSettings)
    app.mount(container)
    await nextTick()
    await nextTick()
    const section = ruleButton(container, "rarityItems").parentElement!.parentElement!
    expect(section.style.marginLeft).toBe("")
    expect(section.querySelector('[data-field-path="jewellery.amuletMaxAreaLevel"]')).not.toBeNull()
    expect(container.textContent).not.toContain("Non-Rare max area level")
    const styles = [...container.querySelectorAll("section")].find((entry) => entry.querySelector("h2")?.textContent === "Styles")!
    expect(styles.classList.contains("@container")).toBe(true)
    const flasks = [...styles.querySelectorAll("button")].find((entry) => entry.textContent?.includes("Flasks & Tinctures"))!
    flasks.click()
    await nextTick()
    expect(styles.querySelector("[class*='@min-[48rem]:grid']")).not.toBeNull()
    expect(styles.querySelector("[class*='@min-[48rem]:hidden']")).not.toBeNull()
  })

  it.each([
    { titles: ["Equipment", "genericEquipment", "Normal", "Socketable Equipment"], path: "normalItems.rules.equipment" },
    { titles: ["Equipment", "genericEquipment", "Normal", "Belts"], path: "normalItems.rules.belts" },
    { titles: ["Equipment", "genericEquipment", "Rare", "1×1"], path: "rareItems.rules.1x1" },
    { titles: ["Early", "socketRules", "2 Sockets · Armour"], path: "early.rules.armour2" },
    { titles: ["Links", "linkRules", "3 Links · Generic"], path: "links.rules.3generic" },
  ])("shows only paired global maximum and size controls for $path", async ({ titles, path }) => {
    container = document.createElement("div")
    document.body.append(container)
    app = createApp(GlobalSettings)
    app.mount(container)
    await nextTick()
    await nextTick()
    const scope = titles[0] === "Equipment" ? container.querySelector('[data-defaults-group="equipment"]')! : container
    const button = (name: string) => ruleButton(scope, name)
    for (const title of titles) {
      button(title).click()
      await nextTick()
    }
    const cutoff = container.querySelector(`[data-field-path="${path}.maxAreaLevel"]`)!
    expect(container.querySelector(`[data-field-path="${path}.style"]`)).toBeNull()
    const size = container.querySelector(`[data-field-path="${path}.size"]`)!
    expect(size.closest("details")).toBeNull()
    expect(size.parentElement!.parentElement).toBe(cutoff.parentElement!.parentElement)
    expect(size.parentElement!.parentElement!.classList.contains("sm:grid-cols-2")).toBe(true)
    const shortcut = size.querySelector<HTMLInputElement>("input")!
    shortcut.value = "31"
    shortcut.dispatchEvent(new Event("input", { bubbles: true }))
    await nextTick()
    expect(shortcut.value).toBe("31")
    const rule = button(titles.at(-1)!).parentElement!.parentElement!
    expect(rule.querySelectorAll("[data-field-path]")).toHaveLength(2)
    expect(rule.querySelector("select")).toBeNull()
    expect(rule.querySelector("summary")).toBeNull()
    expect(container.querySelector('[data-field-path="rareItems.partOneMaxAreaLevel"]')).toBeNull()
    expect(button(titles.at(-1)!).parentElement!.parentElement!.textContent).not.toContain("Preferred Armour")
  })

  it("has no visibility or Advanced disclosure inside global rule rows", async () => {
    container = document.createElement("div")
    document.body.append(container)
    app = createApp(GlobalSettings)
    app.mount(container)
    await nextTick()
    await nextTick()
    const button = (name: string) => ruleButton(container!, name)
    button("Early").click()
    await nextTick()
    button("socketRules").click()
    await nextTick()
    const parentHeader = button("socketRules").parentElement!
    expect(parentHeader.parentElement!.style.marginLeft).toBe("")
    expect(button("socketRules").querySelector("span")!.classList.contains("uppercase")).toBe(true)
    expect(button("socketRules").querySelector("span")!.classList.contains("tracking-wide")).toBe(true)
    button("2 Sockets · Armour").click()
    await nextTick()
    const header = button("2 Sockets · Armour").parentElement!
    const rule = header.parentElement!
    expect(header.classList.contains("px-4")).toBe(false)
    expect(rule.style.marginLeft).toBe("")
    expect(button("2 Sockets · Armour").querySelector("span")!.classList.contains("uppercase")).toBe(false)
    expect(rule.textContent).not.toContain("Body armour, gloves, boots and helmets.")
    expect(rule.querySelector("summary")).toBeNull()
    expect(rule.querySelector("details")).toBeNull()
    expect(rule.querySelectorAll("button[aria-expanded]")).toHaveLength(1)
    expect(rule.querySelector("select")).toBeNull()
    expect(rule.querySelector('input[type="checkbox"]')).toBeNull()
    expect(rule.textContent).not.toContain("using default")
  })

  it("keeps global early cutoffs linked until explicitly overridden, and restores them with Use default", async () => {
    container = document.createElement("div")
    document.body.append(container)
    app = createApp(GlobalSettings)
    app.mount(container)
    await nextTick()
    await nextTick()
    const row = (name: string) =>
      [...container!.querySelectorAll("label")].find((label) => label.querySelector("span")?.textContent === name)!.closest(".space-y-1")!
    const setNumber = async (name: string, value: string) => {
      const input = row(name).querySelector("input")!
      input.value = value
      input.dispatchEvent(new Event("input", { bubbles: true }))
      await nextTick()
    }
    await setNumber("Early Max Area Level", "22")
    expect(row("Three Sockets").querySelector("input")?.value).toBe("22")
    expect(row("Early Shields · Max Area Level").querySelector("input")?.value).toBe("22")
    await setNumber("Three Sockets", "16")
    await setNumber("Early Max Area Level", "24")
    expect(row("Three Sockets").querySelector("input")?.value).toBe("16")
    row("Three Sockets").querySelector<HTMLButtonElement>('button[title="Use default"]')!.click()
    await nextTick()
    expect(row("Three Sockets").querySelector("input")?.value).toBe("24")
    expect(row("Three Sockets").textContent).not.toContain("using default")
    expect(row("Three Sockets").querySelector('button[title="Use default"]')).toBeNull()
  })

  it("restores a style on undo and still recognizes subsequent edits for discard", async () => {
    const exposed = ref<InstanceType<typeof GlobalSettings> | null>(null)
    container = document.createElement("div")
    document.body.append(container)
    app = createApp(defineComponent({ setup: () => () => h(GlobalSettings, { ref: exposed }) }))
    app.mount(container)
    await nextTick()
    await nextTick()
    const flaskSection = [...container.querySelectorAll<HTMLButtonElement>("button")].find((button) =>
      button.textContent?.includes("Flasks & Tinctures"),
    )!
    flaskSection.click()
    await nextTick()

    const getLifeFlaskTextColor = () => {
      const name = [...container!.querySelectorAll<HTMLElement>("span")].find((span) => span.textContent?.trim() === "lifeFlask")!
      return name.parentElement!.querySelector<HTMLInputElement>('input[type="color"]')!
    }
    const changeColor = (color: string) => {
      const input = getLifeFlaskTextColor()
      input.value = color
      input.dispatchEvent(new Event("input", { bubbles: true }))
    }

    changeColor("#aabbcc")
    await nextTick()
    expect(exposed.value?.dirty).toBe(true)
    ;(exposed.value as unknown as { undo: () => void }).undo()
    await nextTick()
    expect(getLifeFlaskTextColor().value).toBe(originalColor)
    expect(exposed.value?.dirty).toBe(false)

    changeColor("#445566")
    await nextTick()
    expect(exposed.value?.dirty).toBe(true)
    ;(exposed.value as unknown as { discard: () => void }).discard()
    await nextTick()
    expect(getLifeFlaskTextColor().value).toBe(originalColor)
    expect(exposed.value?.dirty).toBe(false)
  })
})
