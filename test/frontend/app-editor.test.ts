// @vitest-environment jsdom
import { afterEach, expect, it, vi } from "vitest"
import { createApp, nextTick, type App as VueApp } from "vue"
import App from "../../frontend/src/App.vue"
import reference from "../../internal/data/reference.json"
import defaults from "../../internal/data/defaults.json"
import styles from "../../internal/data/styles.json"
import templates from "../../internal/data/templates.json"

const preview = vi.hoisted(() => vi.fn())
vi.mock("../../frontend/src/components/GlobalSettings.vue", () => ({ default: { template: "<div />" } }))
vi.mock("../../frontend/src/components/AppUpdates.vue", () => ({ default: { template: "<div />" } }))
vi.mock("../../frontend/src/api", () => ({
  api: {
    preview,
    reference: async () => ({ ...reference, styles, sounds: [], styleNames: Object.keys(styles) }),
    getDefaults: async () => ({ defaults }),
    filterDefaults: async () => ({ defaults }),
    getSettings: async () => ({ welcomeComplete: true, filterPath: "", soundsFolder: "", tts: { locale: "en-US", speed: 1.6 } }),
    audioStatus: async () => ({ ffmpegAvailable: true }),
    listFilters: async () => ({
      filters: [{ name: "Caster Template", hasConfigJson: true }],
      devMode: false,
      templates: ["Caster Template"],
    }),
    getConfig: async () => structuredClone(templates["Caster Template"]),
    setDirty: async () => {},
    onCloseRequested: () => () => {},
  },
}))

let app: VueApp
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
  localStorage.clear()
  vi.useRealTimers()
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})
async function settle() {
  for (let i = 0; i < 8; i++) await nextTick()
}

it.each([true, false])("has no output preview or background compilation in desktop=%s", async (desktop) => {
  vi.useFakeTimers()
  vi.stubGlobal("matchMedia", () => ({ matches: desktop, addEventListener() {}, removeEventListener() {} }))
  root = document.createElement("div")
  document.body.append(root)
  app = createApp(App)
  app.mount(root)
  await settle()
  expect(root.querySelector("main img")?.getAttribute("src")).toBe("/favicon.svg")
  root.querySelector<HTMLButtonElement>("nav > button")!.click()
  await settle()
  expect(root.querySelector('[data-editor-group="buildProfile"]')).not.toBeNull()
  expect(root.textContent).not.toContain("Output Preview")
  expect(root.querySelector('[title="Show output preview"]')).toBeNull()
  expect(root.querySelector("main + aside")).toBeNull()
  const input = root.querySelector<HTMLInputElement>('[data-field-path="links.threeLinkMaxAreaLevel"] input')!
  input.value = "35"
  input.dispatchEvent(new Event("input", { bubbles: true }))
  await settle()
  await vi.advanceTimersByTimeAsync(1000)
  expect(preview).not.toHaveBeenCalled()
})
