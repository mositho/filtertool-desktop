package compiler

import (
	"strings"
	"testing"

	"github.com/mositho/filtertool-desktop/internal/data"
)

func compileCurrencyTest(t *testing.T, options Object) string {
	t.Helper()
	out, err := New(nil, nil, "").Compile(Object{"buildProfile": Object{}, "buildSpecificOptions": options})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func currencyBlocks(output, name string) []string {
	result := []string{}
	for _, block := range strings.Split(output, "\n\n") {
		for _, match := range quotedCurrencyName.FindAllStringSubmatch(block, -1) {
			if match[1] == name && strings.Contains(block, "BaseType == ") {
				result = append(result, block)
				break
			}
		}
	}
	return result
}

func TestKnownCurrencyCoverageAndOwnership(t *testing.T) {
	out := compileCurrencyTest(t, Object{})
	ref := data.Load[struct{ BaseTypesByClass map[string][]string }]("reference.json")
	for _, name := range append(ref.BaseTypesByClass["Stackable Currency"], ref.BaseTypesByClass["Delve Stackable Socketable Currency"]...) {
		blocks := currencyBlocks(out, name)
		want := 1
		if name == "Scroll of Wisdom" || name == "Portal Scroll" {
			want = 2
		}
		if len(blocks) != want {
			t.Errorf("%s: got %d rules, want %d", name, len(blocks), want)
		}
	}
	for _, def := range leagueRules {
		for _, name := range def.BaseTypes {
			if len(currencyBlocks(out, name)) != 1 {
				t.Errorf("league item %s must have exactly one owner", name)
			}
			if _, ok := currencyCatalog()[name]; ok {
				t.Errorf("league item %s also appears in currency", name)
			}
		}
	}
	if !strings.HasPrefix(out, "### League Mechanic") || !strings.HasSuffix(out, New(nil, nil, "").fixed("showUnknownItems")) {
		t.Fatal("league priority or unknown fallback changed")
	}
}

func TestCurrencyMoveAndHide(t *testing.T) {
	out := compileCurrencyTest(t, Object{"currency": Object{
		"assignments": Object{"Scroll Fragment": "s", "Chaos Orb": "low"},
		"tiers":       Object{"s": Object{"style": Object{"text": "#123456"}}, "low": Object{"visibility": "Hide"}},
		"items":       Object{"Divine Orb": Object{"visibility": "Hide"}, "Alteration Shard": Object{"visibility": "Show"}},
	}})
	fragment := currencyBlocks(out, "Scroll Fragment")
	if len(fragment) != 1 || !strings.Contains(fragment[0], "SetTextColor 18 52 86") {
		t.Fatal("move did not apply destination styling exactly once")
	}
	for _, name := range []string{"Chaos Orb", "Divine Orb"} {
		blocks := currencyBlocks(out, name)
		if len(blocks) != 1 || !strings.HasPrefix(blocks[0], "Hide\n") || strings.Contains(blocks[0], "AlertSound") || strings.Contains(blocks[0], "MinimapIcon") {
			t.Fatalf("%s not explicitly hidden without callouts", name)
		}
	}
	if !strings.HasPrefix(currencyBlocks(out, "Alteration Shard")[0], "Show\n") {
		t.Fatal("individual visibility did not override tier")
	}
}

func TestCurrencyCalloutsAndOverrides(t *testing.T) {
	c := New(nil, nil, "custom-folder")
	out := c.currency(Object{})
	for name, lines := range c.currencyCallouts() {
		if _, ok := currencyCatalog()[name]; !ok {
			continue
		}
		block := currencyBlocks(out, name)
		if len(block) != 1 {
			t.Fatalf("missing %s", name)
		}
		for _, line := range lines {
			if !strings.Contains(block[0], line) {
				t.Errorf("%s lost %s", name, line)
			}
		}
	}
	out = c.currency(Object{"items": Object{"Chaos Orb": Object{"icon": Object{"enabled": false}, "sound": Object{"enabled": false}}}})
	if block := currencyBlocks(out, "Chaos Orb")[0]; strings.Contains(block, "AlertSound") || strings.Contains(block, "MinimapIcon") {
		t.Fatal("callout disable did not remove inherited presentation")
	}
	out = c.currency(Object{"items": Object{"Chaos Orb": Object{"sound": Object{"soundId": float64(7)}}}})
	if block := currencyBlocks(out, "Chaos Orb")[0]; !strings.Contains(block, "PlayAlertSound 7 240") || strings.Contains(block, "CustomAlertSound") {
		t.Fatal("custom sound did not replace inherited sound")
	}
}

func TestLeaguestartStyleIsIndependentFromBTier(t *testing.T) {
	c := New(nil, Object{"currencyLeaguestart": Object{"text": "#123456"}}, "")
	out := c.currency(Object{})
	if !strings.Contains(currencyBlocks(out, "Primitive Chaotic Resonator")[0], "SetTextColor 18 52 86") {
		t.Fatal("Leaguestart preset not applied")
	}
	if strings.Contains(currencyBlocks(out, "Chromatic Orb")[0], "SetTextColor 18 52 86") {
		t.Fatal("Leaguestart style leaked into B Tier")
	}
}

func TestCurrencySoundDefaultsMatchCompiledCallouts(t *testing.T) {
	c := New(nil, nil, "test-sounds")
	defaults := c.CurrencySoundDefaults()
	if Obj(defaults["Chromatic Orb"])["tts"] != "chromatic_orb" {
		t.Fatal("Chromatic Orb must expose its actual TTS template")
	}
	if _, exists := defaults["Scroll Fragment"]; exists {
		t.Fatal("items without a built-in sound must be allowed to inherit a tier fallback")
	}
	callouts := c.currencyCallouts()
	for name := range currencyCatalog() {
		lines := callouts[name]
		if len(lines) == 0 && strings.Contains(name, "Essence") {
			lines = callouts["Essence"]
		}
		want := []string{}
		for _, line := range lines {
			if strings.Contains(line, "AlertSound ") {
				want = append(want, line)
			}
		}
		got := c.sound(newRule(), Obj(defaults[name])).lines
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("%s: editor sound differs from compiled callout: %v vs %v", name, got, want)
		}
	}
}

