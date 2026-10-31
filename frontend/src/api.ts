import { Call, Events } from "@wailsio/runtime"

export type FilterInfo = { name: string; hasConfigJson: boolean }
export type ReferenceData = {
  itemClasses: string[]
  baseTypesByClass: Record<string, string[]>
  skillGems: string[]
  supportGems: string[]
  rarities: string[]
  linkColors: string[]
  armourTypes: string[]
  weaponClasses: string[]
  amulets: string[]
  colors: string[]
  shapes: string[]
  operators: string[]
  shieldProgressionModes: string[]
  styleNames: string[]
  styles: Record<string, Record<string, unknown>>
  ttsLocales: string[]
  ttsSpeed: { min: number; max: number }
  sounds: { id: string; name: string; hasFile?: boolean }[]
}
export type AppSettings = {
  filterPath?: string
  soundsFolder?: string
  welcomeComplete?: boolean
  tts: { locale: string; speed: number }
}
export type Highlight = Record<string, unknown>
export type FilterConfig = {
  buildProfile: Record<string, unknown>
  buildSpecificOptions: Record<string, unknown>
  localDefaults?: Record<string, unknown>
  localStyles?: Record<string, unknown>
}
export type ImportResult = { imported: string[]; errors: { name: string; error: string }[] }
type ExportResult = { exported: number; total: number; errors: { name: string; error: string }[] }

async function call<T>(action: string, input: Record<string, unknown> = {}): Promise<T> {
  // Detach Vue proxies and omit undefined values at the native boundary.
  return (await Call.ByName("main.App.Dispatch", action, JSON.parse(JSON.stringify(input)))) as T
}
export const api = {
  listFilters: () => call<{ filters: FilterInfo[]; devMode: boolean }>("listFilters"),
  getConfig: (name: string) => call<FilterConfig>("getConfig", { name }),
  saveConfig: (name: string, config: FilterConfig) => call<FilterConfig>("saveConfig", { name, config }),
  createFilter: (name: string, template?: string) => call<{ name: string }>("createFilter", { name, template }),
  duplicateFilter: (name: string, newName: string) => call<{ name: string }>("duplicateFilter", { name, newName }),
  renameFilter: (name: string, newName: string) => call<{ name: string }>("renameFilter", { name, newName }),
  deleteFilter: (name: string, deleteGameFile: boolean) => call<{ ok: boolean }>("deleteFilter", { name, deleteGameFile }),
  reference: (name?: string) => call<ReferenceData>("reference", { name }),
  filterDefaults: (name: string) => call<{ defaults: Record<string, unknown> }>("filterDefaults", { name }),
  preview: (name: string, config?: FilterConfig) => call<{ output: string }>("preview", { name, config }),
  exportFilter: (name: string, config?: FilterConfig) =>
    call<{ fileName: string; filterPath: string; soundFolder: string }>("exportFilter", { name, config }),
  exportAllFilters: (onProgress?: (done: number, total: number, name: string) => void) => {
    const off = onProgress ? Events.On("export-progress", ({ data }) => onProgress(data.done, data.total, data.name)) : undefined
    return call<ExportResult>("exportAllFilters").finally(() => off?.())
  },
  getSettings: () => call<AppSettings>("getSettings"),
  saveSettings: (settings: AppSettings) => call<AppSettings>("saveSettings", { settings }),
  getDefaults: () =>
    call<{ defaults: Record<string, unknown>; baseDefaults: Record<string, unknown>; userDefaults: Record<string, unknown> }>(
      "getDefaults",
    ),
  saveDefaults: (defaults: Record<string, unknown>) => call<Record<string, unknown>>("saveDefaults", { defaults }),
  getStyles: () =>
    call<{ styles: Record<string, unknown>; baseStyles: Record<string, unknown>; userStyles: Record<string, unknown> }>("getStyles"),
  saveStyles: (styles: Record<string, unknown>) => call<Record<string, unknown>>("saveStyles", { styles }),
  sounds: () => call<{ sounds: { id: string; name: string; hasFile: boolean }[] }>("sounds"),
  importSound: () => call<string>("importSound"),
  soundUrl: (id: string, game = false) => call<string>("sound", { id, game }),
  generateSounds: async (onProgress: (done: number, total: number, text: string) => void) => {
    const off = Events.On("sound-progress", ({ data }) => onProgress(data.done, data.total, data.text))
    try {
      return await call<{ generated: number }>("generateSounds")
    } finally {
      off()
    }
  },
  syncSounds: () => call<{ removed: number }>("syncSounds"),
  chooseFilterPath: (path = "") => call<string>("chooseFilterPath", { path }),
  importJSON: (name: string, value: Record<string, unknown>) => call<{ name: string }>("importJSON", { name, value }),
  importFiles: () => call<ImportResult>("importFiles"),
  importFolder: () => call<ImportResult>("importFolder"),
  share: (name: string, config: FilterConfig, includeGlobals: boolean) =>
    call<{ text: string; characters: number }>("share", { name, config, includeGlobals }),
  saveShareFile: (name: string, config: FilterConfig, includeGlobals: boolean) =>
    call<string>("saveShareFile", { name, config, includeGlobals }),
  clipboardRead: () => call<string>("clipboardRead"),
  clipboardWrite: (text: string) => call<{ ok: boolean }>("clipboardWrite", { text }),
  setDirty: (dirty: boolean) => call<void>("setDirty", { dirty }),
  close: () => call<void>("close"),
  onCloseRequested: (handler: () => void) => Events.On("close-requested", handler),
}
