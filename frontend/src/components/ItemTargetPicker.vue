<script setup lang="ts">
import { computed, ref } from "vue"
import type { ReferenceData } from "../api"

const props = defineProps<{ classes: string[]; bases: string[]; reference: ReferenceData; weaponOnly?: boolean }>()
const emit = defineEmits<{ "update:classes": [value: string[]]; "update:bases": [value: string[]] }>()
const search = ref("")
const classSearch = ref("")
const selectedClass = ref("")
const classesOpen = ref(false)
const basesOpen = ref(false)

const classes = computed(() =>
  props.weaponOnly ? [...(props.reference.weaponClasses ?? []), "Shields"] : (props.reference.itemClasses ?? []),
)
const filteredClasses = computed(() => {
  const query = classSearch.value.trim().toLowerCase()
  return query ? classes.value.filter((itemClass) => itemClass.toLowerCase().includes(query)) : classes.value
})
function dropLevel(base: string): number | undefined {
  const value = props.reference.dropLevelsByBaseType?.[base]
  return typeof value === "number" && Number.isFinite(value) && value >= 0 ? value : undefined
}
function compareBases(a: string, b: string): number {
  return (dropLevel(a) ?? Infinity) - (dropLevel(b) ?? Infinity) || a.localeCompare(b)
}
const selectedBaseNames = computed(() => new Set(props.bases))
const selectedClassNames = computed(() => new Set(props.classes))
const baseIndex = computed(() => {
  const byClass = new Map<string, string[]>()
  const classByBase = new Map<string, string>()
  for (const itemClass of classes.value) {
    const bases = [...new Set(props.reference.baseTypesByClass[itemClass] ?? [])]
    byClass.set(itemClass, bases)
    for (const base of bases) {
      // Preserve the first matching class if a name appears in multiple classes.
      if (!classByBase.has(base)) classByBase.set(base, itemClass)
    }
  }
  return { byClass, classByBase }
})
function classBases(itemClass: string): string[] {
  return baseIndex.value.byClass.get(itemClass) ?? []
}
function classForBase(base: string): string | undefined {
  return baseIndex.value.classByBase.get(base)
}
function baseSelected(base: string): boolean {
  const itemClass = classForBase(base)
  return selectedBaseNames.value.has(base) || Boolean(itemClass && selectedClassNames.value.has(itemClass))
}
function baseIsCovered(base: string): boolean {
  const itemClass = classForBase(base)
  return Boolean(itemClass && selectedClassNames.value.has(itemClass))
}
const selectedBases = computed(() => [...props.bases].filter((base) => !baseIsCovered(base)).sort(compareBases))
function classFullySelected(itemClass: string): boolean {
  const itemBases = classBases(itemClass)
  return (
    selectedClassNames.value.has(itemClass) || Boolean(itemBases.length > 0 && itemBases.every((base) => selectedBaseNames.value.has(base)))
  )
}
const visibleSelectionState = computed(() => {
  if (!bases.value.length) return "none"
  const count = bases.value.filter(baseSelected).length
  return count === 0 ? "none" : count === bases.value.length ? "all" : "some"
})
const bases = computed(() => {
  const candidates = selectedClass.value ? classBases(selectedClass.value) : [...baseIndex.value.classByBase.keys()]
  const query = search.value.trim().toLowerCase()
  return [...new Set(query ? candidates.filter((base) => base.toLowerCase().includes(query)) : candidates)].sort(compareBases)
})
const allBasesSelected = computed(() => !bases.value.length || visibleSelectionState.value === "all")
const noBasesSelected = computed(() => !bases.value.length || visibleSelectionState.value === "none")
function normalizeSelections(selectedClasses: string[], selectedBases: string[]) {
  const classesResult = [...selectedClasses]
  const basesResult = new Set(selectedBases)
  for (const itemClass of classes.value) {
    const itemBases = classBases(itemClass)
    if (classesResult.includes(itemClass) || (itemBases.length && itemBases.every((base) => basesResult.has(base)))) {
      classesResult.push(itemClass)
      for (const base of itemBases) basesResult.delete(base)
    }
  }
  return { classes: [...new Set(classesResult)], bases: [...basesResult] }
}
function setVisibleBases(selected: boolean) {
  const classesToExpand = new Set<string>()
  if (!selected) {
    for (const base of bases.value) {
      const itemClass = classForBase(base)
      if (itemClass && props.classes.includes(itemClass)) classesToExpand.add(itemClass)
    }
  }

  let nextClasses = [...props.classes]
  let nextBases = [...props.bases]
  for (const itemClass of classesToExpand) {
    nextClasses = nextClasses.filter((entry) => entry !== itemClass)
    nextBases = [...new Set([...nextBases, ...classBases(itemClass)])]
  }

  const visible = new Set(bases.value)
  nextBases = selected ? [...new Set([...nextBases, ...bases.value])] : nextBases.filter((base) => !visible.has(base))
  const normalized = selected ? normalizeSelections(nextClasses, nextBases) : { classes: nextClasses, bases: nextBases }
  emit("update:classes", normalized.classes)
  emit("update:bases", normalized.bases)
}

function toggleClass(itemClass: string) {
  if (classFullySelected(itemClass)) {
    emit(
      "update:classes",
      props.classes.filter((entry) => entry !== itemClass),
    )
    emit(
      "update:bases",
      props.bases.filter((base) => !classBases(itemClass).includes(base)),
    )
    return
  }
  emit("update:classes", [...props.classes, itemClass])
  emit(
    "update:bases",
    props.bases.filter((base) => !classBases(itemClass).includes(base)),
  )
}

