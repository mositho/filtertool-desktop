package compiler

import (
	"strings"
	"testing"
)

func TestPerRarityDefaultsAndExplicitOverrides(t *testing.T) {
	c := New(nil, nil, "")
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
