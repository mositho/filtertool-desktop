import { nextTick } from "vue"

export async function editColor(button: HTMLButtonElement, value: string) {
  button.click()
  await nextTick()
  const picker = document.getElementById(button.getAttribute("aria-controls")!)!
  const hex = picker.querySelector<HTMLInputElement>('input[type="text"]')!
  hex.value = value
  hex.dispatchEvent(new Event("input", { bubbles: true }))
  hex.dispatchEvent(new Event("change", { bubbles: true }))
  picker.querySelector<HTMLButtonElement>('[aria-label="Close color picker"]')!.click()
  await nextTick()
}
