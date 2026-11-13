<script setup lang="ts">
import { computed, nextTick, ref, useId, watch } from "vue"
import { CURRENCY_TIERS, currencyCatalog, effectiveCurrencyAssignments } from "../../../src/config/currency"
import type { SchemaField } from "../../../src/config/form-schema"
import { getPath, setPath, deletePath } from "../path"
import type { ReferenceData } from "../api"
import FieldInput from "./FieldInput.vue"
import { layoutFields } from "../formLayout"
import { useCollapsedState } from "../viewState"
import StyleShortcut from "./StyleShortcut.vue"
import { setEditedValue } from "../edits"

const props = defineProps<{
  root: Record<string, unknown>
  reference: ReferenceData
  filterName: string
  defaults?: Record<string, unknown>
  defaultsEditor?: boolean
}>()
const emit = defineEmits<{ change: [] }>()
const search = ref("")
const selected = ref<string[]>([])
const activeTier = ref<string | null>(null)
const tierDefinition = computed(() => CURRENCY_TIERS.find((tier) => tier.key === activeTier.value))
const dragged = ref<string[]>([])
const dropTarget = ref<string | null>(null)
const detailPanel = ref<HTMLElement | null>(null)
const panelId = useId()
const storageKey = computed(() => `filtertool:currency-tiers:${props.filterName}`)

const collapsed = useCollapsedState(storageKey, () => Object.fromEntries(CURRENCY_TIERS.map((tier) => [tier.key, true])))
const searchExpanded = ref<Record<string, boolean>>({})
watch(
  () => props.filterName,
  () => {
    activeTier.value = null
    selected.value = []
    searchExpanded.value = {}
  },
)
watch(search, () => {
  searchExpanded.value = {}
})

function tierExpanded(tier: { key: string; items: string[] }): boolean {
  if (search.value.trim()) return searchExpanded.value[tier.key] ?? (tier.items.length > 0 || !collapsed.value[tier.key])
  return !collapsed.value[tier.key]
}

function toggleTier(tier: { key: string; items: string[] }) {
  const expanded = !tierExpanded(tier)
  if (search.value.trim()) searchExpanded.value[tier.key] = expanded
  else collapsed.value[tier.key] = !expanded
}

function setAllExpanded(expanded: boolean) {
  if (search.value.trim()) searchExpanded.value = Object.fromEntries(CURRENCY_TIERS.map((tier) => [tier.key, expanded]))
  else collapsed.value = Object.fromEntries(CURRENCY_TIERS.map((tier) => [tier.key, !expanded]))
}
const catalog = computed(() =>
  currencyCatalog([
    ...(props.reference.baseTypesByClass["Stackable Currency"] ?? []),
    ...(props.reference.baseTypesByClass["Delve Stackable Socketable Currency"] ?? []),
  ]),
)
const shared = computed(() => (getPath(props.defaults ?? {}, "currency.assignments") ?? {}) as Record<string, string>)
const local = computed(() => (getPath(props.root, "currency.assignments") ?? {}) as Record<string, string>)
const assignments = computed(() => effectiveCurrencyAssignments(catalog.value, shared.value, local.value))
const visible = computed(() =>
  Object.keys(catalog.value)
    .sort()
    .filter((name) => name.toLowerCase().includes(search.value.trim().toLowerCase())),
)
const tiers = computed(() =>
  CURRENCY_TIERS.map((tier) => ({
    ...tier,
    items: visible.value.filter((name) => assignments.value[name] === tier.key),
  })),
)
const active = computed(() => (selected.value.length === 1 ? selected.value[0] : undefined))
const canResetSelected = computed(() =>
  selected.value.some(
    (name) =>
      getPath(props.root, `currency.assignments.${name}`) !== undefined || getPath(props.root, `currency.items.${name}`) !== undefined,
  ),
)
const selectedTier = computed(() => {
  const tier = assignments.value[selected.value[0]] ?? ""
  return selected.value.every((name) => assignments.value[name] === tier) ? tier : ""
})
const selectedVisibility = computed(() => {
  if (!selected.value.length) return ""
  const state = visibility(selected.value[0])
  return selected.value.every((name) => visibility(name) === state) ? state : ""
})

