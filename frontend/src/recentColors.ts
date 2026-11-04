import { ref } from "vue"

const STORAGE_KEY = "filtertool:recent-colors"
const LIMIT = 12

function load(): string[] {
  try {
    const saved: unknown = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "[]")
    if (!Array.isArray(saved)) return []
    return [
      ...new Set(
        saved
          .filter((value): value is string => typeof value === "string" && /^#[0-9a-f]{6}$/i.test(value))
          .map((value) => value.toLowerCase()),
      ),
    ].slice(0, LIMIT)
  } catch {
    return []
  }
}

// UI history only: shared across pickers, never part of a filter or its undo.
export const recentColors = ref(load())

export function rememberColor(color: string) {
  if (!/^#[0-9a-f]{6}$/i.test(color)) return
  color = color.toLowerCase()
  recentColors.value = [color, ...recentColors.value.filter((value) => value !== color)].slice(0, LIMIT)
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(recentColors.value))
  } catch {
    /* Color editing must still work when optional UI storage is unavailable. */
  }
}
