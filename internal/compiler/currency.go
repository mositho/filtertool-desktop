package compiler

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/mositho/filtertool-desktop/internal/data"
)

type currencyTier struct {
	Key, Title, Style   string
	BaseTypes, Contains []string
	Sound               Object
}
type leagueRule struct {
	Key, Title, ItemClass string
	BaseTypes             []string
	Stack                 bool
}

var currencyTiers = data.Load[struct{ Tiers []currencyTier }]("currency.json").Tiers
var leagueRules = data.Load[[]leagueRule]("league-mechanic.json")

// MergeDefaults preserves atomic currency/league sound choices at every layer,
// including imported local defaults, before the compiler resolves item rules.
func MergeDefaults(base, override Object) Object {
	out := Merge(base, override)
	for _, key := range []string{"currency", "leagueMechanic"} {
		if value, ok := override[key].(map[string]any); ok {
			out[key] = mergeCurrencySettings(Obj(base[key]), value)
		}
	}
	return out
}

// A sound record represents one choice. Do not retain an inherited TTS/file
// source when a more specific setting selects a different sound or silence.
func mergeCurrencySettings(base, override Object) Object {
	out := Clone(base)
	for key, value := range override {
		if object, ok := value.(map[string]any); ok {
			if key == "sound" && len(object) > 0 {
				out[key] = Clone(object)
			} else {
				out[key] = mergeCurrencySettings(Obj(out[key]), object)
			}
		} else {
			out[key] = value
		}
	}
	return out
}

func currencyCatalog() map[string]string {
	ref := data.Load[struct{ BaseTypesByClass map[string][]string }]("reference.json")
	excluded := map[string]bool{"Scroll of Wisdom": true, "Portal Scroll": true}
	for _, group := range leagueRules {
		for _, name := range group.BaseTypes {
			excluded[name] = true
		}
	}
	out := map[string]string{}
	for _, name := range append(ref.BaseTypesByClass["Stackable Currency"], ref.BaseTypesByClass["Delve Stackable Socketable Currency"]...) {
		if excluded[name] {
			continue
		}
		tier := "leaguestart"
		for _, def := range currencyTiers {
			if contains(def.BaseTypes, name) {
				tier = def.Key
			}
			for _, part := range def.Contains {
				if strings.Contains(name, part) {
					tier = def.Key
				}
			}
		}
		out[name] = tier
	}
	return out
}

var quotedCurrencyName = regexp.MustCompile(`"([^"]+)"`)

// Preserve historical item callouts, never the old broad BaseType matching.
func (c *Compiler) currencyCallouts() map[string][]string {
	out := map[string][]string{}
	for _, r := range c.template(Text(c.templates["currency"])) {
		var names, lines []string
		for _, line := range r.lines {
			if strings.HasPrefix(line, "BaseType ") {
				for _, match := range quotedCurrencyName.FindAllStringSubmatch(line, -1) {
					names = append(names, match[1])
				}
			}
			if strings.HasPrefix(line, "MinimapIcon ") || strings.HasPrefix(line, "PlayAlertSound ") || strings.HasPrefix(line, "CustomAlertSound ") {
				lines = append(lines, line)
			}
		}
		for _, name := range names {
			out[name] = lines
		}
	}
	return out
}

// CurrencySoundDefaults exposes the actual built-in choices to the editor,
// using the same legacy rules as compilation rather than a second sound list.
// User/tier overrides are resolved separately by the editor and compiler.
func (c *Compiler) CurrencySoundDefaults() Object {
	out := Object{}
	callouts := c.currencyCallouts()
	for name := range currencyCatalog() {
		var sound Object
		lines := callouts[name]
		if len(lines) == 0 && strings.Contains(name, "Essence") {
			lines = callouts["Essence"]
		}
		for _, line := range lines {
			if strings.HasPrefix(line, "PlayAlertSound ") {
				fields := strings.Fields(line)
				if len(fields) < 2 {
					continue
				}
				if id, err := strconv.Atoi(fields[1]); err == nil {
					sound = Object{"enabled": true, "soundId": float64(id)}
				}
			}
			if strings.HasPrefix(line, "CustomAlertSound ") {
				match := quotedCurrencyName.FindStringSubmatch(line)
				if len(match) < 2 {
					continue
				}
				file := match[1][strings.LastIndex(match[1], "/")+1:]
				id := strings.TrimSuffix(file, ".mp3")
				if _, exists := c.manifest[id]; exists {
					sound = Object{"enabled": true, "tts": id}
				} else {
					sound = Object{"enabled": true, "soundFileName": file}
				}
			}
		}
		if sound != nil {
			out[name] = sound
		}
	}
	return out
}

