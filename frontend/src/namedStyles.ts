import type { ComputedRef, InjectionKey, Ref } from "vue"
import type { SchemaField } from "@schema/form-schema"
import { DEFAULT_STYLE_SETTINGS, MIN_STYLE_SIZE } from "../../src/style-defaults"
import aliases from "../../internal/data/style-aliases.json"
import rarityStyles from "../../internal/data/rarity-text-styles.json"
import { CURRENCY_TIERS } from "../../src/config/currency"

const currencyStyleLabels: Record<string, string> = Object.fromEntries(
  CURRENCY_TIERS.map((tier) => [tier.style, `Currency · ${tier.key === "low" ? "Low value" : tier.title}`]),
)

/** Keep saved preset keys stable while displaying the currency editor's tiers. */
export function styleLabel(key: string): string {
  if (key === "highlightedEquipment") return "highlight"
  if (key === "fatedItems") return "Fated Items"
  if (isPaletteStyle(key)) return `Shared rarity text · ${key.slice("sharedRarity".length)}`
  return currencyStyleLabels[key] ?? key
}

export const PALETTE_KEYS = ["sharedRarityNormal", "sharedRarityMagic", "sharedRarityRare"]
export function isPaletteStyle(key: string): boolean {
  return PALETTE_KEYS.includes(key)
}

export function sharedRarityText(styles: StyleMap, rarity: string): unknown {
  return styles[`sharedRarity${rarity}`]?.text
}

export function styleTextHelp(key: string): string {
  return ["normal", "magic", "rare"].includes(key) ? "Used when no earlier equipment rule matches." : styleLabel(key)
}

export type Style = Record<string, unknown>
export type StyleMap = Record<string, Style>
export const STYLE_ALIASES: Record<string, string> = aliases
export function isLegacyStyle(key: string): boolean {
  return Object.hasOwn(STYLE_ALIASES, key)
}

export function canonicalStyleName(key: string): string {
  const target = STYLE_ALIASES[key]
  return target && !["twoLink", "threeLink", "fourLink"].includes(target) ? target : key
}

export function styleAliasesFor(key: string): string[] {
  return Object.keys(STYLE_ALIASES).filter((old) => old !== key && canonicalStyleName(old) === key)
}

function normalizeStyles(styles: StyleMap): StyleMap {
  const result = { ...styles }
  const names = Object.keys(styles)
    .filter((key) => canonicalStyleName(key) !== key)
    .sort()
  for (const old of names) {
    const key = canonicalStyleName(old)
    result[key] = { ...result[key], ...styles[old] }
    delete result[old]
  }
  for (const old of names) {
    const key = canonicalStyleName(old)
    result[key] = { ...result[key], ...styles[key] }
  }
  return result
}

// Renames inherit old overrides; retired link-tier overrides remain separate.
export function mergeStyles(base: StyleMap, overrides: StyleMap): StyleMap {
  const result = normalizeStyles(base)
  for (const [key, style] of Object.entries(normalizeStyles(overrides))) result[key] = { ...result[key], ...style }
  for (const [key, rarities] of Object.entries(rarityStyles)) {
    if (rarities.length > 0 && result[key] && result[`sharedRarity${rarities[0]}`]) {
      result[key] = { ...result[key], text: sharedRarityText(result, rarities[0]) }
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
      "highlightedEquipment",
      "rareArmour",
      "normal",
      "magic",
      "magicTier2",
      "magicTier3",
      "magicTier4",
      "magicTier5",
      "rare",
      "rareTier2",
      "rareTier3",
      "rareTier4",
      "rareTier5",
    ],
  },
  {
    title: "Links",
    keys: ["twoLink", "threeLink", "fourLink", "fiveLink", "sixLink"],
  },
  { title: "Flasks & Tinctures", keys: ["lifeFlask", "manaFlask", "utilityFlask", "tincture"] },
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
  { title: "League Mechanic", keys: ["leagueMechanic", "fatedItems"] },
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
  const other = keys.filter((key) => !used.has(key) && !isLegacyStyle(key) && !isPaletteStyle(key)).sort()
  return STYLE_SECTIONS.map((section) => (section.title === "Miscellaneous" ? { ...section, keys: [...section.keys, ...other] } : section))
}

export function usesRarityText(key: string): boolean {
  return Object.hasOwn(rarityStyles, canonicalStyleName(key))
}

export function rarityTextPreviews(key: string): string[] {
  return (rarityStyles as Record<string, string[]>)[canonicalStyleName(key)] ?? []
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
  const tier = /^(magic|rare)Items\.tiers\.([2-5])\./.exec(field.path)
  if (tier) return [`${tier[1]}Tier${tier[2]}`]
  if (field.control === "style") {
    const preset = (value as Style | undefined)?.preset ?? (field.defaultValue as Style | undefined)?.preset
    return typeof preset === "string" ? [canonicalStyleName(preset)] : []
  }
  if (field.sizePreset) return [STYLE_ALIASES[field.sizePreset] ?? field.sizePreset]
  const path = field.path
  if (path === "early.earlyMaxAreaLevel") return ["normal", "magic", "rare", "earlyShieldBase"]
  if (path === "early.twoSocketMaxAreaLevel" || path === "early.threeSocketMaxAreaLevel" || path.startsWith("early.rules.armour"))
    return [...PALETTE_KEYS, "normal", "magic", "rare"]
  if (/^links\.(two|three|four)Link/.test(path)) return linkStyles(path.split(".")[1].match(/^(two|three|four)/)![0])
  if (path.startsWith("links.genericThree")) return ["threeLink"]
  if (path.startsWith("links.genericFour")) return ["fourLink"]
  if (path === "preferredColors" || path === "preferredArmour") return [...linkStyles("two"), ...linkStyles("three"), ...linkStyles("four")]
  if (path === "shieldProgression" || path === "early.shieldMaxAreaLevel") return ["earlyShieldBase"]
  if (path === "misc.whetstoneRecipe") return ["whetstoneRecipe"]
  if (path === "misc.showRusticSash") return ["jewellery"]
  if (path === "early.earlyBootsMaxAreaLevel") return ["rareArmour"]
  if (path.startsWith("jewellery.")) return ["jewellery"]
  if (path === "normalItems.maxAreaLevel") return ["normal"]
  if (path.startsWith("magicItems.")) return ["magic"]
  if (path.startsWith("rareItems.")) return ["rare"]
  if (path.startsWith("flasks.life")) return ["lifeFlask"]
  if (path.startsWith("flasks.mana")) return ["manaFlask"]
  if (path === "flasks.utilityFlasks") return ["utilityFlask"]
  if (path.startsWith("tinctures.")) return ["tincture"]
  if (path.startsWith("gemCallouts.")) return ["gemHighlight"]
  if (path.startsWith("leagueMechanic.rules.fated.")) return ["fatedItems"]
  if (path.startsWith("leagueMechanic.")) return ["leagueMechanic"]
  if (path.startsWith("currency.gold.highStack")) return ["goldStackLarge"]
  if (path.startsWith("currency.gold.stack")) return ["goldStack"]
  if (path.startsWith("currency.gold.")) return ["gold", "goldStack", "goldStackLarge"]
  if (path.startsWith("currency.scrolls.wisdom.")) return ["wisdom"]
  if (path.startsWith("currency.scrolls.portal.")) return ["portal"]
  return []
}
