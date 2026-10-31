type AreaLevelRange = { minAreaLevel?: number; maxAreaLevel?: number }
type ItemLevelRange = { minItemLevel?: number; maxItemLevel?: number }
type HighlightRange = AreaLevelRange & ItemLevelRange

type WeaponPhase = AreaLevelRange & {
  name?: string
  itemClasses?: readonly unknown[]
  baseTypes?: readonly unknown[]
  minAps?: number
}

type ConfigWithRanges = {
  buildProfile?: { weaponPhases?: readonly WeaponPhase[] }
  buildSpecificOptions?: { highlightedEquipment?: { highlights?: readonly HighlightRange[] } }
}

function rangeError(min: number | undefined, max: number | undefined, label: string): string | null {
  return min !== undefined && max !== undefined && min > max ? `Min ${label} must not exceed Max ${label}.` : null
}

export function areaLevelRangeError({ minAreaLevel, maxAreaLevel }: AreaLevelRange): string | null {
  return rangeError(minAreaLevel, maxAreaLevel, "Area Level")
}

export function itemLevelRangeError({ minItemLevel, maxItemLevel }: ItemLevelRange): string | null {
  return rangeError(minItemLevel, maxItemLevel, "Item Level")
}

function weaponPhaseLabel(phase: WeaponPhase, index: number): string {
  return phase.name?.trim() || `Phase ${index + 1}`
}

export function filterRangeErrors(config: ConfigWithRanges): string[] {
  const errors: string[] = []
  config.buildProfile?.weaponPhases?.forEach((phase, index) => {
    const label = weaponPhaseLabel(phase, index)
    if (!(phase.itemClasses?.length || phase.baseTypes?.length)) {
      errors.push(`${label}: select at least one item class or base type.`)
    }
    const error = areaLevelRangeError(phase)
    if (error) errors.push(`${label}: ${error}`)
  })
  config.buildSpecificOptions?.highlightedEquipment?.highlights?.forEach((highlight, index) => {
    for (const error of [areaLevelRangeError(highlight), itemLevelRangeError(highlight)]) {
      if (error) errors.push(`Highlight ${index + 1}: ${error}`)
    }
  })
  return errors
}
