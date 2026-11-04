package compiler

import (
	"regexp"
	"strings"
	"testing"

	"github.com/mositho/filtertool-desktop/internal/data"
)

func TestStarterSupportsAlwaysGetStyleAndAllowSoundOverride(t *testing.T) {
	c := New(nil, Object{"gemHighlight": Object{"text": "#123456"}}, "")
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
			if strings.Contains(block, match) && !strings.Contains(block, "SetTextColor 18 52 86") {
				t.Fatalf("starter %s should use gemHighlight: %s", gem, block)
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

func TestUnselectedGemsUseFullDefaultStyle(t *testing.T) {
	c := New(nil, Object{
		"gem":          Object{"text": "#123456", "size": float64(42)},
		"gemHighlight": Object{"text": "#abcdef"},
	}, "")
	out := c.gems(Object{})
	reference := data.Load[struct {
		ItemClasses []string `json:"itemClasses"`
	}]("reference.json")
	quoted := regexp.MustCompile(`"([^"]+)"`)
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "Class ") {
			continue
		}
		for _, match := range quoted.FindAllStringSubmatch(line, -1) {
			if !contains(reference.ItemClasses, match[1]) {
				t.Fatalf("compiled gem class %q is not in reference data", match[1])
			}
		}
	}
	for _, block := range strings.Split(out, "\n\n") {
		if !strings.Contains(block, "Class ") {
			continue
		}
		if !strings.Contains(block, "SetTextColor 18 52 86") || !strings.Contains(block, "SetFontSize 42") {
			t.Fatal("generic gem rule must use the full default gem style and actual gem classes", block)
		}
	}
	if !strings.Contains(out, "Show\nClass == \"Skill Gems\" \"Support Gems\"\nSetTextColor 18 52 86") {
		t.Fatal("missing catch-all for active and support gems", out)
	}
	if strings.Contains(out, `BaseType == "Heavy Strike"`) {
		t.Fatal("unselected starter active gems must fall through to the default gem style")
	}
}

func TestSelectedGemsUseHighlightStyleWithSoundPriority(t *testing.T) {
	c := New(nil, Object{"gemHighlight": Object{"text": "#abcdef", "size": float64(43)}}, "")
	for _, gem := range []string{"Fireball", "Added Cold Damage Support", "Momentum Support"} {
		for _, sound := range []bool{false, true} {
			cfg := Object{"styleOnly": []any{gem, gem}}
			if sound {
				cfg["gems"] = []any{gem}
			}
			out := c.gems(cfg)
			match := `BaseType == "` + gem + `"`
			if strings.Count(out, match) != 1 {
				t.Fatalf("%s must have exactly one selected rule", gem)
			}
			for _, block := range strings.Split(out, "\n\n") {
				if !strings.Contains(block, match) {
					continue
				}
				if !strings.Contains(block, "SetTextColor 171 205 239") || !strings.Contains(block, "SetFontSize 43") {
					t.Fatal("selected gem must use gemHighlight", block)
				}
				if strings.Contains(block, "CustomAlertSound") != sound || strings.Contains(block, "MinimapIcon") != sound {
					t.Fatal("sound selection must take priority over style-only selection", block)
				}
			}
		}
	}
}
