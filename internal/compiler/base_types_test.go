package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/mositho/filtertool-desktop/internal/data"
)

func assertCurrentBaseTypes(t *testing.T, output string) {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "BaseType ") {
			continue
		}
		for _, match := range quotedCurrencyName.FindAllStringSubmatch(line, -1) {
			if _, obsolete := obsoleteBaseTypes[match[1]]; obsolete {
				t.Errorf("obsolete name reached output: %s", line)
			}
		}
	}
}

func TestEveryRemovedAndRenamedBaseType(t *testing.T) {
	for _, name := range baseTypeChanges.Remove {
		t.Run(name, func(t *testing.T) {
			for _, action := range []string{"Show", "Hide"} {
				input := action + "\nClass == \"Stackable Currency\"\nBaseType == " + strconv.Quote(name) + "\nSetFontSize 39"
				if got := currentBaseTypeOutput(input); got != "" {
					t.Fatal("retired-only selection became a broad rule", got)
				}
			}
		})
	}
	for old, renamed := range baseTypeChanges.Rename {
		t.Run(old, func(t *testing.T) {
			// Reused names remain valid independently of their historical identity.
			for _, target := range baseTypeChanges.Rename {
				if target == old {
					renamed = old
				}
			}
			input := "Show\nBaseType == " + strconv.Quote(old)
			want := "Show\nBaseType == " + strconv.Quote(renamed)
			if got := currentBaseTypeOutput(input); got != want || currentBaseTypeOutput(got) != got {
				t.Fatalf("rename is incorrect or cascades: %s; want %s", got, want)
			}
		})
	}
}

func TestBaseTypeCleanupPreservesMatchingAndRuleOrder(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"Show\nBaseType == \"Orb of Binding\" \"Coin of Skill\" \"Soul Core of Finesse\" \"Chaos Orb\"\nAreaLevel <= 12", "Show\nBaseType == \"Soul Core of Finesse\" \"Chaos Orb\"\nAreaLevel <= 12"},
		{"Hide\nBaseType \"Her Mask\" \"The Gambler\"\nClass \"Divination Cards\"", "Hide\nBaseType \"The Gambler\"\nClass \"Divination Cards\""},
		{"Show\nBaseType != \"Orb of Binding\"\nClass == \"Wands\"", "Show\nClass == \"Wands\""},
		{"Hide\nBaseType != \"Orb of Binding\" \"Coin of Skill\"", "Hide\nBaseType != \"Soul Core of Finesse\""},
		{"Show\nBaseType \"Mortal\" \"Future Unknown Base\"", "Show\nBaseType \"Mortal\" \"Future Unknown Base\""},
		{"### Highlights\n\nShow\nBaseType \"Orb of Binding\"\n\nHide\nBaseType \"Chaos Orb\"\n\n### Fallback\n\nShow", "### Highlights\n\nHide\nBaseType \"Chaos Orb\"\n\n### Fallback\n\nShow"},
	} {
		if got := currentBaseTypeOutput(tc.input); got != tc.want {
			t.Errorf("cleanup changed match semantics or priority:\n%s\nwant:\n%s", got, tc.want)
		}
	}
}

func TestReexportOldSelectionsAndDefaults(t *testing.T) {
	names := []any{}
	for _, name := range baseTypeChanges.Remove {
		names = append(names, name)
	}
	for name, current := range baseTypeChanges.Rename {
		names = append(names, name, current)
	}
	names = append(names, "Future Unknown Base")
	defaults := Object{
		"gemCallouts": Object{"styleOnly": names},
		"tinctures":   Object{"baseTypes": names},
		"currency":    Object{"assignments": Object{"Coin of Skill": "s"}, "items": Object{"Coin of Skill": Object{"visibility": "Hide"}}},
	}
	config := Object{
		"buildProfile": Object{"weaponPhases": []any{Object{"baseTypes": names, "minAreaLevel": float64(70)}}},
		"buildSpecificOptions": Object{
			"gemCallouts":          Object{"gems": names},
			"highlightedEquipment": Object{"highlights": []any{Object{"baseTypes": names, "maxAreaLevel": float64(69)}}},
		},
	}
	beforeConfig, _ := json.Marshal(config)
	beforeDefaults, _ := json.Marshal(defaults)
	c := New(defaults, nil, "")
	out, err := c.Compile(config)
	if err != nil {
		t.Fatal(err)
	}
	assertCurrentBaseTypes(t, out)
	for _, current := range baseTypeChanges.Rename {
		if !strings.Contains(out, strconv.Quote(current)) {
			t.Error("lost renamed or still-valid selection", current)
		}
	}
	if !strings.Contains(out, `"Future Unknown Base"`) || !strings.HasSuffix(out, c.fixed("showUnknownItems")) {
		t.Fatal("unknown selections or final fallback changed")
	}
	if at := strings.Index(out, "### Highlighted Equipment"); at < 0 || at > strings.Index(out, "### Weapon Phases") {
		t.Fatal("section priority changed")
	}
	afterConfig, _ := json.Marshal(config)
	afterDefaults, _ := json.Marshal(defaults)
	if string(beforeConfig) != string(afterConfig) || string(beforeDefaults) != string(afterDefaults) {
		t.Fatal("compilation rewrote saved selections or defaults")
	}
}