function endDrag() {
  dragged.value = []
  dropTarget.value = null
}
watch(
  () => props.root,
  () => {
    selected.value = []
    activeTier.value = null
    endDrag()
  },
)

function moveTo(tier: string, names = selected.value) {
  if (!CURRENCY_TIERS.some((definition) => definition.key === tier)) return
  const inherited = effectiveCurrencyAssignments(catalog.value, shared.value)
  for (const name of names) {
    if (!(name in catalog.value)) continue
    setEditedValue(props.root, `currency.assignments.${name}`, tier, inherited[name])
  }
  emit("change")
}

function choose(name: string, event: MouseEvent) {
  activeTier.value = null
  if (event.ctrlKey || event.metaKey || event.shiftKey) {
    selected.value = selected.value.includes(name) ? selected.value.filter((item) => item !== name) : [...selected.value, name]
  } else selected.value = [name]
  if (selected.value.length === 1) void nextTick(() => detailPanel.value?.scrollIntoView?.({ block: "nearest", behavior: "smooth" }))
}

function startDrag(event: DragEvent, name: string) {
  activeTier.value = null
  if (!selected.value.includes(name)) selected.value = [name]
  dragged.value = [...selected.value]
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = "move"
    event.dataTransfer.setData("application/x-filtertool-currency", JSON.stringify(dragged.value))
  }
}

function dragOver(event: DragEvent, tier: string) {
  if (!dragged.value.length) return
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = "move"
  dropTarget.value = tier
}

function drop(event: DragEvent, tier: string) {
  // Only accept a drag started here, never external text/files or stale payloads.
  if (!dragged.value.length) return
  event.preventDefault()
  moveTo(tier, dragged.value)
  endDrag()
}

function tierVisibility(tier: string): string {
  return String(
    getPath(props.root, `currency.tiers.${tier}.visibility`) ??
      getPath(props.defaults ?? {}, `currency.tiers.${tier}.visibility`) ??
      "Show",
  )
}

function setTierVisibility(tier: string, value: string) {
  const path = `currency.tiers.${tier}.visibility`
  setEditedValue(props.root, path, value, getPath(props.defaults ?? {}, path) ?? "Show")
  emit("change")
}

function effectiveTierSetting(tier: string, key: string, includeLocal = true): unknown {
  let value: unknown = key === "sound" ? CURRENCY_TIERS.find((entry) => entry.key === tier)?.sound : undefined
  for (const source of [
    getPath(props.defaults ?? {}, `currency.tiers.${tier}.${key}`),
    includeLocal ? getPath(props.root, `currency.tiers.${tier}.${key}`) : undefined,
  ]) {
    if (source === undefined || (source !== null && typeof source === "object" && !Object.keys(source).length)) continue
    if (source === null) {
      value = key === "sound" ? CURRENCY_TIERS.find((entry) => entry.key === tier)?.sound : undefined
      continue
    }
    value = key === "icon" && typeof source === "object" ? { ...((value ?? {}) as object), ...source } : source
  }
  return value
}

function effectiveSetting(name: string, key: string, includeLocal = true): unknown {
  const tier = assignments.value[name]
  const builtInSound = props.reference.currencySounds?.[name]
  const builtIn = key === "sound" ? builtInSound : key === "icon" ? props.reference.currencyIcons?.[name] : undefined
  let value: unknown = builtIn ?? effectiveTierSetting(tier, key)
  for (const source of [
    getPath(props.defaults ?? {}, `currency.items.${name}.${key}`),
    includeLocal ? getPath(props.root, `currency.items.${name}.${key}`) : undefined,
  ]) {
    if (source === undefined) continue
    if (key === "sound") {
      if (source === null) value = builtInSound ?? effectiveTierSetting(tier, key)
      else if (typeof source === "object" && Object.keys(source).length) {
        const sound = source as Record<string, unknown>
        value =
          sound.enabled === true && sound.soundEnabled !== false && !sound.tts && !sound.soundId && !sound.soundFileName
            ? (builtInSound ?? effectiveTierSetting(tier, key))
            : sound
      }
    } else if (source === null) value = builtIn ?? effectiveTierSetting(tier, key)
    else value = source && typeof source === "object" ? { ...((value ?? {}) as object), ...source } : source
  }
  return value
}

