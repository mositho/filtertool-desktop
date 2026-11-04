// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest"
import { createApp, h, nextTick, reactive, type App } from "vue"
import ItemTargetPicker from "../../frontend/src/components/ItemTargetPicker.vue"
import type { ReferenceData } from "../../frontend/src/api"
import referenceData from "../../internal/data/reference.json"

let app: App
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
})
const reference = {
  itemClasses: ["Wands", "Bows"],
  weaponClasses: ["Wands", "Bows"],
  baseTypesByClass: {
    Wands: ["Sage Wand", "Spiraled Wand", "Goat's Horn", "Unknown Wand", "Driftwood Wand", "Assembler Wand", "Driftwood Wand"],
    Bows: ["Short Bow"],
  },
  dropLevelsByBaseType: referenceData.dropLevelsByBaseType,
} as unknown as ReferenceData

function mount(bases: string[] = [], weaponOnly = false, initialClasses: string[] = [], data = reference) {
  const state = reactive({ bases: [...bases], classes: [...initialClasses] })
  root = document.createElement("div")
  document.body.append(root)
  app = createApp({
    setup: () => () =>
      h(ItemTargetPicker, {
        "reference": data,
        weaponOnly,
        "bases": state.bases,
        "classes": state.classes,
        "onUpdate:bases": (value) => {
          state.bases = value
        },
        "onUpdate:classes": (value) => {
          state.classes = value
        },
      }),
  })
  app.mount(root)
  return state
}
const options = () => [...root.querySelectorAll<HTMLLabelElement>("details:nth-of-type(2) label")]
const optionNames = () => options().map((label) => label.querySelector("span")!.textContent)
async function openPicker(index: number) {
  const details = root.querySelectorAll("details")[index]
  details.open = true
  details.dispatchEvent(new Event("toggle"))
  await nextTick()
  return details
}
async function filterClass(name: string) {
  await openPicker(1)
  const select = root.querySelector<HTMLSelectElement>('[aria-label="Filter bases by class"]')!
  select.value = name
  select.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
}

it("sorts options and selected chips by actual minimum drop level without rewriting saved order", async () => {
  const saved = ["Sage Wand", "Unknown Wand", "Driftwood Wand", "Assembler Wand", "Goat's Horn"]
  const state = mount(saved)
  await filterClass("Wands")
  expect(optionNames()).toEqual(["Driftwood Wand", "Goat's Horn", "Spiraled Wand", "Assembler Wand", "Sage Wand", "Unknown Wand"])
  expect(options().map((label) => label.lastElementChild!.textContent)).toEqual(["Lvl 1", "Lvl 6", "Lvl 24", "Lvl 30", "Lvl 30", "Lvl —"])
  const chips = [...root.querySelectorAll<HTMLButtonElement>('[aria-label^="Remove base type "]')]
  expect(chips.map((chip) => chip.getAttribute("aria-label")!.replace("Remove base type ", ""))).toEqual([
    "Driftwood Wand",
    "Goat's Horn",
    "Assembler Wand",
    "Sage Wand",
    "Unknown Wand",
  ])
  expect(chips.map((chip) => chip.querySelector("span")!.textContent)).toEqual(["Lvl 1", "Lvl 6", "Lvl 30", "Lvl 30", "Lvl —"])
  expect(state.bases).toEqual(saved)
  expect(root.querySelector('[title*="level"]')).toBeNull()
})

it.each([false, true])("selects every visible base without altering class selections, weaponOnly=%s", async (weaponOnly) => {
  const state = mount(["Short Bow"], weaponOnly, ["Bows"])
  await filterClass("Wands")
  const details = root.querySelector<HTMLDetailsElement>("details:nth-of-type(2)")!
  details.open = true
  const button = [...details.querySelectorAll<HTMLButtonElement>("button")].find((entry) => entry.textContent?.trim() === "Select results")!
  expect(button.textContent?.trim()).toBe("Select results")
  button.click()
  await nextTick()
  expect(state.bases).toEqual([])
  expect(state.classes).toEqual(["Bows", "Wands"])
  expect(button.disabled).toBe(true)
  expect(details.open).toBe(true)
  await openPicker(0)
  const wandClass = [...root.querySelectorAll<HTMLLabelElement>("details:nth-of-type(1) label")].find((label) =>
    label.textContent?.includes("Wands"),
  )!
  wandClass.querySelector<HTMLInputElement>("input")!.click()
  await nextTick()
  expect(state.classes).toEqual(["Bows"])
})

it("adds only search results while preserving other selections", async () => {
  const state = mount(["Goat's Horn", "Short Bow"])
  await openPicker(1)
  const search = root.querySelector<HTMLInputElement>('[aria-label="Search base types"]')!
  search.value = "wand"
  search.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  const details = root.querySelector<HTMLDetailsElement>("details:nth-of-type(2)")!
  details.open = true
  const button = [...details.querySelectorAll<HTMLButtonElement>("button")].find((entry) => entry.textContent?.trim() === "Select results")!
  expect(button.textContent).toBe("Select results")
  button.click()
  await nextTick()
  expect(state.classes).toContain("Wands")
  expect(state.classes).toContain("Bows")
  expect(state.bases).toEqual([])
  search.value = "no matching base"
  search.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  expect(button.disabled).toBe(true)
})

