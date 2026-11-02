package compiler

import (
	"strings"

	"github.com/mositho/filtertool-desktop/internal/data"
)

// Flask progression is shared with the editor; nil means explicitly uncapped,
// while an absent override inherits the catalog's existing progression limit.
func (c *Compiler) flaskProgression(cfg Object) []*rule {
	catalog := data.Load[map[string][]Object]("flask-progression.json")
	rules := []*rule{}
	for _, kind := range []string{"life", "mana"} {
		class := strings.ToUpper(kind[:1]) + kind[1:] + " Flasks"
		color := "Red"
		if kind == "mana" {
			color = "Blue"
		}
		for _, flask := range catalog[kind] {
			maximum := Val(Obj(cfg[kind+"MaxAreaLevels"]), Text(flask["key"]), flask["maxAreaLevel"])
			if maximum == float64(0) {
				continue
			}
			for _, rarity := range []string{"Normal", "Magic"} {
				preset := kind + "Flask"
				if rarity == "Magic" {
					preset = "magic" + strings.ToUpper(preset[:1]) + preset[1:]
				}
				r := newRule().quoted("Class", true, class).
					quoted("BaseType", true, Text(flask["name"])+" "+strings.TrimSuffix(class, "s")).
					set("Rarity", "== "+rarity).number("AreaLevel", "<=", maximum).
					icon(color, "Raindrop", 2).style(Obj(c.Styles[preset]))
				rules = append(rules, c.tts(r, Text(flask["sound"])))
			}
		}
	}
	// Stop normal/magic flasks after their cutoffs, without swallowing uniques
	// that still belong to the later dedicated unique-item section.
	return append(rules, newRule().quoted("Class", true, "Life Flasks", "Mana Flasks").set("Rarity", "<= Magic").hide())
}
