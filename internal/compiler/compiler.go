// Package compiler renders ordered Path of Exile 1 filter rules without filesystem side effects.
package compiler

import (
	"fmt"
	"strings"

	"github.com/mositho/filtertool-desktop/internal/data"
)

type Weapon struct {
	BaseType  string  `json:"baseType"`
	ItemClass string  `json:"itemClass"`
	DropLevel float64 `json:"dropLevel"`
	APS       float64 `json:"aps"`
}
type Compiler struct {
	Defaults, Styles Object
	SoundFolder      string
	templates        Object
	weapons          []Weapon
	manifest         map[string]string
	starterSupports  []string
	rarityDefaults   Object
}

func New(defaults, styles Object, folder string) *Compiler {
	if folder == "" {
		folder = "poeft-sounds-v2"
	}
	c := &Compiler{Defaults: Merge(data.Load[Object]("defaults.json"), defaults), Styles: Merge(data.Load[Object]("styles.json"), styles), SoundFolder: folder, templates: data.Load[Object]("rules.json"), weapons: data.Load[[]Weapon]("weapons.json"), manifest: map[string]string{}}
	c.starterSupports = data.Load[[]string]("starter-supports.json")
	c.rarityDefaults = data.Load[Object]("highlight-defaults.json")
	for _, s := range data.Load[[]Object]("manifest.json") {
		c.manifest[Text(s["id"])] = Text(s["text"])
	}
	return c
}
func (c *Compiler) template(text string) []*rule {
	out := []*rule{}
	for _, block := range strings.Split(strings.TrimSpace(text), "\n\n") {
		lines := strings.Split(block, "\n")
		if lines[0] != "Show" && lines[0] != "Hide" {
			continue
		}
		r := newRule()
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "@style ") {
				r.style(Obj(c.Styles[strings.TrimPrefix(line, "@style ")]))
				continue
			}
			key, value, ok := strings.Cut(line, " ")
			if ok {
				value = strings.ReplaceAll(value, "poeft-sounds-v2/", c.SoundFolder+"/")
				r.set(key, value)
			}
		}
		if lines[0] == "Hide" {
			r.hide()
		}
		out = append(out, r)
	}
	return out
}
func (c *Compiler) fixed(key string) string {
	parts := []string{}
	for _, block := range strings.Split(Text(c.templates[key]), "\n\n") {
		rules := c.template(block)
		if len(rules) > 0 {
			parts = append(parts, rules[0].render())
		} else {
			parts = append(parts, block)
		}
	}
	return strings.Join(parts, "\n\n")
}
func (c *Compiler) section(options Object, name string) Object {
	base, override := Obj(c.Defaults[name]), Obj(options[name])
	out := Merge(base, override)
	rules := Clone(Obj(base["rules"]))
	for key, value := range Obj(override["rules"]) {
		entry := Clone(Obj(rules[key]))
		for field, v := range Obj(value) {
			entry[field] = v
		}
		rules[key] = entry
	}
	out["rules"] = rules
	return out
}
func (c *Compiler) Compile(config Object) (string, error) {
	if err := Validate(config); err != nil {
		return "", err
	}
	p := Obj(config["buildProfile"])
	o := Obj(config["buildSpecificOptions"])
	sections := []string{c.fixed("currency"), c.fixed("scrolls"), c.gems(Obj(o["gemCallouts"])), c.links(p, o), c.highlights(Obj(o["highlightedEquipment"])), c.phases(List(p["weaponPhases"])), c.fixed("sixSockets"), c.jewellery(c.section(o, "jewellery")), c.early(p, c.section(o, "early")), c.earlySockets(c.section(o, "early")), c.flasks(c.section(o, "flasks")), c.tinctures(c.section(o, "tinctures")), c.fixed("chromaticItems"), c.fixed("uniques"), c.equipment("Rare", c.section(o, "rareItems")), c.equipment("Magic", c.section(o, "magicItems")), c.whetstone(p), c.normal(c.section(o, "normalItems")), c.fixed("questItems"), c.fixed("divinationCards"), c.fixed("misc"), c.fixed("twilightStrand"), c.fixed("hideEquipment"), c.fixed("showUnknownItems")}
	result := []string{}
	for _, s := range sections {
		if s = strings.TrimSpace(s); s != "" {
			result = append(result, s)
		}
	}
	return strings.Join(result, "\n\n"), nil
}
func (c *Compiler) gems(cfg Object) string {
	cfg = Merge(Obj(c.Defaults["gemCallouts"]), cfg)
	rules := []*rule{}
	sounding := Strings(cfg["gems"])
	for _, gem := range sounding {
		r := newRule().quoted("BaseType", true, gem).icon("Cyan", "Star", 2).style(Obj(c.Styles["gemCallout"]))
		rules = append(rules, c.tts(r, strings.TrimSuffix(gem, " Support")))
	}
	for _, gem := range unique(append(Strings(cfg["styleOnly"]), c.starterSupports...)) {
		if contains(sounding, gem) {
			continue
		}
		rules = append(rules, newRule().quoted("BaseType", true, gem).style(Obj(c.Styles["gem"])))
	}
	rules = append(rules, c.template(Text(c.templates["gems"]))...)
	return heading("Gems", rules...)
}
func (c *Compiler) jewellery(cfg Object) string {
	rules := c.template(Text(c.templates["jewellery"]))
	last := rules[len(rules)-1]
	rules = rules[:len(rules)-1]
	for _, r := range rules {
		for i, line := range r.lines {
			if strings.HasPrefix(line, "AreaLevel <= ") {
				text := strings.Join(r.lines, "\n")
				key := "beltMaxAreaLevel"
				if strings.Contains(text, `Class "Rings"`) {
					key = "elementalRingMaxAreaLevel"
					if strings.Contains(text, `BaseType "Iron"`) || strings.Contains(text, `BaseType "Coral"`) {
						key = "basicRingMaxAreaLevel"
					}
				}
				r.lines[i] = "AreaLevel <= " + fmt.Sprint(cfg[key])
			}
		}
	}
	for _, a := range List(cfg["amulets"]) {
		name := Text(a)
		if name == "" {
			name = Text(Obj(a)["shortBaseType"])
		}
		for _, r := range c.template(Text(Obj(c.templates["amulets"])[name])) {
			for i, line := range r.lines {
				if strings.HasPrefix(line, "AreaLevel <= ") {
					r.lines[i] = "AreaLevel <= " + fmt.Sprint(cfg["amuletMaxAreaLevel"])
				}
			}
			rules = append(rules, r)
		}
	}
	rules = append(rules, last)
	return heading("Jewellery", rules...)
}
func (c *Compiler) flasks(cfg Object) string {
	rules := c.template(Text(c.templates["flasks"]))
	last := rules[len(rules)-1]
	rules = rules[:len(rules)-1]
	// Preserve the canonical utility-flask order, independent of picker order.
	for _, name := range []string{"Quicksilver", "Bismuth", "Stibnite", "Amethyst", "Ruby", "Sapphire", "Topaz", "Silver", "Aquamarine", "Granite", "Jade", "Quartz", "Diamond", "Gold", "Corundum", "Iron"} {
		if contains(Strings(cfg["utilityFlasks"]), name) {
			rules = append(rules, c.template(Text(Obj(c.templates["utilityFlasks"])[name]))...)
		}
	}
	return heading("Flasks", append(rules, last)...)
}
func (c *Compiler) tinctures(cfg Object) string {
	rules := []*rule{}
	for _, base := range Strings(cfg["baseTypes"]) {
		rules = append(rules, newRule().quoted("BaseType", true, base).icon("Red", "Raindrop", 2).style(Obj(c.Styles["tincture"])).set("PlayAlertSound", "6 240"))
	}
	return heading("Tinctures", append(rules, newRule().quoted("Class", false, "Tinctures").hide())...)
}

