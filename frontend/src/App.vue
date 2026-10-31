<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from "vue"
import { api, type AppSettings, type FilterConfig, type FilterInfo, type ReferenceData } from "./api"
import { EMPTY_FILTER_CONFIG } from "@schema/form-schema"
import { filterRangeErrors } from "@schema/range-validation"
import { isValidFilterName } from "@schema/filter-name"
import { pruneEmpty, stableStringify } from "./path"
import Sidebar from "./components/Sidebar.vue"
import ConfigForm from "./components/ConfigForm.vue"
import PreviewPane from "./components/PreviewPane.vue"
import GlobalSettings from "./components/GlobalSettings.vue"

const filters = ref<FilterInfo[]>([])
const templateMode = ref(false)
const selected = ref<string | null>(null)
const config = ref<FilterConfig>(clone(EMPTY_FILTER_CONFIG))
const savedConfig = ref<FilterConfig>(clone(EMPTY_FILTER_CONFIG))
const reference = ref<ReferenceData | null>(null)
const defaults = ref<Record<string, unknown>>({})
const preview = ref("")
const previewOpen = ref(false)
const previewSection = ref<string | null>(null)
const view = ref<"filter" | "global">("filter")
const sidebarOpen = ref(false)
type FlashKind = "notice" | "create" | "duplicate" | "rename" | "delete" | "save" | "export" | "error"

const FLASH_META: Record<FlashKind, { icon: string; tone: string }> = {
  notice: { icon: "ℹ", tone: "border-green-800 bg-green-950/90 text-green-300" },
  create: { icon: "+", tone: "border-green-800 bg-green-950/90 text-green-300" },
  duplicate: { icon: "⧉", tone: "border-green-800 bg-green-950/90 text-green-300" },
  rename: { icon: "✎", tone: "border-green-800 bg-green-950/90 text-green-300" },
  delete: { icon: "✕", tone: "border-amber-800 bg-amber-950/90 text-amber-300" },
  save: { icon: "✓", tone: "border-green-800 bg-green-950/90 text-green-300" },
  export: { icon: "↑", tone: "border-green-800 bg-green-950/90 text-green-300" },
  error: { icon: "⚠", tone: "border-red-800 bg-red-950/90 text-red-300" },
}

const toast = ref<{ message: string; kind: FlashKind } | null>(null)
let toastTimer: ReturnType<typeof setTimeout> | undefined

const exporting = ref(false)
const exportingAll = ref(false)

const settings = ref<AppSettings | null>(null)
const showQuickstart = ref(false)
const quickstartPath = ref("")

const adding = ref(false)
const renaming = ref<string | null>(null)

const showImport = ref(false)
const importName = ref("")
const importJson = ref("")
const importError = ref("")
const showShare = ref(false)
const shareGlobals = ref(true)
const shareText = ref("")
const shareCharacters = ref(0)
const shareError = ref("")
const importBusy = ref(false)
const pendingClose = ref(false)

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value))
}

function flash(message: string, kind: FlashKind = "notice") {
  toast.value = { message, kind }
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toast.value = null
  }, 4000)
}

const dirty = computed(() => stableStringify(pruneEmpty(config.value)) !== stableStringify(pruneEmpty(savedConfig.value)))

const undoStack = ref<string[]>([])
const redoStack = ref<string[]>([])
let lastSnapshot = JSON.stringify(config.value)
let lastRecordTime = 0
const RECORD_COALESCE_MS = 500
const MAX_HISTORY = 200

const canUndo = computed(() => undoStack.value.length > 0)
const canRedo = computed(() => redoStack.value.length > 0)

function snapshot(): string {
  return JSON.stringify(config.value)
}

function resetHistory() {
  undoStack.value = []
  redoStack.value = []
  lastSnapshot = snapshot()
  lastRecordTime = 0
}

function recordChange() {
  const current = snapshot()
  if (current === lastSnapshot) return
  const now = Date.now()
  if (now - lastRecordTime >= RECORD_COALESCE_MS) {
    undoStack.value.push(lastSnapshot)
    if (undoStack.value.length > MAX_HISTORY) undoStack.value.shift()
    redoStack.value = []
  }
  lastSnapshot = current
  lastRecordTime = now
}

watch(config, recordChange, { deep: true })

function restore(snapshotJson: string) {
  lastSnapshot = snapshotJson
  config.value = JSON.parse(snapshotJson)
}

function undo() {
  if (!canUndo.value) return
  redoStack.value.push(snapshot())
  restore(undoStack.value.pop()!)
}

function redo() {
  if (!canRedo.value) return
  undoStack.value.push(snapshot())
  restore(redoStack.value.pop()!)
}

function discardChanges() {
  if (!dirty.value) return
  undoStack.value.push(snapshot())
  redoStack.value = []
  restore(JSON.stringify(savedConfig.value))
}

