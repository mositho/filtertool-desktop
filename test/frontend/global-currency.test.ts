// @vitest-environment jsdom
import { afterEach, beforeEach, expect, it, vi } from "vitest"
import { createApp, defineComponent, h, nextTick, ref, type App } from "vue"
import GlobalSettings from "../../frontend/src/components/GlobalSettings.vue"
import reference from "../../internal/data/reference.json"
import styles from "../../internal/data/styles.json"
import sounds from "../../internal/data/manifest.json"
import { getPath } from "../../frontend/src/path"
import { ruleButton } from "./ruleButtons"

const mock = vi.hoisted(() => ({
  getSettings: vi.fn(),
  getDefaults: vi.fn(),
  getStyles: vi.fn(),
  reference: vi.fn(),
  saveSettings: vi.fn(),
  saveDefaults: vi.fn(),
  saveStyles: vi.fn(),
  generateSounds: vi.fn(),
}))
vi.mock("../../frontend/src/api", () => ({ api: mock }))

let app: App
let root: HTMLDivElement
const exposed = ref<InstanceType<typeof GlobalSettings> | null>(null)
let saved: Record<string, unknown>[]

beforeEach(() => {
  vi.resetAllMocks()
  saved = []
  mock.getSettings.mockResolvedValue({ filterPath: "", soundsFolder: "", tts: { locale: "en-US", speed: 1.6 } })
  mock.getDefaults.mockResolvedValue({
    userDefaults: {},
    baseDefaults: {
      currency: { assignments: { "Chromatic Orb": "leaguestart" }, gold: { minStackSize: 5 } },
      leagueMechanic: { rules: { charts: { baseTypes: ["Sandy Seabed Chart"] } } },
    },
  })
  mock.getStyles.mockResolvedValue({ baseStyles: styles, userStyles: {} })
  mock.reference.mockResolvedValue({
    ...reference,
    styles,
    sounds,
    baseTypesByClass: { "Stackable Currency": ["Chromatic Orb", "Chaos Orb", "Scroll Fragment"] },
    currencySounds: { "Chromatic Orb": { enabled: true, tts: "chromatic_orb" } },
  })
  mock.saveSettings.mockResolvedValue({})
  mock.saveStyles.mockResolvedValue({})
  mock.saveDefaults.mockImplementation(async (value) => {
    saved.push(JSON.parse(JSON.stringify(value)))
    return {}
  })
})

afterEach(() => {
  app?.unmount()
  root?.remove()
  localStorage.clear()
})

async function mount(devMode = false) {
  root = document.createElement("div")
  document.body.append(root)
  app = createApp(defineComponent({ setup: () => () => h(GlobalSettings, { ref: exposed, devMode }) }))
  app.mount(root)
  await nextTick()
  await nextTick()
  for (const key of ["currency", "leagueMechanic"]) {
    root.querySelector<HTMLButtonElement>(`[data-defaults-group="${key}"] > button`)!.click()
    await nextTick()
  }
}

it("offers one Generate & Sync action with a short description", async () => {
  let finish!: () => void
  mock.generateSounds.mockImplementation(
    () =>
      new Promise<void>((resolve) => {
        finish = resolve
      }),
  )
  await mount()
  expect(root.textContent).toContain("Update TTS sounds, clean the local cache, and copy sounds to your PoE folder.")
  const button = [...root.querySelectorAll<HTMLButtonElement>("button")].find((entry) => entry.textContent?.trim() === "Generate & Sync")!
  expect([...root.querySelectorAll("button")].some((entry) => entry.textContent?.trim() === "Sync sounds")).toBe(false)
  button.click()
  await nextTick()
  expect(mock.saveSettings).toHaveBeenCalledOnce()
  expect(mock.generateSounds).toHaveBeenCalledOnce()
  expect(button.disabled).toBe(true)
  expect(button.textContent).toContain("Generating & Syncing")
  finish()
  await nextTick()
  await nextTick()
  expect(button.disabled).toBe(false)
  expect(root.textContent).toContain("Sounds generated and synced.")
})

