package compiler

import (
	"strings"
	"testing"
)

func TestMinApsNarrowsSelectedTargets(t *testing.T) {
	c := New(nil, nil, "")
	c.weapons = []Weapon{
		{BaseType: "Slow Axe", ItemClass: "One Hand Axes", DropLevel: 1, APS: 1.2},
		{BaseType: "Fast Axe", ItemClass: "One Hand Axes", DropLevel: 5, APS: 1.5},
		{BaseType: "Other Axe", ItemClass: "One Hand Axes", DropLevel: 10, APS: 1.6},
		{BaseType: "Fast Sword", ItemClass: "One Hand Swords", DropLevel: 5, APS: 1.5},
		{BaseType: "Fast Claw", ItemClass: "Claws", DropLevel: 5, APS: 1.6},
	}
	for _, cutoff := range []bool{false, true} {
		for _, tt := range []struct {
			name    string
			bases   []any
			classes []any
			want    []string
		}{
			{name: "individual bases", bases: []any{"Slow Axe", "Fast Axe"}, want: []string{"Fast Axe"}},
			{name: "class", classes: []any{"One Hand Axes"}, want: []string{"Fast Axe", "Other Axe"}},
			{name: "mixed selection", bases: []any{"Slow Axe", "Fast Sword"}, classes: []any{"One Hand Axes"}, want: []string{"Fast Axe", "Other Axe", "Fast Sword"}},
			{name: "no qualifying bases", bases: []any{"Slow Axe"}},
			{name: "nonweapon selection", bases: []any{"Iron Hat"}, classes: []any{"Boots"}, want: []string{"Iron Hat", "Boots"}},
		} {
			name := tt.name
			if cutoff {
				name += " with cutoff"
			}
			t.Run(name, func(t *testing.T) {
				cfg := Object{"baseTypes": tt.bases, "itemClasses": tt.classes, "minAps": float64(1.5), "modules": []any{"minAps"}}
				if cutoff {
					cfg["modules"] = []any{"minAps", "weaponCutoff"}
				}
				output := c.phases([]any{cfg})
				for _, target := range []string{"Slow Axe", "Fast Axe", "Other Axe", "Fast Sword", "Fast Claw", "Iron Hat", "Boots"} {
					if got, want := strings.Contains(output, `"`+target+`"`), contains(tt.want, target); got != want {
						t.Fatalf("target %q: present = %v, want %v\n%s", target, got, want, output)
					}
				}
				if len(tt.want) == 0 && output != "" {
					t.Fatalf("no qualifying targets should emit no rules: %s", output)
				}
				if cutoff && len(tt.want) > 0 && tt.name != "nonweapon selection" && !strings.Contains(output, "AreaLevel <= 15") {
					t.Fatalf("weapon cutoff was not retained: %s", output)
				}
			})
		}
	}
}