async function refreshFilters(selectName?: string) {
  const { filters: list, templateMode: templates } = await api.listFilters()
  templateMode.value = templates
  filters.value = list
  if (selectName && list.some((f) => f.name === selectName)) {
    selected.value = selectName
  } else if (!selected.value || !list.some((f) => f.name === selected.value)) {
    selected.value = list[0]?.name ?? null
  }
}

async function loadConfig() {
  if (!selected.value) {
    config.value = clone(EMPTY_FILTER_CONFIG)
    savedConfig.value = clone(EMPTY_FILTER_CONFIG)
    preview.value = ""
    resetHistory()
    return
  }
  const name = selected.value
  const [loaded, refData, defaultsData] = await Promise.all([api.getConfig(name), api.reference(name), api.filterDefaults(name)])
  if (selected.value !== name) return
  config.value = loaded
  reference.value = refData
  defaults.value = defaultsData.defaults
  savedConfig.value = clone(config.value)
  resetHistory()
  void refreshPreview()
}

watch(selected, () => void loadConfig())

async function refreshPreview() {
  if (!selected.value) {
    preview.value = ""
    return
  }
  try {
    const { output } = await api.preview(selected.value, config.value)
    preview.value = output
  } catch {
    /* preview is best-effort */
  }
}

let debounce: ReturnType<typeof setTimeout> | undefined
watch(
  config,
  () => {
    clearTimeout(debounce)
    debounce = setTimeout(() => void refreshPreview(), 400)
  },
  { deep: true },
)

async function saveOnly() {
  if (!selected.value) return
  const [error] = filterRangeErrors(config.value)
  if (error) throw new Error(error)
  await api.saveConfig(selected.value, config.value)
  savedConfig.value = clone(config.value)
}

async function onSave() {
  if (!selected.value) return
  try {
    await saveOnly()
    flash("Saved", "save")
  } catch (e) {
    flash((e as Error).message, "error")
  }
}

const globalRef = ref<InstanceType<typeof GlobalSettings> | null>(null)

const globalDirty = computed(() => globalRef.value?.dirty ?? false)
const globalCanUndo = computed(() => globalRef.value?.canUndo ?? false)
const globalCanRedo = computed(() => globalRef.value?.canRedo ?? false)
const globalSaving = computed(() => globalRef.value?.saving ?? false)
const globalExporting = computed(() => globalRef.value?.exporting ?? false)

async function onGlobalSave() {
  const result = await globalRef.value?.save()
  if (result) flash(result.message, result.kind)
  await refreshDefaults()
}

async function onGlobalExport() {
  const result = await globalRef.value?.saveAndExport()
  if (result) flash(result.message, result.kind)
  await refreshDefaults()
}

async function refreshDefaults() {
  try {
    const defaultsData = selected.value ? await api.filterDefaults(selected.value) : await api.getDefaults()
    defaults.value = (defaultsData.defaults ?? {}) as Record<string, unknown>
    reference.value = await api.reference(selected.value ?? undefined)
    settings.value = await api.getSettings()
  } catch {
    /* best-effort */
  }
}

async function onSaveAndExport() {
  if (!selected.value) return
  exporting.value = true
  try {
    const [error] = filterRangeErrors(config.value)
    if (error) throw new Error(error)
    await api.saveConfig(selected.value, config.value)
    savedConfig.value = clone(config.value)
    const result = await api.exportFilter(selected.value, config.value)
    flash(`Exported ${result.fileName} to ${result.filterPath}`, "export")
  } catch (e) {
    flash((e as Error).message, "error")
  } finally {
    exporting.value = false
  }
}

async function onExportAll() {
  exportingAll.value = true
  try {
    if (selected.value && dirty.value) {
      const [error] = filterRangeErrors(config.value)
      if (error) throw new Error(error)
      await api.saveConfig(selected.value, config.value)
      savedConfig.value = clone(config.value)
    }
    const result = await api.exportAllFilters()
    if (result.total === 0) {
      flash("No filters to export")
    } else if (result.errors.length > 0) {
      flash(`Exported ${result.exported}/${result.total} filters (${result.errors.length} failed)`, "error")
    } else {
      flash(`Exported ${result.exported} filter${result.exported === 1 ? "" : "s"}`, "export")
    }
  } catch (e) {
    flash((e as Error).message, "error")
  } finally {
    exportingAll.value = false
  }
}

function confirmExportAll() {
  showExportAll.value = false
  void onExportAll()
}

const showUnsaved = ref(false)
const showExportAll = ref(false)
const pendingSelect = ref<string | null>(null)
const pendingView = ref<"filter" | "global">("filter")