it("uses the current currency/league editors and keeps styles in the Styles panel", async () => {
  await mount()
  const currency = root.querySelector('[data-defaults-group="currency"]')!
  const league = root.querySelector('[data-defaults-group="leagueMechanic"]')!
  expect(currency.querySelectorAll("[data-currency-tier]")).toHaveLength(5)
  expect(currency.querySelector('[aria-label="Currency item settings"]')?.textContent).toContain("Select Currency")
  expect(league.textContent).toContain("Curse of the Allflame")
  expect(league.querySelectorAll('[data-field-path$=".baseTypes"]')).toHaveLength(6)
  expect(currency.querySelector("summary")).toBeNull()
  expect(league.querySelector("summary")).toBeNull()
  expect(currency.querySelector('[data-field-path$=".style"]')).toBeNull()
  expect(league.querySelector('[data-field-path$=".style"]')).toBeNull()
  const stylePanel = [...root.querySelectorAll("section")].find((section) => section.querySelector("h2")?.textContent === "Styles")!
  for (const name of ["Currency", "League Mechanic"]) {
    ;[...stylePanel.querySelectorAll<HTMLButtonElement>("button")]
      .find((button) => button.querySelector("span")?.textContent === name)!
      .click()
    await nextTick()
  }
  expect(stylePanel.textContent).toContain("currencyLeaguestart")
  expect(stylePanel.textContent).toContain("leagueMechanic")
  ;[...stylePanel.querySelectorAll<HTMLButtonElement>("button")].find((button) => button.textContent?.includes("Links"))!.click()
  await nextTick()
  expect(stylePanel.textContent).toContain("fiveLink")
  expect(stylePanel.textContent).toContain("sixLink")
  expect([...root.querySelectorAll("[data-defaults-group]")].map((group) => group.getAttribute("data-defaults-group"))).toEqual([
    "buildProfile",
    "weaponPhaseTemplate",
    "early",
    "links",
    "equipment",
    "currency",
    "leagueMechanic",
  ])
  expect(exposed.value?.dirty).toBe(false)
  expect(root.textContent).not.toContain("using default")
})

it("edits and saves a reusable default weapon phase template", async () => {
  await mount()
  const group = root.querySelector<HTMLElement>('[data-defaults-group="weaponPhaseTemplate"]')!
  expect(group.querySelector("button")?.textContent).toContain("Weapon Phase")
  expect(group.querySelector("button")?.textContent).not.toContain("Default Weapon Phase")
  group.querySelector<HTMLButtonElement>("button")!.click()
  await nextTick()
  expect(group.querySelector("[data-weapon-phase-template]")).not.toBeNull()
  const advanced = [...group.querySelectorAll<HTMLDetailsElement>("details")].find(
    (details) => details.querySelector("summary")?.textContent === "Advanced",
  )!
  advanced.open = true
  const addModule = [...group.querySelectorAll<HTMLButtonElement>("button")].find(
    (button) => button.textContent?.trim() === "+ Add module",
  )!
  addModule.click()
  await nextTick()
  ;[...group.querySelectorAll<HTMLButtonElement>("button")].find((button) => button.textContent?.trim() === "Minimum Attack Speed")!.click()
  await nextTick()
  expect(group.textContent).toContain("Minimum Attack Speed")
  const module = group.querySelector<HTMLButtonElement>('[title="Remove Minimum Attack Speed"]')!.parentElement!.parentElement!
  const speed = module.querySelector<HTMLInputElement>('input[inputmode="decimal"]')!
  speed.value = "1.5"
  speed.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  await exposed.value!.save()
  expect(getPath(saved[0], "weaponPhaseTemplate.perRarityCustomization")).toBe(true)
  expect(getPath(saved[0], "weaponPhaseTemplate.modules")).toContain("minAps")
  expect(getPath(saved[0], "weaponPhaseTemplate.minAps")).toBe(1.5)
})

it("saves tier fallback settings globally without replacing existing item TTS", async () => {
  await mount()
  root.querySelector<HTMLButtonElement>('[aria-label="B Tier settings"]')!.click()
  await nextTick()
  const tierSound = root.querySelector('[data-field-path="currency.tiers.b.sound"]')!
  const mode = tierSound.querySelector<HTMLSelectElement>("select")!
  expect(mode.value).toBe("alert")
  const soundId = tierSound.querySelectorAll("select")[1]
  expect(soundId.value).toBe("2")
  soundId.value = "8"
  soundId.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  const icon = root.querySelector('[data-field-path="currency.tiers.b.icon"]')!
  icon.querySelector<HTMLButtonElement>('[title="Star"]')!.click()
  await nextTick()
  await exposed.value!.save()
  expect(getPath(saved[0], "currency.tiers.b.sound.soundId")).toBe(8)
  expect(getPath(saved[0], "currency.tiers.b.icon.iconShape")).toBe("Star")
  root.querySelector<HTMLButtonElement>('[data-currency-item="Chromatic Orb"]')!.click()
  await nextTick()
  const sound = root.querySelector<HTMLSelectElement>('[data-field-path="currency.items.Chromatic Orb.sound"] select')!
  expect(sound.value).toBe("preset")
  expect(getPath(saved[0], "currency.items.Chromatic Orb.sound")).toBeUndefined()
})