func TestRenamedCurrencyLayerPrecedence(t *testing.T) {
	old, current := "Coin of Skill", "Soul Core of Finesse"
	for _, savedKey := range []string{old, current} {
		c := New(Object{"currency": Object{
			"assignments": Object{savedKey: "s"},
			"items":       Object{savedKey: Object{"visibility": "Hide", "sound": Object{"soundId": float64(8)}}},
		}}, nil, "")
		for _, tc := range []struct {
			items         Object
			action, sound string
		}{
			{Object{}, "Hide", ""},
			{Object{old: Object{"visibility": "Show", "sound": Object{"soundId": float64(7)}}}, "Show", "PlayAlertSound 7 240"},
			{Object{old: Object{"visibility": "Hide"}, current: Object{"visibility": "Show", "sound": Object{"soundId": float64(6)}}}, "Show", "PlayAlertSound 6 240"},
		} {
			cfg := Object{"buildProfile": Object{}, "buildSpecificOptions": Object{"currency": Object{"items": tc.items}}}
			out, err := c.Compile(cfg)
			if err != nil {
				t.Fatal(err)
			}
			blocks := currencyBlocks(out, current)
			if len(blocks) != 1 || !strings.HasPrefix(blocks[0], tc.action+"\n") || !strings.Contains(blocks[0], tc.sound) {
				t.Fatal("renamed currency lost settings/precedence", blocks)
			}
			if Obj(Obj(c.Defaults["currency"])["assignments"])[current] != "s" {
				t.Fatal("renamed tier assignment lost")
			}
		}
	}
}

func TestCurrentOutputAuditForAllBuiltInAndHistoricalFilters(t *testing.T) {
	configs := data.Load[map[string]Object]("templates.json")
	configs["blank"] = Object{"buildProfile": Object{}, "buildSpecificOptions": Object{}}
	files, err := filepath.Glob("testdata/*.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var cfg Object
		if err := json.Unmarshal(b, &cfg); err != nil {
			t.Fatal(err)
		}
		configs[file] = cfg
	}
	for name, cfg := range configs {
		t.Run(name, func(t *testing.T) {
			c := New(MergeDefaults(Obj(cfg["_defaults"]), Obj(cfg["localDefaults"])), nil, "")
			out, err := c.Compile(cfg)
			if err != nil {
				t.Fatal(err)
			}
			assertCurrentBaseTypes(t, out)
			cleaned := CurrentItemConfig(cfg)
			if err := Validate(cleaned); err != nil {
				t.Fatal("cleaned historical config is invalid", err)
			}
			if !reflect.DeepEqual(cleaned, CurrentItemConfig(cleaned)) {
				t.Fatal("cleanup is not idempotent")
			}
			if !strings.HasSuffix(out, c.fixed("showUnknownItems")) {
				t.Fatal("lost final unknown-item fallback")
			}
		})
	}
}

