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
			if !strings.HasPrefix(block, "Show\n") || strings.Contains(block, `Class "Belts"`) {
				continue
			}
			if !strings.Contains(block, "SetTextColor "+item.color) || (!strings.Contains(block, "AreaLevel <= 27") && !strings.Contains(block, "AreaLevel <= 12")) {
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
				if strings.HasPrefix(block, "Show\n") && !strings.Contains(block, `Class "Belts"`) && !strings.Contains(block, "SetFontSize "+fmt.Sprint(size)) {
					t.Fatalf("%s equipment did not inherit style size %v: %s", rarity, size, block)
				}
			}
		}
	}
}

func TestEquipmentFootprintSizesStartAfterGlobalEarlyMaximum(t *testing.T) {
	for _, rarity := range []string{"Rare", "Magic"} {
		section := strings.ToLower(rarity) + "Items"
		for _, earlyMax := range []any{float64(12), float64(5), nil} {
			for _, visibleMax := range []float64{7, 20} {
				c := New(Object{
					"early": Object{"earlyMaxAreaLevel": earlyMax},
					section: Object{"rules": Object{"2x3": Object{"size": float64(30), "maxAreaLevel": visibleMax}}},
				}, Object{strings.ToLower(rarity): Object{"size": float64(44)}}, "")
				out := c.equipment(rarity, c.section(Object{}, section))
				for _, area := range []int{5, 6, 7, 8, 12, 13, 20, 21} {
					got := ""
					for _, block := range strings.Split(out, "\n\n") {
						if !strings.HasPrefix(block, "Show\n") || !strings.Contains(block, "Width == 2\nHeight == 3\n") {
							continue
						}
						matches := true
						for _, line := range strings.Split(block, "\n") {
							var bound int
							if _, err := fmt.Sscanf(line, "AreaLevel <= %d", &bound); err == nil && area > bound {
								matches = false
							}
							if _, err := fmt.Sscanf(line, "AreaLevel > %d", &bound); err == nil && area <= bound {
								matches = false
							}
						}
						if matches {
							got = block
							break
						}
					}
					if float64(area) > visibleMax {
						if got != "" {
							t.Fatalf("%s remained visible past cutoff: %s", rarity, got)
						}
						continue
					}
					size := 44
					if earlyMax != nil && float64(area) > earlyMax.(float64) {
						size = 30
					}
					if !strings.Contains(got, fmt.Sprintf("SetFontSize %d", size)) {
						t.Fatalf("%s early=%v visible=%v area=%d: expected size %d, got %s", rarity, earlyMax, visibleMax, area, size, got)
					}
				}
			}
		}
	}
}

func TestEarlyBeltsUseJewelleryStyleAndExplicitSize(t *testing.T) {
	for _, override := range []any{nil, float64(39)} {
		defaults := Object{}
		if override != nil {
			defaults["normalItems"] = Object{"rules": Object{"belts": Object{"size": override}}}
		}
		c := New(defaults, Object{
			"normal":    Object{"size": float64(25), "text": "#112233"},
			"jewellery": Object{"size": float64(43), "text": "#abcdef", "border": "#123456", "background": "#654321"},
		}, "")
		out := c.normal(c.section(Object{}, "normalItems"))
		size := override
		if size == nil {
			size = float64(43)
		}
		for _, block := range strings.Split(out, "\n\n") {
			if !strings.Contains(block, `Class "Belts"`) {
				continue
			}
			for _, line := range []string{"AreaLevel <= 4", "Rarity == Normal", "SetFontSize " + fmt.Sprint(size), "SetTextColor 171 205 239", "SetBorderColor 18 52 86", "SetBackgroundColor 101 67 33"} {
				if !strings.Contains(block, line) {
					t.Fatal("belt jewellery styling or cutoff missing", block)
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