it("reserves reset space without inheritance labels in global default fields", async () => {
  await mount()
  const field = root.querySelector('[data-field-path="currency.gold.minStackSize"]')!
  const header = field.firstElementChild!
  const resetSlot = header.lastElementChild!
  expect(resetSlot.classList.contains("w-5")).toBe(true)
  expect(resetSlot.classList.contains("h-5")).toBe(true)
  expect(resetSlot.querySelector("button")).toBeNull()
  const input = field.querySelector("input")!
  input.value = "10"
  input.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  expect(header.lastElementChild).toBe(resetSlot)
  resetSlot.querySelector<HTMLButtonElement>('button[title="Use default"]')!.click()
  await nextTick()
  expect(input.value).toBe("5")
  expect(header.lastElementChild).toBe(resetSlot)
  expect(resetSlot.querySelector("button")).toBeNull()
  expect(root.textContent).not.toContain("using default")
})

it("rebases saved dev defaults and styles so reset uses the newly authored source", async () => {
  let minimum = 5
  const diskStyles = structuredClone(styles)
  mock.getDefaults.mockImplementation(async () => ({ baseDefaults: { currency: { gold: { minStackSize: minimum } } }, userDefaults: {} }))
  mock.getStyles.mockImplementation(async () => ({ baseStyles: structuredClone(diskStyles), userStyles: {} }))
  mock.saveDefaults.mockImplementation(async (value) => {
    const next = getPath(value, "currency.gold.minStackSize")
    if (typeof next === "number") minimum = next
    return {}
  })
  mock.saveStyles.mockImplementation(async (value) => {
    const next = getPath(value, "lifeFlask.text")
    if (typeof next === "string") diskStyles.lifeFlask.text = next
    return {}
  })
  await mount(true)
  const field = root.querySelector('[data-field-path="currency.gold.minStackSize"]')!
  const input = field.querySelector("input")!
  input.value = "9"
  input.dispatchEvent(new Event("input", { bubbles: true }))
  const stylePanel = [...root.querySelectorAll("section")].find((section) => section.querySelector("h2")?.textContent === "Styles")!
  ;[...stylePanel.querySelectorAll<HTMLButtonElement>("button[aria-expanded]")]
    .find((button) => button.querySelector("span")?.textContent === "Flasks & Tinctures")!
    .click()
  await nextTick()
  const row = () => stylePanel.querySelector('span[title="lifeFlask"]')!.parentElement!
  const color = row().querySelector<HTMLInputElement>('input[type="color"]')!
  color.value = "#123456"
  color.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  expect(field.querySelector('button[title="Use default"]')).not.toBeNull()
  expect(row().querySelector<HTMLButtonElement>('[title="Reset style"]')!.disabled).toBe(false)
  expect(exposed.value?.dirty).toBe(true)
  expect((await exposed.value!.save()).kind).toBe("notice")
  await nextTick()
  expect(input.value).toBe("9")
  expect(color.value).toBe("#123456")
  expect(field.querySelector('button[title="Use default"]')).toBeNull()
  expect(row().querySelector('[title="Reset style"]')?.classList.contains("invisible")).toBe(true)
  expect(exposed.value?.dirty).toBe(false)
  expect(exposed.value?.canUndo).toBe(false)
  input.value = "12"
  input.dispatchEvent(new Event("input", { bubbles: true }))
  color.value = "#abcdef"
  color.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  field.querySelector<HTMLButtonElement>('button[title="Use default"]')!.click()
  row().querySelector<HTMLButtonElement>('[title="Reset style"]')!.click()
  await nextTick()
  expect(input.value).toBe("9")
  expect(color.value).toBe("#123456")
  expect(exposed.value?.dirty).toBe(false)
})