function chooseTier(tier: string) {
  selected.value = []
  activeTier.value = tier
  void nextTick(() => detailPanel.value?.scrollIntoView?.({ block: "nearest", behavior: "smooth" }))
}
function selectResults() {
  activeTier.value = null
  selected.value = [...new Set([...selected.value, ...visible.value])]
}
function clearSelection() {
  activeTier.value = null
  selected.value = []
}
const canResetTier = computed(
  () => activeTier.value && ["sound", "icon"].some((key) => getPath(props.root, `currency.tiers.${activeTier.value}.${key}`) !== undefined),
)

function visibility(name: string): string {
  return String(effectiveSetting(name, "visibility") ?? "Show")
}

function setVisibility(value: string) {
  if (value !== "Show" && value !== "Hide") return
  for (const name of selected.value) {
    const path = `currency.items.${name}.visibility`
    setEditedValue(props.root, path, value, effectiveSetting(name, "visibility", false) ?? "Show")
  }
  emit("change")
}

function resetSelected() {
  if (activeTier.value) {
    for (const key of ["sound", "icon"]) deletePath(props.root, `currency.tiers.${activeTier.value}.${key}`)
    emit("change")
    return
  }
  for (const name of selected.value) {
    deletePath(props.root, `currency.assignments.${name}`)
    deletePath(props.root, `currency.items.${name}`)
  }
  emit("change")
}

const itemFields = computed<SchemaField[]>(() =>
  active.value || activeTier.value
    ? [
        ...(active.value
          ? [
              {
                path: `currency.items.${active.value}.minStackSize`,
                label: "Minimum Stack Size",
                control: "number" as const,
                min: 0,
                integer: true,
                defaultValue: 0,
                tooltip: "Smaller stacks are explicitly hidden.",
              },
            ]
          : []),
        {
          path: activeTier.value ? `currency.tiers.${activeTier.value}.sound` : `currency.items.${active.value}.sound`,
          label: "Sound",
          control: "rule-sound",
          defaultValue: { enabled: false },
          tooltip: activeTier.value ? "Fallback for items without an item-specific sound." : undefined,
        },
        {
          path: activeTier.value ? `currency.tiers.${activeTier.value}.icon` : `currency.items.${active.value}.icon`,
          label: "Minimap icon",
          control: "icon",
          defaultValue: {},
          tooltip: activeTier.value ? "Fallback for items without an item-specific icon." : undefined,
        },
      ]
    : [],
)

function fieldValue(field: SchemaField): unknown {
  if (field.control === "number") return getPath(props.root, field.path)
  if (activeTier.value) return effectiveTierSetting(activeTier.value, field.control === "icon" ? "icon" : "sound") ?? field.defaultValue
  return effectiveSetting(active.value!, field.control === "icon" ? "icon" : "sound") ?? field.defaultValue
}

function updateField(field: SchemaField, value: unknown) {
  const key = field.control === "icon" ? "icon" : "sound"
  const inherited =
    field.control === "number"
      ? (getPath(props.defaults ?? {}, field.path) ?? field.defaultValue)
      : ((activeTier.value ? effectiveTierSetting(activeTier.value, key, false) : effectiveSetting(active.value!, key, false)) ??
        field.defaultValue)
  setEditedValue(props.root, field.path, value, inherited)
  emit("change")
}

function tierStyleKey(tier: (typeof CURRENCY_TIERS)[number]): string {
  const configured = effectiveTierSetting(tier.key, "style") as Record<string, unknown> | undefined
  return typeof configured?.preset === "string" ? configured.preset : tier.style
}
</script>