function tryNavigate(name: string | null, toView: "filter" | "global") {
  pendingClose.value = false
  if (dirty.value || globalDirty.value) {
    pendingSelect.value = name
    pendingView.value = toView
    showUnsaved.value = true
  } else {
    applyNavigation(name, toView)
  }
}

function applyNavigation(name: string | null, toView: "filter" | "global") {
  selected.value = name
  view.value = toView
}

async function confirmUnsavedSave() {
  showUnsaved.value = false
  try {
    if (dirty.value) await saveOnly()
    if (globalDirty.value) {
      const result = await globalRef.value?.save()
      if (result?.kind === "error") throw new Error(result.message)
      await refreshDefaults()
    }
    if (pendingClose.value) {
      await api.close()
      return
    }
    applyNavigation(pendingSelect.value, pendingView.value)
  } catch (e) {
    flash((e as Error).message, "error")
  }
}

function confirmUnsavedDiscard() {
  showUnsaved.value = false
  if (pendingClose.value) {
    void api.close()
    return
  }
  if (globalDirty.value) globalRef.value?.discard()
  applyNavigation(pendingSelect.value, pendingView.value)
}

watch([dirty, globalDirty], ([filterDirty, settingsDirty]) => {
  void api.setDirty(filterDirty || settingsDirty).catch(() => {})
})
const offClose = api.onCloseRequested(() => {
  pendingClose.value = true
  showUnsaved.value = true
})
onUnmounted(offClose)

function onSelect(name: string) {
  sidebarOpen.value = false
  if (name === selected.value && view.value === "filter") return
  tryNavigate(name, "filter")
}

function onGlobal() {
  sidebarOpen.value = false
  tryNavigate(null, "global")
}

function onHome() {
  sidebarOpen.value = false
  tryNavigate(null, "filter")
}

function createFromLanding() {
  sidebarOpen.value = true
  adding.value = true
}

async function confirmQuickstart() {
  const path = quickstartPath.value.trim()
  if (!path) return
  const next: AppSettings = {
    ...settings.value,
    filterPath: path,
    welcomeComplete: true,
    soundsFolder: settings.value?.soundsFolder,
    tts: settings.value?.tts ?? { locale: "en-US", speed: 1.6 },
  }
  try {
    await api.saveSettings(next)
    settings.value = next
    showQuickstart.value = false
    await globalRef.value?.reload()
  } catch (e) {
    flash((e as Error).message, "error")
  }
}

async function skipQuickstart() {
  if (settings.value) {
    settings.value = await api.saveSettings({ ...settings.value, welcomeComplete: true })
  }
  showQuickstart.value = false
}

async function browseQuickstart() {
  try {
    const path = await api.chooseFilterPath(quickstartPath.value)
    if (path) quickstartPath.value = path
  } catch (error) {
    flash((error as Error).message, "error")
  }
}

async function browseHomePath() {
  if (!settings.value) return
  try {
    const path = await api.chooseFilterPath(settings.value.filterPath)
    if (!path) return
    settings.value = await api.saveSettings({ ...settings.value, filterPath: path })
    await globalRef.value?.reload()
  } catch (error) {
    flash((error as Error).message, "error")
  }
}

async function onCreate(name: string, template?: string) {
  try {
    await api.createFilter(name.trim(), template)
    await refreshFilters(name.trim())
    flash(`Created filter "${name}"`, "create")
  } catch (e) {
    flash((e as Error).message, "error")
  }
}

async function updateShare() {
  if (!selected.value) return
  try {
    const result = await api.share(selected.value, config.value, shareGlobals.value)
    shareText.value = result.text
    shareCharacters.value = result.characters
    shareError.value = ""
  } catch (error) {
    shareText.value = ""
    shareError.value = (error as Error).message
  }
}
async function openShare() {
  showShare.value = true
  await updateShare()
}
watch(shareGlobals, () => {
  if (showShare.value) void updateShare()
})
async function copyShare() {
  try {
    await api.clipboardWrite(shareText.value)
    flash("Filter copied", "save")
  } catch (error) {
    shareError.value = (error as Error).message
  }
}
async function saveShareFile() {
  if (!selected.value) return
  try {
    await api.saveShareFile(selected.value, config.value, shareGlobals.value)
  } catch (error) {
    shareError.value = (error as Error).message
  }
}
async function exportTemplateBundle() {
  try {
    if (dirty.value) await saveOnly()
    const path = await api.saveTemplateBundle()
    if (path) flash("Template bundle saved. Review changes before including them in a build.", "save")
  } catch (error) {
    flash((error as Error).message, "error")
  }
}
async function importFromPicker(folder: boolean) {
  importBusy.value = true
  try {
    const result = folder ? await api.importFolder() : await api.importFiles()
    if (result.imported.length) {
      await refreshFilters(result.imported[0])
      showImport.value = false
    }
    if (result.errors.length) {
      importError.value = result.errors.map((error) => `${error.name}: ${error.error}`).join("\n")
      showImport.value = true
    } else if (result.imported.length) flash(`Imported ${result.imported.length} filter(s)`, "create")
  } catch (error) {
    importError.value = (error as Error).message
  } finally {
    importBusy.value = false
  }
}

