import definitions from "../../internal/data/currency.json"
import leagueRules from "../../internal/data/league-mechanic.json"
import reference from "../../internal/data/reference.json"

export const CURRENCY_TIERS = definitions.tiers
export const LEAGUE_RULES = leagueRules.map((rule) => ({
  ...rule,
  baseTypes: rule.itemClass === "Viscera" ? reference.baseTypesByClass.Viscera : rule.baseTypes,
}))

/** One owner per base type; other known currencies belong to Leaguestart. */
export function currencyCatalog(baseTypes: string[]): Record<string, string> {
  const excluded = new Set(["Scroll of Wisdom", "Portal Scroll", ...LEAGUE_RULES.flatMap((rule) => rule.baseTypes)])
  return Object.fromEntries(
    baseTypes
      .filter((name) => !excluded.has(name))
      .map((name) => {
        let tier = "leaguestart"
        for (const definition of CURRENCY_TIERS) {
          if (definition.baseTypes.includes(name) || definition.contains?.some((part) => name.includes(part))) tier = definition.key
        }
        return [name, tier]
      }),
  )
}

export function effectiveCurrencyAssignments(
  catalog: Record<string, string>,
  shared: Record<string, string> = {},
  local: Record<string, string> = {},
): Record<string, string> {
  const tiers = new Set(CURRENCY_TIERS.map((tier) => tier.key))
  return Object.fromEntries(
    Object.entries(catalog).map(([name, tier]) => [name, [local[name], shared[name]].find((id) => tiers.has(id)) ?? tier]),
  )
}
