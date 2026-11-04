package compiler

import (
	"strings"

	"github.com/mositho/filtertool-desktop/internal/data"
)

var styleAliases = data.Load[map[string]string]("style-aliases.json")

// Retired names remain readable in saved overrides and preset references, but
// are not offered for new selections. Link-tier overrides stay tier-specific.
func LegacyStyle(name string) bool {
	_, ok := styleAliases[name]
	return ok
}

func MergeStyles(base, overrides Object) Object {
	out := Merge(base, overrides)
	for old, name := range map[string]string{"gemCallout": "gemHighlight", "earlyShieldLink": "earlyShieldBase", "goldHighStack": "goldStackLarge"} {
		if value, exists := overrides[old]; exists {
			out[name] = Merge(Merge(Obj(base[name]), Obj(value)), Obj(overrides[name]))
		}
	}
	return out
}

func ResolveStyles(overrides Object) Object {
	styles := MergeStyles(Merge(data.Load[Object]("legacy-styles.json"), data.Load[Object]("styles.json")), overrides)
	for _, rarity := range []string{"Normal", "Magic", "Rare"} {
		Obj(styles["highlightedEquipment"+rarity])["text"] = Obj(styles[strings.ToLower(rarity)])["text"]
	}
	for old, name := range map[string]string{"gemCallout": "gemHighlight", "earlyShieldLink": "earlyShieldBase", "goldHighStack": "goldStackLarge"} {
		styles[old] = Clone(Obj(styles[name]))
	}
	return styles
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
