// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest"
import { createApp, h, nextTick, reactive, type App } from "vue"
import SoundSelector from "../../frontend/src/components/SoundSelector.vue"

let app: App
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
})
it("only offers available templates, preserves a saved missing template and refreshes availability", async () => {
  const state = reactive({
    model: { tts: "divine_orb" },
    sounds: [
      { id: "divine_orb", name: "Divine Orb", hasFile: false },
      { id: "chaos_orb", name: "Chaos Orb", hasFile: true },
      { id: "old", name: "Old template", hasFile: false },
    ],
  })
  const updates: unknown[] = []
  root = document.createElement("div")
  document.body.append(root)
  app = createApp({
    setup: () => () =>
      h(SoundSelector, { "modelValue": state.model, "sounds": state.sounds, "onUpdate:modelValue": (value) => updates.push(value) }),
  })
  app.mount(root)
  const select = root.querySelectorAll("select")[1]
  expect(select.value).toBe("divine_orb")
  expect(select.querySelector<HTMLOptionElement>('option[value="divine_orb"]')!.disabled).toBe(true)
  expect(select.textContent).toContain("Divine Orb (unavailable)")
  expect(select.querySelector('option[value="old"]')).toBeNull()
  expect(select.querySelector<HTMLOptionElement>('option[value="chaos_orb"]')!.disabled).toBe(false)
  expect(updates).toEqual([])
  state.sounds[0].hasFile = true
  await nextTick()
  expect(select.querySelector<HTMLOptionElement>('option[value="divine_orb"]')!.disabled).toBe(false)
  expect(select.value).toBe("divine_orb")
  expect(updates).toEqual([])
})
