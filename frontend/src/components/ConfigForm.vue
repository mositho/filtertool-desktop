<script setup lang="ts">
import { computed, provide, ref } from "vue"
import {
  BUILD_PROFILE_SCHEMA,
  BUILD_SPECIFIC_SCHEMA,
  filterEditorGroup,
  editorSectionRank,
  type FieldGroup,
  type FieldSubsection,
  type SchemaField,
} from "@schema/form-schema"
import type { FilterConfig, ReferenceData } from "../api"
import { getPath, setPath, deletePath } from "../path"
import FieldInput from "./FieldInput.vue"
import HighlightEditor from "./HighlightEditor.vue"
import WeaponProgression from "./WeaponProgression.vue"
import { fieldDefault, defaultSource } from "../fieldDefaults"
import RuleHeader from "./RuleHeader.vue"
import starterSupports from "../../../internal/data/starter-supports.json"
import CurrencyEditor from "./CurrencyEditor.vue"
import { layoutFields } from "../formLayout"
import { useCollapsedState } from "../viewState"
import { isLegacyStyle, mergeStyles, namedStylesKey, STYLE_ALIASES, type StyleMap } from "../namedStyles"
import NamedStylesPanel from "./NamedStylesPanel.vue"

const props = defineProps<{ modelValue: FilterConfig; reference: ReferenceData; filterName: string; defaults?: Record<string, unknown> }>()
const emit = defineEmits<{ "update:modelValue": [value: FilterConfig] }>()
const stylesPanel = ref<InstanceType<typeof NamedStylesPanel> | null>(null)

const localStyles = computed(() => mergeStyles({}, (props.modelValue.localStyles ?? {}) as StyleMap))
const effectiveReference = computed(() => {
  const styles = mergeStyles(props.reference.globalStyles ?? props.reference.styles, localStyles.value)
  return {
    ...props.reference,
    styles,
    styleNames: Object.keys(styles)
      .filter((key) => !isLegacyStyle(key))
      .sort(),
  }
})
function setLocalStyle(key: string, value: Record<string, unknown> | undefined) {
  const next = { ...localStyles.value }
  for (const old of ["gemCallout", "earlyShieldLink"]) if (STYLE_ALIASES[old] === key) delete next[old]
  if (value && Object.keys(value).length) next[key] = value
  else delete next[key]
  if (Object.keys(next).length) props.modelValue.localStyles = next
  else delete props.modelValue.localStyles
  emitChange()
}
provide(namedStylesKey, {
  openPopover: ref(null),
  styles: computed(() => effectiveReference.value.styles),
  overrides: localStyles,
  set(key, property, value) {
    const next = { ...localStyles.value[key] }
    if (value === undefined || value === null || value === "") delete next[property]
    else next[property] = value
    setLocalStyle(key, next)
  },
  reset: (key) => setLocalStyle(key, undefined),
  paste: setLocalStyle,
  reveal(key) {
    stylesPanel.value?.reveal(key)
  },
})

function emitChange() {
  emit("update:modelValue", props.modelValue)
}

const buildProfile = computed(() => props.modelValue.buildProfile)
const buildSpecific = computed(() => props.modelValue.buildSpecificOptions)

function optionsFor(field: SchemaField): string[] {
  if (field.control !== "select" && field.control !== "multiselect") return []
  if (field.options && field.options.length > 0) return field.options
  switch (field.path) {
    case "preferredArmour":
      return props.reference.armourTypes
    case "jewellery.amulets":
      return props.reference.amulets
    case "tinctures.baseTypes":
      return props.reference.baseTypesByClass["Tinctures"] ?? []
    case "gemCallouts.gems":
      return [...props.reference.skillGems, ...props.reference.supportGems]
    case "gemCallouts.styleOnly":
      return [...props.reference.skillGems, ...props.reference.supportGems].filter((gem) => !starterSupports.includes(gem))
    default:
      return []
  }
}

function defaultFor(root: Record<string, unknown>, field: SchemaField): unknown {
  return fieldDefault(field, root, props.defaults, effectiveReference.value.styles)
}

function fieldIsSet(root: Record<string, unknown>, field: SchemaField): boolean {
  return getPath(root, field.path) !== undefined
}

function usesDefault(root: Record<string, unknown>, field: SchemaField): boolean {
  const fallback = defaultFor(root, field)
  return (
    fallback !== undefined && (!fieldIsSet(root, field) || (field.path === "shieldProgression" && getPath(root, field.path) === fallback))
  )
}

