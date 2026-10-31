package compiler

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
)

type Object = map[string]any

func Obj(v any) Object {
	if m, ok := v.(map[string]any); ok && m != nil {
		return m
	}
	return Object{}
}
func List(v any) []any {
	if a, ok := v.([]any); ok {
		return a
	}
	return []any{}
}
func Strings(v any) []string {
	out := []string{}
	for _, x := range List(v) {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
func Text(v any) string { s, _ := v.(string); return s }
func Num(v any, fallback float64) float64 {
	if n, ok := v.(float64); ok {
		return n
	}
	return fallback
}
func Val(m Object, key string, fallback any) any {
	if v, ok := m[key]; ok {
		return v
	}
	return fallback
}
func Clone(m Object) Object {
	b, _ := json.Marshal(m)
	var out Object
	_ = json.Unmarshal(b, &out)
	if out == nil {
		return Object{}
	}
	return out
}
func Merge(base, override Object) Object {
	out := Clone(base)
	for k, v := range override {
		if _, ok := v.(map[string]any); ok {
			out[k] = Merge(Obj(out[k]), Obj(v))
		} else {
			out[k] = v
		}
	}
	return out
}

// Difference retains explicit nulls and empty arrays; arrays replace rather than merge.
func Difference(value, base Object) Object {
	out := Object{}
	for k, v := range value {
		if reflect.DeepEqual(v, base[k]) {
			continue
		}
		if _, ok := v.(map[string]any); ok {
			d := Difference(Obj(v), Obj(base[k]))
			if len(d) > 0 {
				out[k] = d
			}
		} else {
			out[k] = v
		}
	}
	return out
}

func Validate(config Object) error {
	b, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if len(b) > 2*1024*1024 {
		return fmt.Errorf("filter must be smaller than 2 MB")
	}
	for _, key := range []string{"buildProfile", "buildSpecificOptions"} {
		if _, ok := config[key].(map[string]any); !ok {
			return fmt.Errorf("%s must be a JSON object", key)
		}
	}
	if phases, ok := Obj(config["buildProfile"])["weaponPhases"].([]any); ok {
		for i, value := range phases {
			phase, ok := value.(map[string]any)
			if !ok {
				continue // The recursive validation below reports the malformed entry.
			}
			if len(Strings(phase["itemClasses"])) == 0 && len(Strings(phase["baseTypes"])) == 0 {
				name := strings.TrimSpace(Text(phase["name"]))
				if name == "" {
					name = fmt.Sprintf("Phase %d", i+1)
				}
				return fmt.Errorf("%s: select at least one item class or base type", name)
			}
		}
	}
	var walk func(any, string) error
	walk = func(v any, path string) error {
		switch x := v.(type) {
		case map[string]any:
			for _, key := range []string{"buildProfile", "buildSpecificOptions", "localDefaults", "localStyles", "style", "icon", "sound", "normal", "magic", "rare", "misc", "links", "jewellery", "flasks", "rareItems", "magicItems", "normalItems", "tinctures", "highlightedEquipment", "gemCallouts", "early", "rules"} {
				if value, ok := x[key]; ok && value != nil {
					if _, ok := value.(map[string]any); !ok {
						return fmt.Errorf("%s.%s must be an object", path, key)
					}
				}
			}
			for _, key := range []string{"preferredArmour", "preferredColors", "weaponPhases", "baseTypes", "itemClasses", "rarities", "amulets", "utilityFlasks", "highlights", "gems", "styleOnly", "modules"} {
				if value, ok := x[key]; ok && value != nil {
					if _, ok := value.([]any); !ok {
						return fmt.Errorf("%s.%s must be a list", path, key)
					}
				}
			}
			for _, pair := range [][2]string{{"minAreaLevel", "maxAreaLevel"}, {"minItemLevel", "maxItemLevel"}} {
				a, aok := x[pair[0]].(float64)
				b, bok := x[pair[1]].(float64)
				if aok && bok && a > b {
					return fmt.Errorf("%s: minimum cannot exceed maximum", path)
				}
			}
			for k, item := range x {
				if text, ok := item.(string); ok {
					if (k == "text" || k == "border" || k == "background") && !regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`).MatchString(text) {
						return fmt.Errorf("%s.%s must be a hex colour", path, k)
					}
					if k == "operator" && text != "==" && text != ">=" && text != "<=" {
						return fmt.Errorf("%s has an invalid size operator", path)
					}
					if (k == "tts" || k == "soundFileName") && strings.ContainsAny(text, "/\\:") {
						return fmt.Errorf("%s.%s must not contain a path", path, k)
					}
				}
				if n, ok := item.(float64); ok {
					if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 {
						return fmt.Errorf("%s.%s must be a nonnegative number", path, k)
					}
					if k != "minAps" && k != "backgroundOpacity" && math.Trunc(n) != n {
						return fmt.Errorf("%s.%s must be an integer", path, k)
					}
					if k == "backgroundOpacity" && n > 1 {
						return fmt.Errorf("%s opacity must be between 0 and 1", path)
					}
					if k == "soundId" && (n < 1 || n > 16) {
						return fmt.Errorf("%s sound must be between 1 and 16", path)
					}
				}
				if err := walk(item, path+"."+k); err != nil {
					return err
				}
			}
		case []any:
			for i, item := range x {
				if err := walk(item, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		case string:
			for _, r := range x {
				if r == '"' || r == '\n' || r == '\r' || r == 0 {
					return fmt.Errorf("%s contains an unsafe character", path)
				}
			}
		}
		return nil
	}
	return walk(config, "filter")
}