function toggleBase(base: string) {
  const itemClass = classForBase(base)
  if (baseIsCovered(base) && itemClass) {
    emit(
      "update:classes",
      props.classes.filter((entry) => entry !== itemClass),
    )
    emit("update:bases", [...new Set([...props.bases, ...classBases(itemClass).filter((entry) => entry !== base)])])
    return
  }
  const nextBases = props.bases.includes(base) ? props.bases.filter((entry) => entry !== base) : [...props.bases, base]
  const normalized = normalizeSelections(props.classes, nextBases)
  emit("update:classes", normalized.classes)
  emit("update:bases", normalized.bases)
}
</script>

<template>
  <div class="space-y-2 rounded border border-neutral-800 p-2">
    <p class="text-xs text-neutral-500">Select {{ weaponOnly ? "weapon" : "item" }} classes, individual bases, or both.</p>
    <div v-if="props.classes.length || props.bases.length" class="flex max-h-40 flex-wrap gap-1 overflow-y-auto">
      <button
        v-for="itemClass in props.classes"
        :key="`class-${itemClass}`"
        type="button"
        class="rounded bg-blue-900/70 px-2 py-1 text-xs text-blue-100 hover:bg-blue-800"
        :aria-label="`Remove item class ${itemClass}`"
        @click="toggleClass(itemClass)"
      >
        {{ itemClass }} ×
      </button>
      <button
        v-for="base in selectedBases"
        :key="`base-${base}`"
        type="button"
        class="rounded bg-neutral-800 px-2 py-1 text-xs text-neutral-200 hover:bg-neutral-700"
        :aria-label="`Remove base type ${base}`"
        @click="toggleBase(base)"
      >
        {{ base }} <span class="ml-1 whitespace-nowrap text-neutral-400">Lvl {{ dropLevel(base) ?? "—" }}</span> ×
      </button>
    </div>
    <details class="rounded border border-neutral-700" @toggle="classesOpen = ($event.target as HTMLDetailsElement).open">
      <summary class="rounded px-3 py-2 text-xs text-neutral-300">Choose {{ weaponOnly ? "weapon" : "item" }} classes</summary>
      <div v-if="classesOpen" class="space-y-2 px-2 pb-2">
        <input
          v-model="classSearch"
          type="search"
          aria-label="Search item classes"
          :placeholder="`Search ${weaponOnly ? 'weapon' : 'item'} classes`"
          class="w-full rounded border border-neutral-700 bg-neutral-900 px-2 py-1 text-xs"
        />
        <div class="grid max-h-36 grid-cols-1 gap-1 overflow-y-auto sm:grid-cols-2">
          <label v-for="itemClass in filteredClasses" :key="itemClass" class="flex items-center gap-2 text-xs text-neutral-300">
            <input
              type="checkbox"
              :checked="classFullySelected(itemClass)"
              :indeterminate="!classFullySelected(itemClass) && classBases(itemClass).some((base) => selectedBaseNames.has(base))"
              @change="toggleClass(itemClass)"
            />
            {{ itemClass }}
          </label>
          <span v-if="!filteredClasses.length" class="text-xs text-neutral-500">No matching classes</span>
        </div>
      </div>
    </details>
    <details class="rounded border border-neutral-700" @toggle="basesOpen = ($event.target as HTMLDetailsElement).open">
      <summary class="rounded px-3 py-2 text-xs text-neutral-300">Choose individual bases</summary>
      <div v-if="basesOpen" class="space-y-2 px-2 pb-2">
        <div class="flex flex-col gap-2 sm:flex-row">
          <select
            v-model="selectedClass"
            aria-label="Filter bases by class"
            class="rounded border border-neutral-700 bg-neutral-900 px-2 py-1 text-xs"
          >
            <option value="">All {{ weaponOnly ? "weapon" : "item" }} classes</option>
            <option v-for="itemClass in classes" :key="itemClass" :value="itemClass">{{ itemClass }}</option>
          </select>
          <input
            v-model="search"
            type="search"
            aria-label="Search base types"
            placeholder="Search bases"
            class="min-w-0 flex-1 rounded border border-neutral-700 bg-neutral-900 px-2 py-1 text-xs"
          />
        </div>
        <div class="flex items-center justify-end gap-3 text-xs">
          <button
            type="button"
            :disabled="allBasesSelected"
            class="text-neutral-300 hover:text-white disabled:opacity-40"
            @click="setVisibleBases(true)"
          >
            {{ search.trim() || selectedClass ? "Select results" : "Select all" }}
          </button>
          <button
            type="button"
            :disabled="noBasesSelected"
            class="text-neutral-300 hover:text-white disabled:opacity-40"
            @click="setVisibleBases(false)"
          >
            {{ search.trim() || selectedClass ? "Deselect results" : "Deselect all" }}
          </button>
        </div>
        <div class="grid max-h-40 grid-cols-1 gap-1 overflow-y-auto sm:grid-cols-2">
          <label v-for="base in bases" :key="base" class="flex items-center gap-2 text-xs text-neutral-300">
            <input type="checkbox" :checked="baseSelected(base)" @change="toggleBase(base)" />
            <span>{{ base }}</span
            ><span class="whitespace-nowrap text-neutral-500">Lvl {{ dropLevel(base) ?? "—" }}</span>
          </label>
          <span v-if="!bases.length" class="text-xs text-neutral-500">No matching bases</span>
        </div>
      </div>
    </details>
  </div>
</template>
