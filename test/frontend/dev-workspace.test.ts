// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest"
import { createApp, h, type App } from "vue"
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
      }),
  })
  app.mount(root)
  const globals = [...root.querySelectorAll<HTMLButtonElement>("button")].find(
    (button) => button.textContent?.trim() === "Global Settings",
  )!
  globals.click()
  expect(openedGlobals).toBe(true)
  expect(root.querySelector('[title="New filter"]')).toBeNull()
  expect(root.querySelector('[title="Import filter from JSON"]')).toBeNull()
  expect(root.querySelector('[title="Rename"]')).toBeNull()
  expect(root.textContent).toContain("Caster Template")
  expect(root.textContent).toContain("Attack Template")
  const exportAll = root.querySelector<HTMLButtonElement>('[title="Export is disabled in Dev Editor"]')!
  expect(exportAll.disabled).toBe(true)
  expect(exportAll.classList.contains("disabled:bg-neutral-800")).toBe(true)
  exportAll.click()
  expect(exported).toBe(false)
})
