import { afterEach, expect, it, vi } from "vitest"
import { createApp, nextTick, type App } from "vue"
import { api, type UpdateStatus } from "../../frontend/src/api"
import AppUpdates from "../../frontend/src/components/AppUpdates.vue"

vi.mock("../../frontend/src/api", () => ({
  api: { checkUpdates: vi.fn(), downloadUpdate: vi.fn(), restartUpdate: vi.fn(), setDirty: vi.fn() },
}))
vi.mock("@wailsio/runtime", () => ({ Browser: { OpenURL: vi.fn() } }))

let app: App | undefined
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.resetAllMocks()
})

const major: UpdateStatus = {
  enabled: true,
  version: "1.0.0",
  ready: false,
  release: { version: "2.0.0", major: true, notes: "Recreate filters using the new templates." },
}

async function settle() {
  await nextTick()
  await nextTick()
  await nextTick()
}

async function mount(status: UpdateStatus, unsaved = false) {
  vi.mocked(api.checkUpdates).mockResolvedValue(status)
  root = document.createElement("div")
  document.body.append(root)
  app = createApp(AppUpdates, { unsaved, busy: false })
  app.mount(root)
  await settle()
}

function button(label: string) {
  return [...root.querySelectorAll<HTMLButtonElement>("button")].find((b) => b.textContent?.includes(label))!
}

it("checks quietly, warns about major changes, and requires explicit consent before downloading", async () => {
  await mount(major)
  expect(root.querySelector('[role="dialog"]')).toBeNull()
  expect(api.downloadUpdate).not.toHaveBeenCalled()
  button("Version 2.0.0").click()
  await settle()
  expect(root.textContent).toContain("migrated or recreated")
  expect(root.textContent).toContain("%APPDATA%\\Filtertool")
  expect(button("Download update").disabled).toBe(true)
  root.querySelector<HTMLInputElement>('input[type="checkbox"]')!.click()
  await settle()
  expect(button("Download update").disabled).toBe(false)
  vi.mocked(api.downloadUpdate).mockResolvedValue({ ...major, ready: true })
  button("Download update").click()
  await settle()
  expect(api.downloadUpdate).toHaveBeenCalledWith("2.0.0", true)
  expect(api.restartUpdate).not.toHaveBeenCalled()
  expect(button("Restart & Apply")).toBeTruthy()
})

it("blocks restart when there are unsaved edits", async () => {
  await mount({ ...major, ready: true }, true)
  button("Update ready").click()
  await settle()
  expect(button("Restart & Apply").disabled).toBe(true)
  expect(root.textContent).toContain("Save or discard")
  button("Restart & Apply").click()
  expect(api.restartUpdate).not.toHaveBeenCalled()
})

it("does not require the major-update acknowledgement for a minor release", async () => {
  await mount({ ...major, release: { version: "1.1.0", major: false, notes: "Small improvements." } })
  button("Version 1.1.0").click()
  await settle()
  expect(root.querySelector('input[type="checkbox"]')).toBeNull()
  expect(button("Download update").disabled).toBe(false)
})

it("makes the version and update label one full-width clickable block", async () => {
  await mount({ enabled: true, version: "1.0.0", ready: false })
  const control = button("Filtertool 1.0.0")
  expect(control.textContent).toContain("Check for updates")
  expect(control.classList.contains("w-full")).toBe(true)
  expect(control.querySelector("button")).toBeNull()
  control.querySelector<HTMLElement>("span")!.click()
  await settle()
  expect(api.checkUpdates).toHaveBeenCalledTimes(2)
  expect(root.querySelector('[role="dialog"]')).not.toBeNull()
})

it("dismisses only on backdrop clicks or Return, including while a download continues", async () => {
  await mount(major)
  button("Version 2.0.0").click()
  await settle()
  root.querySelector<HTMLElement>('[role="dialog"]')!.click()
  await settle()
  expect(root.querySelector('[role="dialog"]')).not.toBeNull()
  button("Return").click()
  await settle()
  expect(root.querySelector('[role="dialog"]')).toBeNull()
  button("Version 2.0.0").click()
  await settle()
  root.querySelector<HTMLInputElement>('input[type="checkbox"]')!.click()
  await settle()
  let finish!: (status: UpdateStatus) => void
  vi.mocked(api.downloadUpdate).mockReturnValue(
    new Promise((resolve) => {
      finish = resolve
    }),
  )
  button("Download update").click()
  await settle()
  root.querySelector<HTMLElement>('[role="dialog"]')!.parentElement!.click()
  await settle()
  expect(root.querySelector('[role="dialog"]')).toBeNull()
  finish({ ...major, ready: true })
  await settle()
  expect(root.querySelector('[role="dialog"]')).toBeNull()
  expect(button("Update ready")).toBeTruthy()
})
