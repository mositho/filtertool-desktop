package compiler

import (
	"fmt"
	"strings"
	"testing"
)

func assertRuleStyle(t *testing.T, block string, style Object) {
	t.Helper()
	for _, line := range newRule().style(style).lines {
		if !strings.Contains("\n"+block+"\n", "\n"+line+"\n") {
			t.Fatalf("rule did not inherit configured style line %q: %s", line, block)
		}
	}
}

func TestOrdinaryRarityStylesReachEquipmentWithExplicitCutoffs(t *testing.T) {
	earlyMaximum, maximum := float64(8), float64(27)
	c := New(Object{"early": Object{"earlyMaxAreaLevel": earlyMaximum}}, Object{
		"normal": Object{"text": "#112233", "size": float64(31)},
		"magic":  Object{"text": "#445566", "size": float64(34)},
		"rare":   Object{"text": "#778899", "size": float64(38)},
	}, "")
	for _, rarity := range []string{"Normal", "Magic", "Rare"} {
		cfg := Object{"maxAreaLevel": maximum, "bigMaxAreaLevel": maximum, "smallMaxAreaLevel": maximum}
		out := ""
		if rarity == "Normal" {
			out = c.normal(cfg)
		} else {
			out = c.equipment(rarity, cfg)
		}
		for _, block := range strings.Split(out, "\n\n") {
			if !strings.HasPrefix(block, "Show\n") || strings.Contains(block, `Class "Belts"`) || strings.Contains(block, "UnidentifiedItemTier") {
				continue
			}
			assertRuleStyle(t, block, Obj(c.Styles[strings.ToLower(rarity)]))
			cutoff := maximum
			if !strings.Contains(block, "AreaLevel <= "+fmt.Sprint(cutoff)) {
				t.Fatal("configured cutoff not applied", block)
			}
		}
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
				if strings.HasPrefix(block, "Show\n") && !strings.Contains(block, `Class "Belts"`) && !strings.Contains(block, "UnidentifiedItemTier") && !strings.Contains(block, "SetFontSize "+fmt.Sprint(size)) {
					t.Fatalf("%s equipment did not inherit style size %v: %s", rarity, size, block)
				}
			}
		}
	}
}

func TestEquipmentFootprintSizesStartAtTheirMinimumWithoutChangingVisibility(t *testing.T) {
	raritySize, footprintSize := float64(44), float64(30)
	for _, rarity := range []string{"Rare", "Magic"} {
		section := strings.ToLower(rarity) + "Items"
		for _, minimum := range []any{float64(0), float64(12), float64(7), float64(5), nil} {
			for _, visibleMax := range []float64{7, 20} {
				c := New(Object{
					section: Object{"footprintMinAreaLevel": minimum, "maxAreaLevel": visibleMax, "rules": Object{"2x3": Object{"size": footprintSize, "maxAreaLevel": visibleMax}}},
				}, Object{strings.ToLower(rarity): Object{"size": raritySize}}, "")
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
							if _, err := fmt.Sscanf(line, "AreaLevel < %d", &bound); err == nil && area >= bound {
								matches = false
							}
							if _, err := fmt.Sscanf(line, "AreaLevel >= %d", &bound); err == nil && area < bound {
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
					size := raritySize
					if minimum != nil && float64(area) >= minimum.(float64) {
						size = footprintSize
					}
					if !strings.Contains(got, "SetFontSize "+fmt.Sprint(size)) {
						t.Fatalf("%s minimum=%v visible=%v area=%d: expected size %v, got %s", rarity, minimum, visibleMax, area, size, got)
					}
				}
			}
		}
	}
}