<template>
  <div class="space-y-3 @container">
    <input
      v-model="search"
      aria-label="Search currency"
      placeholder="Find currency"
      class="w-full rounded border border-neutral-700 bg-neutral-900 px-2 py-1.5 text-sm"
    />
    <div class="flex flex-wrap items-center gap-2 text-xs">
      <button type="button" class="text-neutral-300 hover:text-white" @click="selectResults">Select results</button>
      <button type="button" class="text-neutral-400 hover:text-white" @click="clearSelection">Clear selection</button>
      <span class="text-neutral-500" aria-live="polite">{{ selected.length }} selected · {{ visible.length }} results</span>
      <button type="button" class="ml-auto text-neutral-400 hover:text-white" @click="setAllExpanded(false)">Collapse tiers</button>
      <button type="button" class="text-neutral-400 hover:text-white" @click="setAllExpanded(true)">Expand tiers</button>
    </div>

    <div class="grid items-start gap-3 @min-[48rem]:grid-cols-[minmax(0,1fr)_20rem]">
      <div class="min-w-0 space-y-3">
        <section
          v-for="tier in tiers"
          :key="tier.key"
          :data-currency-tier="tier.key"
          :aria-label="tier.title"
          class="rounded border bg-neutral-950/40 transition-colors"
          :class="dropTarget === tier.key ? 'border-blue-400 bg-blue-950/30' : 'border-neutral-800'"
          @dragover="dragOver($event, tier.key)"
          @dragleave.self="dropTarget = null"
          @drop.stop="drop($event, tier.key)"
        >
          <div class="flex items-center gap-2 px-3 py-2" :class="tierExpanded(tier) ? 'border-b border-neutral-800' : ''">
            <h3 class="min-w-0 flex-1">
              <button
                type="button"
                class="flex w-full flex-wrap items-center gap-2 text-left"
                :aria-label="tier.title"
                :aria-expanded="tierExpanded(tier)"
                :aria-controls="`${panelId}-${tier.key}`"
                @click="toggleTier(tier)"
              >
                <span aria-hidden="true" class="text-neutral-400">{{ tierExpanded(tier) ? "▾" : "▸" }}</span>
                <span class="text-sm font-semibold text-neutral-300">{{ tier.title }}</span>
              </button>
            </h3>
            <div class="ml-auto flex min-w-0 flex-wrap items-center justify-end gap-2">
              <select
                :aria-label="`${tier.title} visibility`"
                :value="tierVisibility(tier.key)"
                class="ml-auto rounded border border-neutral-700 bg-neutral-900 px-2 py-1 text-xs"
                @change="setTierVisibility(tier.key, ($event.target as HTMLSelectElement).value)"
              >
                <option value="Show">Show</option>
                <option value="Hide">Hide</option>
              </select>
              <button
                type="button"
                :aria-label="`${tier.title} settings`"
                :title="`${tier.title} settings`"
                class="flex h-7 w-7 shrink-0 items-center justify-center rounded text-neutral-400 hover:bg-neutral-700 hover:text-white"
                :aria-pressed="activeTier === tier.key"
                @click="chooseTier(tier.key)"
              >
                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                  <path d="M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8Z" />
                  <path d="m9 3-1 3-3 1-2 3 2 2-1 3 2 3 3-1 3 2 3-1 1-3 3-1 2-3-2-2 1-3-2-3-3 1-3-2-3 1Z" />
                </svg>
              </button>
              <StyleShortcut :style-keys="[tierStyleKey(tier)]" />
            </div>
          </div>
          <div
            :id="`${panelId}-${tier.key}`"
            v-show="tierExpanded(tier)"
            :aria-label="`${tier.title} currencies`"
            class="flex max-h-64 min-h-14 flex-wrap content-start gap-1.5 overflow-y-auto p-3"
          >
            <button
              v-for="name in tier.items"
              :key="name"
              type="button"
              draggable="true"
              :data-currency-item="name"
              :aria-pressed="selected.includes(name)"
              :aria-label="`${name}${visibility(name) === 'Hide' ? ' (hidden)' : ''}`"
              class="rounded border px-2 py-1 text-xs transition-colors"
              :class="[
                selected.includes(name)
                  ? 'border-blue-400 bg-blue-950/60 text-blue-100 ring-1 ring-blue-400/40'
                  : 'border-neutral-700 bg-neutral-800 text-neutral-200 hover:border-neutral-500',
                visibility(name) === 'Hide' ? 'opacity-50' : '',
              ]"
              @click="choose(name, $event)"
              @dragstart="startDrag($event, name)"
              @dragend="endDrag"
            >
              {{ name }}<span v-if="visibility(name) === 'Hide'" class="ml-1 text-amber-400">· Hidden</span>
            </button>
            <p v-if="!tier.items.length" class="self-center text-xs text-neutral-500">
              {{ search ? "No matches" : "Empty" }}
            </p>
          </div>
        </section>
      </div>
      <aside
        aria-label="Currency details"
        class="min-w-0 space-y-3 @container @min-[48rem]:sticky @min-[48rem]:top-3 @min-[48rem]:min-h-[28rem]"
      >
        <section
          ref="detailPanel"
          class="min-h-48 space-y-3 rounded border border-blue-900/60 bg-neutral-950/60 p-3"
          aria-label="Currency item settings"
        >
          <div class="flex items-start justify-between gap-3 border-b border-neutral-800 pb-3">
            <h3 class="min-w-0 break-words text-sm font-semibold text-neutral-200">
              {{ tierDefinition?.title ?? active ?? (selected.length ? `${selected.length} currencies selected` : "Select Currency") }}
            </h3>
            <button
              type="button"
              :disabled="activeTier ? !canResetTier : !selected.length || (defaultsEditor && !canResetSelected)"
              :class="defaultsEditor && !(activeTier ? canResetTier : canResetSelected) ? 'invisible' : ''"
              class="flex h-7 w-7 shrink-0 items-center justify-center rounded text-neutral-400 hover:bg-neutral-700 hover:text-white disabled:opacity-50"
              :title="
                activeTier
                  ? 'Reset tier sound and icon to defaults.'
                  : 'Reset tier, visibility, sound, icon, and stack settings to defaults.'
              "
              :aria-label="activeTier ? `Reset ${tierDefinition?.title} settings` : 'Reset selected currency'"
              @click="resetSelected"
            >
              <svg
                class="h-4 w-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
              >
                <polyline points="1 4 1 10 7 10"></polyline>
                <path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"></path>
              </svg>
            </button>
          </div>
          <select
            v-if="!activeTier"
            :value="selectedTier"
            :disabled="!selected.length"
            aria-label="Currency tier"
            class="w-full min-w-0 rounded border border-neutral-700 bg-neutral-900 px-2 py-1.5 text-sm text-neutral-200 disabled:opacity-50"
            @change="moveTo(($event.target as HTMLSelectElement).value)"
          >
            <option v-if="!selectedTier" value="" disabled>{{ selected.length ? "Mixed tiers" : "—" }}</option>
            <option v-for="tier in CURRENCY_TIERS" :key="tier.key" :value="tier.key">{{ tier.title }}</option>
          </select>

          <select
            v-if="!activeTier"
            :value="selectedVisibility"
            :disabled="!selected.length"
            aria-label="Currency visibility"
            class="w-full rounded border border-neutral-700 bg-neutral-900 px-2 py-1.5 text-sm disabled:opacity-50"
            @change="setVisibility(($event.target as HTMLSelectElement).value)"
          >
            <option v-if="!selectedVisibility" value="" disabled>—</option>
            <option value="Show">Show</option>
            <option value="Hide">Hide</option>
          </select>

          <div v-if="active || activeTier" class="mt-3 grid grid-cols-1 gap-3 @min-[28rem]:grid-cols-2">
            <FieldInput
              v-for="{ field, fullWidth } in layoutFields(itemFields)"
              :key="field.path"
              :class="fullWidth ? '@min-[28rem]:col-span-2' : ''"
              :field="field"
              :defaults-editor="defaultsEditor"
              :value="fieldValue(field)"
              :options="field.options ?? []"
              :reference="reference"
              :defaulted="getPath(root, field.path) === undefined"
              :can-reset="getPath(root, field.path) !== undefined"
              :placeholder="field.control === 'number' && active ? String(effectiveSetting(active, 'minStackSize') ?? 0) : undefined"
              @update:value="updateField(field, $event)"
              @reset="updateField(field, undefined)"
            />
          </div>
        </section>
      </aside>
    </div>
  </div>
</template>
