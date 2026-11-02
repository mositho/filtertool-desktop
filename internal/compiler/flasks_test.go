package compiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mositho/filtertool-desktop/internal/data"
)

func flaskBlocks(output, base string) []string {
	blocks := []string{}
	for _, block := range strings.Split(output, "\n\n") {
		if strings.Contains(block, `BaseType == "`+base+`"`) {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

func TestFlaskProgressionDefaultsCoverAllBasesAndPreserveCallouts(t *testing.T) {
	c := New(nil, nil, "test-sounds")
	catalog := data.Load[map[string][]Object]("flask-progression.json")
	ref := data.Load[Object]("reference.json")
	out := c.flasks(c.section(Object{}, "flasks"))
	for _, kind := range []string{"life", "mana"} {
		class := strings.ToUpper(kind[:1]) + kind[1:] + " Flasks"
		bases := Strings(Obj(ref["baseTypesByClass"])[class])
		if len(catalog[kind]) != len(bases) {
			t.Fatal("flask editor missing base types", kind)
		}
		for _, flask := range catalog[kind] {
			if actual, exists := Obj(Obj(c.Defaults["flasks"])[kind+"MaxAreaLevels"])[Text(flask["key"])]; !exists || actual != flask["maxAreaLevel"] {
				t.Fatal("bundled defaults differ from editor progression", kind, flask["key"])
			}
			base := Text(flask["name"]) + " " + strings.TrimSuffix(class, "s")
			if !contains(bases, base) || c.manifest[Text(flask["sound"])] == "" {
				t.Fatal("unknown flask or missing TTS template", base)
			}
			blocks := flaskBlocks(out, base)
			if flask["maxAreaLevel"] == float64(0) {
				if len(blocks) != 0 {
					t.Fatal("previously hidden life flask shown by default", base)
				}
				continue
			}
			if len(blocks) != 2 {
				t.Fatal("expected normal and magic rules", base)
			}
			for _, block := range blocks {
				if !strings.Contains(block, `CustomAlertSound "test-sounds/`+Text(flask["sound"])+`.mp3" 300`) {
					t.Fatal("default callout changed", base)
				}
				if flask["maxAreaLevel"] == nil {
					if strings.Contains(block, "AreaLevel") {
						t.Fatal("uncapped flask gained a cutoff", base)
					}
				} else if !strings.Contains(block, fmt.Sprint("AreaLevel <= ", flask["maxAreaLevel"])) {
					t.Fatal("default cutoff changed", base)
				}
			}
		}
	}
	if !strings.Contains(out, "Hide\nClass == \"Life Flasks\" \"Mana Flasks\"\nRarity <= Magic") {
		t.Fatal("expired progression must stop before later generic rules, leaving uniques untouched")
	}
}

func TestFlaskCutoffInheritanceNullAndZero(t *testing.T) {
	c := New(Object{"flasks": Object{"lifeMaxAreaLevels": Object{"small": float64(18)}, "manaMaxAreaLevels": Object{"eternal": float64(72)}}}, nil, "")
	for _, test := range []struct {
		name   string
		value  any
		want   string
		hidden bool
	}{
		{"override", float64(25), "AreaLevel <= 25", false},
		{"zero hides", float64(0), "", true},
		{"null removes cutoff", nil, "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := Object{"flasks": Object{"lifeMaxAreaLevels": Object{"small": test.value}}}
			cfg := c.section(options, "flasks")
			out := c.flasks(cfg)
			blocks := flaskBlocks(out, "Small Life Flask")
			if test.hidden {
				if len(blocks) != 0 {
					t.Fatal("zero must not show flask")
				}
				return
			}
			if len(blocks) != 2 {
				t.Fatal("normal and magic rules missing")
			}
			for _, block := range blocks {
				if test.want == "" && strings.Contains(block, "AreaLevel") {
					t.Fatal("null inherited a cutoff")
				}
				if test.want != "" && !strings.Contains(block, test.want) {
					t.Fatal("override ignored")
				}
			}
			if !strings.Contains(flaskBlocks(out, "Eternal Mana Flask")[0], "AreaLevel <= 72") {
				t.Fatal("other global cutoff lost")
			}
		})
	}
	if !strings.Contains(flaskBlocks(c.flasks(c.section(Object{}, "flasks")), "Small Life Flask")[0], "AreaLevel <= 18") {
		t.Fatal("global cutoff not inherited")
	}
}

func TestFlaskCutoffValidation(t *testing.T) {
	for _, value := range []any{"12", true, float64(-1), float64(1.5), []any{}} {
		cfg := Object{"buildProfile": Object{}, "buildSpecificOptions": Object{"flasks": Object{"lifeMaxAreaLevels": Object{"small": value}}}}
		if err := Validate(cfg); err == nil {
			t.Fatalf("accepted invalid flask cutoff %#v", value)
		}
	}
	for _, value := range []any{nil, float64(0), float64(20)} {
		cfg := Object{"buildProfile": Object{}, "buildSpecificOptions": Object{"flasks": Object{"manaMaxAreaLevels": Object{"small": value}}}}
		if err := Validate(cfg); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDifferenceRetainsExplicitNullWithoutAnInheritedKey(t *testing.T) {
	diff := Difference(Object{"flasks": Object{"lifeMaxAreaLevels": Object{"small": nil}}}, Object{})
	value, exists := Obj(Obj(diff["flasks"])["lifeMaxAreaLevels"])["small"]
	if !exists || value != nil {
		t.Fatal("explicit uncapped override lost when the base has no cutoff key")
	}
}
