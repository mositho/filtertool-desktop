package compiler

import (
	"fmt"
	"strings"
	"testing"
)

func TestTieredEquipmentDefaultsOverridesAndPriority(t *testing.T) {
	c := New(nil, Object{"rareTier3": Object{"size": float64(41), "text": "#123456"}}, "")
	options := Object{"rareItems": Object{"maxAreaLevel": float64(1), "tiers": Object{"2": Object{"enabled": false}, "3": Object{"style": Object{"size": float64(39)}}}},
		"magicItems": Object{"maxAreaLevel": float64(1), "tiers": Object{"5": Object{"enabled": true}}}}
	out, err := c.Compile(Object{"buildProfile": Object{}, "buildSpecificOptions": options})
	if err != nil {
		t.Fatal(err)
	}
	for _, rarity := range []string{"Rare", "Magic"} {
		section := out[strings.Index(out, "### "+rarity+" Items"):]
		section = strings.SplitN(section, "\n\n### ", 2)[0]
		for tier := 2; tier <= 5; tier++ {
			want := rarity == "Rare" && tier != 2 || rarity == "Magic" && tier == 5
			match := fmt.Sprintf("UnidentifiedItemTier == %d", tier)
			found := false
			for _, block := range strings.Split(section, "\n\n") {
				if !strings.Contains(block, match) {
					continue
				}
				found = true
				if !strings.HasPrefix(block, "Show\n") || !strings.Contains(block, "Rarity == "+rarity) || strings.Contains(block, "AreaLevel") || strings.Contains(block, "Width") || strings.Contains(block, "AlertSound") || strings.Contains(block, "MinimapIcon") {
					t.Fatal("tier highlight acquired a cutoff, hide or unsolicited callout", block)
				}
				style := Obj(c.Styles[strings.ToLower(rarity)+fmt.Sprint("Tier", tier)])
				if rarity == "Rare" && tier == 3 {
					style = Merge(style, Object{"size": float64(39)})
				}
				assertRuleStyle(t, block, style)
				if strings.Index(section, match) > strings.Index(section, "Width == ") {
					t.Fatal("tier highlight must precede generic equipment")
				}
			}
			if found != want {
				t.Fatalf("%s tier %d: highlight=%v want=%v", rarity, tier, found, want)
			}
		}
	}
	for _, heading := range []string{"### Links", "### Highlighted Equipment", "### Weapon Phases", "### Jewellery"} {
		if at := strings.Index(out, heading); at >= 0 && at > strings.Index(out, "### Rare Items") {
			t.Fatal("earlier section moved behind tier rules", heading)
		}
	}
	defaults := New(nil, nil, "")
	if strings.Contains(defaults.equipment("Magic", defaults.section(Object{}, "magicItems")), "UnidentifiedItemTier") {
		t.Fatal("magic tier highlights must default off")
	}
	if got := strings.Count(defaults.equipment("Rare", defaults.section(Object{}, "rareItems")), "UnidentifiedItemTier"); got != 4 {
		t.Fatal("rare tier highlights must default on", got)
	}
}

func TestCovensConditionsAndSoundInheritance(t *testing.T) {
	c := New(nil, nil, "")
	out := c.leagueMechanic(Object{})
	for _, block := range strings.Split(out, "\n\n")[1:] {
		if !strings.Contains(block, "PlayAlertSound 2 240") || strings.Contains(block, "CustomAlertSound") {
			t.Fatal("league default sound missing", block)
		}
	}
	if !strings.Contains(out, "Fated True") || !strings.Contains(out, `Class == "Viscera"`) || !strings.Contains(out, `BaseType == "Cursed Poppet"`) || strings.Contains(out, "Allflame") {
		t.Fatal("incorrect current league conditions", out)
	}
	// The existing sound selector's Default option is an enabled-only record,
	// not an instruction to go silent or retain a previous custom source.
	defaulted := c.leagueMechanic(Object{"sound": Object{"enabled": true}})
	if !strings.Contains(defaulted, "PlayAlertSound 2 240") {
		t.Fatal("default league sound choice went silent")
	}
	silent := c.leagueMechanic(Object{"sound": Object{"enabled": false}})
	if strings.Contains(silent, "AlertSound") {
		t.Fatal("explicit None did not silence the league")
	}
	out = c.leagueMechanic(Object{"sound": Object{"tts": "custom"}, "rules": Object{"fated": Object{"sound": Object{"soundId": float64(8)}}}})
	for _, block := range strings.Split(out, "\n\n") {
		if strings.Contains(block, "Fated True") && (!strings.Contains(block, "PlayAlertSound 8 240") || strings.Contains(block, "CustomAlertSound")) {
			t.Fatal("per-rule sound must replace the shared source atomically", block)
		}
	}
	out = c.leagueMechanic(Object{"rules": Object{"fated": Object{"visibility": "Hide"}}})
	if !strings.Contains(out, "Hide\nFated True") {
		t.Fatal("Fated visibility override ignored")
	}
	chance := currencyBlocks(compileCurrencyTest(t, Object{}), "Orb of Chance")[0]
	if currencyCatalog()["Orb of Chance"] != "a" || !strings.Contains(chance, "PlayAlertSound 1 240") || strings.Contains(chance, "CustomAlertSound") {
		t.Fatal("Chance must remain A with game sound 1", chance)
	}
	chance = currencyBlocks(compileCurrencyTest(t, Object{"currency": Object{"items": Object{"Orb of Chance": Object{"sound": Object{"tts": "custom"}}}}}), "Orb of Chance")[0]
	if !strings.Contains(chance, "custom.mp3") {
		t.Fatal("deliberate item sound override lost")
	}
}
