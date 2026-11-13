package basedata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratePreservesSourceData(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join("internal", "data"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"BaseTypes.csv":  "Class,BaseType,DropLevel,Game:APS\nMap Fragments,Divine Vessel,12,\nStackable Currency,Coin of Skill,1,\nStackable Currency,Orb of Binding,1,\nWands,Synthetic Wand,4,1.2\n",
		"reference.json": `{"sounds":[]}`,
	} {
		if err := os.WriteFile(filepath.Join("internal", "data", name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Generate(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("internal/data/reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct{ BaseTypesByClass map[string][]string }
	if err := json.Unmarshal(b, &ref); err != nil {
		t.Fatal(err)
	}
	if got := ref.BaseTypesByClass["Map Fragments"]; len(got) != 1 || got[0] != "Divine Vessel" {
		t.Fatal("upstream class was overridden", got)
	}
	if got := ref.BaseTypesByClass["Stackable Currency"]; len(got) != 2 || got[0] != "Coin of Skill" || got[1] != "Orb of Binding" {
		t.Fatal("upstream names were supplemented, renamed or removed", got)
	}
	b, err = os.ReadFile("internal/data/weapons.json")
	if err != nil {
		t.Fatal(err)
	}
	var weapons []struct {
		BaseType       string
		DropLevel, APS float64
	}
	if err := json.Unmarshal(b, &weapons); err != nil {
		t.Fatal(err)
	}
	if len(weapons) != 1 || weapons[0].BaseType != "Synthetic Wand" || weapons[0].DropLevel != 4 || weapons[0].APS != 1.2 {
		t.Fatal("upstream weapon metadata changed", weapons)
	}
}

func TestMinimumDropLevelsUsesActualEarliestKnownValue(t *testing.T) {
	levels := minimumDropLevels([]map[string]string{
		{"Class": "Wands", "BaseType": "Driftwood Wand", "DropLevel": "1"},
		{"Class": "Wands", "BaseType": "Driftwood Wand", "DropLevel": "12"},
		{"Class": "Wands", "BaseType": "Goat's Horn", "DropLevel": "8"},
		{"Class": "Wands", "BaseType": "Goat's Horn", "DropLevel": "6"},
		{"Class": "Wands", "BaseType": "Unknown", "DropLevel": ""},
		{"Class": "Wands", "BaseType": "Invalid", "DropLevel": "-1"},
		{"Class": "Wands", "BaseType": "Zero", "DropLevel": "0"},
	})
	if levels["Driftwood Wand"] != 1 || levels["Goat's Horn"] != 6 {
		t.Fatal("did not retain the actual minimum drop level", levels)
	}
	if _, exists := levels["Unknown"]; exists {
		t.Fatal("missing level must not be invented")
	}
	if _, exists := levels["Invalid"]; exists {
		t.Fatal("invalid level must not be shown")
	}
	if level, exists := levels["Zero"]; !exists || level != 0 {
		t.Fatal("a recorded zero must not be treated as missing")
	}
}
