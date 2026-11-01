import { afterEach, expect, it } from "vitest"
import { createApp, h, nextTick, ref, type App } from "vue"
import Sidebar from "../../frontend/src/components/Sidebar.vue"

let app: App | undefined
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
})

async function mount() {
  root = document.createElement("div")
  document.body.append(root)
  app = createApp({
    setup() {
      const adding = ref(true)
      return () =>
        h("div", [
          h(Sidebar, {
            filters: [],
            selected: null,
            view: "filter",
            exporting: false,
            adding: adding.value,
            renaming: null,
            onCancelAdd: () => {
              adding.value = false
            },
          }),
          h("button", { id: "outside" }, "Outside"),
        ])
    },
  })
  app.mount(root)
  await nextTick()
  await nextTick()
}

it("keeps creation open for template controls and cancels when clicking outside", async () => {
  await mount()
  root.querySelector("select")!.dispatchEvent(new Event("pointerdown", { bubbles: true }))
  await nextTick()
  expect(root.querySelector('[placeholder="Filter name"]')).not.toBeNull()
  root.querySelector("#outside")!.dispatchEvent(new Event("pointerdown", { bubbles: true }))
  await nextTick()
  expect(root.querySelector('[placeholder="Filter name"]')).toBeNull()
})

it("also cancels when keyboard focus leaves creation", async () => {
  await mount()
  root.querySelector<HTMLButtonElement>("#outside")!.focus()
  await nextTick()
  expect(root.querySelector('[placeholder="Filter name"]')).toBeNull()
})