func TestEarlyBeltsUseJewelleryStyleAndExplicitSize(t *testing.T) {
	for _, override := range []any{nil, float64(39)} {
		maximum := float64(6)
		defaults := Object{"normalItems": Object{"maxAreaLevel": maximum, "rules": nil}}
		if override != nil {
			Obj(defaults["normalItems"])["rules"] = Object{"belts": Object{"size": override}}
		}
		c := New(defaults, Object{
			"normal":    Object{"size": float64(25), "text": "#112233"},
			"jewellery": Object{"size": float64(43), "text": "#abcdef", "border": "#123456", "background": "#654321"},
		}, "")
		out := c.normal(c.section(Object{}, "normalItems"))
		style := c.rarityStyle(Obj(c.Styles["jewellery"]), "Normal")
		if override != nil {
			style["size"] = override
		}
		found := false
		for _, block := range strings.Split(out, "\n\n") {
			if !strings.Contains(block, `Class "Belts"`) {
				continue
			}
			found = true
			assertRuleStyle(t, block, style)
			for _, line := range []string{"AreaLevel <= " + fmt.Sprint(maximum), "Rarity == Normal"} {
				if !strings.Contains(block, line) {
					t.Fatal("belt jewellery styling or cutoff missing", block)
				}
			}
			if !found {
				t.Fatal("missing early belt rule")
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
	if Obj(c.Styles["earlyShieldBase"])["border"] != "#654321" || c.namedStyle("earlyShieldLink")["border"] != "#654321" {
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

func TestEarlySocketsOnlyMatchNormalItemsAndRetainStyleAndCutoffs(t *testing.T) {
	c := New(nil, Object{
		"normal": Object{"text": "#112233", "size": float64(28)},
		"magic":  Object{"text": "#445566", "size": float64(32)},
		"rare":   Object{"text": "#778899", "size": float64(36), "border": "#abcdef"},
	}, "")
	maximums := Object{"twoSocketMaxAreaLevel": float64(7), "earlyMaxAreaLevel": float64(12)}
	out := c.earlySockets(maximums)
	for _, block := range strings.Split(out, "\n\n") {
		if strings.HasPrefix(block, "Show\n") && !strings.Contains(block, "\nRarity == Normal\n") {
			t.Fatal("early socket rule must exclude magic, rare and unique items", block)
		}
	}
	for _, sockets := range []struct{ count, cutoff string }{{"2", "twoSocketMaxAreaLevel"}, {"3", "earlyMaxAreaLevel"}} {
		found := false
		for _, block := range strings.Split(out, "\n\n") {
			if strings.Contains(block, "Sockets == "+sockets.count) && strings.Contains(block, "Rarity == Normal") {
				found = true
				style := Merge(Obj(c.Styles["normal"]), Object{"text": c.sharedRarityText("Normal")})
				assertRuleStyle(t, block, style)
				if !strings.Contains(block, "AreaLevel <= "+fmt.Sprint(maximums[sockets.cutoff])) {
					t.Fatal("socket rule did not retain its configured cutoff", block)
				}
			}
		}
		if !found {
			t.Fatal("missing normal socket rule", sockets.count)
		}
	}
}

func TestConsolidatedStylesResolveAliasesOnlyAtTheBoundary(t *testing.T) {
	for _, tc := range []struct{ old, name string }{
		{"magicJewellery", "jewellery"}, {"rareJewellery", "jewellery"},
		{"magicLifeFlask", "lifeFlask"}, {"magicManaFlask", "manaFlask"},
		{"highlightedEquipmentNormal", "highlightedEquipment"},
		{"highlightedEquipmentMagic", "highlightedEquipment"},
		{"highlightedEquipmentRare", "highlightedEquipment"},
	} {
		t.Run(tc.old, func(t *testing.T) {
			merged := MergeStyles(Object{tc.name: Object{"background": "#111111", "size": float64(31)}}, Object{
				tc.old:  Object{"background": "#222222", "border": "#123456"},
				tc.name: Object{"border": "#654321"},
			})
			style := Obj(merged[tc.name])
			if style["background"] != "#222222" || style["border"] != "#654321" || style["size"] != float64(31) {
				t.Fatal("local aliases must override base properties; explicit canonical properties win", merged)
			}
			c := New(nil, merged, "")
			if _, exists := c.Styles[tc.old]; exists {
				t.Fatal("alias must not remain an internal style", tc.old)
			}
			if c.namedStyle(tc.old)["border"] != "#654321" {
				t.Fatal("old preset references must resolve to the canonical style")
			}
		})
	}
}

func TestFootprintMinimumDefaultsAndOverrides(t *testing.T) {
	c := New(Object{"early": Object{"earlyMaxAreaLevel": float64(17)}}, nil, "")
	if got := c.section(Object{}, "magicItems")["footprintMinAreaLevel"]; got != float64(0) {
		t.Fatal("magic footprint sizes must start immediately", got)
	}
	if got := c.section(Object{}, "rareItems")["footprintMinAreaLevel"]; got != float64(17) {
		t.Fatal("rare footprint minimum must inherit Early Max Area Level", got)
	}
	options := Object{"early": Object{"earlyMaxAreaLevel": float64(23)}}
	if got := c.section(options, "rareItems")["footprintMinAreaLevel"]; got != float64(23) {
		t.Fatal("rare minimum must follow a filter's early cutoff", got)
	}
	options["rareItems"] = Object{"footprintMinAreaLevel": float64(0)}
	if got := c.section(options, "rareItems")["footprintMinAreaLevel"]; got != float64(0) {
		t.Fatal("explicit zero must override inheritance", got)
	}
	c = New(Object{"early": Object{"earlyMaxAreaLevel": float64(17)}, "rareItems": Object{"footprintMinAreaLevel": float64(9)}}, nil, "")
	if got := c.section(Object{"early": Object{"earlyMaxAreaLevel": float64(23)}}, "rareItems")["footprintMinAreaLevel"]; got != float64(9) {
		t.Fatal("explicit shared minimum must remain independent of the early cutoff", got)
	}
}
