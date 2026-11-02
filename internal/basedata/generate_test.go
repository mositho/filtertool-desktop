package basedata

import "testing"

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