func (c *Compiler) CurrencyIconDefaults() Object {
	out := Object{}
	callouts := c.currencyCallouts()
	for name := range currencyCatalog() {
		lines := callouts[name]
		if len(lines) == 0 && strings.Contains(name, "Essence") {
			lines = callouts["Essence"]
		}
		for _, line := range lines {
			if !strings.HasPrefix(line, "MinimapIcon ") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[1] == "-1" {
				out[name] = Object{"enabled": false}
				continue
			}
			if len(fields) != 4 {
				continue
			}
			if size, err := strconv.Atoi(fields[1]); err == nil {
				out[name] = Object{"enabled": true, "iconSize": float64(size), "iconColor": fields[2], "iconShape": fields[3]}
			}
		}
	}
	return out
}

func stripLines(r *rule, prefixes ...string) {
	keep := []string{}
	for _, line := range r.lines {
		remove := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(line, prefix+" ") {
				remove = true
			}
		}
		if !remove {
			keep = append(keep, line)
		}
	}
	r.lines = keep
}

func (c *Compiler) currencyPresentation(r *rule, cfg Object, preset string) *rule {
	i, s := Obj(cfg["icon"]), Obj(cfg["sound"])
	if len(i) > 0 {
		stripLines(r, "MinimapIcon")
	}
	if s["enabled"] == false || s["soundEnabled"] == false || s["soundId"] != nil || Text(s["tts"]) != "" || Text(s["soundFileName"]) != "" {
		stripLines(r, "PlayAlertSound", "CustomAlertSound")
	}
	c.presentation(r, cfg, Obj(c.Styles[preset]))
	if cfg["visibility"] == "Hide" {
		r.hide()
	}
	return r
}

func (c *Compiler) currency(cfg Object) string {
	cfg = mergeCurrencySettings(Obj(c.Defaults["currency"]), cfg)
	catalog := currencyCatalog()
	for name, value := range Obj(cfg["assignments"]) {
		if _, ok := catalog[name]; ok {
			for _, tier := range currencyTiers {
				if value == tier.Key {
					catalog[name] = tier.Key
				}
			}
		}
	}
	names := make([]string, 0, len(catalog))
	for name := range catalog {
		names = append(names, name)
	}
	sort.Strings(names)
	callouts := c.currencyCallouts()
	builtInSounds, builtInIcons := c.CurrencySoundDefaults(), c.CurrencyIconDefaults()
	parts := []string{"### Currency", c.gold(Obj(cfg["gold"]))}
	for _, tier := range currencyTiers {
		parts = append(parts, "# "+tier.Title)
		settings := Obj(Obj(cfg["tiers"])[tier.Key])
		fallback := mergeCurrencySettings(Object{"sound": tier.Sound, "icon": Object{}}, settings)
		if fallback["sound"] == nil {
			fallback["sound"] = tier.Sound
		}
		for _, name := range names {
			if catalog[name] != tier.Key {
				continue
			}
			r := newRule().quoted("Class", true, "Stackable Currency", "Delve Stackable Socketable Currency").quoted("BaseType", true, name)
			r.lines = append(r.lines, callouts[name]...)
			if len(callouts[name]) == 0 && strings.Contains(name, "Essence") {
				r.lines = append(r.lines, callouts["Essence"]...)
			}
			// Tier settings are fallbacks, never replacements for existing item choices.
			inherited := mergeCurrencySettings(fallback, Object{})
			if sound, exists := builtInSounds[name]; exists {
				inherited["sound"] = sound
			}
			if icon, exists := builtInIcons[name]; exists {
				inherited["icon"] = icon
			}
			item := Obj(Obj(cfg["items"])[name])
			itemSettings := mergeCurrencySettings(inherited, item)
			for _, key := range []string{"sound", "icon"} {
				if value, exists := item[key]; exists && value == nil {
					itemSettings[key] = inherited[key]
				}
			}
			sound := Obj(item["sound"])
			if sound["enabled"] == true && sound["soundEnabled"] != false && sound["soundId"] == nil && Text(sound["tts"]) == "" && Text(sound["soundFileName"]) == "" {
				itemSettings["sound"] = inherited["sound"]
			}
			minimum := Num(itemSettings["minStackSize"], 0)
			if minimum > 0 && itemSettings["visibility"] != "Hide" {
				r.number("StackSize", ">=", minimum)
			}
			parts = append(parts, c.currencyPresentation(r, itemSettings, tier.Style).render())
			if minimum > 0 && itemSettings["visibility"] != "Hide" {
				parts = append(parts, newRule().quoted("Class", true, "Stackable Currency", "Delve Stackable Socketable Currency").quoted("BaseType", true, name).hide().render())
			}
		}
	}
	return strings.Join(parts, "\n\n")
}

