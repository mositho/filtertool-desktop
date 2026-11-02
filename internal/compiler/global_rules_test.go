package compiler

import (
	"strings"
	"testing"
)

func TestGlobalRuleTuningOnlyUsesMaximumAndSize(t *testing.T) {
	c := New(Object{"early": Object{"rules": Object{"armour2": Object{"maxAreaLevel": float64(17), "size": float64(31), "enabled": false, "style": Object{"text": "#123456"}, "sound": Object{"soundId": float64(8)}, "icon": Object{"iconShape": "Star"}}}}}, nil, "")
	out := c.earlySockets(c.section(Object{}, "early"))
	if !strings.Contains(out, "AreaLevel <= 17") || !strings.Contains(out, "SetFontSize 31") {
		t.Fatal("global maximum/size missing", out)
	}
	if strings.Contains(out, "Hide\n") || strings.Contains(out, "SetTextColor 18 52 86") || strings.Contains(out, "AlertSound") || strings.Contains(out, "MinimapIcon") {
		t.Fatal("obsolete overrides still affect rules", out)
	}
	entry := Obj(Obj(Obj(c.Defaults["early"])["rules"])["armour2"])
	if len(entry) != 2 {
		t.Fatal("obsolete global values not discarded", entry)
	}
}

func TestFilterRuleOverridesAreDiscardedInsteadOfMigrated(t *testing.T) {
	c := New(Object{"links": Object{"rules": Object{"3generic": Object{"maxAreaLevel": float64(20), "size": float64(29)}}}}, nil, "")
	options := Object{"links": Object{"rules": Object{"3generic": Object{"maxAreaLevel": float64(2), "size": float64(22), "enabled": false}}}}
	base := c.links(Object{}, Object{})
	if got := c.links(Object{}, options); got != base {
		t.Fatal("per-filter rules still override global tuning")
	}
	if !strings.Contains(base, "SetFontSize 29") {
		t.Fatal("global link size missing")
	}
	cfg := Object{"buildSpecificOptions": options, "localDefaults": Object{"early": Object{"rules": Object{"armour2": Object{"size": float64(30)}}}}, "currency": Object{"items": Object{}}}
	DiscardFilterRuleOverrides(cfg)
	if _, exists := Obj(Obj(options["links"]))["rules"]; exists {
		t.Fatal("filter override retained")
	}
	if _, exists := Obj(Obj(Obj(cfg["localDefaults"])["early"]))["rules"]; exists {
		t.Fatal("filter-local snapshot override retained")
	}
}