it.each([false, true])("selects, expands and collapses a whole class, weaponOnly=%s", async (weaponOnly) => {
  const state = mount([], weaponOnly)
  const classDetails = await openPicker(0)
  const wandCheckbox = [...classDetails.querySelectorAll<HTMLInputElement>('input[type="checkbox"]')].find((input) =>
    input.parentElement?.textContent?.includes("Wands"),
  )!
  wandCheckbox.click()
  await nextTick()
  expect(state.classes).toEqual(["Wands"])
  await openPicker(1)
  expect(
    options()
      .filter((label) => !label.textContent?.includes("Short Bow"))
      .every((label) => label.querySelector("input")!.checked),
  ).toBe(true)
  expect(
    options()
      .find((label) => label.textContent?.includes("Short Bow"))!
      .querySelector("input")!.checked,
  ).toBe(false)
  expect(root.querySelectorAll('[aria-label^="Remove base type "]')).toHaveLength(0)
  options()
    .find((label) => label.textContent?.includes("Goat's Horn"))!
    .querySelector("input")!
    .click()
  await nextTick()
  expect(state.classes).toEqual([])
  expect(state.bases).toHaveLength(5)
  expect(state.bases).not.toContain("Goat's Horn")
  expect(
    options()
      .filter((label) => !label.textContent?.includes("Short Bow"))
      .every((label) => label.querySelector("input")!.checked === !label.textContent?.includes("Goat's Horn")),
  ).toBe(true)
  expect(root.querySelectorAll('[aria-label^="Remove base type "]')).toHaveLength(5)
  options()
    .find((label) => label.textContent?.includes("Goat's Horn"))!
    .querySelector("input")!
    .click()
  await nextTick()
  expect(state.classes).toEqual(["Wands"])
  expect(state.bases).toEqual([])
  expect(root.querySelectorAll('[aria-label^="Remove base type "]')).toHaveLength(0)
})

it.each([false, true])("shows partial selection as mixed and selecting it replaces bases, weaponOnly=%s", async (weaponOnly) => {
  const state = mount(["Goat's Horn", "Driftwood Wand"], weaponOnly)
  const classDetails = await openPicker(0)
  const wandCheckbox = [...classDetails.querySelectorAll<HTMLInputElement>('input[type="checkbox"]')].find((input) =>
    input.parentElement?.textContent?.includes("Wands"),
  )!
  expect(wandCheckbox.indeterminate).toBe(true)
  wandCheckbox.click()
  await nextTick()
  expect(state.classes).toEqual(["Wands"])
  expect(state.bases).toEqual([])
  expect(wandCheckbox.checked).toBe(true)
  expect(wandCheckbox.indeterminate).toBe(false)
})

it.each([false, true])("deselects search results from a class without removing other bases, weaponOnly=%s", async (weaponOnly) => {
  const state = mount([], weaponOnly, ["Wands", "Bows"])
  await openPicker(1)
  const search = root.querySelector<HTMLInputElement>('[aria-label="Search base types"]')!
  search.value = "Sage"
  search.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  const button = [...root.querySelectorAll<HTMLButtonElement>("details:nth-of-type(2) button")].find(
    (entry) => entry.textContent?.trim() === "Deselect results",
  )!
  button.click()
  await nextTick()
  expect(state.classes).toEqual(["Bows"])
  expect(state.bases).toHaveLength(5)
  expect(state.bases).not.toContain("Sage Wand")
  expect(state.bases).toContain("Goat's Horn")
})

it("defers full reference lists until opened and preserves search and selections across reopening", async () => {
  const state = mount(["Goat's Horn"], false, [], referenceData as unknown as ReferenceData)
  expect(root.querySelector('[aria-label="Remove base type Goat\'s Horn"]')).not.toBeNull()
  expect(root.querySelectorAll('input[type="checkbox"]')).toHaveLength(0)
  expect(root.querySelector('[aria-label="Search base types"]')).toBeNull()

  await openPicker(0)
  expect(root.querySelectorAll('input[type="checkbox"]').length).toBe(referenceData.itemClasses.length)
  expect(root.querySelector('[aria-label="Search base types"]')).toBeNull()

  await filterClass("Wands")
  const search = root.querySelector<HTMLInputElement>('[aria-label="Search base types"]')!
  search.value = "Goat's Horn"
  search.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  expect(optionNames()).toEqual(["Goat's Horn"])
  expect(options()[0].querySelector("input")!.checked).toBe(true)

  const details = root.querySelectorAll("details")[1]
  details.open = false
  details.dispatchEvent(new Event("toggle"))
  await nextTick()
  expect(options()).toHaveLength(0)
  await openPicker(1)
  expect(root.querySelector<HTMLInputElement>('[aria-label="Search base types"]')!.value).toBe("Goat's Horn")
  expect(root.querySelector<HTMLSelectElement>('[aria-label="Filter bases by class"]')!.value).toBe("Wands")
  expect(options()[0].querySelector("input")!.checked).toBe(true)
  expect(state.bases).toEqual(["Goat's Horn"])
})
