import { afterEach, expect, it, vi } from "vitest"
import { createApp, h, nextTick, ref, type App } from "vue"
import ColorField from "../../frontend/src/components/ColorField.vue"
import { recentColors } from "../../frontend/src/recentColors"

let app: App
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
  recentColors.value = []
  localStorage.clear()
  vi.useRealTimers()
})

async function mount(initial = "#ff0000") {
  const value = ref(initial)
  const updates = vi.fn((color: string) => (value.value = color))
  root = document.createElement("div")
  document.body.append(root)
  app = createApp({
    setup: () => () => h(ColorField, { "modelValue": value.value, "label": "Text color", "onUpdate:modelValue": updates }),
  })
  app.mount(root)
  root.querySelector<HTMLButtonElement>("button")!.click()
  await nextTick()
  const hex = root.querySelector<HTMLInputElement>('[aria-label="Text color hex"]')!
  const edit = async (color: string) => {
    hex.value = color
    hex.dispatchEvent(new Event("input", { bubbles: true }))
    await nextTick()
  }
  return { value, updates, hex, edit }
}

it("opens an in-app picker on click, focuses Hex, and accepts valid colors without clicking away", async () => {
  const { value, hex, edit } = await mount()
  expect(root.querySelector('[role="dialog"]')?.getAttribute("aria-label")).toBe("Choose Text color")
  expect(document.activeElement).toBe(hex)
  expect(value.value).toBe("#ff0000")
  await edit("1122FF")
  expect(value.value).toBe("#1122ff")
  expect(document.activeElement).toBe(hex)
  await edit("#12")
  expect(value.value).toBe("#1122ff")
  expect(hex.getAttribute("aria-invalid")).toBe("true")
  hex.dispatchEvent(new FocusEvent("blur"))
  await nextTick()
  expect(hex.value).toBe("#1122ff")
})

it("continues updating during sustained input at most every 100 ms and flushes the final value on completion", async () => {
  vi.useFakeTimers()
  const { value, updates, edit, hex } = await mount()
  await edit("#111111")
  expect(value.value).toBe("#111111")
  for (let i = 2; i <= 10; i++) {
    await vi.advanceTimersByTimeAsync(20)
    await edit(`#${String(i).padStart(6, "0")}`)
  }
  expect(updates).toHaveBeenCalledTimes(2)
  expect(value.value).toBe("#000005")
  hex.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(value.value).toBe("#000010")
  await vi.runAllTimersAsync()
  expect(updates).toHaveBeenCalledTimes(3)
})

it("uses keyboard-accessible HSV sliders without losing the selected hue for black", async () => {
  const { value } = await mount("#000000")
  const slide = async (name: string, amount: number) => {
    const input = root.querySelector<HTMLInputElement>(`[aria-label="Text color ${name}"]`)!
    input.value = String(amount)
    input.dispatchEvent(new Event("input", { bubbles: true }))
    input.dispatchEvent(new Event("change", { bubbles: true }))
    await nextTick()
  }
  await slide("hue", 120)
  await slide("saturation", 100)
  await slide("brightness", 100)
  expect(value.value).toBe("#00ff00")
  await slide("hue", 240)
  expect(value.value).toBe("#0000ff")
})

it("selects colors by dragging the saturation/brightness square and finishes on pointer release", async () => {
  const { value } = await mount()
  const square = root.querySelector<HTMLElement>('[aria-hidden="true"].cursor-crosshair')!
  square.setPointerCapture = vi.fn()
  square.hasPointerCapture = vi.fn(() => true)
  vi.spyOn(square, "getBoundingClientRect").mockReturnValue({ left: 0, top: 0, width: 100, height: 100 } as DOMRect)
  const point = (type: string, x: number, y: number) => {
    square.dispatchEvent(new MouseEvent(type, { clientX: x, clientY: y, button: 0, bubbles: true }))
  }
  point("pointerdown", 100, 0)
  point("pointermove", 50, 50)
  point("pointerup", 50, 50)
  await nextTick()
  expect(value.value).toBe("#804040")
})

it("cancels delayed edits when the parent restores a color, without reapplying them after undo/reset", async () => {
  vi.useFakeTimers()
  const { value, updates, edit } = await mount()
  await edit("#111111")
  await edit("#222222")
  value.value = "#abcdef"
  await nextTick()
  await vi.runAllTimersAsync()
  expect(value.value).toBe("#abcdef")
  expect(updates).toHaveBeenCalledTimes(1)
  expect(root.querySelector<HTMLInputElement>('[aria-label="Text color hex"]')!.value).toBe("#abcdef")
})

it.each(["Escape", "outside"])("flushes the last edit on %s dismissal and restores focus only for Escape", async (action) => {
  vi.useFakeTimers()
  const { value, hex, edit, updates } = await mount()
  await edit("#111111")
  await edit("#222222")
  await vi.advanceTimersByTimeAsync(1)
  if (action === "Escape") hex.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }))
  else document.body.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }))
  await nextTick()
  expect(value.value).toBe("#222222")
  expect(root.querySelector('[role="dialog"]')).toBeNull()
  if (action === "Escape") expect(document.activeElement).toBe(root.querySelector("button"))
  await vi.runAllTimersAsync()
  expect(updates).toHaveBeenCalledTimes(2)
})

it("shares final recent colors with other fields and applies a swatch immediately without recording drag intermediates", async () => {
  const first = await mount()
  await first.edit("#111111")
  await first.edit("#223344")
  first.hex.dispatchEvent(new Event("change", { bubbles: true }))
  expect(recentColors.value).toEqual([])
  root.querySelector<HTMLButtonElement>('[aria-label="Close color picker"]')!.click()
  await nextTick()
  expect(recentColors.value).toEqual(["#223344"])
  expect(JSON.parse(localStorage.getItem("filtertool:recent-colors")!)).toEqual(["#223344"])
  app.unmount()
  root.remove()

  const second = await mount("#abcdef")
  root.querySelector<HTMLButtonElement>('[aria-label="Use #223344"]')!.click()
  await nextTick()
  expect(second.value.value).toBe("#223344")
  expect(second.updates).toHaveBeenCalledOnce()
  root.querySelector<HTMLButtonElement>('[aria-label="Close color picker"]')!.click()
  expect(recentColors.value).toEqual(["#223344"])
})
