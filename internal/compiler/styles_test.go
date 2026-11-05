package compiler

import (
	"fmt"
	"strings"
	"testing"
)

func TestOrdinaryRarityStylesReachEquipmentWithFallbackSizesAndExplicitCutoffs(t *testing.T) {
	c := New(nil, Object{
		// Explicitly omit the size: this tests compiler fallbacks, not shipped tuning.
		"normal": Object{"text": "#112233", "size": nil},
		"magic":  Object{"text": "#445566"},
		"rare":   Object{"text": "#778899"},
	}, "")
	for _, item := range []struct{ rarity, color string }{
		{"Normal", "17 34 51"}, {"Magic", "68 85 102"}, {"Rare", "119 136 153"},
	} {
		cfg := Object{"maxAreaLevel": float64(27), "bigMaxAreaLevel": float64(27), "smallMaxAreaLevel": float64(27)}
		out := ""
		if item.rarity == "Normal" {
			out = c.normal(cfg)
		} else {
			out = c.equipment(item.rarity, cfg)
		}
		for _, block := range strings.Split(out, "\n\n") {
			if !strings.HasPrefix(block, "Show\n") {
				continue
			}
			if !strings.Contains(block, "SetTextColor "+item.color) || !strings.Contains(block, "AreaLevel <= 27") {
				t.Fatal("ordinary rarity style or cutoff not applied", block)
			}
		}
	}
	if !strings.Contains(c.normal(Object{}), "SetFontSize 40") || !strings.Contains(c.normal(Object{}), "SetFontSize 45") {
		t.Fatal("normal equipment and belts must retain their distinct fallback sizes when no style size is set")
	}
}

func TestOrdinaryRarityStyleSizesReachEquipment(t *testing.T) {
	for _, size := range []float64{28, 36, 42} {
		for _, rarity := range []string{"Normal", "Magic", "Rare"} {
			c := New(nil, Object{strings.ToLower(rarity): Object{"size": size}}, "")
			out := c.equipment(rarity, Object{})
			if rarity == "Normal" {
				out = c.normal(Object{})
			}
			for _, block := range strings.Split(out, "\n\n") {
				if strings.HasPrefix(block, "Show\n") && !strings.Contains(block, "SetFontSize "+fmt.Sprint(size)) {
					t.Fatalf("%s equipment did not inherit style size %v: %s", rarity, size, block)
				}
			}
		}
	}
}

func TestRetiredStyleOverridesRemainCompatible(t *testing.T) {
	c := New(nil, Object{
		"gemCallout":        Object{"text": "#123456"},
		"earlyShieldLink":   Object{"border": "#654321"},
		"selectedThreeLink": Object{"background": "#112233"},
	}, "")
	if !strings.Contains(c.gems(Object{"gems": []any{"Fireball"}}), "SetTextColor 18 52 86") {
		t.Fatal("legacy gemCallout override was lost after the rename")
	}
	if Obj(c.Styles["earlyShieldBase"])["border"] != "#654321" || Obj(c.Styles["earlyShieldLink"])["border"] != "#654321" {
		t.Fatal("retired shield name must resolve to the base style with its overrides")
	}
	if c.linkStyle("three", "selected") != "selectedThreeLink" || c.linkStyle("three", "good") != "threeLink" {
		t.Fatal("explicit old link overrides must stay tier-specific; other tiers use the unified style")
	}
	out := c.links(Object{"preferredColors": []any{"R"}, "preferredArmour": []any{"armour"}}, Object{})
	if !strings.Contains(out, "SetBackgroundColor 17 34 51") {
		t.Fatal("legacy selected-link appearance was lost")
	}
	if New(nil, nil, "").linkStyle("three", "selected") != "threeLink" {
		t.Fatal("new configurations must use the unified style")
	}
}

func TestEarlySocketsUseFullRarityStylesAndRetainCutoffs(t *testing.T) {
	c := New(nil, Object{
		"normal": Object{"text": "#112233", "size": float64(28)},
		"magic":  Object{"text": "#445566", "size": float64(32)},
		"rare":   Object{"text": "#778899", "size": float64(36), "border": "#abcdef"},
	}, "")
	out := c.earlySockets(Object{"twoSocketMaxAreaLevel": float64(7), "earlyMaxAreaLevel": float64(12)})
	for _, sockets := range []struct{ count, cutoff string }{{"2", "7"}, {"3", "12"}} {
		for _, rarity := range []struct{ name, color, size string }{{"Normal", "17 34 51", "28"}, {"Magic", "68 85 102", "32"}, {"Rare", "119 136 153", "36"}} {
			found := false
			for _, block := range strings.Split(out, "\n\n") {
				if strings.Contains(block, "Sockets == "+sockets.count) && strings.Contains(block, "Rarity == "+rarity.name) {
					found = true
					for _, line := range []string{"AreaLevel <= " + sockets.cutoff, "SetTextColor " + rarity.color, "SetFontSize " + rarity.size} {
						if !strings.Contains(block, line) {
							t.Fatal("socket rule did not inherit its rarity style and cutoff", block)
						}
					}
					if rarity.name == "Rare" && !strings.Contains(block, "SetBorderColor 171 205 239") {
						t.Fatal("socket rule did not inherit full rare style", block)
					}
				}
			}
			if !found {
				t.Fatal("missing socket rarity rule", sockets.count, rarity.name)
			}
		}
	}
}