it("mirrors flask progression and paired field layouts using global reset semantics", async () => {
  await mount()
  ruleButton(root, "flasks").click()
  await nextTick()
  for (const kind of ["life", "mana"]) {
    const section = root.querySelector(`[data-editor-subsection="${kind}Flasks"]`)!
    expect(section.querySelectorAll("[data-field-path]")).toHaveLength(12)
    expect(section.querySelector(".grid")?.classList.contains("sm:grid-cols-2")).toBe(true)
  }
  const field = root.querySelector('[data-field-path="flasks.lifeMaxAreaLevels.small"]')!
  const input = field.querySelector("input")!
  expect(input.value).toBe("12")
  expect(field.querySelector('button[title="Use default"]')).toBeNull()
  input.value = ""
  input.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  expect(input.value).toBe("")
  await exposed.value!.save()
  expect(getPath(saved[0], "flasks.lifeMaxAreaLevels.small")).toBeNull()
  field.querySelector<HTMLButtonElement>('button[title="Use default"]')!.click()
  await nextTick()
  expect(input.value).toBe("12")
  expect(field.querySelector('button[title="Use default"]')).toBeNull()
  // Retyping the schema default must not leave a redundant global override.
  input.value = "12"
  input.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  await exposed.value!.save()
  expect(getPath(saved[1], "flasks.lifeMaxAreaLevels.small")).toBeUndefined()
  expect(root.textContent).not.toContain("using default")
  for (const key of ["socketRules", "linkRules", "genericEquipment"]) {
    expect(root.querySelector(`[data-editor-subsection="${key}"] > div > button[aria-expanded]`)?.textContent).toContain("Advanced Rules")
  }
  const cutoff = root.querySelector('[data-field-path="jewellery.amuletMaxAreaLevel"]')!
  expect(cutoff.parentElement?.parentElement?.classList.contains("sm:grid-cols-2")).toBe(true)
})

it("exposes every bundled style preset including flask and tincture styles", async () => {
  await mount()
  const panel = [...root.querySelectorAll("section")].find((section) => section.querySelector("h2")?.textContent === "Styles")!
  for (const button of panel.querySelectorAll<HTMLButtonElement>("button[aria-expanded]")) {
    button.click()
  }
  await nextTick()
  for (const key of Object.keys(styles)) {
    expect(panel.querySelector(`span[title="${key}"]`)).not.toBeNull()
  }
})

it("shows inherited highlight rarity text in link style controls and previews", async () => {
  await mount()
  const panel = [...root.querySelectorAll("section")].find((section) => section.querySelector("h2")?.textContent === "Styles")!
  for (const button of panel.querySelectorAll<HTMLButtonElement>("button[aria-expanded]")) button.click()
  await nextTick()
  for (const key of [
    "twoLink",
    "goodTwoLink",
    "selectedTwoLink",
    "threeLink",
    "goodThreeLink",
    "selectedThreeLink",
    "fourLink",
    "goodFourLink",
    "selectedFourLink",
    "fiveLink",
    "sixLink",
  ]) {
    const row = panel.querySelector(`[data-style-key="${key}"]`)!
    // Both responsive layouts retain background/border controls, but no
    // misleading independent text-color picker.
    expect(row.querySelectorAll('input[type="color"]')).toHaveLength(4)
    for (const rarity of ["Normal", "Magic", "Rare"]) {
      const previews = row.querySelectorAll<HTMLElement>(`[data-rarity-preview="${rarity}"]`)
      expect(previews).toHaveLength(2)
      const expected = document.createElement("span")
      expected.style.color = styles[`highlightedEquipment${rarity}` as keyof typeof styles].text!
      for (const preview of previews) expect(preview.style.color).toBe(expected.style.color)
    }
  }
  expect(panel.querySelector('[data-style-key="sixSocket"]')!.querySelectorAll('input[type="color"]')).toHaveLength(6)
})

it("persists global defaults, nested disclosures, Settings and Styles without marking edits", async () => {
  await mount()
  const equipment = () => root.querySelector<HTMLButtonElement>('[data-defaults-group="equipment"] > button')!
  equipment().click()
  ruleButton(root, "flasks").click()
  await nextTick()
  ruleButton(root, "lifeFlasks").click()
  ruleButton(root, "tinctures").click()
  const settings = () => root.querySelector<HTMLButtonElement>("section > button[aria-expanded]")!
  settings().click()
  const stylesPanel = () => [...root.querySelectorAll("section")].find((section) => section.querySelector("h2")?.textContent === "Styles")!
  const styleGroup = () =>
    [...stylesPanel().querySelectorAll<HTMLButtonElement>("button[aria-expanded]")].find(
      (button) => button.querySelector("span")?.textContent === "Flasks & Tinctures",
    )!
  styleGroup().click()
  await nextTick()
  expect(exposed.value?.dirty).toBe(false)
  app.unmount()
  root.remove()
  await mount()
  expect(equipment().getAttribute("aria-expanded")).toBe("true")
  expect(ruleButton(root, "lifeFlasks").getAttribute("aria-expanded")).toBe("true")
  expect(ruleButton(root, "tinctures").getAttribute("aria-expanded")).toBe("true")
  expect(styleGroup().getAttribute("aria-expanded")).toBe("true")
  expect(settings().getAttribute("aria-expanded")).toBe("true")
  expect(exposed.value?.dirty).toBe(false)
  const defaultsPanel = equipment().closest("section")!
  const allButton = (title: string) =>
    [...defaultsPanel.querySelectorAll<HTMLButtonElement>("button")].find((button) => button.textContent?.trim() === title)!
  allButton("Collapse all").click()
  allButton("Expand all").click()
  await nextTick()
  expect(ruleButton(root, "lifeFlasks").getAttribute("aria-expanded")).toBe("true")
  expect(ruleButton(root, "tinctures").getAttribute("aria-expanded")).toBe("true")
})