async function openImport() {
  importName.value = ""
  importJson.value = ""
  importError.value = ""
  showImport.value = true
  try {
    const text = await api.clipboardRead()
    if (text && text.trim()) importJson.value = text
  } catch {
    /* clipboard read may be blocked; leave the textarea empty */
  }
}

async function confirmImport() {
  const name = importName.value.trim()
  if (!isValidFilterName(name)) {
    importError.value = "Only letters, numbers, spaces, underscores and hyphens are allowed."
    return
  }
  let parsed: unknown
  try {
    parsed = JSON.parse(importJson.value)
  } catch {
    importError.value = "That isn't valid JSON."
    return
  }
  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
    importError.value = "Expected a JSON object with buildProfile / buildSpecificOptions."
    return
  }
  try {
    await api.importJSON(name, parsed as Record<string, unknown>)
    showImport.value = false
    await refreshFilters(name)
    flash(`Imported filter "${name}"`, "create")
  } catch (e) {
    importError.value = (e as Error).message
  }
}

async function onDuplicate(name: string) {
  const newName = `${name} copy`
  try {
    await api.duplicateFilter(name, newName)
    await refreshFilters(newName)
    renaming.value = newName
    flash(`Duplicated "${name}"`, "duplicate")
  } catch (e) {
    flash((e as Error).message, "error")
  }
}

async function onRename(name: string, newName: string) {
  try {
    await api.renameFilter(name, newName.trim())
    await refreshFilters(newName.trim())
    flash(`Renamed to "${newName}"`, "rename")
  } catch (e) {
    flash((e as Error).message, "error")
  }
}

const deleteTarget = ref<string | null>(null)

function onDelete(name: string) {
  deleteTarget.value = name
}

async function confirmDelete(deleteGameFile: boolean) {
  const name = deleteTarget.value
  deleteTarget.value = null
  if (!name) return
  try {
    await api.deleteFilter(name, deleteGameFile)
    await refreshFilters()
    flash(`Deleted "${name}"`, "delete")
  } catch (e) {
    flash((e as Error).message, "error")
  }
}

const title = computed(() => (view.value === "global" ? "Global Settings" : (selected.value ?? "Home")))

void (async () => {
  try {
    const [refData, defaultsData, settingsData] = await Promise.all([api.reference(), api.getDefaults(), api.getSettings()])
    reference.value = refData
    defaults.value = (defaultsData.defaults ?? {}) as Record<string, unknown>
    settings.value = settingsData
    if (!settingsData.welcomeComplete) {
      showQuickstart.value = true
      quickstartPath.value = settingsData.filterPath ?? ""
    }
    await refreshFilters()
    selected.value = null
  } catch (e) {
    flash((e as Error).message, "error")
  }
})()

function onBeforeUnload(event: BeforeUnloadEvent) {
  if (!dirty.value && !globalDirty.value) return
  event.preventDefault()
  event.returnValue = ""
}

function onKeyDown(event: KeyboardEvent) {
  const target = event.target as HTMLElement | null
  const tag = target?.tagName?.toLowerCase()
  if (tag === "input" || tag === "textarea" || tag === "select" || target?.isContentEditable) return
  const mod = event.ctrlKey || event.metaKey
  if (!mod || event.altKey) return
  const key = event.key.toLowerCase()

  if (view.value === "global") {
    if (key === "z") {
      event.preventDefault()
      if (event.shiftKey) globalRef.value?.redo()
      else globalRef.value?.undo()
    } else if (key === "y") {
      event.preventDefault()
      globalRef.value?.redo()
    } else if (key === "s") {
      event.preventDefault()
      if (event.shiftKey) void onGlobalExport()
      else void onGlobalSave()
    }
    return
  }

  if (!selected.value) return
  if (key === "z") {
    event.preventDefault()
    if (event.shiftKey) redo()
    else undo()
  } else if (key === "y") {
    event.preventDefault()
    redo()
  } else if (key === "s") {
    event.preventDefault()
    if (event.shiftKey) void onSaveAndExport()
    else void onSave()
  }
}

window.addEventListener("beforeunload", onBeforeUnload)
window.addEventListener("keydown", onKeyDown)
onUnmounted(() => {
  window.removeEventListener("beforeunload", onBeforeUnload)
  window.removeEventListener("keydown", onKeyDown)
  clearTimeout(toastTimer)
  clearTimeout(debounce)
})
</script>