func TestGoldAndLeagueSelectionControls(t *testing.T) {
	out := compileCurrencyTest(t, Object{
		"currency":       Object{"gold": Object{"minStackSize": float64(100), "highStackSize": float64(50), "highStack": Object{"style": Object{"text": "#123456"}}}},
		"leagueMechanic": Object{"style": Object{"text": "#654321"}, "rules": Object{"viscera": Object{"baseTypes": []any{"Scorched Arm"}}}},
	})
	if !strings.Contains(out, "StackSize >= 100") || !strings.Contains(out, "Hide\nClass == \"Gold\"") {
		t.Fatal("gold minimum must explicitly hide small stacks")
	}
	if !strings.HasPrefix(currencyBlocks(out, "Severed Rhoa Head")[0], "Hide\n") {
		t.Fatal("deselected viscera not hidden")
	}
	if !strings.Contains(currencyBlocks(out, "Scorched Arm")[0], "SetTextColor 101 67 33") {
		t.Fatal("shared league style not inherited")
	}
}

func TestGoldStackStyleTiers(t *testing.T) {
	out := compileCurrencyTest(t, Object{"currency": Object{"gold": Object{"stackSize": float64(12), "highStackSize": float64(75)}}})
	gold := strings.Split(out[strings.Index(out, "# Gold"):], "\n\n")
	if len(gold) < 4 || !strings.Contains(gold[1], "StackSize >= 75") || !strings.Contains(gold[2], "StackSize >= 12") || !strings.Contains(gold[3], "SetFontSize") {
		t.Fatalf("gold tiers should order large, stack, then regular: %v", gold[:4])
	}
}

func TestScrollCutoffsAndRemovedVisibilityOverride(t *testing.T) {
	c := New(nil, Object{"wisdom": Object{"text": "#123456"}}, "")
	out, err := c.Compile(Object{"buildProfile": Object{}, "buildSpecificOptions": Object{"currency": Object{"scrolls": Object{
		"wisdom": Object{"soundCutoffLevel": float64(12), "style": Object{"size": float64(30)}, "sound": Object{"enabled": true}},
		"portal": Object{"visibility": "Hide"},
	}}}})
	if err != nil {
		t.Fatal(err)
	}
	wisdom := currencyBlocks(out, "Scroll of Wisdom")
	if len(wisdom) != 2 || !strings.Contains(wisdom[0], "AreaLevel <= 12") || !strings.Contains(wisdom[0], "CustomAlertSound") || strings.Contains(wisdom[1], "AlertSound") {
		t.Fatal("scroll cutoff or inherited sound changed")
	}
	for _, block := range wisdom {
		if !strings.Contains(block, "SetTextColor 18 52 86") || !strings.Contains(block, "SetFontSize 30") {
			t.Fatal("scroll size override did not retain the preset text colour")
		}
	}
	portal := currencyBlocks(out, "Portal Scroll")
	if len(portal) != 2 || !strings.HasPrefix(portal[0], "Show\n") || !strings.HasPrefix(portal[1], "Show\n") {
		t.Fatal("obsolete portal visibility override must be discarded")
	}
}

func TestCurrencyValidation(t *testing.T) {
	for _, currency := range []any{
		[]any{},
		Object{"assignments": Object{"Chaos Orb": "invalid"}},
		Object{"tiers": Object{"a": Object{"visibility": "Disabled"}}},
		Object{"tiers": Object{"a": "Hide"}},
		Object{"gold": Object{"minStackSize": "50"}},
		Object{"gold": Object{"highStackSize": float64(-1)}},
	} {
		if err := Validate(Object{"buildProfile": Object{}, "buildSpecificOptions": Object{"currency": currency}}); err == nil {
			t.Errorf("accepted invalid currency: %#v", currency)
		}
	}
}

