package compiler

import (
	"fmt"
	"math"
	"strings"
)

func (c *Compiler) links(p, o Object) string {
	cfg := c.section(o, "links")
	early := c.section(o, "early")
	colors := []string{}
	for _, s := range Strings(Val(p, "preferredColors", c.Defaults["preferredColors"])) {
		if !contains(colors, s) {
			colors = append(colors, s)
		}
	}
	kinds := Strings(Val(p, "preferredArmour", c.Defaults["preferredArmour"]))
	rules := []*rule{}
	if c.legacyEquipmentSelections {
		rules = append(rules,
			c.tts(newRule().set("LinkedSockets", "= 6").icon("Red", "Diamond", 2).style(Obj(c.Styles["currencyA"])), "six_link"),
			c.tts(newRule().set("LinkedSockets", "= 5").icon("Orange", "Diamond", 2).style(Obj(c.Styles["currencyB"])), "five_link"),
		)
	} else {
		for _, link := range []struct {
			sockets int
			style   string
			color   string
			sound   string
		}{{6, "sixLink", "Red", "six_link"}, {5, "fiveLink", "Orange", "five_link"}} {
			for _, rarity := range []string{"Normal", "Magic", "Rare"} {
				style := Clone(Obj(c.Styles[link.style]))
				style["text"] = Obj(c.Styles[strings.ToLower(rarity)])["text"]
				r := newRule().set("LinkedSockets", fmt.Sprintf("= %d", link.sockets)).set("Rarity", "== "+rarity).icon(link.color, "Diamond", 2).style(style)
				rules = append(rules, c.tts(r, link.sound))
			}
		}
	}
	for _, n := range []int{4, 3, 2} {
		word := map[int]string{2: "two", 3: "three", 4: "four"}[n]
		max := Num(cfg[word+"LinkMaxAreaLevel"], 0)
		cutoff := Num(cfg[word+"LinkTtsCutoffLevel"], 0)
		classes := armour
		if n == 2 {
			classes = []string{""}
		}
		ruleConfig := func(tier string) Object { return c.ruleConfig(cfg, fmt.Sprint(n)+tier) }
		buildBase := func(class, kind, color, tier, style string, maximum float64) []*rule {
			rc := ruleConfig(tier)
			if rc["enabled"] == false && c.legacyEquipmentSelections {
				return nil
			}
			base := func() *rule {
				cl := armour
				if class != "" {
					cl = []string{class}
				}
				r := newRule().quoted("Class", false, cl...).number("LinkedSockets", "==", n)
				if kind != "" {
					defence(r, kind)
				}
				if color != "" {
					r.set("SocketGroup", ">= "+fmt.Sprintf("%q", color))
				}
				return r
			}
			silentConfig := Clone(rc)
			delete(silentConfig, "sound")
			silent := c.presentation(base(), silentConfig, Obj(c.Styles[style])).number("AreaLevel", "<=", maximum)
			if n == 2 {
				return []*rule{c.presentation(silent, rc, Obj(c.Styles[style]))}
			}
			r := base()
			sound := Obj(rc["sound"])
			if sound["enabled"] != false && Text(sound["tts"]) == "" && Text(sound["soundFileName"]) == "" && sound["soundId"] == nil {
				slot := map[string]string{"Body Armours": "body", "Gloves": "gloves", "Boots": "boots", "Helmets": "helm"}[class]
				c.tts(r, fmt.Sprintf("%d_%s", n, slot))
			}
			audible := c.presentation(r, rc, Obj(c.Styles[style])).number("AreaLevel", "<=", math.Min(Num(rc["soundCutoffLevel"], cutoff), maximum))
			return []*rule{audible, silent}
		}
		build := func(class, kind, color, tier, style string, maximum float64) []*rule {
			baseRules := buildBase(class, kind, color, tier, style, maximum)
			if c.legacyEquipmentSelections {
				return baseRules
			}
			result := []*rule{}
			for _, base := range baseRules {
				for _, rarity := range []string{"Normal", "Magic", "Rare"} {
					r := &rule{hidden: base.hidden}
					for _, line := range base.lines {
						if !strings.HasPrefix(line, "SetTextColor ") {
							r.lines = append(r.lines, line)
						}
					}
					r.set("Rarity", "== "+rarity)
					if text := Obj(c.Styles[strings.ToLower(rarity)])["text"]; text != nil {
						r.set("SetTextColor", rgb(Text(text)))
					}
					result = append(result, r)
				}
				// Preserve the original presentation for unique items.
				result = append(result, base)
			}
			return result
		}
		maxFor := func(tier string) float64 { return Num(ruleConfig(tier)["maxAreaLevel"], max) }
		for _, class := range classes {
			for _, kind := range kinds {
				for _, color := range colors {
					rules = append(rules, build(class, kind, color, "selected", c.linkStyle(word, "selected"), maxFor("selected"))...)
				}
			}
		}
		for _, class := range classes {
			for _, color := range colors {
				rules = append(rules, build(class, "", color, "good", c.linkStyle(word, "good"), maxFor("good"))...)
			}
		}
		for _, class := range classes {
			for _, kind := range kinds {
				rules = append(rules, build(class, kind, "", "good", c.linkStyle(word, "good"), maxFor("good"))...)
			}
		}
		generic := func(maximum float64) {
			for _, class := range classes {
				rules = append(rules, build(class, "", "", "generic", word+"Link", maximum)...)
			}
		}
		if n == 3 {
			generic(math.Min(maxFor("generic"), Num(cfg["genericThreeLinkMaxAreaLevel"], Num(early["earlyMaxAreaLevel"], 12))))
		}
		if n == 2 || cfg["generic"+title(word)+"LinksEnabled"] == true {
			generic(maxFor("generic"))
		}
	}
	return heading("Links", rules...)
}
func title(s string) string {
	if s == "two" {
		return "Two"
	}
	if s == "three" {
		return "Three"
	}
	return "Four"
}