<template>
  <div class="flex h-full">
    <div v-if="sidebarOpen" class="fixed inset-0 z-40 md:hidden">
      <div class="absolute inset-0 bg-black/60" @click="sidebarOpen = false"></div>
      <div class="absolute inset-y-0 left-0 flex w-64">
        <Sidebar
          :template-mode="templateMode"
          :filters="filters"
          :selected="selected"
          :view="view"
          :exporting="exportingAll"
          :adding="adding"
          :renaming="renaming"
          @select="onSelect"
          @export-all="showExportAll = true"
          @create="onCreate"
          @start-add="adding = true"
          @cancel-add="adding = false"
          @start-rename="renaming = $event"
          @cancel-rename="renaming = null"
          @duplicate="onDuplicate"
          @rename="onRename"
          @delete="onDelete"
          @global="onGlobal"
          @home="onHome"
          @import="openImport"
        />
      </div>
    </div>

    <div class="hidden shrink-0 md:flex">
      <Sidebar
        :template-mode="templateMode"
        :filters="filters"
        :selected="selected"
        :view="view"
        :exporting="exportingAll"
        :adding="adding"
        :renaming="renaming"
        @select="onSelect"
        @export-all="showExportAll = true"
        @create="onCreate"
        @start-add="adding = true"
        @cancel-add="adding = false"
        @start-rename="renaming = $event"
        @cancel-rename="renaming = null"
        @duplicate="onDuplicate"
        @rename="onRename"
        @delete="onDelete"
        @global="onGlobal"
        @home="onHome"
        @import="openImport"
      />
    </div>

    <main class="flex-1 flex flex-col min-w-0">
      <div
        v-if="templateMode"
        class="flex items-center justify-between gap-3 border-b border-amber-800 bg-amber-950/40 px-4 py-2 text-xs text-amber-200"
      >
        <span>Template Editor — isolated from your personal filters. Save edits, then export to update the bundled templates.</span>
        <button
          type="button"
          @click="exportTemplateBundle"
          class="shrink-0 rounded border border-amber-700 px-3 py-1 hover:bg-amber-900/40"
        >
          Export template bundle…
        </button>
      </div>
      <header class="flex h-14 items-center justify-between gap-2 border-b border-neutral-800 px-3 sm:px-6">
        <div class="flex min-w-0 items-center gap-2">
          <button
            type="button"
            @click="sidebarOpen = true"
            class="flex h-8 w-8 shrink-0 items-center justify-center rounded border border-neutral-700 text-neutral-300 md:hidden"
            title="Filters"
          >
            ☰
          </button>
          <span
            v-if="(view === 'filter' && dirty) || (view === 'global' && globalDirty)"
            class="h-2 w-2 shrink-0 rounded-full bg-amber-400"
            title="Unsaved changes"
          ></span>
          <h1 class="truncate text-lg font-semibold">{{ title }}</h1>
        </div>
        <div class="flex shrink-0 items-center gap-2">
          <template v-if="view === 'filter' && selected">
            <button
              type="button"
              @click="undo"
              :disabled="!canUndo"
              class="flex h-8 w-8 items-center justify-center rounded border border-neutral-700 text-neutral-300 hover:bg-neutral-700 disabled:cursor-not-allowed disabled:opacity-30"
              title="Undo (Ctrl+Z)"
            >
              <svg
                class="h-4 w-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <polyline points="1 4 1 10 7 10"></polyline>
                <path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"></path>
              </svg>
            </button>
            <button
              type="button"
              @click="redo"
              :disabled="!canRedo"
              class="flex h-8 w-8 items-center justify-center rounded border border-neutral-700 text-neutral-300 hover:bg-neutral-700 disabled:cursor-not-allowed disabled:opacity-30"
              title="Redo (Ctrl+Shift+Z)"
            >
              <svg
                class="h-4 w-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <polyline points="23 4 23 10 17 10"></polyline>
                <path d="M20.49 15a9 9 0 1 1-2.13-9.36L23 10"></path>
              </svg>
            </button>
            <button
              @click="discardChanges"
              :disabled="!dirty"
              class="rounded px-3 py-1.5 text-sm text-red-300 hover:bg-red-900/40 hover:text-red-200 disabled:cursor-not-allowed disabled:opacity-40"
              title="Discard unsaved changes"
            >
              Discard
            </button>
            <button
              type="button"
              @click="openShare"
              class="flex h-8 items-center justify-center gap-2 rounded border border-neutral-700 px-3 text-sm text-neutral-300 hover:bg-neutral-700"
              title="Share filter"
              aria-label="Share filter"
            >
              <svg
                class="h-4 w-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <circle cx="18" cy="5" r="3"></circle>
                <circle cx="6" cy="12" r="3"></circle>
                <circle cx="18" cy="19" r="3"></circle>
                <path d="m8.6 10.5 6.8-4m-6.8 7 6.8 4"></path>
              </svg>
              Share
            </button>
            <span class="h-6 w-px bg-neutral-700"></span>
            <button @click="onSave" class="rounded bg-neutral-800 px-3 py-1.5 text-sm hover:bg-neutral-700" title="Save config (Ctrl+S)">
              Save
            </button>
            <button
              @click="onSaveAndExport"
              :disabled="exporting"
              class="rounded bg-green-700 px-3 py-1.5 text-sm font-medium text-white hover:bg-green-600 disabled:opacity-50"
              title="Save and export to Path of Exile (Ctrl+Shift+S)"
            >
              {{ exporting ? "Exporting…" : "Export" }}
            </button>
            <button
              @click="previewOpen = true"
              class="rounded bg-neutral-800 px-3 py-1.5 text-sm hover:bg-neutral-700 md:hidden"
              title="Show output preview"
            >
              Preview
            </button>
          </template>
          <template v-else-if="view === 'global'">
            <button
              type="button"
              @click="globalRef?.undo()"
              :disabled="!globalCanUndo"
              class="flex h-8 w-8 items-center justify-center rounded border border-neutral-700 text-neutral-300 hover:bg-neutral-700 disabled:cursor-not-allowed disabled:opacity-30"
              title="Undo (Ctrl+Z)"
            >
              <svg
                class="h-4 w-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <polyline points="1 4 1 10 7 10"></polyline>
                <path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"></path>
              </svg>
            </button>
            <button
              type="button"
              @click="globalRef?.redo()"
              :disabled="!globalCanRedo"
              class="flex h-8 w-8 items-center justify-center rounded border border-neutral-700 text-neutral-300 hover:bg-neutral-700 disabled:cursor-not-allowed disabled:opacity-30"
              title="Redo (Ctrl+Shift+Z)"
            >
              <svg
                class="h-4 w-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <polyline points="23 4 23 10 17 10"></polyline>
                <path d="M20.49 15a9 9 0 1 1-2.13-9.36L23 10"></path>
              </svg>
            </button>
            <button
              @click="globalRef?.discard()"
              :disabled="!globalDirty"
              class="rounded px-3 py-1.5 text-sm text-red-300 hover:bg-red-900/40 hover:text-red-200 disabled:cursor-not-allowed disabled:opacity-40"
              title="Discard unsaved changes"
            >
              Discard
            </button>
            <span class="h-6 w-px bg-neutral-700"></span>
            <button
              @click="onGlobalSave"
              :disabled="globalSaving || globalExporting"
              class="rounded bg-neutral-800 px-3 py-1.5 text-sm hover:bg-neutral-700 disabled:cursor-not-allowed disabled:opacity-50"
              title="Save settings, defaults and styles (Ctrl+S)"
            >
              {{ globalSaving ? "Saving…" : "Save" }}
            </button>
            <button
              @click="onGlobalExport"
              :disabled="globalSaving || globalExporting"
              class="rounded bg-green-700 px-3 py-1.5 text-sm font-medium text-white hover:bg-green-600 disabled:opacity-50"
              title="Save settings and export all filters (Ctrl+Shift+S)"
            >
              {{ globalExporting ? "Exporting…" : "Export" }}
            </button>
          </template>
        </div>
      </header>

      <div
        v-if="toast"
        class="pointer-events-none fixed left-1/2 top-4 z-50 flex max-w-[calc(100vw-2rem)] -translate-x-1/2 items-center gap-2 rounded border px-3 py-2 text-sm shadow-lg"
        :class="FLASH_META[toast.kind].tone"
      >
        <span class="shrink-0 leading-none">{{ FLASH_META[toast.kind].icon }}</span>
        <span>{{ toast.message }}</span>
      </div>

      <div class="flex-1 overflow-auto p-4 sm:p-6">
        <GlobalSettings v-show="view === 'global'" ref="globalRef" />
        <div v-if="view !== 'global' && selected && reference" class="space-y-4">
          <ConfigForm
            :key="selected"
            v-model="config"
            :reference="reference"
            :filter-name="selected"
            :defaults="defaults"
            @preview-section="previewSection = $event"
          />
        </div>
        <div v-else-if="view !== 'global' && !selected" class="flex h-full items-center justify-center">
          <div class="w-full max-w-md rounded border border-neutral-800 bg-neutral-900/40 p-10 text-center">
            <img src="/favicon.svg" alt="" class="mx-auto mb-4 h-14 w-14" />
            <h1 class="mb-6 text-2xl font-semibold">Welcome to Filtertool</h1>
            <div class="mb-6 text-left">
              <label class="mb-1 block text-xs text-neutral-400">PoE Filter Path</label>
              <div class="flex gap-2">
                <input
                  :value="settings?.filterPath ?? ''"
                  readonly
                  aria-label="PoE Filter Path"
                  placeholder="Choose your filter folder"
                  class="min-w-0 flex-1 rounded border border-neutral-700 bg-neutral-950 px-2 py-2 text-xs"
                />
                <button @click="browseHomePath" class="rounded border border-neutral-700 px-3 text-sm hover:bg-neutral-800">Browse…</button>
              </div>
            </div>
            <div v-if="!templateMode" class="flex flex-col items-stretch gap-3">
              <button
                type="button"
                @click="createFromLanding"
                class="rounded bg-blue-700 px-6 py-3 text-base font-semibold text-white hover:bg-blue-600"
              >
                Create Filter
              </button>
              <button
                type="button"
                @click="openImport"
                class="rounded border border-blue-700 px-6 py-3 text-base font-medium text-blue-300 hover:bg-blue-900/40"
              >
                Import Filter
              </button>
              <button
                type="button"
                @click="onGlobal"
                class="rounded bg-neutral-300 px-6 py-3 text-base font-medium text-black hover:bg-neutral-200"
              >
                Global Settings
              </button>
            </div>
          </div>
        </div>
        <p v-else-if="view !== 'global'" class="text-neutral-500">Loading…</p>
      </div>
    </main>

    <aside
      v-if="view === 'filter' && selected"
      class="hidden shrink-0 flex-col border-l border-neutral-800 bg-neutral-950 md:flex"
      :class="previewOpen ? 'w-96' : 'w-10'"
    >
      <PreviewPane :preview="preview" v-model:open="previewOpen" :section="previewSection" />
    </aside>

    <div v-if="view === 'filter' && selected && previewOpen" class="fixed inset-0 z-40 flex flex-col bg-neutral-950 md:hidden">
      <PreviewPane :preview="preview" v-model:open="previewOpen" :section="previewSection" />
    </div>

    <div v-if="deleteTarget" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="deleteTarget = null">
      <div class="w-96 max-w-[calc(100vw-2rem)] rounded border border-neutral-700 bg-neutral-900 p-5">
        <h2 class="mb-4 text-lg font-semibold">Delete "{{ deleteTarget }}"</h2>
        <div class="flex flex-col gap-2">
          <button @click="confirmDelete(false)" class="rounded bg-neutral-700 px-3 py-2 text-sm hover:bg-neutral-600">
            Delete local filter only
          </button>
          <button @click="confirmDelete(true)" class="rounded bg-red-700 px-3 py-2 text-sm hover:bg-red-600">
            Delete local + exported .filter
          </button>
          <button @click="deleteTarget = null" class="rounded border border-neutral-700 px-3 py-2 text-sm hover:bg-neutral-800">
            Cancel
          </button>
        </div>
      </div>
    </div>

    <div v-if="showExportAll" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="showExportAll = false">
      <div class="w-96 max-w-[calc(100vw-2rem)] rounded border border-neutral-700 bg-neutral-900 p-5">
        <h2 class="mb-2 text-lg font-semibold">Export All Filters</h2>
        <p class="mb-4 text-sm text-neutral-400">Save and export all filters to Path of Exile?</p>
        <div class="flex flex-col gap-2">
          <button @click="confirmExportAll" class="rounded bg-green-700 px-3 py-2 text-sm font-medium text-white hover:bg-green-600">
            Save &amp; Export
          </button>
          <button @click="showExportAll = false" class="rounded border border-neutral-700 px-3 py-2 text-sm hover:bg-neutral-800">
            Cancel
          </button>
        </div>
      </div>
    </div>

    <div v-if="showUnsaved" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="showUnsaved = false">
      <div class="w-96 max-w-[calc(100vw-2rem)] rounded border border-neutral-700 bg-neutral-900 p-5">
        <h2 class="mb-2 text-lg font-semibold">Unsaved changes</h2>
        <p class="mb-4 text-sm text-neutral-400">You have unsaved changes. Save them before leaving?</p>
        <div class="flex flex-col gap-2">
          <button @click="confirmUnsavedSave" class="rounded bg-green-700 px-3 py-2 text-sm font-medium text-white hover:bg-green-600">
            Save
          </button>
          <button @click="confirmUnsavedDiscard" class="rounded bg-red-700 px-3 py-2 text-sm font-medium text-white hover:bg-red-600">
            Discard
          </button>
          <button @click="showUnsaved = false" class="rounded border border-neutral-700 px-3 py-2 text-sm hover:bg-neutral-800">
            Cancel
          </button>
        </div>
      </div>
    </div>

    <div v-if="showQuickstart" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div class="w-full max-w-lg rounded border border-neutral-700 bg-neutral-900 p-5">
        <img src="/favicon.svg" alt="" class="mb-4 h-12 w-12" />
        <h2 class="mb-5 text-xl font-semibold">Welcome to Filtertool</h2>
        <div class="mb-4">
          <label class="mb-1 block text-xs text-neutral-400" title="Where exported .filter files and sounds are written"
            >PoE Filter Path</label
          >
          <div class="flex gap-2">
            <input
              v-model="quickstartPath"
              class="w-full rounded border border-neutral-700 bg-neutral-950 px-2 py-1.5 text-sm"
              placeholder="Choose your Path of Exile filter folder"
              @keydown.enter.prevent="confirmQuickstart"
            /><button @click="browseQuickstart" class="rounded border border-neutral-700 px-3 text-sm hover:bg-neutral-800">Browse…</button>
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <button @click="skipQuickstart" class="rounded border border-neutral-700 px-3 py-1.5 text-sm hover:bg-neutral-800">
            Skip for now
          </button>
          <button
            @click="confirmQuickstart"
            :disabled="!quickstartPath.trim()"
            class="rounded bg-green-700 px-3 py-1.5 text-sm font-medium text-white hover:bg-green-600 disabled:cursor-not-allowed disabled:opacity-50"
          >
            Save &amp; Continue
          </button>
        </div>
      </div>
    </div>

    <div v-if="showShare" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="showShare = false">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="share-title"
        class="w-full max-w-lg rounded border border-neutral-700 bg-neutral-900 p-5"
      >
        <h2 id="share-title" class="mb-4 text-lg font-semibold">Share filter</h2>
        <label class="mb-4 flex items-center gap-2 text-sm">
          <input type="checkbox" v-model="shareGlobals" class="accent-green-600" />
          Include global defaults and custom styles
        </label>
        <textarea
          :value="shareText"
          readonly
          rows="5"
          aria-label="Shared filter JSON"
          class="mb-2 w-full rounded border border-neutral-700 bg-neutral-950 p-2 font-mono text-xs"
        ></textarea>
        <p class="mb-4 text-xs text-neutral-400">
          {{ shareCharacters.toLocaleString() }} characters<span v-if="shareCharacters > 2000">
            — over Discord's standard message limit; share as a file.</span
          >
        </p>
        <p v-if="shareError" class="mb-4 text-sm text-red-400">{{ shareError }}</p>
        <div class="flex justify-end gap-2">
          <button @click="showShare = false" class="rounded border border-neutral-700 px-3 py-2 text-sm">Close</button>
          <button
            @click="saveShareFile"
            :disabled="!shareText"
            class="rounded border border-neutral-700 px-3 py-2 text-sm disabled:opacity-50"
          >
            Save JSON…
          </button>
          <button @click="copyShare" :disabled="!shareText" class="rounded bg-green-700 px-3 py-2 text-sm disabled:opacity-50">Copy</button>
        </div>
      </div>
    </div>

    <div v-if="showImport" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="showImport = false">
      <div class="w-full max-w-lg rounded border border-neutral-700 bg-neutral-900 p-5">
        <h2 class="mb-2 text-lg font-semibold">Import filter</h2>
        <div class="mb-4 flex gap-2">
          <button
            @click="importFromPicker(false)"
            :disabled="importBusy"
            class="rounded border border-neutral-700 px-3 py-2 text-sm disabled:opacity-50"
          >
            JSON files…
          </button>
          <button
            @click="importFromPicker(true)"
            :disabled="importBusy"
            class="rounded border border-neutral-700 px-3 py-2 text-sm disabled:opacity-50"
          >
            Old web app folder…
          </button>
        </div>
        <div class="mb-4">
          <label class="mb-1 block text-xs text-neutral-400">Filter name</label>
          <input
            v-model="importName"
            class="w-full rounded border border-neutral-700 bg-neutral-950 px-2 py-1.5 text-sm"
            placeholder="My Filter"
          />
        </div>
        <div class="mb-4">
          <label class="mb-1 block text-xs text-neutral-400">Or paste shared filter / JSON</label>
          <textarea
            v-model="importJson"
            rows="8"
            class="w-full resize-y rounded border border-neutral-700 bg-neutral-950 px-2 py-1.5 font-mono text-xs"
            placeholder='{ "buildProfile": { … }, "buildSpecificOptions": { … } }'
          ></textarea>
          <p v-if="importError" class="mt-1 whitespace-pre-line text-xs text-red-400">{{ importError }}</p>
        </div>
        <div class="flex justify-end gap-2">
          <button @click="showImport = false" class="rounded border border-neutral-700 px-3 py-1.5 text-sm hover:bg-neutral-800">
            Cancel
          </button>
          <button
            @click="confirmImport"
            :disabled="!importName.trim() || !importJson.trim()"
            class="rounded bg-green-700 px-3 py-1.5 text-sm font-medium text-white hover:bg-green-600 disabled:cursor-not-allowed disabled:opacity-50"
          >
            Import
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