function effectiveValue(root: Record<string, unknown>, field: SchemaField): unknown {
  const value = getPath(root, field.path)
  return value !== undefined ? value : defaultFor(root, field)
}

/** Number fields show the default as a placeholder; other controls show it as a value. */
function displayValue(root: Record<string, unknown>, field: SchemaField): unknown {
  if (field.path === "gemCallouts.styleOnly") {
    const gems = effectiveValue(root, field)
    return Array.isArray(gems) ? gems.filter((gem) => !starterSupports.includes(gem)) : gems
  }
  return field.control === "number" && !field.emptyMeansUncapped ? getPath(root, field.path) : effectiveValue(root, field)
}

function setFieldValue(root: Record<string, unknown>, field: SchemaField, value: unknown) {
  if (field.path === "shieldProgression" && value === defaultFor(root, field)) value = undefined
  if (value === undefined) deletePath(root, field.path)
  else setPath(root, field.path, value)
  emitChange()
}

function placeholderFor(root: Record<string, unknown>, field: SchemaField): string | undefined {
  if (field.placeholder) return field.placeholder
  const when = field.placeholderWhen
  if (!when) return undefined
  const { path, equals, notEquals, defaultValue, value } = when
  const related = getPath(root, path)
  const resolved = related === undefined ? defaultValue : related
  if (equals !== undefined && resolved === equals) return value
  if (notEquals !== undefined && resolved !== notEquals) return value
  return undefined
}

function defaultValueLabel(value: unknown): string | undefined {
  if (value === undefined || value === null) return undefined
  if (Array.isArray(value)) return value.length ? value.join(", ") : undefined
  if (typeof value === "boolean") return value ? "On" : "Off"
  return String(value)
}

/** Placeholder for an unset field: its explicit placeholder, otherwise its default value. */
function displayPlaceholder(root: Record<string, unknown>, field: SchemaField): string | undefined {
  if (field.placeholder === "uncapped" && defaultFor(root, field) !== undefined) return defaultValueLabel(defaultFor(root, field))
  const explicit = placeholderFor(root, field)
  if (explicit !== undefined) return explicit
  return fieldIsSet(root, field) ? undefined : defaultValueLabel(defaultFor(root, field))
}

const highlightPath = "highlightedEquipment.highlights"
const highlights = computed(() => {
  const raw = getPath(buildSpecific.value, highlightPath)
  return Array.isArray(raw) ? (raw as Record<string, unknown>[]) : []
})

const weaponPhases = computed(() => (Array.isArray(buildProfile.value.weaponPhases) ? buildProfile.value.weaponPhases : []))

function setWeaponPhases(value: Record<string, unknown>[]) {
  if (value.length === 0) deletePath(buildProfile.value, "weaponPhases")
  else setPath(buildProfile.value, "weaponPhases", value)
  emitChange()
}

function setHighlights(value: Record<string, unknown>[]) {
  setPath(buildSpecific.value, highlightPath, value)
  emitChange()
}

function subsectionRoot(groupRoot: Record<string, unknown>, subsection: FieldSubsection): Record<string, unknown> {
  if (subsection.root === "buildProfile") return buildProfile.value
  if (subsection.root === "buildSpecificOptions") return buildSpecific.value
  return groupRoot
}

function isVisible(root: Record<string, unknown>, field: SchemaField): boolean {
  if (!field.visibleWhen) return true
  const { path, equals, notEquals, defaultValue } = field.visibleWhen
  const value = getPath(root, path)
  const resolved = value === undefined ? defaultValue : value
  if (equals !== undefined && resolved !== equals) return false
  if (notEquals !== undefined && resolved === notEquals) return false
  return true
}

function visibleFields(root: Record<string, unknown>, fields: SchemaField[]): SchemaField[] {
  return fields.filter((field) => isVisible(root, field))
}

type SubsectionEntry = {
  key: string
  subsection: FieldSubsection
  visibilityField?: SchemaField
  group: FieldGroup
  root: Record<string, unknown>
  depth: number
  hasSiblingAbove: boolean
}

