package compiler

import (
	"strings"
	"testing"
)

func TestPerRarityDefaultsAndExplicitOverrides(t *testing.T) {
	c := New(nil, nil, "")
	// Deliberate test settings: exercise inheritance without freezing shipped icon/sound choices.
	c.rarityDefaults = Object{
		"normal": Object{"iconColor": "Grey", "iconShape": "Star", "iconSize": float64(2), "soundEnabled": false},
		"magic":  Object{"iconColor": "Blue", "iconShape": "Star", "iconSize": float64(2), "soundEnabled": false},
		"rare":   Object{"iconColor": "Yellow", "iconShape": "Star", "iconSize": float64(2), "soundId": float64(3)},
	}
	cfg := Object{"itemClasses": []any{"Boots"}, "perRarityCustomization": true}
	rules := c.highlight(cfg)
	for i, color := range []string{"Grey", "Blue", "Yellow"} {
		output := rules[i].render()
		if !strings.Contains(output, "MinimapIcon 2 "+color+" Star") {
			t.Fatalf("missing per-rarity icon: %s", output)
		}
		if strings.Contains(output, "PlayAlertSound") != (i == 2) {
			t.Fatalf("only rare should play a default sound: %s", output)
		}
	}
	if !strings.Contains(rules[2].render(), "PlayAlertSound 3 240") {
		t.Fatal("rare should use game sound 3")
	}
	cfg["normal"] = Object{"iconColor": "Red"}
	cfg["magic"] = Object{"iconShape": ""}
	cfg["rare"] = Object{"soundEnabled": false, "iconSize": float64(0)}
	rules = c.highlight(cfg)
	if !strings.Contains(rules[0].render(), "MinimapIcon 2 Red Star") || strings.Contains(rules[1].render(), "MinimapIcon") {
		t.Fatal("explicit icon overrides were ignored")
	}
	if strings.Contains(rules[2].render(), "PlayAlertSound") || !strings.Contains(rules[2].render(), "MinimapIcon 0 Yellow Star") {
		t.Fatal("explicit no-sound or zero icon size was ignored")
	}
	cfg["rare"] = Object{"tts": "My Rare", "iconShape": "Diamond"}
	if output := c.highlight(cfg)[2].render(); !strings.Contains(output, "My_Rare.mp3") || strings.Contains(output, "PlayAlertSound") || !strings.Contains(output, "Yellow Diamond") {
		t.Fatal("explicit sound/shape should replace defaults", output)
	}
	delete(cfg, "perRarityCustomization")
	for _, r := range c.highlight(cfg) {
		if output := r.render(); strings.Contains(output, "MinimapIcon") || strings.Contains(output, "AlertSound") {
			t.Fatal("per-rarity defaults applied while customization was off", output)
		}
	}
}

