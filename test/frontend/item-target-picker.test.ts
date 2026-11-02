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

function mount(bases: string[] = [], weaponOnly = false, initialClasses: string[] = []) {
  const state = reactive({ bases: [...bases], classes: [...initialClasses] })
  root = document.createElement("div")
  document.body.append(root)
  app = createApp({
    setup: () => () =>
      h(ItemTargetPicker, {
        reference,
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
async function filterClass(name: string) {
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
  if (weaponOnly) {
    expect(state.bases).toEqual([])
    expect(state.classes).toEqual(["Bows", "Wands"])
  } else {
    expect(state.bases).toEqual([
      "Short Bow",
      "Driftwood Wand",
      "Goat's Horn",
      "Spiraled Wand",
      "Assembler Wand",
      "Sage Wand",
      "Unknown Wand",
    ])
    expect(state.classes).toEqual(["Bows"])
  }
  expect(button.disabled).toBe(true)
  expect(details.open).toBe(true)
  if (weaponOnly) {
    const wandClass = [...root.querySelectorAll<HTMLLabelElement>("details:nth-of-type(1) label")].find((label) =>
      label.textContent?.includes("Wands"),
    )!
    wandClass.querySelector<HTMLInputElement>("input")!.click()
    await nextTick()
    expect(state.classes).toEqual(["Bows"])
  } else {
    root.querySelector<HTMLButtonElement>('[aria-label="Remove base type Sage Wand"]')!.click()
    await nextTick()
    expect(state.bases).not.toContain("Sage Wand")
    expect(button.disabled).toBe(false)
    expect(
      options()
        .find((label) => label.textContent?.includes("Sage Wand"))!
        .querySelector("input")!.checked,
    ).toBe(false)
  }
})

it("adds only search results while preserving other selections", async () => {
  const state = mount(["Goat's Horn", "Short Bow"])
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
  expect(state.bases).toContain("Goat's Horn")
  expect(state.bases).toContain("Short Bow")
  expect(state.bases).toHaveLength(7)
  search.value = "no matching base"
  search.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  expect(button.disabled).toBe(true)
})

it("selects a whole class and expands it to individual bases when one is deselected", async () => {
  const state = mount([], true)
  const classDetails = root.querySelector<HTMLDetailsElement>("details:nth-of-type(1)")!
  classDetails.open = true
  const wandCheckbox = [...classDetails.querySelectorAll<HTMLInputElement>('input[type="checkbox"]')].find((input) =>
    input.parentElement?.textContent?.includes("Wands"),
  )!
  wandCheckbox.click()
  await nextTick()
  expect(state.classes).toEqual(["Wands"])
  const basesDetails = root.querySelector<HTMLDetailsElement>("details:nth-of-type(2)")!
  basesDetails.open = true
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

it("shows a partially selected class as mixed and selecting it replaces base selections", async () => {
  const state = mount(["Goat's Horn", "Driftwood Wand"], true)
  const classDetails = root.querySelector<HTMLDetailsElement>("details:nth-of-type(1)")!
  classDetails.open = true
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
