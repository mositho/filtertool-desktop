package compiler

import (
	"fmt"
	"strings"
	"testing"
)

func TestDisabledGenericLinkExtensionsFallThroughToEquipment(t *testing.T) {
	// Explicit false must also override an enabled global default.
	c := New(Object{"links": Object{
		"genericThreeLinksEnabled": true, "genericFourLinksEnabled": true,
		"threeLinkMaxAreaLevel": float64(50), "genericThreeLinkMaxAreaLevel": float64(12), "rules": nil,
	}, "magicItems": Object{"rules": nil}, "normalItems": Object{"rules": nil}}, nil, "")
	profile := Object{"preferredColors": []any{}, "preferredArmour": []any{}}
	options := Object{
		"links":       Object{"genericThreeLinksEnabled": false, "genericFourLinksEnabled": false},
		"early":       Object{"earlyMaxAreaLevel": float64(12)},
		"magicItems":  Object{"bigMaxAreaLevel": float64(100)},
		"normalItems": Object{"maxAreaLevel": float64(100)},
	}
	links := c.links(profile, options)
	for _, block := range strings.Split(links, "\n\n") {
		if strings.HasPrefix(block, "Hide\n") {
			t.Fatal("disabled generic links must never generate a Hide rule", block)
		}
		if strings.Contains(block, "LinkedSockets == 4") {
			t.Fatal("disabled generic four-link rule was emitted", block)
		}
		if strings.Contains(block, "LinkedSockets == 3") && !strings.Contains(block, "AreaLevel <= 12") {
			t.Fatal("disabled three-link extension exceeded the separate early cutoff", block)
		}
	}
	// Rarity rules must remain available after the links section. At area level
	// 30, early three-link rules have expired and cannot intercept these items.
	out, err := c.Compile(Object{"buildProfile": profile, "buildSpecificOptions": options})
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range []string{
		"Show\nRarity == Rare\nWidth == 2\nHeight == 3",
		"Show\nRarity == Magic\nWidth == 2\nHeight == 3\nAreaLevel <= 100",
		"Show\nRarity == Normal\nClass \"Body Armours\" \"Gloves\" \"Boots\" \"Helmets\" \"Shields\"\nAreaLevel <= 100",
	} {
		if !strings.Contains(out, match) {
			t.Fatal("later equipment Show rule missing", match)
		}
	}
}

func TestGenericLinkExtensionsOnlyAddShowRules(t *testing.T) {
	c := New(Object{"links": Object{"threeLinkMaxAreaLevel": float64(50), "genericThreeLinkMaxAreaLevel": float64(50), "rules": Object{
		"3generic": Object{"maxAreaLevel": float64(33)},
		"4generic": Object{"maxAreaLevel": float64(61)},
	}}}, nil, "")
	profile := Object{"preferredColors": []any{}, "preferredArmour": []any{}}
	out := c.links(profile, Object{"links": Object{"genericThreeLinksEnabled": true, "genericFourLinksEnabled": true}})
	foundThree, foundFour := false, false
	for _, block := range strings.Split(out, "\n\n") {
		if strings.HasPrefix(block, "Hide\n") {
			t.Fatal("generic-link extension emitted a Hide rule")
		}
		if strings.Contains(block, "LinkedSockets == 3") && strings.Contains(block, "AreaLevel <= 33") {
			foundThree = true
		}
		if strings.Contains(block, "LinkedSockets == 4") && strings.Contains(block, "AreaLevel <= 61") {
			foundFour = true
		}
	}
	if !foundThree || !foundFour {
		t.Fatal("enabled generic extensions missing")
	}
}

