package compiler

import (
	"fmt"
	"math"
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
	rules := []*rule{c.tts(newRule().set("LinkedSockets", "= 6").icon("Red", "Diamond", 2).style(Obj(c.Styles["currencyA"])), "six_link"), c.tts(newRule().set("LinkedSockets", "= 5").icon("Orange", "Diamond", 2).style(Obj(c.Styles["currencyB"])), "five_link")}
	for _, n := range []int{4, 3, 2} {
		word := map[int]string{2: "two", 3: "three", 4: "four"}[n]
		max := Num(cfg[word+"LinkMaxAreaLevel"], 0)
		cutoff := Num(cfg[word+"LinkTtsCutoffLevel"], 0)
		classes := armour
		if n == 2 {
			classes = []string{""}
		}
		ruleConfig := func(tier string) Object { return Obj(Obj(cfg["rules"])[fmt.Sprint(n)+tier]) }
		build := func(class, kind, color, tier, style string, maximum float64) []*rule {
			rc := ruleConfig(tier)
			if rc["enabled"] == false {
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
		maxFor := func(tier string) float64 { return Num(ruleConfig(tier)["maxAreaLevel"], max) }
		for _, class := range classes {
			for _, kind := range kinds {
				for _, color := range colors {
					rules = append(rules, build(class, kind, color, "selected", "selected"+title(word)+"Link", maxFor("selected"))...)
				}
			}
		}
		for _, class := range classes {
			for _, color := range colors {
				rules = append(rules, build(class, "", color, "good", "good"+title(word)+"Link", maxFor("good"))...)
			}
		}
		for _, class := range classes {
			for _, kind := range kinds {
				rules = append(rules, build(class, kind, "", "good", "good"+title(word)+"Link", maxFor("good"))...)
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
