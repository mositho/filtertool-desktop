package compiler

import (
	"sort"
	"strconv"
	"strings"

	"github.com/mositho/filtertool-desktop/internal/data"
)

// Saved-selection and output compatibility: the upstream catalogue is never patched.
var baseTypeChanges = data.Load[struct {
	Rename map[string]string
	Remove []string
}]("base-type-changes.json")

var obsoleteBaseTypes = func() map[string]string {
	names := map[string]string{}
	for old, current := range baseTypeChanges.Rename {
		names[old] = current
	}
	// Some captured-soul names are reused by different souls. With no source
	// game version in saved JSON, a still-valid name must remain unchanged.
	for _, current := range baseTypeChanges.Rename {
		delete(names, current)
	}
	for _, removed := range baseTypeChanges.Remove {
		names[removed] = ""
	}
	return names
}()

func currentBaseType(name string) string {
	if current, obsolete := obsoleteBaseTypes[name]; obsolete {
		return current
	}
	return name
}

// Resolve currency keys before merging each settings layer. Explicit current
// keys win within a layer; more-specific saved settings still win across layers.
func currentCurrencyNames(cfg Object) Object {
	out := Clone(cfg)
	for _, key := range []string{"assignments", "items"} {
		values, ok := out[key].(map[string]any)
		if !ok {
			continue
		}
		names := make([]string, 0, len(values))
		for name := range values {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, old := range names {
			current := currentBaseType(old)
			if current == old {
				continue
			}
			if current != "" {
				if _, exists := values[current]; !exists {
					values[current] = values[old]
				}
			}
			delete(values, old)
		}
	}
	return out
}

// CurrentItemConfig returns a copy with only known item-selection fields updated.
// Labels, style keys, sound filenames and unknown/partial item names are untouched.
func CurrentItemConfig(cfg Object) Object {
	out := Clone(cfg)
	currentItemTargets(Obj(out["buildProfile"]), "weaponPhases")
	currentItemOptions(Obj(out["buildSpecificOptions"]))
	if defaults, ok := out["localDefaults"].(map[string]any); ok {
		out["localDefaults"] = CurrentItemDefaults(defaults)
	}
	return out
}

// CurrentItemDefaults also works on sparse user overrides, preserving unset,
// null and explicit empty arrays instead of restoring inherited selections.
func CurrentItemDefaults(defaults Object) Object {
	out := Clone(defaults)
	currentItemOptions(out)
	currentItemSelection(Obj(out["weaponPhaseTemplate"]), "baseTypes")
	return out
}

func currentItemSelection(cfg Object, key string) bool {
	values, ok := cfg[key].([]any)
	if !ok {
		return false
	}
	out := []any{}
	seen := map[string]bool{}
	for _, value := range values {
		name, ok := value.(string)
		if !ok {
			out = append(out, value) // Leave malformed values for normal validation.
			continue
		}
		name = currentBaseType(name)
		if name != "" && !seen[name] {
			out = append(out, name)
			seen[name] = true
		}
	}
	cfg[key] = out
	return len(values) > 0 && len(out) == 0
}

func currentItemTargets(cfg Object, key string) {
	values, ok := cfg[key].([]any)
	if !ok {
		return
	}
	out := []any{}
	for _, value := range values {
		target := Obj(value)
		emptied := currentItemSelection(target, "baseTypes")
		if emptied && len(Strings(target["itemClasses"])) == 0 {
			continue // Do not turn retired-only targets into broad rules/invalid phases.
		}
		out = append(out, value)
	}
	cfg[key] = out
}

func currentItemOptions(options Object) {
	currentItemSelection(Obj(options["gemCallouts"]), "gems")
	currentItemSelection(Obj(options["gemCallouts"]), "styleOnly")
	currentItemSelection(Obj(options["tinctures"]), "baseTypes")
	currentItemSelection(Obj(options["flasks"]), "utilityFlasks")
	currentItemSelection(Obj(options["jewellery"]), "amulets")
	currentItemTargets(Obj(options["highlightedEquipment"]), "highlights")
	if currency, ok := options["currency"].(map[string]any); ok {
		options["currency"] = currentCurrencyNames(currency)
	}
	for _, value := range Obj(Obj(options["leagueMechanic"])["rules"]) {
		currentItemSelection(Obj(value), "baseTypes")
	}
}

// Apply once to the complete production output, including static Show/Hide
// templates and user selections. Unknown names and partial matching stay valid.
func currentBaseTypeOutput(output string) string {
	blocks := []string{}
	for _, block := range strings.Split(output, "\n\n") {
		lines := []string{}
		keep := true
		for _, line := range strings.Split(block, "\n") {
			if strings.HasPrefix(line, "BaseType ") {
				names := []string{}
				for _, match := range quotedCurrencyName.FindAllStringSubmatch(line, -1) {
					name := currentBaseType(match[1])
					if name != "" && !contains(names, strconv.Quote(name)) {
						names = append(names, strconv.Quote(name))
					}
				}
				if len(names) == 0 {
					// Removing an obsolete exclusion is harmless. An emptied positive
					// condition, however, must discard the entire rule, even a Hide.
					if strings.HasPrefix(line, "BaseType != ") {
						continue
					}
					keep = false
					break
				}
				line = line[:strings.IndexByte(line, '"')] + strings.Join(names, " ")
			}
			lines = append(lines, line)
		}
		if keep {
			blocks = append(blocks, strings.Join(lines, "\n"))
		}
	}
	return strings.Join(blocks, "\n\n")
}
