import { describe, expect, it } from "vitest"
import reference from "../internal/data/reference.json"
import { CURRENCY_TIERS, LEAGUE_RULES, currencyCatalog, effectiveCurrencyAssignments } from "../src/config/currency"
import { BUILD_SPECIFIC_SCHEMA, DEFAULTS_SCHEMA, filterEditorGroup, type FieldSubsection } from "../src/config/form-schema"

describe("currency catalog", () => {
  const bases = [...reference.baseTypesByClass["Stackable Currency"], ...reference.baseTypesByClass["Delve Stackable Socketable Currency"]]
  const catalog = currencyCatalog(bases)

  it("assigns every known currency to exactly one editor owner", () => {
    const excluded = new Set(["Portal Scroll", "Scroll of Wisdom", ...LEAGUE_RULES.flatMap((rule) => rule.baseTypes)])
    expect(Object.keys(catalog)).toHaveLength(bases.filter((base) => !excluded.has(base)).length)
    const tiers = CURRENCY_TIERS.map((tier) => tier.key)
    for (const [name, tier] of Object.entries(catalog)) {
      expect(excluded.has(name)).toBe(false)
      expect(tiers).toContain(tier)
    }
    expect(catalog["Scroll Fragment"]).toBe("low")
    expect(catalog["Transmutation Shard"]).toBe("low")
    expect(catalog["Chaos Orb"]).toBe("a")
    expect(catalog["Primitive Chaotic Resonator"]).toBe("leaguestart")
  })

  it("merges per-item moves without duplicating items or discarding inherited assignments", () => {
    const effective = effectiveCurrencyAssignments(catalog, { "Chaos Orb": "s", "Scroll Fragment": "b" }, { "Chaos Orb": "low" })
    expect(effective["Chaos Orb"]).toBe("low")
    expect(effective["Scroll Fragment"]).toBe("b")
    expect(Object.keys(effective)).toEqual(Object.keys(catalog))
    expect(effectiveCurrencyAssignments(catalog, {}, { "Chaos Orb": "bad" })["Chaos Orb"]).toBe("a")
  })

  it("keeps all three chart biomes backed by exact known base types", () => {
    const charts = LEAGUE_RULES.filter((rule) => rule.itemClass === "Chart").flatMap((rule) => rule.baseTypes)
    expect(charts.sort()).toEqual([...reference.baseTypesByClass.Chart].sort())
  })

  it("keeps currency styles in Global Settings only", () => {
    const fields = (sections: FieldSubsection[]): string[] =>
      sections.flatMap((section) => [
        ...section.fields.filter((field) => field.control === "style").map((field) => field.path),
        ...fields(section.subsections ?? []),
      ])
    const currency = BUILD_SPECIFIC_SCHEMA.find((group) => group.key === "currency")!
    expect(fields(filterEditorGroup(currency).subsections ?? [])).toEqual([])
    const shared = DEFAULTS_SCHEMA.find((group) => group.key === "currency")!
    expect(fields(shared.subsections ?? [])).toContain("currency.tiers.s.style")
    expect(fields(shared.subsections ?? [])).toContain("currency.gold.highStack.style")
    expect(fields(shared.subsections ?? [])).toContain("currency.scrolls.wisdom.style")
  })

  it("gives Leaguestart a separate style from B Tier", () => {
    expect(CURRENCY_TIERS.find((tier) => tier.key === "leaguestart")?.style).toBe("currencyLeaguestart")
    expect(CURRENCY_TIERS.find((tier) => tier.key === "b")?.style).toBe("currencyC")
  })

  it("keeps the league style editable in Global Settings, not individual filters", () => {
    const league = BUILD_SPECIFIC_SCHEMA.find((group) => group.key === "leagueMechanic")!
    expect(filterEditorGroup(league).fields.some((field) => field.control === "style")).toBe(false)
    expect(
      DEFAULTS_SCHEMA.find((group) => group.key === "leagueMechanic")?.fields.some((field) => field.path === "leagueMechanic.style"),
    ).toBe(true)
  })
})