func TestCurrentItemConfigPreservesSelectionSemantics(t *testing.T) {
	names := []any{"Orb of Binding", "Coin of Skill", "Soul Core of Finesse", "Mortal", "Future Unknown Base"}
	for name := range baseTypeChanges.Rename {
		names = append(names, name)
	}
	for _, name := range baseTypeChanges.Remove {
		names = append(names, name)
	}
	options := Object{
		"gemCallouts": Object{"gems": names, "styleOnly": []any{"Orb of Binding"}},
		"tinctures":   Object{"baseTypes": []any{}},
		"highlightedEquipment": Object{"highlights": []any{
			Object{"baseTypes": []any{"Orb of Binding"}, "minAps": float64(1)},
			Object{"baseTypes": []any{"Orb of Binding"}, "itemClasses": []any{"Wands"}},
		}},
		"leagueMechanic": Object{"rules": Object{"poppet": Object{"baseTypes": []any{"Orb of Binding"}}}},
		"currency": Object{
			"assignments": Object{"Coin of Skill": "s", "Soul Core of Finesse": "a", "Orb of Binding": "b"},
			"items":       Object{"Coin of Skill": Object{"visibility": "Hide"}, "Soul Core of Finesse": Object{"visibility": "Show"}},
		},
	}
	cfg := Object{
		"buildProfile": Object{"weaponPhases": []any{
			Object{"name": "Orb of Binding", "baseTypes": []any{"Orb of Binding"}},
			Object{"name": "Coin of Skill", "baseTypes": names, "tts": "Coin of Skill", "soundFileName": "Orb of Binding.mp3", "style": Object{"preset": "Coin of Skill"}},
		}},
		"buildSpecificOptions": options,
		"localDefaults":        options,
		"localStyles":          Object{"Coin of Skill": Object{"size": float64(40)}},
	}
	before, _ := json.Marshal(cfg)
	got := CurrentItemConfig(cfg)
	if err := Validate(got); err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(cfg)
	if string(before) != string(after) || !reflect.DeepEqual(got, CurrentItemConfig(got)) {
		t.Fatal("cleanup mutated input or cascaded")
	}
	phases := List(Obj(got["buildProfile"])["weaponPhases"])
	if len(phases) != 1 {
		t.Fatal("removed-only phase survived", phases)
	}
	phase := Obj(phases[0])
	for _, key := range []string{"name", "tts", "soundFileName", "style"} {
		if !reflect.DeepEqual(phase[key], Obj(List(Obj(cfg["buildProfile"])["weaponPhases"])[1])[key]) {
			t.Fatal("rewrote non-selection", key)
		}
	}
	for _, key := range []string{"buildSpecificOptions", "localDefaults"} {
		o := Obj(got[key])
		want := []string{}
		for _, name := range Strings(names) {
			if current := currentBaseType(name); current != "" && !contains(want, current) {
				want = append(want, current)
			}
		}
		if !reflect.DeepEqual(Strings(Obj(o["gemCallouts"])["gems"]), want) {
			t.Fatal("lost unknown/partial names, renamed incorrectly, or failed to deduplicate")
		}
		for _, selection := range []any{Obj(o["gemCallouts"])["styleOnly"], Obj(o["tinctures"])["baseTypes"], Obj(Obj(Obj(o["leagueMechanic"])["rules"])["poppet"])["baseTypes"]} {
			if !reflect.DeepEqual(selection, []any{}) {
				t.Fatal("explicit empty selection was lost", selection)
			}
		}
		highlights := List(Obj(o["highlightedEquipment"])["highlights"])
		if len(highlights) != 1 || !reflect.DeepEqual(Obj(highlights[0])["itemClasses"], []any{"Wands"}) {
			t.Fatal("lost independent class targets or retained empty highlight", highlights)
		}
		currency := Obj(o["currency"])
		if !reflect.DeepEqual(currency["assignments"], Object{"Soul Core of Finesse": "a"}) || !reflect.DeepEqual(currency["items"], Object{"Soul Core of Finesse": Object{"visibility": "Show"}}) {
			t.Fatal("current currency keys did not win", currency)
		}
	}
	if !reflect.DeepEqual(got["localStyles"], cfg["localStyles"]) {
		t.Fatal("renamed a style key")
	}
	if got := CurrentItemDefaults(Object{"tinctures": Object{"baseTypes": nil}, "weaponPhaseTemplate": Object{"baseTypes": []any{"Orb of Binding"}}}); Obj(got["tinctures"])["baseTypes"] != nil || !reflect.DeepEqual(Obj(got["weaponPhaseTemplate"])["baseTypes"], []any{}) {
		t.Fatal("lost null or empty template targets", got)
	}
}