var armour = []string{"Body Armours", "Gloves", "Boots", "Helmets"}

func contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
func (c *Compiler) early(p, cfg Object) string {
	rules := []*rule{c.tts(newRule().quoted("Class", false, "Boots").number("AreaLevel", "<=", cfg["earlyBootsMaxAreaLevel"]).set("Rarity", "== Rare").style(Obj(c.Styles["rareArmour"])), "rare_boots")}
	mode := Text(Val(p, "shieldProgression", c.Defaults["shieldProgression"]))
	max := Val(cfg, "shieldMaxAreaLevel", cfg["earlyMaxAreaLevel"])
	if mode != "none" {
		rules = append(rules, newRule().quoted("Class", false, "Shields").number("AreaLevel", "<=", max).style(Obj(c.Styles["earlyShieldBase"])))
	}
	if mode == "full" {
		for _, kind := range Strings(Val(p, "preferredArmour", c.Defaults["preferredArmour"])) {
			rules = append(rules, defence(newRule().quoted("Class", false, "Shields").number("AreaLevel", ">", max).set("Rarity", "== Rare").style(Obj(c.Styles["rareArmour"])), kind))
		}
	}
	if Val(Obj(p["misc"]), "showRusticSash", Obj(c.Defaults["misc"])["showRusticSash"]) == true {
		rules = append(rules, c.tts(newRule().quoted("BaseType", false, "Rustic").quoted("Class", false, "Belts").set("AreaLevel", "<= 12").icon("White", "Pentagon", 2).style(Obj(c.Styles["jewellery"])), "rustic_sash"))
	}
	return heading("Early", rules...)
}
func (c *Compiler) earlySockets(cfg Object) string {
	rules := []*rule{}
	for i, count := range []int{2, 3} {
		key := fmt.Sprintf("armour%d", count)
		rc := Obj(Obj(cfg["rules"])[key])
		if rc["enabled"] == false {
			continue
		}
		cut := cfg["twoSocketMaxAreaLevel"]
		if count == 3 {
			cut = Val(cfg, "threeSocketMaxAreaLevel", cfg["earlyMaxAreaLevel"])
		}
		r := newRule().number("Sockets", "==", count).quoted("Class", false, armour...).number("AreaLevel", "<=", Val(rc, "maxAreaLevel", cut))
		rules = append(rules, c.presentation(r, rc, Object{"size": float64(40 + i*5)}))
	}
	return heading("Early Sockets", rules...)
}
func (c *Compiler) equipment(rarity string, cfg Object) string {
	rules := []*rule{}
	for _, shape := range []struct {
		key        string
		w, h, size int
	}{{"1x1", 1, 1, 45}, {"1x2", 1, 2, 43}, {"2x1", 2, 1, 43}, {"1x3", 1, 3, 40}, {"1x4", 1, 4, 37}, {"2x2", 2, 2, 37}, {"2x3", 2, 3, 33}, {"2x4", 2, 4, 30}} {
		rc := Obj(Obj(cfg["rules"])[shape.key])
		r := newRule().set("Rarity", "== "+rarity).number("Width", "==", shape.w).number("Height", "==", shape.h)
		var maximum any
		var color string
		if rarity == "Rare" {
			maximum = cfg["maxAreaLevel"]
			color = "#FFFF77"
		} else {
			maximum = cfg["bigMaxAreaLevel"]
			if shape.h <= 2 {
				maximum = cfg["smallMaxAreaLevel"]
			}
			color = "#8888FF"
		}
		maximum = Val(rc, "maxAreaLevel", maximum)
		if rarity == "Rare" && cfg["maxAreaLevel"] != nil && Num(maximum, 1e9) > Num(cfg["maxAreaLevel"], 1e9) {
			maximum = cfg["maxAreaLevel"]
		}
		r.number("AreaLevel", "<=", maximum)
		if rc["enabled"] == false {
			r.hide()
		} else {
			c.presentation(r, rc, Object{"size": float64(shape.size), "text": color})
		}
		rules = append(rules, r)
	}
	return heading(rarity+" Items", rules...)
}
func (c *Compiler) whetstone(p Object) string {
	if Val(Obj(p["misc"]), "whetstoneRecipe", Obj(c.Defaults["misc"])["whetstoneRecipe"]) != true {
		return ""
	}
	classes := Strings(data.Load[Object]("reference.json")["weaponClasses"])
	r := newRule().set("Quality", ">= 20").set("Rarity", "== Normal").quoted("Class", false, classes...).style(Obj(c.Styles["whetstoneRecipe"]))
	return heading("Whetstone Recipe", c.tts(r, "whet_recipe"))
}
func (c *Compiler) normal(cfg Object) string {
	rules := []*rule{}
	for _, key := range []string{"equipment", "belts"} {
		rc := Obj(Obj(cfg["rules"])[key])
		if rc["enabled"] == false {
			continue
		}
		var r *rule
		var style Object
		if key == "equipment" {
			r = newRule().set("Rarity", "== Normal").quoted("Class", false, append(append([]string{}, armour...), "Shields")...)
			style = Object{"size": float64(40)}
		} else {
			r = newRule().quoted("Class", false, "Belts").set("Rarity", "== Normal")
			style = Obj(c.Styles["jewellery"])
		}
		r.number("AreaLevel", "<=", Val(rc, "maxAreaLevel", cfg["maxAreaLevel"]))
		rules = append(rules, c.presentation(r, rc, style))
	}
	return heading("Normal Items", rules...)
}
