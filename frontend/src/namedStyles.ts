import type { ComputedRef, InjectionKey, Ref } from "vue"
import type { SchemaField } from "@schema/form-schema"
import { DEFAULT_STYLE_SETTINGS, MIN_STYLE_SIZE } from "../../src/style-defaults"
import aliases from "../../internal/data/style-aliases.json"

export type Style = Record<string, unknown>
export type StyleMap = Record<string, Style>
export const STYLE_ALIASES: Record<string, string> = aliases
export function isLegacyStyle(key: string): boolean {
  return Object.hasOwn(STYLE_ALIASES, key)
}

// Renames inherit old overrides; retired link-tier overrides remain separate.
export function mergeStyles(base: StyleMap, overrides: StyleMap): StyleMap {
  const result = { ...base }
  for (const [key, style] of Object.entries(overrides)) result[key] = { ...base[key], ...style }
  for (const old of ["gemCallout", "earlyShieldLink"]) {
    if (overrides[old]) {
      const key = STYLE_ALIASES[old]
      result[key] = { ...base[key], ...overrides[old], ...overrides[key] }
    }
    if (base[old]) result[old] = { ...result[STYLE_ALIASES[old]] }
  }
  return result
}
export type NamedStyles = {
  openPopover: Ref<string | null>
  styles: ComputedRef<StyleMap>
  overrides: ComputedRef<StyleMap>
  set: (key: string, property: string, value: unknown) => void
  reset: (key: string) => void
  paste: (key: string, style: Style) => void
  reveal: (key: string) => void
}
export const namedStylesKey: InjectionKey<NamedStyles> = Symbol("namedStyles")
export const STYLE_GRID_TEMPLATE = "minmax(7rem, 1fr) 2.75rem 2.75rem 2.75rem 3.5rem 4rem 7rem minmax(10rem, 2fr)"

export const STYLE_SECTIONS = [
  {
    title: "Currency",
    keys: ["currencyA", "currencyB", "currencyC", "currencyLeaguestart", "currencyLow", "gold", "goldHighStack", "wisdom", "portal"],
  },
  { title: "League Mechanic", keys: ["leagueMechanic"] },
  {
    title: "Links",
    keys: ["twoLink", "threeLink", "fourLink", "fiveLink", "sixLink"],
  },
  { title: "Shields", keys: ["earlyShieldBase"] },
  { title: "Flasks & Tinctures", keys: ["lifeFlask", "magicLifeFlask", "manaFlask", "magicManaFlask", "utilityFlask", "tincture"] },
  { title: "Jewellery", keys: ["jewellery", "magicJewellery", "rareJewellery"] },
  { title: "Rarities", keys: ["normal", "magic", "rare"] },
  { title: "Highlights", keys: ["highlightedEquipmentRare", "highlightedEquipmentMagic", "highlightedEquipmentNormal"] },
]

export function usesRarityText(key: string): boolean {
  return /^(?:(?:good|selected)?(?:Two|Three|Four)|two|three|four|five|six)Link$/.test(key)
}

export function styleSwatch(style: Style): Record<string, string> {
  const resolved = { ...DEFAULT_STYLE_SETTINGS, ...style }
  const hex = String(resolved.background).replace(/^#/, "")
  const rgb = /^[0-9a-f]{6}$/i.test(hex) ? Number.parseInt(hex, 16) : 0
  const fontSize = Math.round(Math.max(MIN_STYLE_SIZE, Number(resolved.size)) * 0.42)
  return {
    color: String(resolved.text ?? DEFAULT_STYLE_SETTINGS.text),
    backgroundColor: `rgba(${(rgb >> 16) & 255},${(rgb >> 8) & 255},${rgb & 255},${resolved.backgroundOpacity})`,
    borderColor: String(resolved.border),
    fontSize: `${fontSize}px`,
    minHeight: `${Math.max(28, fontSize + 18)}px`,
  }
}

const linkStyles = (word: string) => [word + "Link"]

/** Only map fields with a named compiler style; inline-only rules have no named target. */
export function fieldStyleKeys(field: SchemaField, value?: unknown): string[] {
  if (field.control === "style") {
    const preset = (value as Style | undefined)?.preset ?? (field.defaultValue as Style | undefined)?.preset
    return typeof preset === "string" ? [preset] : []
  }
  if (field.sizePreset) return [STYLE_ALIASES[field.sizePreset] ?? field.sizePreset]
  const path = field.path
  if (path === "early.earlyMaxAreaLevel") return ["normal", "magic", "rare", "earlyShieldBase"]
  if (path === "early.twoSocketMaxAreaLevel" || path === "early.threeSocketMaxAreaLevel" || path.startsWith("early.rules.armour"))
    return ["normal", "magic", "rare"]
  if (/^links\.(two|three|four)Link/.test(path)) return linkStyles(path.split(".")[1].match(/^(two|three|four)/)![0])
  if (path.startsWith("links.genericThree")) return ["threeLink"]
  if (path.startsWith("links.genericFour")) return ["fourLink"]
  if (path === "preferredColors" || path === "preferredArmour") return [...linkStyles("two"), ...linkStyles("three"), ...linkStyles("four")]
  if (path === "shieldProgression" || path === "early.shieldMaxAreaLevel") return ["earlyShieldBase"]
  if (path === "misc.whetstoneRecipe") return ["whetstoneRecipe"]
  if (path === "misc.showRusticSash") return ["jewellery"]
  if (path === "early.earlyBootsMaxAreaLevel") return ["rareArmour"]
  if (path.startsWith("jewellery.")) return ["jewellery", "magicJewellery"]
  if (path === "normalItems.maxAreaLevel") return ["normal"]
  if (path.startsWith("magicItems.")) return ["magic"]
  if (path.startsWith("rareItems.")) return ["rare"]
  if (path.startsWith("flasks.life")) return ["lifeFlask", "magicLifeFlask"]
  if (path.startsWith("flasks.mana")) return ["manaFlask", "magicManaFlask"]
  if (path === "flasks.utilityFlasks") return ["utilityFlask"]
  if (path.startsWith("tinctures.")) return ["tincture"]
  if (path === "gemCallouts.styleOnly") return ["gem"]
  if (path.startsWith("gemCallouts.")) return ["gemHighlight"]
  if (path.startsWith("leagueMechanic.")) return ["leagueMechanic"]
  if (path.startsWith("currency.gold.highStack")) return ["goldHighStack"]
  if (path.startsWith("currency.gold.")) return ["gold", "goldHighStack"]
  if (path.startsWith("currency.scrolls.wisdom.")) return ["wisdom"]
  if (path.startsWith("currency.scrolls.portal.")) return ["portal"]
  return []
}
