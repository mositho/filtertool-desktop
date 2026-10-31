// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest"
import { createApp, h, nextTick, ref, type App } from "vue"
import PreviewPane from "../../frontend/src/components/PreviewPane.vue"

let app: App | undefined
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
})

it("toggles from the full-height rail in either state without making output clickable", async () => {
  root = document.createElement("div")
  document.body.append(root)
  const open = ref(false)
  app = createApp({
    setup: () => () =>
      h(PreviewPane, {
        "preview": "Show\n  Rarity Rare",
        "open": open.value,
        "onUpdate:open": (value: boolean) => (open.value = value),
      }),
  })
  app.mount(root)
  const rail = () => root.querySelector<HTMLButtonElement>("button[aria-label]")!
  expect(rail().getAttribute("aria-expanded")).toBe("false")
  expect(rail().textContent).toContain("Output Preview")
  rail().click()
  await nextTick()
  expect(rail().getAttribute("aria-expanded")).toBe("true")
  const output = root.querySelector<HTMLElement>(".font-mono")!
  output.click()
  expect(open.value).toBe(true)
  rail().click()
  await nextTick()
  expect(open.value).toBe(false)
  expect(root.querySelector(".font-mono")).toBeNull()
})
