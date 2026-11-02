package compiler

import (
	"strings"
	"testing"

	"github.com/mositho/filtertool-desktop/internal/data"
)

func TestUtilityFlaskSelectionShowsWithTTSAndHidesTheRest(t *testing.T) {
	c := New(nil, nil, "test-sounds")
	names := []string{"Quicksilver", "Bismuth", "Stibnite", "Amethyst", "Ruby", "Sapphire", "Topaz", "Silver", "Aquamarine", "Granite", "Jade", "Quartz", "Sulphur", "Basalt", "Diamond", "Gold", "Corundum", "Iron"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			out := c.flasks(Object{"utilityFlasks": []any{name}})
			if !strings.Contains(out, `BaseType == "`+name+` Flask"`) || !strings.Contains(out, `CustomAlertSound "test-sounds/`+strings.ToLower(name)+`_flask.mp3" 300`) {
				t.Fatal("selected flask missing exact match or TTS", out)
			}
			if !strings.HasSuffix(out, "Hide\nClass == \"Utility Flasks\"") {
				t.Fatal("unselected utility flasks must be hidden", out)
			}
			if !strings.Contains(out, `Class == "Life Flasks"`) || !strings.Contains(out, `Class == "Mana Flasks"`) {
				t.Fatal("life/mana progression was removed")
			}
		})
	}
	out := c.flasks(Object{"utilityFlasks": []any{}})
	if strings.Contains(out, `Class "Utility Flasks"`) || !strings.HasSuffix(out, "Hide\nClass == \"Utility Flasks\"") {
		t.Fatal("empty selection must hide every utility flask")
	}
}

func TestTinctureSelectionUsesFullNameTTSTemplates(t *testing.T) {
	c := New(nil, nil, "test-sounds")
	ref := data.Load[Object]("reference.json")
	for _, base := range Strings(Obj(ref["baseTypesByClass"])["Tinctures"]) {
		id := strings.ReplaceAll(strings.ToLower(base), " ", "_")
		if c.manifest[id] != base {
			t.Fatalf("missing full-name TTS template for %s", base)
		}
		out := c.tinctures(Object{"baseTypes": []any{base}})
		if !strings.Contains(out, `BaseType == "`+base+`"`) || !strings.Contains(out, `CustomAlertSound "test-sounds/`+id+`.mp3" 300`) || strings.Contains(out, "PlayAlertSound") {
			t.Fatal("selected tincture must use exact matching and full-name TTS", out)
		}
		if !strings.HasSuffix(out, "Hide\nClass \"Tinctures\"") {
			t.Fatal("unselected tinctures must remain hidden")
		}
	}
	if out := c.tinctures(Object{"baseTypes": []any{}}); strings.Contains(out, "Show\n") || !strings.Contains(out, "Hide\n") {
		t.Fatal("empty tincture selection must hide all tinctures", out)
	}
}

func TestEquipmentSelectionsRespectGlobalDefaultsAndEmptyOverrides(t *testing.T) {
	c := New(Object{"flasks": Object{"utilityFlasks": []any{"Basalt"}}, "tinctures": Object{"baseTypes": []any{"Oakbranch Tincture"}}}, nil, "")
	options := Object{}
	if !strings.Contains(c.flasks(c.section(options, "flasks")), `BaseType == "Basalt Flask"`) || !strings.Contains(c.tinctures(c.section(options, "tinctures")), `BaseType == "Oakbranch Tincture"`) {
		t.Fatal("global selections not inherited")
	}
	options = Object{"flasks": Object{"utilityFlasks": []any{}}, "tinctures": Object{"baseTypes": []any{}}}
	if strings.Contains(c.flasks(c.section(options, "flasks")), "basalt_flask.mp3") || strings.Contains(c.tinctures(c.section(options, "tinctures")), "oakbranch_tincture.mp3") {
		t.Fatal("explicit empty filter selections must override global selections")
	}
}
