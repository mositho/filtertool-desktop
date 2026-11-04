<script setup lang="ts">
import HighlightCard from "./HighlightCard.vue"
import NumberField from "./NumberField.vue"
import type { ReferenceData } from "../api"
import { areaLevelRangeError } from "@schema/range-validation"

const props = defineProps<{ modelValue: Record<string, unknown>; reference: ReferenceData }>()
const emit = defineEmits<{ "update:modelValue": [value: Record<string, unknown>] }>()

function updateArea(key: "minAreaLevel" | "maxAreaLevel", value: number | undefined) {
  const next = { ...props.modelValue }
  if (value === undefined) delete next[key]
  else next[key] = value
  emit("update:modelValue", next)
}

function changed() {
  emit("update:modelValue", JSON.parse(JSON.stringify(props.modelValue)))
}
</script>

<template>
  <div class="space-y-3" data-weapon-phase-template>
    <div class="editor-card grid max-w-lg grid-cols-1 gap-3 sm:grid-cols-2">
      <label class="space-y-1 text-xs text-neutral-400">
        <span>Min Area Level</span>
        <NumberField
          :model-value="modelValue.minAreaLevel as number | undefined"
          :min="0"
          integer
          placeholder="No min area level"
          @update:model-value="updateArea('minAreaLevel', $event)"
        />
      </label>
      <label class="space-y-1 text-xs text-neutral-400">
        <span>Max Area Level</span>
        <NumberField
          :model-value="modelValue.maxAreaLevel as number | undefined"
          :min="0"
          integer
          placeholder="No max area level"
          @update:model-value="updateArea('maxAreaLevel', $event)"
        />
      </label>
    </div>
    <p v-if="areaLevelRangeError(modelValue)" role="alert" class="text-xs text-red-400">{{ areaLevelRangeError(modelValue) }}</p>
    <HighlightCard :highlight="modelValue" :reference="reference" phase-mode @change="changed" />
  </div>
</template>
