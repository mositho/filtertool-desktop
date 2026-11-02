import { ref, watch, type Ref } from "vue"

/** Local UI preferences only: never part of a filter, shared defaults, or undo. */
export function useCollapsedState(key: Readonly<Ref<string>>, defaults: () => Record<string, boolean>) {
  const load = () => {
    const fallback = defaults()
    try {
      const saved = JSON.parse(localStorage.getItem(key.value) ?? "{}")
      if (!saved || typeof saved !== "object" || Array.isArray(saved)) return fallback
      return { ...fallback, ...Object.fromEntries(Object.entries(saved).filter(([, value]) => typeof value === "boolean")) } as Record<
        string,
        boolean
      >
    } catch {
      return fallback
    }
  }
  const state = ref(load())
  watch(
    key,
    () => {
      state.value = load()
    },
    { flush: "sync" },
  )
  watch(
    state,
    (value) => {
      try {
        localStorage.setItem(key.value, JSON.stringify(value))
      } catch {
        /* Optional UI state. */
      }
    },
    { deep: true, flush: "sync" },
  )
  return state
}

export function renameFilterViewState(from: string, to: string) {
  if (from === to) return
  for (const prefix of ["filtertool:collapsed:", "filtertool:currency-tiers:"]) {
    try {
      const saved = localStorage.getItem(prefix + from)
      if (saved !== null) localStorage.setItem(prefix + to, saved)
      else localStorage.removeItem(prefix + to)
      localStorage.removeItem(prefix + from)
    } catch {
      /* Renaming a filter must not depend on browser storage. */
    }
  }
}

export function forgetFilterViewState(name: string) {
  for (const prefix of ["filtertool:collapsed:", "filtertool:currency-tiers:"]) {
    try {
      localStorage.removeItem(prefix + name)
    } catch {
      /* Optional UI state. */
    }
  }
}
