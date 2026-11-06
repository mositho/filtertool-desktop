package compiler

import (
	"sort"

	"github.com/mositho/filtertool-desktop/internal/data"
)

var styleAliases = data.Load[map[string]string]("style-aliases.json")
var rarityTextStyles = data.Load[map[string][]string]("rarity-text-styles.json")

func PaletteStyle(name string) bool {
	return name == "sharedRarityNormal" || name == "sharedRarityMagic" || name == "sharedRarityRare"
}

// Retired names remain readable in saved overrides and preset references, but
// are not offered for new selections. Link-tier overrides stay tier-specific.
func LegacyStyle(name string) bool {
	_, ok := styleAliases[name]
	return ok
}

func MergeStyles(base, overrides Object) Object {
	return Merge(normalizeStyles(base), normalizeStyles(overrides))
}

// Retired link tiers retain their existing independent overrides. All other
// aliases are resolved at the boundary, never duplicated as internal styles.
func canonicalStyleName(name string) string {
	if target, ok := styleAliases[name]; ok && target != "twoLink" && target != "threeLink" && target != "fourLink" {
		return target
	}
	return name
}

func normalizeStyles(styles Object) Object {
	out := Clone(styles)
	names := []string{}
	for name := range styles {
		if canonicalStyleName(name) != name {
			names = append(names, name)
		}
	}
	// Alias conflicts have deterministic alphabetical precedence; explicit
	// canonical properties always win over every alias in the same layer.
	sort.Strings(names)
	for _, old := range names {
		name := canonicalStyleName(old)
		out[name] = Merge(Obj(out[name]), Obj(styles[old]))
		delete(out, old)
	}
	for _, old := range names {
		name := canonicalStyleName(old)
		out[name] = Merge(Obj(out[name]), Obj(styles[name]))
	}
	return out
}

func ResolveStyles(overrides Object) Object {
	styles := MergeStyles(Merge(data.Load[Object]("legacy-styles.json"), data.Load[Object]("styles.json")), overrides)
	for name, rarities := range rarityTextStyles {
		if len(rarities) > 0 {
			Obj(styles[name])["text"] = Obj(styles["sharedRarity"+rarities[0]])["text"]
		}
	}
	return styles
}

func (c *Compiler) namedStyle(name string) Object {
	if !c.legacyEquipmentSelections {
		name = canonicalStyleName(name)
	}
	return Obj(c.Styles[name])
}

func (c *Compiler) sharedRarityText(rarity string) any {
	return Obj(c.Styles["sharedRarity"+rarity])["text"]
}

func (c *Compiler) rarityStyle(base Object, rarity string) Object {
	style := Clone(base)
	style["text"] = c.sharedRarityText(rarity)
	return style
}

func (c *Compiler) linkStyle(word, tier string) string {
	name := word + "Link"
	if tier == "selected" || tier == "good" {
		old := tier + title(word) + "Link"
		if c.legacyEquipmentSelections || c.legacyLinkStyles[old] {
			return old
		}
	}
	return name
}

// Split only presentation: retain the original conditions, sounds and icons.
// The final rule keeps the original match scope for any other rarities.
func (c *Compiler) rarityTextRules(r *rule, rarities ...string) []*rule {
	if c.legacyEquipmentSelections || r.hidden {
		return []*rule{r}
	}
	out := []*rule{}
	for _, rarity := range rarities[1:] {
		copy := &rule{lines: append([]string{}, r.lines...)}
		copy.set("Rarity", "== "+rarity)
		copy.set("SetTextColor", rgb(Text(c.sharedRarityText(rarity))))
		out = append(out, copy)
	}
	r.set("SetTextColor", rgb(Text(c.sharedRarityText(rarities[0]))))
	return append(out, r)
}
