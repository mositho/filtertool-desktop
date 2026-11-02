// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest"
import { createApp, h, nextTick, ref, type App } from "vue"
import Sidebar from "../../frontend/src/components/Sidebar.vue"

let app: App | undefined
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
})

it("lets the isolated dev workspace open global settings without changing template identities", () => {
  root = document.createElement("div")
  document.body.append(root)
  let openedGlobals = false
  let exported = false
  let deleted: string | undefined
  app = createApp({
    setup: () => () =>
      h(Sidebar, {
        filters: [
          { name: "Caster Template", hasConfigJson: true },
          { name: "Attack Template", hasConfigJson: true },
        ],
        selected: "Caster Template",
        view: "filter",
        exporting: false,
        adding: false,
        renaming: null,
        devMode: true,
        onGlobal: () => {
          openedGlobals = true
        },
        onExportAll: () => {
          exported = true
        },
        onDelete: (name: string) => {
          deleted = name
        },
      }),
  })
  app.mount(root)
  const globals = [...root.querySelectorAll<HTMLButtonElement>("button")].find(
    (button) => button.textContent?.trim() === "Global Settings",
  )!
  globals.click()
  expect(openedGlobals).toBe(true)
  expect(root.querySelector('[title="New filter"]')).not.toBeNull()
  expect(root.querySelector('[title="Import filter from JSON"]')).toBeNull()
  expect(root.querySelector('[title="Rename"]')).toBeNull()
  root.querySelector<HTMLButtonElement>('[title="Delete"]')!.click()
  expect(deleted).toBe("Caster Template")
  expect(root.textContent).toContain("Caster Template")
  expect(root.textContent).toContain("Attack Template")
  const exportAll = root.querySelector<HTMLButtonElement>('[title="Export is disabled in Dev Editor"]')!
  expect(exportAll.disabled).toBe(true)
  expect(exportAll.classList.contains("disabled:bg-neutral-800")).toBe(true)
  exportAll.click()
  expect(exported).toBe(false)
})

it("allows blank test-filter creation from the dev sidebar", async () => {
  root = document.createElement("div")
  document.body.append(root)
  const adding = ref(false)
  let created: string | undefined
  app = createApp({
    setup: () => () =>
      h(Sidebar, {
        filters: [],
        templates: ["Caster Template", "Leveling Template"],
        selected: null,
        view: "filter",
        exporting: false,
        adding: adding.value,
        renaming: null,
        devMode: true,
        onStartAdd: () => {
          adding.value = true
        },
        onCreate: (name: string) => {
          created = name
        },
      }),
  })
  app.mount(root)
  root.querySelector<HTMLButtonElement>('[title="New filter"]')!.click()
  await nextTick()
  expect(root.querySelector<HTMLSelectElement>('[aria-label="Starting template"]')!.value).toBe("")
  expect([...root.querySelectorAll('[aria-label="Starting template"] option')].map((option) => option.textContent)).toEqual([
    "Blank filter",
    "Caster Template",
    "Leveling Template",
  ])
  const input = root.querySelector<HTMLInputElement>('[placeholder="Filter name"]')!
  input.value = "Test Filter"
  input.dispatchEvent(new Event("input", { bubbles: true }))
  input.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }))
  await nextTick()
  expect(created).toBe("Test Filter")
  expect(root.querySelector('[title="Import filter from JSON"]')).toBeNull()
})
