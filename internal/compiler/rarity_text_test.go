package compiler

import (
	"strings"
	"testing"
)

func TestNamedHighlightsInheritRarityText(t *testing.T) {
	c := New(nil, Object{
		"sharedRarityNormal": Object{"text": "#ABCDEF"},
		"sharedRarityMagic":  Object{"text": "#FEDCBA"},
		"sharedRarityRare":   Object{"text": "#123456"},
		"normal":             Object{"text": "#112233", "size": float64(21)},
		"magic":              Object{"text": "#445566", "size": float64(22)},
		"rare":               Object{"text": "#778899", "size": float64(23)},
	}, "test-sounds")
	for _, key := range []string{"earlyShieldBase", "utilityFlask", "tincture"} {
		c.Styles[key] = Object{"text": "#FF0000", "background": "#123456", "border": "#654321", "size": float64(39)}
	}
	for _, tc := range []struct {
		name, output, match, sound string
		rarities                   []string
	}{
		{"shield", c.early(Object{"shieldProgression": "early"}, Object{"shieldMaxAreaLevel": float64(12)}), `Class "Shields"`, "shield", []string{"Magic", "Rare", "Normal"}},
		{"utility template", c.flasks(Object{"utilityFlasks": []any{"Quicksilver"}}), `BaseType == "Quicksilver Flask"`, "quicksilver_flask", []string{"Magic", "Normal"}},
		{"utility generated", c.flasks(Object{"utilityFlasks": []any{"Sulphur"}}), `BaseType == "Sulphur Flask"`, "sulphur_flask", []string{"Magic", "Normal"}},
		{"tincture", c.tinctures(Object{"baseTypes": []any{"Prismatic Tincture"}}), `BaseType == "Prismatic Tincture"`, "prismatic_tincture", []string{"Magic", "Normal"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			blocks := []string{}
			for _, block := range strings.Split(tc.output, "\n\n") {
				if strings.HasPrefix(block, "Show\n") && strings.Contains(block, tc.match) {
					blocks = append(blocks, block)
				}
			}
			if len(blocks) != len(tc.rarities) {
				t.Fatalf("got %d rules, want %d: %s", len(blocks), len(tc.rarities), tc.output)
			}
			for i, rarity := range tc.rarities {
				style := Object{"text": c.sharedRarityText(rarity), "background": "#123456", "border": "#654321", "size": float64(39)}
				assertRuleStyle(t, blocks[i], style)
				if rarity != "Normal" && !strings.Contains(blocks[i], "Rarity == "+rarity) {
					t.Fatal("missing rarity condition", blocks[i])
				}
				if tc.sound != "" && !strings.Contains(blocks[i], `CustomAlertSound "test-sounds/`+tc.sound+`.mp3" 300`) {
					t.Fatal("sound not preserved", blocks[i])
				}
				if tc.name == "shield" && !strings.Contains(blocks[i], "AreaLevel <= 12") {
					t.Fatal("shield cutoff not preserved", blocks[i])
				}
			}
		})
	}
}

func TestFatedItemsUseIndependentStyleWithSharedRarityText(t *testing.T) {
	c := New(nil, Object{
		"sharedRarityNormal": Object{"text": "#ABCDEF"},
		"sharedRarityMagic":  Object{"text": "#FEDCBA"},
		"sharedRarityRare":   Object{"text": "#123456"},
		"fatedItems":         Object{"background": "#102030", "border": "#405060", "size": float64(38)},
	}, "")
	for _, custom := range []Object{{}, {"background": "#203040", "size": float64(40)}} {
		cfg := Object{
			"style": Object{"background": "#FF0000", "text": "#FF0000"},
			"rules": Object{"fated": Object{"style": custom, "icon": Object{"iconColor": "Green", "iconShape": "Circle"}}},
		}
		seen := map[string]bool{}
		for _, block := range strings.Split(c.leagueMechanic(cfg), "\n\n") {
			if !strings.Contains(block, "Fated True") {
				continue
			}
			rarity := "Normal"
			for _, value := range []string{"Magic", "Rare"} {
				if strings.Contains(block, "Rarity == "+value) {
					rarity = value
				}
			}
			seen[rarity] = true
			want := c.rarityStyle(Merge(Obj(c.Styles["fatedItems"]), custom), rarity)
			assertRuleStyle(t, block, want)
			if !strings.HasPrefix(block, "Show\n") || !strings.Contains(block, "PlayAlertSound 2 240") || !strings.Contains(block, "MinimapIcon 2 Green Circle") {
				t.Fatal("Fated condition, visibility, sound or icon lost", block)
			}
		}
		if len(seen) != 3 {
			t.Fatal("missing Fated rarity variants", seen)
		}
	}
	out := c.leagueMechanic(Object{"rules": Object{"fated": Object{"visibility": "Hide"}}})
	if strings.Count(out, "Fated True") != 1 || !strings.Contains(out, "Hide\nFated True") {
		t.Fatal("hidden Fated items must retain a single unrestricted Hide rule", out)
	}
}