func TestHighlightsKeepRarityStylesUnlessStyleModuleIsAdded(t *testing.T) {
	c := New(nil, Object{
		"sharedRarityNormal":         Object{"text": "#112233"},
		"sharedRarityMagic":          Object{"text": "#445566"},
		"sharedRarityRare":           Object{"text": "#778899"},
		"highlightedEquipmentNormal": Object{"text": "#ff0000", "background": "#abcdef", "size": float64(38)},
	}, "")
	styleCheck := func(output string, colors ...string) {
		for _, color := range colors {
			if !strings.Contains(output, "SetTextColor "+color) {
				t.Fatalf("missing rarity style color %s: %s", color, output)
			}
		}
	}

	for _, cfg := range []Object{
		{"itemClasses": []any{"Boots"}},
		{"itemClasses": []any{"Boots"}, "perRarityCustomization": true},
		{"itemClasses": []any{"Boots"}, "perRarityCustomization": true, "rarities": []any{"Magic"}},
	} {
		rules := c.highlight(cfg)
		want := []string{"17 34 51", "68 85 102", "119 136 153"}
		if cfg["rarities"] != nil {
			want = []string{"68 85 102"}
		}
		if len(rules) != len(want) {
			t.Fatalf("unexpected selected rarity rules: got %d want %d", len(rules), len(want))
		}
		for i, r := range rules {
			styleCheck(r.render(), want[i])
			if strings.Contains(r.render(), "Rarity == Normal") &&
				(!strings.Contains(r.render(), "SetBackgroundColor 171 205 239") || !strings.Contains(r.render(), "SetFontSize 38")) {
				t.Fatal("highlight must retain its own non-text styling", r.render())
			}
		}
	}

	for _, makeOutput := range []func(Object) string{
		func(cfg Object) string { return c.highlights(Object{"highlights": []any{cfg}}) },
		func(cfg Object) string { return c.phases([]any{cfg}) },
	} {
		cfg := Object{"itemClasses": []any{"Test Weapons"}, "modules": []any{}, "style": Object{"text": "#123456"}}
		output := makeOutput(cfg)
		styleCheck(output, "17 34 51", "68 85 102", "119 136 153")
		if strings.Contains(output, "SetTextColor 18 52 86") {
			t.Fatalf("inactive style module overrode rarity styles: %s", output)
		}
		cfg["modules"] = []any{"style"}
		output = makeOutput(cfg)
		if !strings.Contains(output, "SetTextColor 18 52 86") {
			t.Fatalf("explicit style module did not override rarity styles: %s", output)
		}
	}
}

func TestOptionalModulesOnlyApplyWhenAdded(t *testing.T) {
	c := New(nil, nil, "")
	c.weapons = []Weapon{{BaseType: "Test Weapon", ItemClass: "Test Weapons", DropLevel: 5, APS: 1.5}}
	for _, phase := range []Object{
		{"itemClasses": []any{"Test Weapons"}},
		{"itemClasses": []any{"Test Weapons"}, "modules": []any{}},
	} {
		if output := c.phases([]any{phase}); strings.Contains(output, "AreaLevel <=") || !strings.Contains(output, `Class == "Test Weapons"`) {
			t.Fatal("absent weapon cutoff applied", output)
		}
	}
	phase := Object{"itemClasses": []any{"Test Weapons"}, "modules": []any{"weaponCutoff"}}
	if output := c.phases([]any{phase}); !strings.Contains(output, "AreaLevel <= 15") {
		t.Fatal("added weapon cutoff should use its default overlap", output)
	}
	phase["weaponCutoffOverlap"] = float64(0)
	if output := c.phases([]any{phase}); !strings.Contains(output, "AreaLevel <= 5") {
		t.Fatal("explicit zero overlap was ignored", output)
	}
	phase["modules"] = []any{}
	phase["minAreaLevel"] = float64(2)
	phase["maxAreaLevel"] = float64(30)
	phase["minSockets"] = float64(6)
	phase["minAps"] = float64(1)
	phase["iconShape"] = "Star"
	phase["soundId"] = float64(5)
	output := c.phases([]any{phase})
	if !strings.Contains(output, "AreaLevel >= 2") || !strings.Contains(output, "AreaLevel <= 30") || !strings.Contains(output, `Class == "Test Weapons"`) {
		t.Fatal("phase bounds should remain independent of optional modules", output)
	}
	if strings.Contains(output, "Sockets >=") || strings.Contains(output, "MinimapIcon") || strings.Contains(output, "AlertSound") {
		t.Fatal("unadded module fields affected phase output", output)
	}
	if strings.Contains(output, `BaseType == "Test Weapon"`) {
		t.Fatal("unadded minimum attack speed module affected phase output", output)
	}
	if output := c.highlight(phase)[0].render(); strings.Contains(output, "AreaLevel") || strings.Contains(output, `BaseType == "Test Weapon"`) {
		t.Fatal("unadded highlight modules should not affect output when metadata exists", output)
	}
	delete(phase, "modules")
	delete(phase, "minAps")
	if output := c.highlight(phase); len(output) != 3 {
		t.Fatal("explicit fields in legacy JSON should remain supported")
	}
}
