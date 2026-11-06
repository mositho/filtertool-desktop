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
	Defaults, Styles          Object
	SoundFolder               string
	templates                 Object
	weapons                   []Weapon
	manifest                  map[string]string
	starterGems               []string
	rarityDefaults            Object
	legacyCurrency            bool // Only used by the frozen web parity tests.
	legacyEquipmentSelections bool // Only used by the frozen web parity tests.
	legacyLinkStyles          map[string]bool
}

func New(defaults, styles Object, folder string) *Compiler {
	if folder == "" {
		folder = "poeft-sounds-v2"
	}
	c := &Compiler{Defaults: MergeDefaults(data.Load[Object]("defaults.json"), defaults), Styles: ResolveStyles(styles), SoundFolder: folder, templates: data.Load[Object]("rules.json"), weapons: data.Load[[]Weapon]("weapons.json"), manifest: map[string]string{}, legacyLinkStyles: map[string]bool{}}
	for name := range data.Load[Object]("legacy-styles.json") {
		_, c.legacyLinkStyles[name] = styles[name]
	}
	c.starterGems = append(data.Load[[]string]("starter-skills.json"), data.Load[[]string]("starter-supports.json")...)
	c.Defaults = CleanRuleDefaults(c.Defaults)
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
		styleName := ""
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "@style ") {
				styleName = strings.TrimPrefix(line, "@style ")
				r.style(c.namedStyle(styleName))
				continue
			}
			key, value, ok := strings.Cut(line, " ")
			if ok {
				value = strings.ReplaceAll(value, "poeft-sounds-v2/", c.SoundFolder+"/")
				r.set(key, value)
			}
		}
		if !c.legacyEquipmentSelections && len(rarityTextStyles[canonicalStyleName(styleName)]) > 0 {
			for _, line := range r.lines {
				for _, rarity := range []string{"Normal", "Magic", "Rare"} {
					if line == "Rarity == "+rarity {
						r.set("SetTextColor", rgb(Text(c.sharedRarityText(rarity))))
					}
				}
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
	if name == "rareItems" {
		if _, exists := base["footprintMinAreaLevel"]; !exists {
			base = Clone(base)
			base["footprintMinAreaLevel"] = Val(Obj(options["early"]), "earlyMaxAreaLevel", Obj(c.Defaults["early"])["earlyMaxAreaLevel"])
		}
	}
	if !c.legacyEquipmentSelections {
		for _, section := range globalRuleSections {
			if name == section {
				override = Clone(override)
				delete(override, "rules")
				out := Merge(base, override)
				if name == "magicItems" {
					if _, explicit := override["maxAreaLevel"]; explicit {
						delete(out, "smallMaxAreaLevel")
						delete(out, "bigMaxAreaLevel")
					}
				}
				return out
			}
		}
	}
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
	if !c.legacyEquipmentSelections {
		config = Clone(config)
		DiscardFilterRuleOverrides(config)
	}
	if err := Validate(config); err != nil {
		return "", err
	}
	p := Obj(config["buildProfile"])
	o := Obj(config["buildSpecificOptions"])
	sections := []string{c.fixed("currency"), c.fixed("scrolls"), c.gems(Obj(o["gemCallouts"])), c.links(p, o), c.highlights(Obj(o["highlightedEquipment"])), c.phases(List(p["weaponPhases"])), c.fixed("sixSockets"), c.jewellery(c.section(o, "jewellery")), c.early(p, c.section(o, "early")), c.earlySockets(c.section(o, "early")), c.flasks(c.section(o, "flasks")), c.tinctures(c.section(o, "tinctures")), c.fixed("chromaticItems"), c.fixed("uniques"), c.equipment("Rare", c.section(o, "rareItems")), c.equipment("Magic", c.section(o, "magicItems")), c.whetstone(p), c.normal(c.section(o, "normalItems")), c.fixed("questItems"), c.fixed("divinationCards"), c.fixed("misc"), c.fixed("twilightStrand"), c.fixed("hideEquipment"), c.fixed("showUnknownItems")}
	if !c.legacyCurrency {
		currency := mergeCurrencySettings(Obj(c.Defaults["currency"]), Obj(o["currency"]))
		sections[0] = c.leagueMechanic(Obj(o["leagueMechanic"])) + "\n\n" + c.currency(currency)
		sections[1] = c.scrolls(Obj(currency["scrolls"]))
	}
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
		r := newRule().quoted("BaseType", true, gem).icon("Cyan", "Star", 2).style(Obj(c.Styles["gemHighlight"]))
		rules = append(rules, c.tts(r, strings.TrimSuffix(gem, " Support")))
	}
	for _, gem := range unique(Strings(cfg["styleOnly"])) {
		if contains(sounding, gem) {
			continue
		}
		style := "gemHighlight"
		if c.legacyEquipmentSelections {
			style = "gem" // Frozen web-output tests only.
		}
		rules = append(rules, newRule().quoted("BaseType", true, gem).style(Obj(c.Styles[style])))
	}
	for _, gem := range c.starterGems {
		if contains(sounding, gem) || contains(Strings(cfg["styleOnly"]), gem) {
			continue
		}
		rules = append(rules, newRule().quoted("BaseType", true, gem).style(Obj(c.Styles["gemHighlight"])))
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
	if !c.legacyEquipmentSelections {
		rules = c.flaskProgression(cfg)
	}
	// Preserve the canonical utility-flask order, independent of picker order.
	for _, name := range []string{"Quicksilver", "Bismuth", "Stibnite", "Amethyst", "Ruby", "Sapphire", "Topaz", "Silver", "Aquamarine", "Granite", "Jade", "Quartz", "Diamond", "Gold", "Corundum", "Iron"} {
		if contains(Strings(cfg["utilityFlasks"]), name) {
			for _, r := range c.template(Text(Obj(c.templates["utilityFlasks"])[name])) {
				rules = append(rules, c.rarityTextRules(r, "Normal", "Magic")...)
			}
		}
	}
	if !c.legacyEquipmentSelections {
		for _, name := range []string{"Sulphur", "Basalt"} {
			if contains(Strings(cfg["utilityFlasks"]), name) {
				r := c.tts(newRule().quoted("Class", true, "Utility Flasks").quoted("BaseType", true, name+" Flask").icon("Green", "Raindrop", 2).style(Obj(c.Styles["utilityFlask"])), strings.ToLower(name)+"_flask")
				rules = append(rules, c.rarityTextRules(r, "Normal", "Magic")...)
			}
		}
		last = newRule().quoted("Class", true, "Utility Flasks").hide()
	}
	return heading("Flasks", append(rules, last)...)
}
func (c *Compiler) tinctures(cfg Object) string {
	rules := []*rule{}
	for _, base := range Strings(cfg["baseTypes"]) {
		r := newRule().quoted("BaseType", true, base).icon("Red", "Raindrop", 2).style(Obj(c.Styles["tincture"]))
		if c.legacyEquipmentSelections {
			r.set("PlayAlertSound", "6 240")
		} else {
			c.tts(r, strings.ReplaceAll(strings.ToLower(base), " ", "_"))
		}
		rules = append(rules, c.rarityTextRules(r, "Normal", "Magic")...)
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
		r := newRule().quoted("Class", false, "Shields").number("AreaLevel", "<=", max).style(Obj(c.Styles["earlyShieldBase"]))
		rules = append(rules, c.rarityTextRules(r, "Normal", "Magic", "Rare")...)
	}
	if mode == "full" {
		for _, kind := range Strings(Val(p, "preferredArmour", c.Defaults["preferredArmour"])) {
			style := Obj(c.Styles["rare"])
			if c.legacyEquipmentSelections {
				style = Obj(c.Styles["rareArmour"])
			} else {
				style = c.rarityStyle(style, "Rare")
			}
			rules = append(rules, defence(newRule().quoted("Class", false, "Shields").number("AreaLevel", ">", max).set("Rarity", "== Rare").style(style), kind))
		}
	}
	if Val(Obj(p["misc"]), "showRusticSash", Obj(c.Defaults["misc"])["showRusticSash"]) == true {
		r := c.tts(newRule().quoted("BaseType", false, "Rustic").quoted("Class", false, "Belts").set("AreaLevel", "<= 12").icon("White", "Pentagon", 2).style(Obj(c.Styles["jewellery"])), "rustic_sash")
		rules = append(rules, c.rarityTextRules(r, "Normal", "Magic", "Rare")...)
	}
	return heading("Early", rules...)
}
func (c *Compiler) earlySockets(cfg Object) string {
	rules := []*rule{}
	for i, count := range []int{2, 3} {
		key := fmt.Sprintf("armour%d", count)
		rc := c.ruleConfig(cfg, key)
		if rc["enabled"] == false && c.legacyEquipmentSelections {
			continue
		}
		cut := cfg["twoSocketMaxAreaLevel"]
		if count == 3 {
			cut = Val(cfg, "threeSocketMaxAreaLevel", cfg["earlyMaxAreaLevel"])
		}
		base := func() *rule {
			return newRule().number("Sockets", "==", count).quoted("Class", false, armour...).number("AreaLevel", "<=", Val(rc, "maxAreaLevel", cut))
		}
		if c.legacyEquipmentSelections {
			rules = append(rules, c.presentation(base(), rc, Object{"size": float64(40 + i*5)}))
			continue
		}
		for _, rarity := range []string{"Normal", "Magic", "Rare"} {
			rules = append(rules, c.presentation(base().set("Rarity", "== "+rarity), rc, c.rarityStyle(Obj(c.Styles[strings.ToLower(rarity)]), rarity)))
		}
		// Keep the existing catch-all for unique items after the rarity variants.
		rules = append(rules, c.presentation(base(), rc, Object{}))
	}
	return heading("Early Sockets", rules...)
}
func (c *Compiler) equipment(rarity string, cfg Object) string {
	rules := []*rule{}
	for _, shape := range []struct {
		key        string
		w, h, size int
	}{{"1x1", 1, 1, 45}, {"1x2", 1, 2, 43}, {"2x1", 2, 1, 43}, {"1x3", 1, 3, 40}, {"1x4", 1, 4, 37}, {"2x2", 2, 2, 37}, {"2x3", 2, 3, 33}, {"2x4", 2, 4, 30}} {
		rc := c.ruleConfig(cfg, shape.key)
		r := newRule().set("Rarity", "== "+rarity).number("Width", "==", shape.w).number("Height", "==", shape.h)
		var maximum any
		var color string
		if rarity == "Rare" {
			maximum = cfg["maxAreaLevel"]
			color = "#FFFF77"
		} else {
			maximum = Val(cfg, "bigMaxAreaLevel", cfg["maxAreaLevel"])
			if shape.h <= 2 {
				maximum = Val(cfg, "smallMaxAreaLevel", cfg["maxAreaLevel"])
			}
			color = "#8888FF"
		}
		maximum = Val(rc, "maxAreaLevel", maximum)
		// Imported filters can still carry the retired per-filter split. Those
		// explicit cutoffs take precedence over the new shared size defaults.
		if rarity == "Magic" && !c.legacyEquipmentSelections {
			key := "bigMaxAreaLevel"
			if shape.h <= 2 {
				key = "smallMaxAreaLevel"
			}
			maximum = Val(cfg, key, maximum)
		}
		if rarity == "Rare" && cfg["maxAreaLevel"] != nil && Num(maximum, 1e9) > Num(cfg["maxAreaLevel"], 1e9) {
			maximum = cfg["maxAreaLevel"]
		}
		r.number("AreaLevel", "<=", maximum)
		if rc["enabled"] == false {
			r.hide()
		} else {
			style := Object{"size": float64(shape.size), "text": color}
			if !c.legacyEquipmentSelections {
				style = Merge(style, Obj(c.Styles[strings.ToLower(rarity)]))
				var minimum any = float64(0)
				if rarity == "Rare" {
					minimum = Obj(c.Defaults["early"])["earlyMaxAreaLevel"]
				}
				minimum = Val(cfg, "footprintMinAreaLevel", minimum)
				if minimum == nil || Num(minimum, 0) > 0 {
					early := &rule{lines: append([]string{}, r.lines...)}
					early.number("AreaLevel", "<", minimum).style(Obj(c.Styles[strings.ToLower(rarity)]))
					rules = append(rules, early)
					if minimum == nil || (maximum != nil && Num(maximum, 0) < Num(minimum, 0)) {
						continue
					}
					r.number("AreaLevel", ">=", minimum)
				}
			}
			c.presentation(r, rc, style)
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
		rc := c.ruleConfig(cfg, key)
		if rc["enabled"] == false && c.legacyEquipmentSelections {
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
			if !c.legacyEquipmentSelections {
				style = c.rarityStyle(style, "Normal")
			}
		}
		if !c.legacyEquipmentSelections && key == "equipment" {
			style = Clone(Obj(c.Styles["normal"]))
			if style["size"] == nil {
				style["size"] = float64(40)
			}
		}
		r.number("AreaLevel", "<=", Val(rc, "maxAreaLevel", cfg["maxAreaLevel"]))
		if rc["enabled"] == false {
			rules = append(rules, r.hide())
		} else {
			rules = append(rules, c.presentation(r, rc, style))
		}
	}
	return heading("Normal Items", rules...)
}