func (c *Compiler) gold(cfg Object) string {
	base := func() *rule { return newRule().quoted("Class", true, "Gold") }
	if cfg["visibility"] == "Hide" {
		return "# Gold\n\n" + base().hide().render()
	}
	minimum := Num(cfg["minStackSize"], 0)
	high := Num(cfg["highStackSize"], 50)
	if high < minimum {
		high = minimum
	}
	rules := []*rule{
		c.currencyPresentation(base().number("StackSize", ">=", high), Obj(cfg["highStack"]), "goldHighStack"),
		c.currencyPresentation(base().number("StackSize", ">=", minimum), cfg, "gold"),
	}
	if minimum > 0 {
		rules = append(rules, base().hide())
	}
	parts := []string{"# Gold"}
	for _, r := range rules {
		parts = append(parts, r.render())
	}
	return strings.Join(parts, "\n\n")
}

func (c *Compiler) scrolls(cfg Object) string {
	cfg = mergeCurrencySettings(Obj(Obj(c.Defaults["currency"])["scrolls"]), cfg)
	rules := []*rule{}
	for _, entry := range []struct{ key, name string }{{"wisdom", "Scroll of Wisdom"}, {"portal", "Portal Scroll"}} {
		settings := Obj(cfg[entry.key])
		for _, old := range c.template(Text(c.templates["scrolls"])) {
			if !strings.Contains(strings.Join(old.lines, "\n"), fmt.Sprintf("BaseType %q", entry.name)) {
				continue
			}
			old.quoted("BaseType", true, entry.name)
			sounded := false
			for _, line := range old.lines {
				if strings.HasPrefix(line, "AreaLevel <= ") {
					sounded = true
				}
			}
			if settings["visibility"] == "Hide" {
				rules = append(rules, newRule().quoted("BaseType", true, entry.name).hide())
				break
			}
			if sounded {
				old.number("AreaLevel", "<=", Val(settings, "soundCutoffLevel", float64(38)))
			}
			presentation := Clone(settings)
			if !sounded {
				delete(presentation, "sound")
			}
			rules = append(rules, c.currencyPresentation(old, presentation, entry.key))
		}
	}
	return heading("Scrolls", rules...)
}

func (c *Compiler) leagueMechanic(cfg Object) string {
	cfg = mergeCurrencySettings(Obj(c.Defaults["leagueMechanic"]), cfg)
	rules := []*rule{}
	for _, def := range leagueRules {
		settings := Merge(Object{"style": cfg["style"]}, Obj(Obj(cfg["rules"])[def.Key]))
		selection := def.BaseTypes
		if value, exists := settings["baseTypes"]; exists {
			selection = Strings(value)
		}
		selected, hidden := []string{}, []string{}
		for _, name := range def.BaseTypes {
			if contains(selection, name) && settings["visibility"] != "Hide" {
				selected = append(selected, name)
			} else {
				hidden = append(hidden, name)
			}
		}
		base := func(names []string) *rule {
			return newRule().quoted("Class", true, def.ItemClass).quoted("BaseType", true, names...)
		}
		// Never emit a broad Show rule when the selection is explicitly empty.
		if len(selected) > 0 {
			r := base(selected)
			if def.Stack {
				r.number("StackSize", ">=", settings["minStackSize"])
			}
			rules = append(rules, c.currencyPresentation(r, settings, "leagueMechanic"))
			if def.Stack && Num(settings["minStackSize"], 0) > 0 {
				rules = append(rules, base(selected).hide())
			}
		}
		if len(hidden) > 0 {
			rules = append(rules, base(hidden).hide())
		}
	}
	return heading("League Mechanic", rules...)
}