function subsectionEntries(group: FieldGroup, groupRoot: Record<string, unknown>): SubsectionEntry[] {
  const entries: SubsectionEntry[] = []
  const walk = (subsections: FieldSubsection[], parentRoot: Record<string, unknown>, depth: number) => {
    subsections.forEach((subsection, index) => {
      if (subsection.module) return
      const root = subsectionRoot(parentRoot, subsection)
      entries.push({
        key: `${depth}-${subsection.key}`,
        subsection,
        visibilityField: subsection.headerVisibility ? subsection.fields.find((field) => field.path.endsWith(".visibility")) : undefined,
        group,
        root,
        depth,
        hasSiblingAbove: index > 0,
      })
      if (!subsectionCollapsed(group.key, subsection)) walk(subsection.subsections ?? [], root, depth + 1)
    })
  }
  walk(group.subsections ?? [], groupRoot, 0)
  return entries
}

const buildProfileGroups = computed(() => BUILD_PROFILE_SCHEMA.map((group) => ({ group, root: buildProfile.value })))
const buildSpecificGroups = computed(() =>
  BUILD_SPECIFIC_SCHEMA.map((group) => ({ group: filterEditorGroup(group), root: buildSpecific.value })),
)

// Editor navigation order is independent of compiled filter rule priority.
const allGroups = computed(() =>
  [...buildProfileGroups.value, ...buildSpecificGroups.value].sort(
    (a, b) => editorSectionRank(a.group.key) - editorSectionRank(b.group.key),
  ),
)

function defaultCollapsed(): Record<string, boolean> {
  return Object.fromEntries(allGroups.value.map(({ group }) => [group.key, !["buildProfile", "weaponPhases"].includes(group.key)]))
}

const collapsed = useCollapsedState(
  computed(() => `filtertool:collapsed:${props.filterName}`),
  defaultCollapsed,
)

function subsectionCollapsed(group: string, section: FieldSubsection): boolean {
  return collapsed.value[`${group}:${section.key}`] ?? section.collapsed ?? true
}

function toggleSubsection(group: string, section: FieldSubsection) {
  collapsed.value[`${group}:${section.key}`] = !subsectionCollapsed(group, section)
}

function isCollapsed(key: string): boolean {
  return !!collapsed.value[key]
}

function toggleSection(key: string) {
  collapsed.value = { ...collapsed.value, [key]: !collapsed.value[key] }
}

function collapseAll() {
  stylesPanel.value?.collapse()
  const next: Record<string, boolean> = { ...collapsed.value }
  for (const entry of allGroups.value) next[entry.group.key] = true
  collapsed.value = next
}

