import { afterEach, expect, it, vi } from "vitest"
import { createApp, h, nextTick, ref, type App } from "vue"
import NumberField from "../../frontend/src/components/NumberField.vue"

let app: App
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
})

function mount(
  initial: number | undefined,
  options: { min?: number; max?: number; integer?: boolean; decimals?: number; step?: number; fallback?: number } = {},
) {
  const value = ref(initial)
  const updates = vi.fn((next: number | undefined) => (value.value = next))
  root = document.createElement("div")
  document.body.append(root)
  const { fallback, ...props } = options
  app = createApp({
    setup: () => () => h(NumberField, { ...props, "modelValue": value.value ?? fallback, "onUpdate:modelValue": updates }),
  })
  app.mount(root)
  const input = root.querySelector("input")!
  input.focus()
  const type = async (text: string) => {
    input.value = text
    input.dispatchEvent(new Event("input", { bubbles: true }))
    await nextTick()
  }
  return { input, value, updates, type }
}

it("allows replacing a size digit by digit without clamping the unfinished number", async () => {
  const { input, value, updates, type } = mount(35, { min: 20, max: 45, integer: true })
  input.select()
  await type("3")
  expect(input.value).toBe("3")
  expect(value.value).toBe(35)
  expect(updates).not.toHaveBeenCalled()
  await type("32")
  expect(input.value).toBe("32")
  expect(value.value).toBe(32)
  input.blur()
  await nextTick()
  expect(input.value).toBe("32")
  expect(updates).toHaveBeenCalledOnce()
})

it.each([
  { text: "3", action: "blur", expected: 20 },
  { text: "99", action: "Enter", expected: 45 },
])("applies bounds on $action rather than during typing", async ({ text, action, expected }) => {
  const { input, value, type } = mount(35, { min: 20, max: 45, integer: true })
  await type(text)
  expect(input.value).toBe(text)
  expect(value.value).toBe(35)
  if (action === "blur") input.blur()
  else input.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }))
  await nextTick()
  expect(value.value).toBe(expected)
  expect(input.value).toBe(String(expected))
})

it("does not restore an inherited default during temporary clearing, but clears an override when leaving it blank", async () => {
  const { input, value, updates, type } = mount(32, { min: 20, max: 45, integer: true, fallback: 35 })
  await type("")
  expect(input.value).toBe("")
  expect(value.value).toBe(32)
  await type("4")
  expect(input.value).toBe("4")
  await type("40")
  expect(value.value).toBe(40)
  await type("")
  expect(value.value).toBe(40)
  input.blur()
  await nextTick()
  expect(value.value).toBeUndefined()
  expect(input.value).toBe("35")
  expect(updates.mock.calls.map(([value]) => value)).toEqual([40, undefined])
})

it("keeps decimal drafts intact and only rounds/formats on completion", async () => {
  const { input, value, type } = mount(0.94, { min: 0, max: 1, decimals: 2 })
  await type("0")
  await type("0.")
  expect(input.value).toBe("0.")
  await type("0.1")
  expect(value.value).toBe(0.1)
  await type("0.126")
  expect(input.value).toBe("0.126")
  expect(value.value).toBe(0.1)
  input.blur()
  await nextTick()
  expect(input.value).toBe("0.13")
  expect(value.value).toBe(0.13)
})

it("lets external undo/reset replace an unfinished draft without committing it again on blur", async () => {
  const { input, value, updates, type } = mount(35, { min: 20, max: 45, integer: true })
  await type("32")
  await type("4")
  value.value = 35
  await nextTick()
  expect(input.value).toBe("35")
  input.blur()
  await nextTick()
  expect(value.value).toBe(35)
  expect(updates.mock.calls.map(([value]) => value)).toEqual([32])
})

it("does not create overrides merely by focusing and leaving an inherited value", async () => {
  const { input, value, updates } = mount(undefined, { min: 20, max: 45, integer: true, fallback: 35 })
  input.blur()
  await nextTick()
  expect(value.value).toBeUndefined()
  expect(updates).not.toHaveBeenCalled()
})

it("restores an unchanged inherited value after clearing, without interrupting the blank draft", async () => {
  const { input, value, type } = mount(undefined, { min: 20, max: 45, integer: true, fallback: 35 })
  await type("")
  expect(input.value).toBe("")
  input.blur()
  await nextTick()
  expect(value.value).toBeUndefined()
  expect(input.value).toBe("35")
})

it("keeps step and focused-wheel actions immediately bounded while leaving an unfocused wheel alone", async () => {
  const { input, value, type } = mount(35, { min: 20, max: 45, integer: true })
  await type("99")
  root.querySelector<HTMLButtonElement>('button[title="Increase by 1"]')!.click()
  await nextTick()
  expect(value.value).toBe(45)
  expect(input.value).toBe("45")
  input.dispatchEvent(new WheelEvent("wheel", { deltaY: 1, cancelable: true }))
  await nextTick()
  expect(value.value).toBe(44)
  input.blur()
  input.dispatchEvent(new WheelEvent("wheel", { deltaY: 1, cancelable: true }))
  expect(value.value).toBe(44)
})