func TestCurrencyItemSoundAndStackThreshold(t *testing.T) {
	out := compileCurrencyTest(t, Object{"currency": Object{
		"assignments": Object{"Chaos Orb": "s"},
		"items":       Object{"Chaos Orb": Object{"minStackSize": float64(5), "sound": Object{"soundId": float64(8)}, "icon": Object{"enabled": false}}},
	}})
	blocks := currencyBlocks(out, "Chaos Orb")
	if len(blocks) != 2 || !strings.HasPrefix(blocks[0], "Show\n") || !strings.Contains(blocks[0], "StackSize >= 5") || !strings.Contains(blocks[0], "PlayAlertSound 8 240") || strings.Contains(blocks[0], "CustomAlertSound") || strings.Contains(blocks[0], "MinimapIcon") || !strings.HasPrefix(blocks[1], "Hide\n") {
		t.Fatal("item-specific settings or stack fallback not applied after a tier move")
	}
	divine := currencyBlocks(out, "Divine Orb")[0]
	if strings.Contains(divine, "PlayAlertSound 8") || strings.Contains(divine, "StackSize") {
		t.Fatal("item settings leaked to another currency")
	}
}

func TestCurrencyItemSoundReplacesInheritedSoundChoice(t *testing.T) {
	c := New(Object{"currency": Object{
		"tiers": Object{"a": Object{"sound": Object{"tts": "Chaos Orb"}}},
		"items": Object{"Chaos Orb": Object{"sound": Object{"soundFileName": "old.mp3"}}},
	}}, nil, "")
	for _, sound := range []Object{{"soundId": float64(8)}, {"enabled": false}} {
		out, err := c.Compile(Object{"buildProfile": Object{}, "buildSpecificOptions": Object{"currency": Object{"items": Object{"Chaos Orb": Object{"sound": sound}}}}})
		if err != nil {
			t.Fatal(err)
		}
		block := currencyBlocks(out, "Chaos Orb")[0]
		if strings.Contains(block, "CustomAlertSound") {
			t.Fatal("inherited TTS/file took priority over the item's chosen sound")
		}
		if sound["enabled"] == false && strings.Contains(block, "AlertSound") {
			t.Fatal("item silence did not override inherited callout")
		}
		if sound["soundId"] != nil && !strings.Contains(block, "PlayAlertSound 8 240") {
			t.Fatal("item game sound was not applied")
		}
	}
}

func TestLeagueSelectionsShowSelectedAndHideEverythingElse(t *testing.T) {
	for _, def := range leagueRules {
		if def.Fated {
			continue // Condition-based rule, covered separately.
		}
		t.Run(def.Key, func(t *testing.T) {
			for _, selected := range [][]any{nil, {}, {def.BaseTypes[0]}} {
				settings := Object{}
				if selected != nil {
					settings["baseTypes"] = selected
				}
				out := compileCurrencyTest(t, Object{"leagueMechanic": Object{"rules": Object{def.Key: settings}}})
				for _, name := range def.BaseTypes {
					blocks := currencyBlocks(out, name)
					shown := selected == nil || len(selected) > 0 && selected[0] == name
					if len(blocks) != 1 {
						t.Fatalf("%s: expected one rule, got %d", name, len(blocks))
					}
					for _, block := range blocks {
						if shown != strings.HasPrefix(block, "Show\n") {
							t.Fatalf("%s: incorrect visibility for selection %#v", name, selected)
						}
					}
				}
			}
		})
	}
}

func TestLeagueEmptySelectionOverridesSharedDefaults(t *testing.T) {
	c := New(Object{"leagueMechanic": Object{"rules": Object{"viscera": Object{"baseTypes": []any{"Scorched Arm"}}}}}, nil, "")
	out, err := c.Compile(Object{"buildProfile": Object{}, "buildSpecificOptions": Object{"leagueMechanic": Object{"rules": Object{"viscera": Object{"baseTypes": []any{}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range leagueRules[0].BaseTypes {
		blocks := currencyBlocks(out, name)
		if len(blocks) != 1 || !strings.HasPrefix(blocks[0], "Hide\n") {
			t.Fatalf("%s fell through after deselecting everything", name)
		}
	}
}

func TestCurrencyChangesLeaveOtherSectionsUnchanged(t *testing.T) {
	modern := compileCurrencyTest(t, Object{})
	c := New(nil, nil, "")
	c.legacyCurrency = true
	legacy, err := c.Compile(Object{"buildProfile": Object{}, "buildSpecificOptions": Object{}})
	if err != nil {
		t.Fatal(err)
	}
	fromGems := func(s string) string { return s[strings.Index(s, "### Gems"):] }
	if fromGems(modern) != fromGems(legacy) {
		t.Fatal("unrelated rule order/output changed")
	}
}