function expandAll() {
  stylesPanel.value?.expand()
  const next = { ...collapsed.value }
  for (const { group } of allGroups.value) next[group.key] = false
  collapsed.value = next
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-end gap-2">
      <button
        type="button"
        @click="collapseAll"
        class="rounded border border-neutral-700 px-2.5 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
      >
        Collapse all
      </button>
      <button
        type="button"
        @click="expandAll"
        class="rounded border border-neutral-700 px-2.5 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
      >
        Expand all
      </button>
    </div>

    <template v-for="entry in allGroups" :key="entry.group.key">
      <section class="rounded border border-neutral-800 bg-neutral-900/40" :data-editor-group="entry.group.key">
        <button
          type="button"
          @click="toggleSection(entry.group.key)"
          :aria-expanded="!isCollapsed(entry.group.key)"
          class="flex w-full items-center justify-between gap-2 px-4 py-3 text-left hover:bg-neutral-900/40"
        >
          <span class="text-lg font-semibold uppercase tracking-wide text-neutral-200">{{ entry.group.title }}</span>
          <span class="text-neutral-400">{{ isCollapsed(entry.group.key) ? "▸" : "▾" }}</span>
        </button>

        <div v-show="!isCollapsed(entry.group.key)" class="px-4 pb-4">
          <p v-if="entry.group.description" class="mb-3 text-xs text-neutral-500">{{ entry.group.description }}</p>

          <CurrencyEditor
            v-if="entry.group.key === 'currency'"
            :root="buildSpecific"
            :reference="effectiveReference"
            :filter-name="filterName"
            :defaults="defaults"
            @change="emitChange"
          />

          <HighlightEditor
            v-if="entry.group.key === 'highlightedEquipment'"
            :highlights="highlights"
            :reference="effectiveReference"
            @change="emitChange"
            @update:highlights="setHighlights"
          />

          <template v-else-if="entry.group.key === 'weaponPhases'">
            <WeaponProgression
              :phases="weaponPhases"
              :reference="effectiveReference"
              :phase-template="getPath(defaults ?? {}, 'weaponPhaseTemplate') as Record<string, unknown> | undefined"
              @update:phases="setWeaponPhases"
              @change="emitChange"
            />
          </template>

          <template v-else>
            <div
              v-if="visibleFields(entry.root, entry.group.fields).length > 0"
              class="editor-card grid grid-cols-1 items-end gap-3 sm:grid-cols-2"
            >
              <div
                v-for="{ field, fullWidth } in layoutFields(visibleFields(entry.root, entry.group.fields))"
                :key="field.path"
                :class="['min-w-0', fullWidth ? 'sm:col-span-2' : '']"
              >
                <FieldInput
                  :reference="effectiveReference"
                  :default-source="defaultSource(field, defaults)"
                  :field="field"
                  :value="displayValue(entry.root, field)"
                  :options="optionsFor(field)"
                  :placeholder="displayPlaceholder(entry.root, field)"
                  :defaulted="usesDefault(entry.root, field)"
                  :can-reset="fieldIsSet(entry.root, field)"
                  @update:value="setFieldValue(entry.root, field, $event)"
                  @reset="setFieldValue(entry.root, field, undefined)"
                />
              </div>
            </div>

            <div
              v-for="sub in subsectionEntries(entry.group, entry.root)"
              :key="sub.key"
              :data-editor-subsection="sub.subsection.key"
              class="editor-card mt-3"
            >
              <div v-if="sub.visibilityField" class="mb-3 flex items-center justify-between gap-2">
                <RuleHeader
                  class="min-w-0 flex-1"
                  :title="sub.subsection.title"
                  :heading="sub.depth === 0"
                  compact
                  :expanded="!subsectionCollapsed(entry.group.key, sub.subsection)"
                  @toggle="toggleSubsection(entry.group.key, sub.subsection)"
                />
                <select
                  :aria-label="`${sub.subsection.title} visibility`"
                  :data-field-path="sub.visibilityField.path"
                  :value="effectiveValue(sub.root, sub.visibilityField)"
                  class="rounded border border-neutral-700 bg-neutral-900 px-2 py-1 text-xs"
                  @change="
                    setFieldValue(
                      sub.root,
                      sub.visibilityField,
                      ($event.target as HTMLSelectElement).value === defaultFor(sub.root, sub.visibilityField)
                        ? undefined
                        : ($event.target as HTMLSelectElement).value,
                    )
                  "
                >
                  <option value="Show">Show</option>
                  <option value="Hide">Hide</option>
                </select>
              </div>
              <RuleHeader
                v-else
                :title="sub.subsection.title"
                :heading="sub.depth === 0"
                compact
                :expanded="!subsectionCollapsed(entry.group.key, sub.subsection)"
                @toggle="toggleSubsection(entry.group.key, sub.subsection)"
              />
              <p
                v-if="sub.subsection.description && !subsectionCollapsed(entry.group.key, sub.subsection)"
                class="mb-3 text-xs text-neutral-500"
              >
                {{ sub.subsection.description }}
              </p>
              <div v-show="!subsectionCollapsed(entry.group.key, sub.subsection)" class="grid grid-cols-1 items-end gap-3 sm:grid-cols-2">
                <div
                  v-for="{ field, fullWidth } in layoutFields(
                    visibleFields(sub.root, sub.subsection.fields).filter((field) => field !== sub.visibilityField),
                  )"
                  :key="field.path"
                  :class="['min-w-0', fullWidth ? 'sm:col-span-2' : '']"
                >
                  <FieldInput
                    :reference="effectiveReference"
                    :default-source="defaultSource(field, defaults)"
                    :field="field"
                    :value="displayValue(sub.root, field)"
                    :options="optionsFor(field)"
                    :placeholder="displayPlaceholder(sub.root, field)"
                    :defaulted="usesDefault(sub.root, field)"
                    :can-reset="fieldIsSet(sub.root, field)"
                    @update:value="setFieldValue(sub.root, field, $event)"
                    @reset="setFieldValue(sub.root, field, undefined)"
                  />
                </div>
              </div>
            </div>
          </template>
        </div>
      </section>
    </template>
    <NamedStylesPanel ref="stylesPanel" :filter-name="filterName" />
  </div>
</template>
