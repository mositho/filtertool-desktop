import type { ComputedRef, InjectionKey, Ref } from "vue"
import type { SchemaField } from "@schema/form-schema"
import { DEFAULT_STYLE_SETTINGS, MIN_STYLE_SIZE } from "../../src/style-defaults"
import aliases from "../../internal/data/style-aliases.json"
import { CURRENCY_TIERS } from "../../src/config/currency"

const currencyStyleLabels: Record<string, string> = Object.fromEntries(
  CURRENCY_TIERS.map((tier) => [tier.style, tier.key === "low" ? tier.title : `Currency · ${tier.title}`]),
)

/** Keep saved preset keys stable while displaying the currency editor's tiers. */
export function styleLabel(key: string): string {
  return currencyStyleLabels[key] ?? key
}

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
  for (const old of ["gemCallout", "earlyShieldLink", "goldHighStack"]) {
    if (overrides[old]) {
      const key = STYLE_ALIASES[old]
      result[key] = { ...base[key], ...overrides[old], ...overrides[key] }
    }
    if (base[old]) result[old] = { ...result[STYLE_ALIASES[old]] }
  }
  for (const rarity of ["Normal", "Magic", "Rare"]) {
    const key = `highlightedEquipment${rarity}`
    if (result[key] && result[rarity.toLowerCase()]) {
      result[key] = { ...result[key], text: result[rarity.toLowerCase()].text }
    }
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
    title: "Equipment",
    keys: [
      "earlyShieldBase",
      "jewellery",
      "magicJewellery",
      "rareJewellery",
      "normal",
      "magic",
      "rare",
      "highlightedEquipmentNormal",
      "highlightedEquipmentMagic",
      "highlightedEquipmentRare",
    ],
  },
  {
    title: "Links",
    keys: ["twoLink", "threeLink", "fourLink", "fiveLink", "sixLink"],
  },
  { title: "Flasks & Tinctures", keys: ["lifeFlask", "magicLifeFlask", "manaFlask", "magicManaFlask", "utilityFlask", "tincture"] },
  {
    title: "Currency",
    keys: [
      "currencyA",
      "currencyB",
      "currencyC",
      "currencyLeaguestart",
      "currencyLow",
      "gold",
      "goldStack",
      "goldStackLarge",
      "wisdom",
      "portal",
    ],
  },
  { title: "Gems", keys: ["gem", "gemHighlight"] },
  { title: "League Mechanic", keys: ["leagueMechanic"] },
  {
    title: "Miscellaneous",
    keys: [
      "chromatic",
      "divinationCard",
      "fragments",
      "maps",
      "questItem",
      "relics",
      "sixSocket",
      "unique",
      "unknownItem",
      "whetstoneRecipe",
    ],
  },
]

export function groupStyleSections(keys: string[]) {
  const used = new Set(STYLE_SECTIONS.flatMap((section) => section.keys))
  const other = keys.filter((key) => !used.has(key) && !isLegacyStyle(key)).sort()
  return STYLE_SECTIONS.map((section) => (section.title === "Miscellaneous" ? { ...section, keys: [...section.keys, ...other] } : section))
}

export function usesRarityText(key: string): boolean {
  return (
    /^highlightedEquipment(Normal|Magic|Rare)$/.test(key) ||
    /^(?:(?:good|selected)?(?:Two|Three|Four)|two|three|four|five|six)Link$/.test(key)
  )
}

export function rarityTextPreviews(key: string): string[] {
  const rarity = /^highlightedEquipment(Normal|Magic|Rare)$/.exec(key)?.[1]
  return rarity ? [rarity] : ["Normal", "Magic", "Rare"]
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
  if (path.startsWith("gemCallouts.")) return ["gemHighlight"]
  if (path.startsWith("leagueMechanic.")) return ["leagueMechanic"]
  if (path.startsWith("currency.gold.highStack")) return ["goldStackLarge"]
  if (path.startsWith("currency.gold.stack")) return ["goldStack"]
  if (path.startsWith("currency.gold.")) return ["gold", "goldStack", "goldStackLarge"]
  if (path.startsWith("currency.scrolls.wisdom.")) return ["wisdom"]
  if (path.startsWith("currency.scrolls.portal.")) return ["portal"]
  return []
}