it("saves global tier moves and item settings as defaults, and supports discard", async () => {
  await mount()
  const currency = root.querySelector('[data-defaults-group="currency"]')!
  currency.querySelector<HTMLButtonElement>('[data-currency-item="Chromatic Orb"]')!.click()
  await nextTick()
  const tier = currency.querySelector<HTMLSelectElement>('[aria-label="Currency tier"]')!
  expect(tier.value).toBe("leaguestart")
  const sound = currency.querySelector<HTMLSelectElement>('[data-field-path="currency.items.Chromatic Orb.sound"] select')!
  expect(sound.value).toBe("preset")
  tier.value = "a"
  tier.dispatchEvent(new Event("change", { bubbles: true }))
  const visibility = currency.querySelector<HTMLSelectElement>('[aria-label="Currency visibility"]')!
  visibility.value = "Hide"
  visibility.dispatchEvent(new Event("change", { bubbles: true }))
  sound.value = "none"
  sound.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(exposed.value?.dirty).toBe(true)
  await exposed.value!.save()
  expect(getPath(saved[0], "currency.assignments.Chromatic Orb")).toBe("a")
  expect(getPath(saved[0], "currency.items.Chromatic Orb.visibility")).toBe("Hide")
  expect(getPath(saved[0], "currency.items.Chromatic Orb.sound.enabled")).toBe(false)
  expect(saved[0].buildSpecificOptions).toBeUndefined()
  visibility.value = "Show"
  visibility.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  exposed.value!.discard()
  await nextTick()
  currency.querySelector<HTMLButtonElement>('[data-currency-item="Chromatic Orb"]')!.click()
  await nextTick()
  expect(currency.querySelector<HTMLSelectElement>('[aria-label="Currency visibility"]')!.value).toBe("Hide")
  expect(exposed.value?.dirty).toBe(false)
})

it("keeps hidden league items restorable and saves an explicit empty selection", async () => {
  await mount()
  const charts = root.querySelector('[data-defaults-group="leagueMechanic"] [data-field-path="leagueMechanic.rules.charts.baseTypes"]')!
  const sandy = charts.querySelector<HTMLButtonElement>('[aria-label="Sandy Seabed Chart"]')!
  const forest = charts.querySelector<HTMLButtonElement>('[aria-label="Coral Forest Chart"]')!
  expect(sandy.getAttribute("aria-pressed")).toBe("true")
  expect(forest.getAttribute("aria-pressed")).toBe("false")
  forest.click()
  await nextTick()
  expect(forest.getAttribute("aria-pressed")).toBe("true")
  forest.click()
  await nextTick()
  sandy.click()
  await nextTick()
  expect(charts.querySelectorAll('button[aria-pressed="false"]')).toHaveLength(3)
  await exposed.value!.save()
  expect(getPath(saved[0], "leagueMechanic.rules.charts.baseTypes")).toEqual([])
  charts.querySelector<HTMLButtonElement>('button[title="Use default"]')!.click()
  await nextTick()
  expect(sandy.getAttribute("aria-pressed")).toBe("true")
  expect(forest.getAttribute("aria-pressed")).toBe("false")
  await exposed.value!.save()
  expect(getPath(saved[1], "leagueMechanic.rules.charts.baseTypes")).toBeUndefined()
})

it("omits Gold/scroll visibility controls and preserves matching field widths", async () => {
  await mount()
  for (const title of ["Gold", "Scroll of Wisdom", "Portal Scroll"]) {
    expect(root.querySelector(`[data-defaults-group="currency"] select[aria-label="${title} visibility"]`)).toBeNull()
  }
  for (const path of [
    "currency.scrolls.wisdom.soundCutoffLevel",
    "currency.scrolls.portal.soundCutoffLevel",
    "leagueMechanic.rules.sulphur.minStackSize",
  ]) {
    expect(root.querySelector(`[data-field-path="${path}"]`)?.parentElement?.classList.contains("sm:col-span-2")).toBe(true)
  }
  for (const path of ["currency.gold.minStackSize", "currency.gold.highStackSize"]) {
    expect(root.querySelector(`[data-field-path="${path}"]`)?.parentElement?.classList.contains("sm:col-span-2")).toBe(false)
  }
})
