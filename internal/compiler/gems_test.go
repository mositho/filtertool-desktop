package compiler

import (
	"strings"
	"testing"
)

func TestStarterSupportsAlwaysGetStyleAndAllowSoundOverride(t *testing.T) {
	c := New(nil, nil, "")
	starters := []string{"Ruthless Support", "Chance to Bleed Support", "Momentum Support", "Chance to Poison Support", "Arcane Surge Support", "Hallow Support", "Prismatic Burst Support"}
	output := c.gems(Object{})
	for _, gem := range starters {
		match := `BaseType == "` + gem + `"`
		if strings.Count(output, match) != 1 {
			t.Fatalf("missing default rule for %s", gem)
		}
		for _, block := range strings.Split(output, "\n\n") {
			if strings.Contains(block, match) && (strings.Contains(block, "CustomAlertSound") || strings.Contains(block, "MinimapIcon")) {
				t.Fatalf("starter %s should only have style: %s", gem, block)
			}
		}
		outputWithSound := c.gems(Object{"gems": []any{gem}})
		if strings.Count(outputWithSound, match) != 1 {
			t.Fatalf("sound override should replace the style-only rule for %s", gem)
		}
		for _, block := range strings.Split(outputWithSound, "\n\n") {
			if strings.Contains(block, match) && (!strings.Contains(block, "CustomAlertSound") || !strings.Contains(block, "MinimapIcon")) {
				t.Fatalf("sound override was not applied for %s: %s", gem, block)
			}
		}
	}
	if !strings.Contains(c.gems(Object{"styleOnly": []any{}}), `BaseType == "Momentum Support"`) {
		t.Fatal("starter styling must not depend on the editable style-only selection")
	}
	if strings.Count(c.gems(Object{"styleOnly": []any{"Momentum Support"}}), `BaseType == "Momentum Support"`) != 1 {
		t.Fatal("a previously saved starter selection must not create duplicate rules")
	}
}
