package compiler

var globalRuleSections = []string{"early", "links", "normalItems", "magicItems", "rareItems"}

// Built-in rule tuning is global and has exactly two supported values.
// Obsolete presentation/visibility choices are discarded, never translated.
func CleanRuleDefaults(defaults Object) Object {
	out := Clone(defaults)
	normalizeMagicCutoffs(out)
	discardFixedCurrencyVisibility(out)
	for _, section := range globalRuleSections {
		cfg := Obj(out[section])
		if _, exists := cfg["rules"]; !exists {
			continue
		}
		rules := Object{}
		for key, raw := range Obj(cfg["rules"]) {
			entry := Object{}
			for _, field := range []string{"maxAreaLevel", "size"} {
				if value, exists := Obj(raw)[field]; exists {
					entry[field] = value
				}
			}
			if len(entry) > 0 {
				rules[key] = entry
			}
		}
		cfg["rules"] = rules
	}
	return out
}

// Old shared small/big cutoffs become the common default and per-size tuning.
// Existing explicit advanced maxima retain their precedence.
func normalizeMagicCutoffs(defaults Object) {
	cfg := Obj(defaults["magicItems"])
	if _, explicit := cfg["maxAreaLevel"]; !explicit {
		if small, exists := cfg["smallMaxAreaLevel"]; exists {
			cfg["maxAreaLevel"] = small
		}
		if big, exists := cfg["bigMaxAreaLevel"]; exists {
			rules := Clone(Obj(cfg["rules"]))
			for _, key := range []string{"1x3", "1x4", "2x3", "2x4"} {
				entry := Clone(Obj(rules[key]))
				if _, explicit := entry["maxAreaLevel"]; !explicit {
					entry["maxAreaLevel"] = big
				}
				rules[key] = entry
			}
			cfg["rules"] = rules
		}
	}
	delete(cfg, "smallMaxAreaLevel")
	delete(cfg, "bigMaxAreaLevel")
}

func DiscardFilterRuleOverrides(config Object) {
	for _, root := range []string{"buildSpecificOptions", "localDefaults"} {
		discardFixedCurrencyVisibility(Obj(config[root]))
		for _, section := range globalRuleSections {
			delete(Obj(Obj(config[root])[section]), "rules")
		}
	}
}

func discardFixedCurrencyVisibility(settings Object) {
	currency := Obj(settings["currency"])
	delete(Obj(currency["gold"]), "visibility")
	for _, key := range []string{"wisdom", "portal"} {
		delete(Obj(Obj(currency["scrolls"])[key]), "visibility")
	}
}

func (c *Compiler) ruleConfig(cfg Object, key string) Object {
	raw := Obj(Obj(cfg["rules"])[key])
	if c.legacyEquipmentSelections {
		return raw
	} // Frozen web-output tests only.
	out := Object{}
	if maximum, exists := raw["maxAreaLevel"]; exists {
		out["maxAreaLevel"] = maximum
	}
	if size, exists := raw["size"]; exists {
		out["style"] = Object{"size": size}
	}
	return out
}
