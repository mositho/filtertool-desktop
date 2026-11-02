package compiler

import (
	"fmt"
	"strings"
	"testing"
)

func TestCurrencyTierFallbacksPreserveEveryBuiltInCallout(t *testing.T) {
	c := New(nil, nil, "test-sounds")
	c.Defaults = Object{}
	tiers := Object{}
	for _, tier := range currencyTiers {
		tiers[tier.Key] = Object{"sound": Object{"enabled": true, "soundId": float64(16)}, "icon": Object{"enabled": true, "iconShape": "Square", "iconColor": "Purple", "iconSize": float64(0)}}
	}
	out := c.currency(Object{"tiers": tiers})
	for name, raw := range c.CurrencySoundDefaults() {
		choice := Obj(raw)
		want := fmt.Sprintf("PlayAlertSound %v 240", choice["soundId"])
		if choice["tts"] != nil {
			want = `CustomAlertSound "test-sounds/` + Text(choice["tts"]) + `.mp3" 300`
		}
		if choice["soundFileName"] != nil {
			want = `CustomAlertSound "test-sounds/` + Text(choice["soundFileName"]) + `" 300`
		}
		if !strings.Contains(currencyBlocks(out, name)[0], want) {
			t.Fatalf("tier fallback replaced %s's built-in sound %s", name, want)
		}
	}
	for name, raw := range c.CurrencyIconDefaults() {
		icon := Obj(raw)
		want := fmt.Sprintf("MinimapIcon %v %s %s", icon["iconSize"], Text(icon["iconColor"]), Text(icon["iconShape"]))
		if !strings.Contains(currencyBlocks(out, name)[0], want) {
			t.Fatalf("tier fallback replaced %s's built-in icon %s", name, want)
		}
	}
}

func TestCurrencyTierDefaultSoundsAndExplicitItemSilence(t *testing.T) {
	c := New(nil, nil, "test-sounds")
	c.Defaults = Object{}
	for _, tier := range currencyTiers {
		out := c.currency(Object{"assignments": Object{"Scroll Fragment": tier.Key}})
		block := currencyBlocks(out, "Scroll Fragment")[0]
		if tier.Sound["enabled"] == false {
			if strings.Contains(block, "AlertSound") {
				t.Fatal("quiet tier gained a sound", tier.Key)
			}
		} else if !strings.Contains(block, fmt.Sprintf("PlayAlertSound %v 240", tier.Sound["soundId"])) {
			t.Fatal("tier fallback missing", tier.Key, block)
		}
	}
	settings := Object{"assignments": Object{"Scroll Fragment": "s"}, "tiers": Object{"s": Object{"icon": Object{"enabled": true, "iconShape": "Star", "iconColor": "Red", "iconSize": float64(1)}}}}
	block := currencyBlocks(c.currency(settings), "Scroll Fragment")[0]
	if !strings.Contains(block, "MinimapIcon 1 Red Star") {
		t.Fatal("tier icon fallback not applied")
	}
	settings["items"] = Object{"Scroll Fragment": Object{"sound": Object{"enabled": false}, "icon": Object{"enabled": false}}, "Chromatic Orb": Object{"sound": Object{"enabled": false}, "icon": Object{"enabled": false}}}
	for _, name := range []string{"Scroll Fragment", "Chromatic Orb"} {
		block := currencyBlocks(c.currency(settings), name)[0]
		if strings.Contains(block, "AlertSound") || strings.Contains(block, "MinimapIcon") {
			t.Fatal("explicit None did not win over tier and item defaults", name)
		}
	}
}

func TestCurrencyMovesKeepItemChoicesAndUpdateOnlyFallbacks(t *testing.T) {
	c := New(nil, nil, "test-sounds")
	c.Defaults = Object{}
	for _, test := range []struct {
		tier  string
		sound float64
	}{{"b", 2}, {"s", 6}} {
		settings := Object{"assignments": Object{"Scroll Fragment": test.tier, "Chromatic Orb": test.tier}}
		out := c.currency(settings)
		if !strings.Contains(currencyBlocks(out, "Scroll Fragment")[0], fmt.Sprintf("PlayAlertSound %v 240", test.sound)) {
			t.Fatal("move did not update fallback")
		}
		if !strings.Contains(currencyBlocks(out, "Chromatic Orb")[0], "chromatic_orb.mp3") {
			t.Fatal("move lost existing TTS")
		}
		settings["items"] = Object{"Scroll Fragment": Object{"sound": Object{"soundId": float64(8)}}}
		if !strings.Contains(currencyBlocks(c.currency(settings), "Scroll Fragment")[0], "PlayAlertSound 8 240") {
			t.Fatal("move lost explicit item sound")
		}
	}
}
