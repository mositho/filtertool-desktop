package compiler

import (
	"math"
	"strings"
)

func (c *Compiler) weaponClass(name string) bool {
	if name == "Shields" {
		return true
	}
	for _, w := range c.weapons {
		if w.ItemClass == name {
			return true
		}
	}
	return false
}
func (c *Compiler) weaponBase(name string) (Weapon, bool) {
	for _, w := range c.weapons {
		if w.BaseType == name {
			return w, true
		}
	}
	return Weapon{}, false
}
func unique(values []string) []string {
	out := []string{}
	for _, s := range values {
		if !contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}
func (c *Compiler) findWeapons(classes []string, min any) []string {
	out := []string{}
	for _, w := range c.weapons {
		if (len(classes) == 0 || contains(classes, w.ItemClass)) && w.APS >= Num(min, 0) {
			out = append(out, w.BaseType)
		}
	}
	return out
}
func (c *Compiler) highlight(cfg Object) []*rule {
	bases, classes := Strings(cfg["baseTypes"]), Strings(cfg["itemClasses"])
	if len(bases) == 0 && len(classes) == 0 && cfg["minAps"] == nil {
		return nil
	}
	rarities := Strings(cfg["rarities"])
	if len(rarities) == 0 {
		rarities = []string{"Normal", "Magic", "Rare"}
	}
	makeRules := func(bases, classes []string, maximum any) []*rule {
		out := []*rule{}
		for _, rarity := range rarities {
			if !contains([]string{"Normal", "Magic", "Rare"}, rarity) {
				continue
			}
			style := Obj(c.Styles["highlightedEquipment"+rarity])
			if cfg["style"] != nil {
				s := Obj(cfg["style"])
				style = Merge(Obj(c.Styles[Text(s["preset"])]), s)
			}
			r := newRule().style(style)
			presentation := cfg
			if cfg["perRarityCustomization"] == true {
				presentation = Obj(cfg[strings.ToLower(rarity)])
			}
			if presentation["iconColor"] != nil || presentation["iconShape"] != nil {
				r.icon(Text(Val(presentation, "iconColor", "Cyan")), Text(Val(presentation, "iconShape", "UpsideDownHouse")), Val(presentation, "iconSize", float64(2)))
			}
			r.quoted("BaseType", false, bases...).quoted("Class", true, classes...).set("Rarity", "== "+rarity)
			for _, entry := range []struct{ field, key, op string }{{"minAreaLevel", "AreaLevel", ">="}, {"maxAreaLevel", "AreaLevel", "<="}, {"minItemLevel", "ItemLevel", ">="}, {"maxItemLevel", "ItemLevel", "<="}, {"linkedSockets", "LinkedSockets", ">="}, {"minSockets", "Sockets", ">="}} {
				v := cfg[entry.field]
				if entry.field == "maxAreaLevel" {
					v = maximum
				}
				r.number(entry.key, entry.op, v)
			}
			for _, key := range []string{"width", "height"} {
				s := Obj(cfg[key])
				if s["value"] != nil {
					label := "Width"
					if key == "height" {
						label = "Height"
					}
					r.number(label, Text(s["operator"]), s["value"])
				}
			}
			c.sound(r, presentation)
			r.set("Rarity", "== "+rarity)
			out = append(out, r)
		}
		return out
	}
	targets := func(bases, classes []string) []*rule {
		out := []*rule{}
		if len(classes) > 0 {
			out = append(out, makeRules(nil, classes, cfg["maxAreaLevel"])...)
		}
		if len(bases) > 0 {
			out = append(out, makeRules(bases, nil, cfg["maxAreaLevel"])...)
		}
		return out
	}
	wclasses, nonclasses := []string{}, []string{}
	for _, s := range classes {
		if c.weaponClass(s) {
			wclasses = append(wclasses, s)
		} else {
			nonclasses = append(nonclasses, s)
		}
	}
	if cfg["weaponCutoffOverlap"] == nil {
		if cfg["minAps"] != nil {
			classes = nonclasses
			bases = unique(append(bases, c.findWeapons(wclasses, cfg["minAps"])...))
		}
		return targets(bases, classes)
	}
	wbases, nonbases := []string{}, []string{}
	for _, s := range bases {
		if _, ok := c.weaponBase(s); ok {
			wbases = append(wbases, s)
		} else {
			nonbases = append(nonbases, s)
		}
	}
	if len(wclasses) > 0 || cfg["minAps"] != nil {
		wbases = append(wbases, c.findWeapons(wclasses, cfg["minAps"])...)
	}
	wbases = unique(wbases)
	maxima := []float64{}
	grouped := map[float64][]string{}
	for _, base := range wbases {
		w, _ := c.weaponBase(base)
		automatic := w.DropLevel + Num(cfg["weaponCutoffOverlap"], 10)
		if automatic < Num(cfg["minAreaLevel"], 0) {
			continue
		}
		max := math.Min(automatic, Num(cfg["maxAreaLevel"], math.Inf(1)))
		if _, ok := grouped[max]; !ok {
			maxima = append(maxima, max)
		}
		grouped[max] = append(grouped[max], base)
	}
	out := []*rule{}
	for _, max := range maxima {
		out = append(out, makeRules(grouped[max], nil, max)...)
	}
	return append(out, targets(nonbases, nonclasses)...)
}
func (c *Compiler) highlights(cfg Object) string {
	rules := []*rule{}
	for _, h := range List(cfg["highlights"]) {
		rules = append(rules, c.highlight(Obj(h))...)
	}
	if len(rules) == 0 {
		return ""
	}
	return heading("Highlighted Equipment", rules...)
}
func (c *Compiler) phases(phases []any) string {
	remaining := append([]any{}, phases...)
	ordered := []Object{}
	lower := func(p Object) float64 { return Num(p["minAreaLevel"], math.Inf(-1)) }
	upper := func(p Object) float64 { return Num(p["maxAreaLevel"], math.Inf(1)) }
	bounds := func(p Object) int {
		n := 0
		if p["minAreaLevel"] != nil {
			n++
		}
		if p["maxAreaLevel"] != nil {
			n++
		}
		return n
	}
	for len(remaining) > 0 {
		selected := -1
		for i, v := range remaining {
			p := Obj(v)
			candidate := true
			for j, other := range remaining {
				if i == j {
					continue
				}
				q := Obj(other)
				if lower(q) >= lower(p) && upper(q) <= upper(p) && (lower(q) > lower(p) || upper(q) < upper(p)) {
					candidate = false
					break
				}
			}
			if candidate && (selected < 0 || bounds(p) > bounds(Obj(remaining[selected]))) {
				selected = i
			}
		}
		ordered = append(ordered, Clone(Obj(remaining[selected])))
		remaining = append(remaining[:selected], remaining[selected+1:]...)
	}
	rules := []*rule{}
	for _, p := range ordered {
		if p["weaponCutoffOverlap"] == nil {
			p["weaponCutoffOverlap"] = float64(10)
		}
		rules = append(rules, c.highlight(p)...)
	}
	if len(rules) == 0 {
		return ""
	}
	return heading("Weapon Progression", rules...)
}