func TestLeagueMechanicUsesIndependentTextColor(t *testing.T) {
	c := New(nil, Object{
		"sharedRarityNormal": Object{"text": "#ABCDEF"},
		"sharedRarityMagic":  Object{"text": "#FEDCBA"},
		"sharedRarityRare":   Object{"text": "#123456"},
		"leagueMechanic":     Object{"text": "#112233"},
	}, "")
	for _, cfg := range []Object{{}, {"style": Object{"text": "#445566"}}} {
		want := "SetTextColor 17 34 51"
		if len(cfg) > 0 {
			want = "SetTextColor 68 85 102"
		}
		output := c.leagueMechanic(cfg)
		shown := 0
		for _, block := range strings.Split(output, "\n\n") {
			if !strings.HasPrefix(block, "Show\n") || strings.Contains(block, "Fated True") {
				continue
			}
			shown++
			if !strings.Contains(block, want) || strings.Contains(block, "Rarity ") {
				t.Fatal("league text must remain independent of rarity", block)
			}
		}
		if shown != len(leagueRules)-1 {
			t.Fatalf("got %d shown rules, want %d: %s", shown, len(leagueRules)-1, output)
		}
	}
}

func TestSharedPaletteReachesFlasksJewelleryAndEarlyEquipmentWithoutChangingFallbacks(t *testing.T) {
	colors := map[string]string{"Normal": "18 52 86", "Magic": "35 69 103", "Rare": "52 86 120"}
	c := New(nil, Object{
		"sharedRarityNormal": Object{"text": "#123456"},
		"sharedRarityMagic":  Object{"text": "#234567"},
		"sharedRarityRare":   Object{"text": "#345678"},
		"normal":             Object{"text": "#112233"},
		"magic":              Object{"text": "#445566"},
		"rare":               Object{"text": "#778899"},
		// Old independent named text overrides must no longer bypass the palette.
		"lifeFlask":      Object{"text": "#FF0000"},
		"magicManaFlask": Object{"text": "#FF0000"},
		"magicJewellery": Object{"text": "#FF0000"},
		"rareArmour":     Object{"text": "#FF0000"},
	}, "")
	for _, tc := range []struct{ name, output string }{
		{"flasks", heading("Flasks", c.flaskProgression(Object{})...)},
		{"jewellery", c.jewellery(c.section(Object{}, "jewellery"))},
		{"early sockets", c.earlySockets(c.section(Object{}, "early"))},
		{"early equipment", c.early(Object{"shieldProgression": "full", "misc": Object{"showRusticSash": true}}, c.section(Object{}, "early"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checked := 0
			for _, block := range strings.Split(tc.output, "\n\n") {
				if !strings.HasPrefix(block, "Show\n") {
					continue
				}
				for rarity, color := range colors {
					if strings.Contains(block, "Rarity == "+rarity) {
						checked++
						if !strings.Contains(block, "SetTextColor "+color) {
							t.Fatal("shared text not applied", block)
						}
					}
				}
			}
			if checked == 0 {
				t.Fatal("no rarity rules checked")
			}
		})
	}
	for _, rarity := range []string{"Normal", "Magic", "Rare"} {
		cfg := Object{"maxAreaLevel": float64(20), "smallMaxAreaLevel": float64(20), "bigMaxAreaLevel": float64(20)}
		output := c.equipment(rarity, cfg)
		if rarity == "Normal" {
			output = c.normal(cfg)
		}
		found := false
		for _, block := range strings.Split(output, "\n\n") {
			if strings.HasPrefix(block, "Show\n") && !strings.Contains(block, `Class "Belts"`) && !strings.Contains(block, "UnidentifiedItemTier") {
				found = true
				want := rgb(Text(Obj(c.Styles[strings.ToLower(rarity)])["text"]))
				if !strings.Contains(block, "SetTextColor "+want) || want == colors[rarity] {
					t.Fatal("fallback colour must remain independent", block)
				}
			}
		}
		if !found {
			t.Fatal("missing fallback rules", rarity)
		}
	}
}