func TestFiveAndSixLinksUseDedicatedStylesWithSharedRarityTextColors(t *testing.T) {
	defaults := New(nil, nil, "")
	defaultLinks := defaults.links(Object{}, Object{})
	fiveLink := Obj(defaults.Styles["fiveLink"])
	foundDefault := false
	for _, block := range strings.Split(defaultLinks, "\n\n") {
		if !strings.Contains(block, "LinkedSockets = 5\nRarity == Normal\n") {
			continue
		}
		foundDefault = true
		style := Merge(fiveLink, Object{"text": defaults.sharedRarityText("Normal")})
		for _, line := range newRule().style(style).lines {
			if !strings.Contains(block, line) {
				t.Fatal("default five-link rule should use the current dedicated style with shared rarity text", block)
			}
		}
	}
	if !foundDefault {
		t.Fatal("missing default normal five-link rule")
	}
	c := New(nil, Object{
		"sharedRarityNormal": Object{"text": "#123456"},
		"sharedRarityMagic":  Object{"text": "#234567"},
		"sharedRarityRare":   Object{"text": "#345678"},
		"fiveLink":           Object{"background": "#123456", "backgroundOpacity": nil, "border": "#654321", "text": "#abcdef"},
		"sixLink":            Object{"background": "#234567", "backgroundOpacity": nil, "border": "#765432", "text": "#fedcba"},
	}, "custom-sounds")
	out := c.links(Object{}, Object{})
	for _, link := range []struct {
		sockets int
		color   string
		sound   string
		bg      string
		border  string
	}{{5, "Orange", "five_link", "18 52 86", "101 67 33"}, {6, "Red", "six_link", "35 69 103", "118 84 50"}} {
		for _, rarity := range []struct{ name, text string }{{"Normal", "18 52 86"}, {"Magic", "35 69 103"}, {"Rare", "52 86 120"}} {
			want := []string{
				fmt.Sprintf("LinkedSockets = %d", link.sockets),
				"Rarity == " + rarity.name,
				"SetTextColor " + rarity.text,
				"SetBackgroundColor " + link.bg,
				"SetBorderColor " + link.border,
				"MinimapIcon 2 " + link.color + " Diamond",
				`CustomAlertSound "custom-sounds/` + link.sound + `.mp3" 300`,
			}
			found := false
			for _, block := range strings.Split(out, "\n\n") {
				matches := true
				for _, line := range want {
					if !strings.Contains(block, line) {
						matches = false
						break
					}
				}
				if matches {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("missing %d-link %s rarity rule with dedicated style, text color, icon and sound", link.sockets, rarity.name)
			}
		}
	}
}

func TestTwoThroughFourLinksOnlyChangeTextForRarity(t *testing.T) {
	styles := Object{
		"sharedRarityNormal": Object{"text": "#123456"},
		"sharedRarityMagic":  Object{"text": "#234567"},
		"sharedRarityRare":   Object{"text": "#345678"},
	}
	c := New(nil, styles, "custom-sounds")
	profile := Object{"preferredColors": []any{"RGB"}, "preferredArmour": []any{"armour"}}
	options := Object{"links": Object{"genericThreeLinksEnabled": true, "genericFourLinksEnabled": true}}
	out := c.links(profile, options)
	// Compare matching against the frozen renderer, using today's consolidated
	// styles so the intentional style cleanup does not mask matching regressions.
	for old, name := range styleAliases {
		if strings.HasPrefix(old, "good") || strings.HasPrefix(old, "selected") {
			c.Styles[old] = Clone(Obj(c.Styles[name]))
		}
	}
	c.legacyEquipmentSelections = true
	legacy := c.links(profile, options)
	blocks := strings.Split(out, "\n\n")
	checked := 0
	for _, base := range strings.Split(legacy, "\n\n") {
		if !strings.Contains(base, "LinkedSockets == ") {
			continue
		}
		// Every original rule remains as the unique-item fallback. Its three
		// preceding variants must differ only in rarity and text color.
		found := false
		for i, block := range blocks {
			if block != base || i < 3 {
				continue
			}
			for j, rarity := range []string{"Normal", "Magic", "Rare"} {
				variant := blocks[i-3+j]
				text := "SetTextColor " + rgb(Text(c.sharedRarityText(rarity)))
				if !strings.Contains(variant, "Rarity == "+rarity) || !strings.Contains(variant, text) {
					t.Fatalf("missing rarity text in rule: %s", variant)
				}
				stripText := func(s string) string {
					lines := []string{}
					for _, line := range strings.Split(s, "\n") {
						if !strings.HasPrefix(line, "SetTextColor ") && !strings.HasPrefix(line, "Rarity ") {
							lines = append(lines, line)
						}
					}
					return strings.Join(lines, "\n")
				}
				if stripText(variant) != stripText(base) {
					t.Fatalf("rarity styling changed more than text:\n%s\noriginal:\n%s", variant, base)
				}
			}
			found = true
			break
		}
		if !found {
			t.Fatalf("missing variants for original link rule: %s", base)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no two- through four-link rules checked")
	}
}
